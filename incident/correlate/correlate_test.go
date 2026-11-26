package correlate

import (
	"testing"
	"time"

	"github.com/anil-debug/rackpilot-ai/incident/signal"
)

func ts() time.Time { return time.Now().UTC() }

func sig(metric string, src signal.Source, sev signal.Severity, st signal.SignalType, comp string) signal.Signal {
	return signal.Signal{
		SignalID:          metric + "-" + comp,
		Source:            src,
		SignalType:        st,
		Severity:          sev,
		Timestamp:         ts(),
		Component:         comp,
		MetricName:        metric,
		MetricValue:       2,
		ObservedFact:      "test observation",
		EvidenceReference: "test",
	}
}

func TestHealthyNoIncident(t *testing.T) {
	e := New(5 * time.Minute)
	res := e.Evaluate(nil, ts())
	for _, r := range res {
		if r.Active {
			t.Fatalf("unexpected %s", r.RuleID)
		}
	}
}

func TestCoolingIncident(t *testing.T) {
	e := New(5 * time.Minute)
	signals := []signal.Signal{
		sig("rackpilot_bmc_fan_health_status", signal.SourceBMC, signal.SeverityCritical, signal.TypeHealthStatus, "fan-0"),
		sig("rackpilot_bmc_cpu_temperature_celsius", signal.SourceBMC, signal.SeverityWarning, signal.TypeThreshold, "cpu-0"),
		sig("rackpilot_gpu_temperature_celsius", signal.SourceGPU, signal.SeverityWarning, signal.TypeThreshold, "gpu-0"),
	}
	res := e.Evaluate(signals, ts())
	if !anyActive(res, "hardware_cooling") {
		t.Fatal("expected cooling incident")
	}
}

func TestPSUIncident(t *testing.T) {
	e := New(5 * time.Minute)
	signals := []signal.Signal{
		sig("rackpilot_bmc_psu_health_status", signal.SourceBMC, signal.SeverityCritical, signal.TypeHealthStatus, "psu-0"),
		sig("rackpilot_bmc_system_health_status", signal.SourceBMC, signal.SeverityCritical, signal.TypeHealthStatus, "srv"),
	}
	res := e.Evaluate(signals, ts())
	if !anyActive(res, "power_supply") {
		t.Fatal("expected psu incident")
	}
}

func TestKubernetesWorkload(t *testing.T) {
	e := New(5 * time.Minute)
	s := sig("kube_pod_container_status_restarts_total", signal.SourceKubernetes, signal.SeverityWarning, signal.TypeRateIncrease, "pod1")
	s.Namespace = "rackpilot"
	s.Pod = "pod1"
	avail := sig("kube_deployment_status_replicas_available", signal.SourceKubernetes, signal.SeverityCritical, signal.TypeAvailabilityGap, "rackpilot-demo-api")
	avail.Namespace = "rackpilot"
	res := e.Evaluate([]signal.Signal{s, avail}, ts())
	if !anyActive(res, "kubernetes_workload") {
		t.Fatal("expected k8s workload incident")
	}
}

func TestMemoryPressure(t *testing.T) {
	e := New(5 * time.Minute)
	signals := []signal.Signal{
		sig("rackpilot_bmc_memory_errors_total", signal.SourceBMC, signal.SeverityWarning, signal.TypeCounterIncrease, "dimm-0"),
		{
			SignalID: "k8s-mem", Source: signal.SourceKubernetes, SignalType: signal.TypeResourcePressure,
			Severity: signal.SeverityWarning, Timestamp: ts(), Namespace: "rackpilot", Component: "api",
			MetricName: "container_memory_working_set_bytes", MetricValue: 0.9, ObservedFact: "mem pressure", EvidenceReference: "test",
		},
	}
	res := e.Evaluate(signals, ts())
	if !anyActive(res, "memory_pressure") {
		t.Fatal("expected memory incident")
	}
}

func TestUnrelatedNoCooling(t *testing.T) {
	e := New(5 * time.Minute)
	res := e.Evaluate([]signal.Signal{
		sig("rackpilot_gpu_utilization_percent", signal.SourceGPU, signal.SeverityWarning, signal.TypeThreshold, "gpu-0"),
	}, ts())
	if anyActive(res, "hardware_cooling") {
		t.Fatal("should not correlate cooling")
	}
}

func TestOutOfOrderTimestamps(t *testing.T) {
	e := New(5 * time.Minute)
	older := sig("rackpilot_bmc_fan_health_status", signal.SourceBMC, signal.SeverityCritical, signal.TypeHealthStatus, "fan-0")
	older.Timestamp = time.Now().Add(-10 * time.Minute)
	newer := sig("rackpilot_bmc_cpu_temperature_celsius", signal.SourceBMC, signal.SeverityWarning, signal.TypeThreshold, "cpu-0")
	res := e.Evaluate([]signal.Signal{older, newer}, ts())
	// fan still within 5m window? older is 10m ago - should NOT fire
	if anyActive(res, "hardware_cooling") {
		t.Fatal("old signal outside window should not open incident")
	}
}

func anyActive(res []Result, rule string) bool {
	for _, r := range res {
		if r.RuleID == rule && r.Active {
			return true
		}
	}
	return false
}
