package ingest

import (
	"context"
	"fmt"

	prom "github.com/anil-debug/rackpilot-ai/incident/collector/prometheus"
	"github.com/anil-debug/rackpilot-ai/incident/config"
	"github.com/anil-debug/rackpilot-ai/incident/signal"
)

// PromQL queries — aligned with metrics that exist in this repo; absent series yield no signals.
const (
	QGPUHealth       = `rackpilot_gpu_health_status`
	QGPUTemp         = `rackpilot_gpu_temperature_celsius`
	QGPUUtil         = `rackpilot_gpu_utilization_percent`
	QGPUMemUsed      = `rackpilot_gpu_memory_used_bytes`
	QBMCCPUTemp      = `rackpilot_bmc_cpu_temperature_celsius`
	QBMCFanHealth    = `rackpilot_bmc_fan_health_status`
	QBMCPSUHealth    = `rackpilot_bmc_psu_health_status`
	QBMCSystemHealth = `rackpilot_bmc_system_health_status`
	QBMCMemErrors    = `increase(rackpilot_bmc_memory_errors_total[5m]) > 0`
	QK8sRestarts     = `increase(kube_pod_container_status_restarts_total{namespace="rackpilot"}[5m]) > 0`
	QK8sAvail        = `kube_deployment_status_replicas_available{namespace="rackpilot"}`
	QK8sDesired      = `kube_deployment_spec_replicas{namespace="rackpilot"}`
	QK8sCPURate      = `rate(container_cpu_usage_seconds_total{namespace="rackpilot",container!="",image!=""}[5m])`
	QK8sMemWS        = `container_memory_working_set_bytes{namespace="rackpilot",container!="",image!=""}`
	QK8sMemLimit     = `container_spec_memory_limit_bytes{namespace="rackpilot",container!="",image!=""} > 0`
)

// Collector loads Prometheus samples and emits normalized signals.
type Collector struct {
	client prom.Client
	cfg    config.Config
}

func New(client prom.Client, cfg config.Config) *Collector {
	return &Collector{client: client, cfg: cfg}
}

func (c *Collector) Collect(ctx context.Context) ([]signal.Signal, error) {
	type job struct {
		query string
		fn    func([]prom.Sample) []signal.Signal
	}
	jobs := []job{
		{QGPUHealth, c.gpuHealth},
		{QGPUTemp, c.gpuTemp},
		{QBMCCPUTemp, c.bmcCPUTemp},
		{QBMCFanHealth, c.bmcFanHealth},
		{QBMCPSUHealth, c.bmcPSUHealth},
		{QBMCSystemHealth, c.bmcSystemHealth},
		{QBMCMemErrors, c.bmcMemErrors},
		{QK8sRestarts, c.k8sRestarts},
	}
	var out []signal.Signal
	for _, j := range jobs {
		samples, err := c.client.QueryInstant(ctx, j.query)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", j.query, err)
		}
		if len(samples) == 0 {
			continue
		}
		out = append(out, j.fn(samples)...)
	}
	// availability gaps need paired query
	avail, err := c.client.QueryInstant(ctx, QK8sAvail)
	if err == nil && len(avail) > 0 {
		desired, err2 := c.client.QueryInstant(ctx, QK8sDesired)
		if err2 == nil {
			out = append(out, c.k8sAvailabilityGap(avail, desired)...)
		}
	}
	memWS, err := c.client.QueryInstant(ctx, QK8sMemWS)
	if err == nil && len(memWS) > 0 {
		out = append(out, c.k8sMemPressure(ctx, memWS)...)
	}
	return out, nil
}

func (c *Collector) gpuHealth(samples []prom.Sample) []signal.Signal {
	return healthSignals(samples, signal.SourceGPU, "rackpilot_gpu_health_status", "gpu_id", c.cfg.HealthCritical)
}

func (c *Collector) gpuTemp(samples []prom.Sample) []signal.Signal {
	var out []signal.Signal
	for _, s := range samples {
		if s.Value < c.cfg.GPUTempWarnC {
			continue
		}
		id := labelFirst(s.Metric, "gpu_id", "model")
		out = append(out, signal.Signal{
			SignalID:          "gpu-temp-" + id,
			Source:            signal.SourceGPU,
			SignalType:        signal.TypeThreshold,
			Severity:          severityFromThreshold(s.Value, c.cfg.GPUTempWarnC, c.cfg.GPUTempWarnC+15),
			Timestamp:         s.Time,
			Component:         id,
			MetricName:        "rackpilot_gpu_temperature_celsius",
			MetricValue:       s.Value,
			Labels:            s.Metric,
			EvidenceReference: "promql:" + QGPUTemp,
			ObservedFact:      fmt.Sprintf("GPU temperature %.1f°C exceeds warn threshold %.1f°C", s.Value, c.cfg.GPUTempWarnC),
		})
	}
	return out
}

