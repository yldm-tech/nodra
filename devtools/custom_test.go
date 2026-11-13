package devtools

import (
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yldm-tech/nodra"
)

func TestCustomEventsAreRedactedExportedAndIndexed(t *testing.T) {
	i := New(Options{RedactMetadata: func(_ string, metadata Metadata) (Metadata, error) {
		metadata.Name = "masked"
		metadata.Key = "safe"
		metadata.Error = "safe error"
		return metadata, nil
	}})
	defer func() { _ = i.Close() }()
	store := nodra.New(0)
	mustAttach(t, i, "store", store)
	if err := i.Record("store", CustomEvent{Kind: "network", Name: "private request", Key: "private-token", Error: "private failure", Data: json.RawMessage(`{"status":500}`), At: time.Now()}); err != nil {
		t.Fatal(err)
	}
	snapshot := i.Snapshot()
	if snapshot.Events[len(snapshot.Events)-1].Name != "masked" || snapshot.Events[len(snapshot.Events)-1].Key != "safe" || len(i.Search("network safe", 10)) != 1 {
		t.Fatalf("custom event: %+v search=%v", snapshot.Events[len(snapshot.Events)-1], i.Search("masked safe", 10))
	}
	if string(snapshot.Events[len(snapshot.Events)-1].Data) != `{"status":500}` {
		t.Fatal("custom data lost")
	}
}

func TestSearchRequiresEveryTermIncludingAbsentFirstTerm(t *testing.T) {
	i := New(Options{})
	defer func() { _ = i.Close() }()
	mustAttach(t, i, "store", nodra.New(0))
	mustUpdate(t, i.Record("store", CustomEvent{Kind: "network", Name: "request", Data: json.RawMessage(`{}`)}))
	if got := i.Search("network request", 10); len(got) != 1 {
		t.Fatalf("matching terms: %v", got)
	}
	for _, query := range []string{"missing", "network missing", "missing network", "missing other network", "missing missing network"} {
		if got := i.Search(query, 10); len(got) != 0 {
			t.Errorf("Search(%q) = %v, want no matches", query, got)
		}
	}
}

func TestRecordRedactsCustomMetadataOnce(t *testing.T) {
	var calls atomic.Int32
	var counting atomic.Bool
	i := New(Options{RedactMetadata: func(_ string, m Metadata) (Metadata, error) {
		if counting.Load() {
			calls.Add(1)
		}
		if m.Name != "" {
			m.Name = "h(" + m.Name + ")"
		}
		return m, nil
	}})
	defer func() { _ = i.Close() }()
	store := nodra.New(counterState{})
	mustAttach(t, i, "counter", store)
	mustUpdate(t, store.Do("secret", func(v *counterState) error { v.Count++; return nil }))
	counting.Store(true)
	mustUpdate(t, i.Record("counter", CustomEvent{Kind: "custom", Name: "secret", Data: json.RawMessage(`{}`)}))
	counting.Store(false)
	if n := calls.Load(); n != 1 {
		t.Fatalf("RedactMetadata calls for one Record = %d, want 1", n)
	}
	var mutation, custom string
	for _, event := range i.Snapshot().Events {
		switch event.Kind {
		case "mutation":
			mutation = event.Name
		case "custom":
			custom = event.Name
		}
	}
	if mutation != "h(secret)" || custom != mutation {
		t.Fatalf("mutation name %q, custom name %q", mutation, custom)
	}
	if got := i.Search("custom h(secret)", 10); len(got) != 1 {
		t.Fatalf("custom event not indexed under its redacted name: %v", got)
	}
}
