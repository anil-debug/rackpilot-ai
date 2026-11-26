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
	inc "github.com/anil-debug/rackpilot-ai/incident/incident"
	"github.com/anil-debug/rackpilot-ai/incident/ingest"
	"github.com/anil-debug/rackpilot-ai/incident/metrics"
	"github.com/anil-debug/rackpilot-ai/incident/signal"
	"github.com/anil-debug/rackpilot-ai/incident/store"
	promreg "github.com/prometheus/client_golang/prometheus"
)

func coolingIncident(now time.Time) inc.Incident {
	return inc.Incident{
		IncidentID:      "inc-hardware_cooling-hardware_cooling-fan-0",
		Title:           "Hardware cooling incident",
		Status:          inc.StatusOpen,
		FirstSeen:       now,
		LastSeen:        now,
		CorrelationRule: "hardware_cooling",
	}
}

func TestIngestErrorDoesNotResolveIncident(t *testing.T) {
	cfg := config.Default()
	cfg.RecoveryWindow = time.Minute
	mem := store.NewMemory()
	now := time.Now().UTC()
	mem.ApplyTick([]correlate.Result{{
		RuleID: "hardware_cooling", DedupeKey: "hardware_cooling:fan-0", Active: true,
		Incident: coolingIncident(now),
	}}, now, cfg.RecoveryWindow)

	reg := promreg.NewRegistry()
	mreg := metrics.NewRegistrar(reg)
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "error", "error": "down"})
	}))
	t.Cleanup(failSrv.Close)

	runner := NewRunner(cfg, ingest.New(prometheus.NewHTTPClient(failSrv.URL), cfg), correlate.New(cfg.CorrelationWindow), mem, mreg)
	runner.tick(context.Background())

	if mem.CountActive() != 1 {
		t.Fatal("ingest error must not resolve active incidents")
	}
	_, errMsg := runner.LastRun()
	if errMsg == "" {
		t.Fatal("expected last error recorded")
	}
}

func TestDistinctResourcesDistinctIncidents(t *testing.T) {
	mem := store.NewMemory()
	now := time.Now().UTC()
	o, _, _ := mem.ApplyTick([]correlate.Result{
		{RuleID: "hardware_cooling", DedupeKey: "hardware_cooling:fan-0", Active: true, Incident: coolingIncident(now)},
		{RuleID: "hardware_cooling", DedupeKey: "hardware_cooling:fan-1", Active: true, Incident: coolingIncident(now)},
	}, now, time.Minute)
	if o != 2 || len(mem.Active()) != 2 {
		t.Fatalf("opens=%d active=%d", o, len(mem.Active()))
	}
}

func TestAllFourRulesViaCorrelate(t *testing.T) {
	e := correlate.New(5 * time.Minute)
	now := time.Now().UTC()
	ts := func() time.Time { return now }
	mk := func(metric string, sev signal.Severity, st signal.SignalType, comp string) signal.Signal {
		return signal.Signal{
			SignalID: metric + comp, MetricName: metric, Severity: sev, SignalType: st,
			Component: comp, Timestamp: ts(), ObservedFact: "fact", EvidenceReference: "test",
		}
	}
	cases := []struct {
		rule string
		sig  []signal.Signal
	}{
		{"hardware_cooling", []signal.Signal{
			mk("rackpilot_bmc_fan_health_status", signal.SeverityCritical, signal.TypeHealthStatus, "f0"),
		}},
		{"power_supply", []signal.Signal{
			mk("rackpilot_bmc_psu_health_status", signal.SeverityCritical, signal.TypeHealthStatus, "p0"),
		}},
		{"kubernetes_workload", []signal.Signal{
			{MetricName: "kube_pod_container_status_restarts_total", SignalType: signal.TypeRateIncrease, Severity: signal.SeverityWarning, Timestamp: now, Namespace: "rackpilot"},
		}},
		{"memory_pressure", []signal.Signal{
			mk("rackpilot_bmc_memory_errors_total", signal.SeverityWarning, signal.TypeCounterIncrease, "d0"),
			{MetricName: "container_memory_working_set_bytes", SignalType: signal.TypeResourcePressure, Severity: signal.SeverityWarning, Timestamp: now, Namespace: "rackpilot", Component: "c"},
		}},
	}
	for _, tc := range cases {
		res := e.Evaluate(tc.sig, now)
		ok := false
		for _, r := range res {
			if r.RuleID == tc.rule && r.Active {
				ok = true
				if r.Incident.RootCauseDisclaimer == "" {
					t.Errorf("%s missing disclaimer", tc.rule)
				}
			}
		}
		if !ok {
			t.Fatalf("rule %s not active", tc.rule)
		}
	}
}
