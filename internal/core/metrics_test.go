package core

import (
	"context"
	"errors"
	"testing"
	"time"
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

func TestSubscribeMetricsPublishesTerminalAggregatesAndSurvivesPanics(t *testing.T) {
	store := New(0)
	action, err := NewAsyncAction(store, "load", func(context.Context, int, func(float64)) (int, error) { return 1, nil }, func(value *int, result int) error { *value = result; return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = action.Close(context.Background()) }()
	updates := make(chan ActionMetrics, 1)
	stop := action.SubscribeMetrics(func(metrics ActionMetrics) { updates <- metrics })
	defer stop()
	_ = action.SubscribeMetrics(func(ActionMetrics) { panic("exporter") })
	task, err := action.Run(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := task.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case metrics := <-updates:
		if metrics.Completed != 1 || metrics.Succeeded != 1 {
			t.Fatalf("metrics = %+v", metrics)
		}
	case <-time.After(time.Second):
		t.Fatal("metrics subscriber did not receive terminal aggregate")
	}
}

func TestActionMetricsDoNotCountSupersededRunsAsFailed(t *testing.T) {
	gate, entered := make(chan struct{}), make(chan struct{}, 2)
	action, err := NewAsyncAction(New(0), "search", func(ctx context.Context, query int, _ func(float64)) (int, error) {
		entered <- struct{}{}
		if query == 1 {
			<-ctx.Done()
			return 0, ctx.Err()
		}
		<-gate
		return query, nil
	}, func(s *int, v int) error { *s = v; return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = action.Close(context.Background()) }()
	first, err := action.Run(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	second, err := action.Run(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Wait(context.Background()); !errors.Is(err, ErrSuperseded) || first.Status().Get().Phase != TaskSuperseded {
		t.Fatalf("first run = %v, phase %v", err, first.Status().Get().Phase)
	}
	<-entered
	close(gate)
	if err := second.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	metrics := action.Metrics()
	if metrics.Accepted != 2 || metrics.Completed != 2 || metrics.Succeeded != 1 || metrics.Failed != 0 || metrics.Canceled != 0 || metrics.Superseded != 1 {
		t.Fatalf("metrics = %+v", metrics)
	}
}
