package metrics

import (
	"testing"

	"github.com/anil-debug/rackpilot-ai/simulator/bmc/config"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/engine"
	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsGather(t *testing.T) {
	eng := engine.New(config.Default())
	reg := prometheus.NewRegistry()
	r := NewRegistrar(reg, eng)
	r.CollectFromEngine()
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, mf := range mfs {
		names[mf.GetName()] = true
	}
	want := []string{
		"rackpilot_bmc_cpu_temperature_celsius",
		"rackpilot_bmc_system_health_status",
		"rackpilot_bmc_fan_speed_rpm",
	}
	for _, w := range want {
		if !names[w] {
			t.Fatalf("missing %s", w)
		}
	}
}
