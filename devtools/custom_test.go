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
