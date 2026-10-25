package nodra

import "errors"

// MetricsSink is the dependency-free bridge for exporting Nodra action
// metrics. Adapters can translate one snapshot into Prometheus gauges,
// OpenTelemetry instruments, logs, or an application's existing telemetry
// system without making the core package depend on a telemetry SDK.
type MetricsSink interface {
	RecordActionMetrics(name string, metrics ActionMetrics)
}

// ExportMetrics records the current aggregate metrics for one action. It takes
// a snapshot under the action lock and invokes the sink afterward, so sinks may
// reenter the action or perform blocking export work. A nil sink is rejected.
func (a *AsyncAction[T, I, R]) ExportMetrics(sink MetricsSink) error {
	if a == nil || sink == nil {
		return errors.New("nodra: nil metrics sink")
	}
	sink.RecordActionMetrics(a.name, a.Metrics())
	return nil
}
