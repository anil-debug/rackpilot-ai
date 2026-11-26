package metrics

import (
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/domain"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/engine"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "rackpilot"

// Registrar wires simulator gauges into a Prometheus registry.
type Registrar struct {
	temperature *prometheus.GaugeVec
	utilization *prometheus.GaugeVec
	memUsed     *prometheus.GaugeVec
	memTotal    *prometheus.GaugeVec
	power       *prometheus.GaugeVec
	health      *prometheus.GaugeVec
	eng         *engine.Engine
}

// NewRegistrar creates and registers metrics on reg.
func NewRegistrar(reg prometheus.Registerer, eng *engine.Engine) *Registrar {
	r := &Registrar{eng: eng}
	r.temperature = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "gpu_temperature_celsius",
		Help:      "Synthetic simulated GPU temperature in Celsius (not physical hardware).",
	}, []string{"gpu_id", "model"})
	r.utilization = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "gpu_utilization_percent",
		Help:      "Synthetic simulated GPU utilization percentage.",
	}, []string{"gpu_id", "model"})
	r.memUsed = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "gpu_memory_used_bytes",
		Help:      "Synthetic simulated GPU memory used in bytes.",
	}, []string{"gpu_id", "model"})
	r.memTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "gpu_memory_total_bytes",
		Help:      "Synthetic simulated GPU memory capacity in bytes.",
	}, []string{"gpu_id", "model"})
	r.power = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "gpu_power_watts",
		Help:      "Synthetic simulated GPU power draw in watts.",
	}, []string{"gpu_id", "model"})
	r.health = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "gpu_health_status",
		Help:      "Derived synthetic health: 0=healthy, 1=warning, 2=critical.",
	}, []string{"gpu_id", "model"})

	reg.MustRegister(r.temperature, r.utilization, r.memUsed, r.memTotal, r.power, r.health)
	return r
}

// CollectFromEngine updates gauge values from the current engine snapshot.
func (r *Registrar) CollectFromEngine() {
	snap := r.eng.Snapshot()
	for _, t := range snap {
		labels := prometheus.Labels{"gpu_id": t.ID, "model": t.Model}
		r.temperature.With(labels).Set(t.TemperatureCelsius)
		r.utilization.With(labels).Set(t.UtilizationPercent)
		r.memUsed.With(labels).Set(float64(t.MemoryUsedBytes))
		r.memTotal.With(labels).Set(float64(t.MemoryTotalBytes))
		r.power.With(labels).Set(t.PowerWatts)
		r.health.With(labels).Set(float64(domain.HealthToCode(t.HealthStatus)))
	}
}
