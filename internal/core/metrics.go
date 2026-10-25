package core

import (
	"errors"
	"sync"
)

// MetricsSink is the dependency-free bridge for exporting Nodra action metrics. Adapters can translate one snapshot into Prometheus gauges, OpenTelemetry instruments, logs, or an application's existing telemetry system without making the core package depend on a telemetry SDK.
type MetricsSink interface {
	RecordActionMetrics(name string, metrics ActionMetrics)
}

// ExportMetrics records the current aggregate metrics for one action. It takes a snapshot under the action lock and invokes the sink afterward, so sinks may reenter the action or perform blocking export work. A nil sink is rejected.
func (a *AsyncAction[T, I, R]) ExportMetrics(sink MetricsSink) error {
	if a == nil || sink == nil {
		return errors.New("nodra: nil metrics sink")
	}
	sink.RecordActionMetrics(a.name, a.Metrics())
	return nil
}

// SubscribeMetrics observes aggregate metrics after each terminal task. The callback runs outside the action lock and is protected from panics so an exporter cannot strand a task or terminate the worker goroutine. The returned function is idempotent.
func (a *AsyncAction[T, I, R]) SubscribeMetrics(fn func(ActionMetrics)) func() {
	if a == nil || fn == nil {
		return func() {}
	}
	a.mu.Lock()
	if a.metricsListeners == nil {
		a.metricsListeners = make(map[uint64]func(ActionMetrics))
	}
	a.nextMetricsID++
	id := a.nextMetricsID
	a.metricsListeners[id] = fn
	a.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			a.mu.Lock()
			delete(a.metricsListeners, id)
			a.mu.Unlock()
		})
	}
}