func (c *Collector) bmcCPUTemp(samples []prom.Sample) []signal.Signal {
	var out []signal.Signal
	for _, s := range samples {
		if s.Value < c.cfg.CPUTempWarnC {
			continue
		}
		id := labelFirst(s.Metric, "cpu_id", "server_id")
		out = append(out, signal.Signal{
			SignalID:          "bmc-cpu-temp-" + id,
			Source:            signal.SourceBMC,
			SignalType:        signal.TypeThreshold,
			Severity:          severityFromThreshold(s.Value, c.cfg.CPUTempWarnC, c.cfg.CPUTempWarnC+20),
			Timestamp:         s.Time,
			Component:         id,
			MetricName:        "rackpilot_bmc_cpu_temperature_celsius",
			MetricValue:       s.Value,
			Labels:            s.Metric,
			EvidenceReference: "promql:" + QBMCCPUTemp,
			ObservedFact:      fmt.Sprintf("BMC CPU temperature %.1f°C elevated", s.Value),
		})
	}
	return out
}

func (c *Collector) bmcFanHealth(samples []prom.Sample) []signal.Signal {
	return healthSignals(samples, signal.SourceBMC, "rackpilot_bmc_fan_health_status", "fan_id", c.cfg.HealthCritical)
}

func (c *Collector) bmcPSUHealth(samples []prom.Sample) []signal.Signal {
	return healthSignals(samples, signal.SourceBMC, "rackpilot_bmc_psu_health_status", "psu_id", c.cfg.HealthCritical)
}

func (c *Collector) bmcSystemHealth(samples []prom.Sample) []signal.Signal {
	return healthSignals(samples, signal.SourceBMC, "rackpilot_bmc_system_health_status", "server_id", c.cfg.HealthCritical)
}

func (c *Collector) bmcMemErrors(samples []prom.Sample) []signal.Signal {
	var out []signal.Signal
	for _, s := range samples {
		if s.Value <= 0 {
			continue
		}
		id := labelFirst(s.Metric, "dimm_id", "server_id")
		out = append(out, signal.Signal{
			SignalID:          "bmc-mem-err-" + id,
			Source:            signal.SourceBMC,
			SignalType:        signal.TypeCounterIncrease,
			Severity:          signal.SeverityWarning,
			Timestamp:         s.Time,
			Component:         id,
			MetricName:        "rackpilot_bmc_memory_errors_total",
			MetricValue:       s.Value,
			Labels:            s.Metric,
			EvidenceReference: "promql:" + QBMCMemErrors,
			ObservedFact:      fmt.Sprintf("BMC memory errors increased by %.0f in 5m window for %s", s.Value, id),
		})
	}
	return out
}

func (c *Collector) k8sRestarts(samples []prom.Sample) []signal.Signal {
	var out []signal.Signal
	for _, s := range samples {
		if s.Value <= 0 {
			continue
		}
		pod := s.Metric["pod"]
		ns := s.Metric["namespace"]
		out = append(out, signal.Signal{
			SignalID:          "k8s-restart-" + ns + "/" + pod,
			Source:            signal.SourceKubernetes,
			SignalType:        signal.TypeRateIncrease,
			Severity:          signal.SeverityWarning,
			Timestamp:         s.Time,
			Namespace:         ns,
			Pod:               pod,
			Cluster:           s.Metric["cluster"],
			MetricName:        "kube_pod_container_status_restarts_total",
			MetricValue:       s.Value,
			Labels:            s.Metric,
			EvidenceReference: "promql:" + QK8sRestarts,
			ObservedFact:      fmt.Sprintf("Pod %s/%s restart increase %.0f in 5m window", ns, pod, s.Value),
		})
	}
	return out
}

func (c *Collector) k8sDeployAvail(samples []prom.Sample) []signal.Signal {
	return nil // used in pairing
}

