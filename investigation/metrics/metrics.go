package metrics

import "github.com/prometheus/client_golang/prometheus"

const namespace = "rackpilot"

// Registrar exposes root-cause investigator metrics.
type Registrar struct {
	Requests         prometheus.Counter
	Successes        prometheus.Counter
	Failures         prometheus.Counter
	EvidenceDuration prometheus.Histogram
	EvidenceErrors   *prometheus.CounterVec
	LLMErrors        prometheus.Counter
	Duration         prometheus.Histogram
}

func NewRegistrar(reg prometheus.Registerer) *Registrar {
	r := &Registrar{
		Requests: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "investigation_requests_total",
			Help:      "Investigation API requests.",
		}),
		Successes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "investigation_success_total",
			Help:      "Completed or partial investigations with evidence.",
		}),
		Failures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "investigation_failure_total",
			Help:      "Investigations that failed.",
		}),
		EvidenceDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "evidence_collection_duration_seconds",
			Help:      "Time to collect evidence from providers.",
			Buckets:   prometheus.DefBuckets,
		}),
		EvidenceErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "evidence_collection_errors_total",
			Help:      "Evidence provider failures.",
		}, []string{"provider"}),
		LLMErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "investigation_llm_errors_total",
			Help:      "LLM provider failures.",
		}),
		Duration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "investigation_processing_duration_seconds",
			Help:      "End-to-end investigation duration.",
			Buckets:   prometheus.DefBuckets,
		}),
	}
	reg.MustRegister(
		r.Requests, r.Successes, r.Failures, r.EvidenceDuration,
		r.EvidenceErrors, r.LLMErrors, r.Duration,
	)
	return r
}
