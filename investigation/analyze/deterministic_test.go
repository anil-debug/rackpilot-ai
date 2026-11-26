package analyze

import (
	"testing"
	"time"

	inc "github.com/anil-debug/rackpilot-ai/incident/incident"
	"github.com/anil-debug/rackpilot-ai/incident/signal"
	"github.com/anil-debug/rackpilot-ai/investigation/collector"
	"github.com/anil-debug/rackpilot-ai/investigation/evidence"
	"github.com/anil-debug/rackpilot-ai/investigation/report"
)

func TestDeterministicFanFailureReferencesEvidence(t *testing.T) {
	inst := inc.Incident{
		IncidentID:      "inc-fan",
		Title:           "Hardware cooling",
		CorrelationRule: "hardware_cooling",
		Confidence:      "medium",
		FirstSeen:       time.Now().UTC().Add(-2 * time.Minute),
		LastSeen:        time.Now().UTC(),
		ContributingSignals: []signal.Signal{
			{
				MetricName:   "rackpilot_bmc_fan_health_status",
				MetricValue:  2,
				Component:    "fan-0",
				ObservedFact: "Fan fan-0 health critical",
			},
		},
	}
	collected := collector.Result{
		Items: []evidence.Item{
			{EvidenceID: "E001", Name: "rackpilot_bmc_fan_health_status", Value: "2", Description: "fan critical"},
			{EvidenceID: "E002", Name: "rackpilot_bmc_fan_speed_rpm", Value: "0", Description: "speed zero"},
		},
		Sources: []string{"prometheus", "redfish"},
	}
	rep := (&Deterministic{}).Investigate(inst, collected)
	if rep.Status != report.StatusCompleted {
		t.Fatalf("status=%s", rep.Status)
	}
	if len(rep.SupportingEvidenceIDs) == 0 {
		t.Fatal("expected supporting evidence ids")
	}
	if len(rep.ConfirmedRootCauses) != 0 {
		t.Fatal("must not confirm root cause without proof")
	}
}

func TestDeterministicPrometheusUnavailablePartial(t *testing.T) {
	inst := inc.Incident{
		IncidentID:      "inc-partial",
		CorrelationRule: "hardware_cooling",
		FirstSeen:       time.Now().UTC(),
		LastSeen:        time.Now().UTC(),
	}
	collected := collector.Result{
		PrometheusErr: errTest("prom down"),
		Items: []evidence.Item{
			{EvidenceID: "E001", Name: "event_log", Source: "redfish"},
		},
		Sources: []string{"redfish"},
	}
	rep := (&Deterministic{}).Investigate(inst, collected)
	if rep.Status != report.StatusPartial {
		t.Fatalf("expected partial got %s", rep.Status)
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }
