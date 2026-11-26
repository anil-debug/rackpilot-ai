package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anil-debug/rackpilot-ai/incident/collector/prometheus"
	"github.com/anil-debug/rackpilot-ai/incident/config"
	"github.com/anil-debug/rackpilot-ai/incident/correlate"
	"github.com/anil-debug/rackpilot-ai/incident/ingest"
	"github.com/anil-debug/rackpilot-ai/incident/metrics"
	"github.com/anil-debug/rackpilot-ai/incident/store"
	promreg "github.com/prometheus/client_golang/prometheus"
)

func TestEndToEndCoolingFromPrometheus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("query")
		resp := map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "vector",
				"result":     []any{},
			},
		}
		now := float64(time.Now().Unix())
		switch q {
		case ingest.QBMCFanHealth:
			resp["data"] = map[string]any{"resultType": "vector", "result": []any{sample("rackpilot_bmc_fan_health_status", map[string]string{"fan_id": "fan-0"}, "2", now)}}
		case ingest.QBMCCPUTemp:
			resp["data"] = map[string]any{"resultType": "vector", "result": []any{sample("rackpilot_bmc_cpu_temperature_celsius", map[string]string{"cpu_id": "cpu-0"}, "85", now)}}
		case ingest.QGPUTemp:
			resp["data"] = map[string]any{"resultType": "vector", "result": []any{sample("rackpilot_gpu_temperature_celsius", map[string]string{"gpu_id": "gpu-0"}, "90", now)}}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.PrometheusURL = srv.URL
	cfg.CorrelationWindow = 5 * time.Minute
	cfg.RecoveryWindow = time.Minute

	client := prometheus.NewHTTPClient(cfg.PrometheusURL)
	collector := ingest.New(client, cfg)
	correlator := correlate.New(cfg.CorrelationWindow)
	mem := store.NewMemory()
	reg := promreg.NewRegistry()
	mreg := metrics.NewRegistrar(reg)
	runner := NewRunner(cfg, collector, correlator, mem, mreg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.Run(ctx)
	time.Sleep(100 * time.Millisecond)
	runner.tick(ctx)
	active := mem.Active()
	if len(active) == 0 {
		t.Fatal("expected cooling incident from mocked prom")
	}
	if active[0].CorrelationRule != "hardware_cooling" {
		t.Fatalf("rule=%s", active[0].CorrelationRule)
	}
}

func sample(name string, labels map[string]string, val string, ts float64) map[string]any {
	labels["__name__"] = name
	return map[string]any{
		"metric": labels,
		"value":  []any{ts, val},
	}
}
