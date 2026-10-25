package nodra

import (
	"context"
	"testing"
)

type metricsSink struct {
	name    string
	metrics ActionMetrics
}

func (s *metricsSink) RecordActionMetrics(name string, metrics ActionMetrics) {
	s.name, s.metrics = name, metrics
}

func TestExportMetricsUsesDependencyFreeSink(t *testing.T) {
	store := New(0)
	action, err := NewAsyncAction(store, "load", func(context.Context, int, func(float64)) (int, error) { return 4, nil }, func(value *int, result int) error { *value = result; return nil })
	if err != nil {
		t.Fatal(err)
	}
	task, err := action.Run(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := task.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	sink := new(metricsSink)
	if err := action.ExportMetrics(sink); err != nil {
		t.Fatal(err)
	}
	if sink.name != "load" || sink.metrics.Completed != 1 || sink.metrics.Succeeded != 1 {
		t.Fatalf("sink = %+v", sink)
	}
	if err := action.ExportMetrics(nil); err == nil {
		t.Fatal("nil sink accepted")
	}
	_ = action.Close(context.Background())
}
