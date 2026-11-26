package metrics

import "github.com/prometheus/client_golang/prometheus"

const namespace = "rackpilot"

// Registrar exposes incident engine metrics.
type Registrar struct {
	Active      prometheus.Gauge
	Created     prometheus.Counter
	Resolved    prometheus.Counter
	Signals     prometheus.Counter
	Evaluations prometheus.Counter
	Errors      prometheus.Counter
}

func NewRegistrar(reg prometheus.Registerer) *Registrar {
	r := &Registrar{
		Active: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "incidents_active",
			Help:      "Number of open or updated incidents.",
		}),
		Created: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "incidents_created_total",
			Help:      "Incidents opened.",
		}),
		Resolved: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "incidents_resolved_total",
			Help:      "Incidents resolved.",
		}),
		Signals: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "signals_processed_total",
			Help:      "Normalized signals processed.",
		}),
		Evaluations: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "correlation_evaluations_total",
			Help:      "Correlation evaluation ticks.",
		}),
		Errors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "correlation_errors_total",
			Help:      "Failed ingest or correlation cycles.",
		}),
	}
	reg.MustRegister(r.Active, r.Created, r.Resolved, r.Signals, r.Evaluations, r.Errors)
	return r
}
