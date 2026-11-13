package devtools

import (
	"encoding/json"
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
