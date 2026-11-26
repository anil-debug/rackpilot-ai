package metrics

import "github.com/prometheus/client_golang/prometheus"

// RegisterStaticMode exposes a single info-style gauge for static fixture mode.
func RegisterStaticMode(reg prometheus.Registerer) {
	g := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "bmc_fixture_mode_active",
		Help:      "1 when serving DMTF static mockup fixtures (not live synthetic telemetry).",
	})
	g.Set(1)
	reg.MustRegister(g)
}
