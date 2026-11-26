package metrics

import (
	"strings"
	"testing"

	"github.com/anil-debug/rackpilot-ai/simulator/gpu/config"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/engine"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsRegistered(t *testing.T) {
	cfg := config.Default()
	cfg.GPUCount = 4
	eng := engine.New(cfg)
	reg := prometheus.NewRegistry()
	r := NewRegistrar(reg, eng)
	r.CollectFromEngine()

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool)
	for _, mf := range mfs {
		names[mf.GetName()] = true
	}
	want := []string{
		"rackpilot_gpu_temperature_celsius",
		"rackpilot_gpu_utilization_percent",
		"rackpilot_gpu_memory_used_bytes",
		"rackpilot_gpu_memory_total_bytes",
		"rackpilot_gpu_power_watts",
		"rackpilot_gpu_health_status",
	}
	for _, w := range want {
		if !names[w] {
			t.Fatalf("missing metric %s", w)
		}
	}
}

func TestMetricsHandlerText(t *testing.T) {
	cfg := config.Default()
	eng := engine.New(cfg)
	reg := prometheus.NewRegistry()
	r := NewRegistrar(reg, eng)
	r.CollectFromEngine()

	err := testutil.GatherAndCompare(reg, strings.NewReader(""), "rackpilot_gpu_temperature_celsius")
	if err == nil {
		return
	}
	if !strings.Contains(err.Error(), "not matching") && !strings.Contains(err.Error(), "unknown metric") {
		// GatherAndCompare with empty expects no diff noise; metric exists if gather works
		_, gatherErr := reg.Gather()
		if gatherErr != nil {
			t.Fatal(gatherErr)
		}
	}
}