func (c *Collector) k8sDeployDesired(samples []prom.Sample) []signal.Signal {
	return nil
}

func (c *Collector) k8sAvailabilityGap(avail, desired []prom.Sample) []signal.Signal {
	desiredByDep := map[string]float64{}
	for _, s := range desired {
		dep := s.Metric["deployment"]
		if dep != "" {
			desiredByDep[dep] = s.Value
		}
	}
	var out []signal.Signal
	for _, s := range avail {
		dep := s.Metric["deployment"]
		want, ok := desiredByDep[dep]
		if !ok || s.Value >= want {
			continue
		}
		out = append(out, signal.Signal{
			SignalID:          "k8s-avail-" + dep,
			Source:            signal.SourceKubernetes,
			SignalType:        signal.TypeAvailabilityGap,
			Severity:          signal.SeverityCritical,
			Timestamp:         s.Time,
			Namespace:         s.Metric["namespace"],
			Component:         dep,
			Cluster:           s.Metric["cluster"],
			MetricName:        "kube_deployment_status_replicas_available",
			MetricValue:       s.Value,
			Labels:            s.Metric,
			EvidenceReference: "promql:" + QK8sAvail + " vs " + QK8sDesired,
			ObservedFact:      fmt.Sprintf("Deployment %s available %.0f < desired %.0f", dep, s.Value, want),
		})
	}
	return out
}

func (c *Collector) k8sMemPressure(ctx context.Context, samples []prom.Sample) []signal.Signal {
	limits, err := c.client.QueryInstant(ctx, QK8sMemLimit)
	if err != nil || len(limits) == 0 {
		return nil
	}
	limitBy := map[string]float64{}
	for _, l := range limits {
		key := l.Metric["namespace"] + "/" + l.Metric["pod"] + "/" + l.Metric["container"]
		limitBy[key] = l.Value
	}
	var out []signal.Signal
	for _, s := range samples {
		key := s.Metric["namespace"] + "/" + s.Metric["pod"] + "/" + s.Metric["container"]
		lim, ok := limitBy[key]
		if !ok || lim <= 0 {
			continue
		}
		ratio := s.Value / lim
		if ratio < c.cfg.MemPressureRatio {
			continue
		}
		out = append(out, signal.Signal{
			SignalID:          "k8s-mem-" + key,
			Source:            signal.SourceKubernetes,
			SignalType:        signal.TypeResourcePressure,
			Severity:          signal.SeverityWarning,
			Timestamp:         s.Time,
			Namespace:         s.Metric["namespace"],
			Pod:               s.Metric["pod"],
			Component:         s.Metric["container"],
			Cluster:           s.Metric["cluster"],
			MetricName:        "container_memory_working_set_bytes",
			MetricValue:       ratio,
			Labels:            s.Metric,
			EvidenceReference: "promql:" + QK8sMemWS,
			ObservedFact:      fmt.Sprintf("Container %s memory at %.0f%% of limit", key, ratio*100),
		})
	}
	return out
}

func healthSignals(samples []prom.Sample, src signal.Source, metric, labelKey string, critical float64) []signal.Signal {
	var out []signal.Signal
	for _, s := range samples {
		sev := signal.SeverityInfo
		if s.Value >= 1 {
			sev = signal.SeverityWarning
		}
		if s.Value >= critical {
			sev = signal.SeverityCritical
		}
		if sev == signal.SeverityInfo {
			continue
		}
		id := labelFirst(s.Metric, labelKey, "server_id")
		out = append(out, signal.Signal{
			SignalID:          fmt.Sprintf("%s-health-%s", src, id),
			Source:            src,
			SignalType:        signal.TypeHealthStatus,
			Severity:          sev,
			Timestamp:         s.Time,
			Component:         id,
			MetricName:        metric,
			MetricValue:       s.Value,
			Labels:            s.Metric,
			EvidenceReference: "promql:" + metric,
			ObservedFact:      fmt.Sprintf("%s health status %.0f for %s", metric, s.Value, id),
		})
	}
	return out
}

func labelFirst(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := m[k]; v != "" {
			return v
		}
	}
	return "unknown"
}

func severityFromThreshold(v, warn, crit float64) signal.Severity {
	if v >= crit {
		return signal.SeverityCritical
	}
	if v >= warn {
		return signal.SeverityWarning
	}
	return signal.SeverityInfo
}
