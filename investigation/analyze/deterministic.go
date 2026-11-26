package analyze

import (
	"fmt"
	"strings"
	"time"

	inc "github.com/anil-debug/rackpilot-ai/incident/incident"
	"github.com/anil-debug/rackpilot-ai/investigation/collector"
	"github.com/anil-debug/rackpilot-ai/investigation/evidence"
	"github.com/anil-debug/rackpilot-ai/investigation/report"
)

// Deterministic produces evidence-backed investigation reports without an LLM.
type Deterministic struct{}

func (d *Deterministic) Investigate(incident inc.Incident, collected collector.Result) report.Investigation {
	rep := report.Investigation{
		InvestigationID:     fmt.Sprintf("inv-%s-%d", incident.IncidentID, time.Now().Unix()),
		IncidentID:          incident.IncidentID,
		IncidentSummary:     incident.Title,
		IncidentContext:     report.ContextFromIncident(incident),
		InvestigationMode:   "deterministic",
		EvidenceTimeline:    collected.Items,
		ObservedFacts:       uniqueAppend(incident.ObservedFacts, factsFromEvidence(collected.Items)),
		PotentialRootCauses: []string{},
		ConfirmedRootCauses: []string{},
		Confidence:          incident.Confidence,
		MissingEvidence:     []string{},
		RecommendedChecks:   []string{},
		DataSources:         collected.Sources,
		DataClassification: map[string]string{
			"prometheus_metrics": string(evidence.ProvenanceSyntheticLive),
			"redfish":            string(redfishClass(collected.BMCMode)),
		},
		InvestigatedAt:      time.Now().UTC(),
		CausationDisclaimer: report.CausationDisclaimer,
	}
	if collected.PrometheusErr != nil {
		rep.Errors = append(rep.Errors, "prometheus: "+collected.PrometheusErr.Error())
		rep.MissingEvidence = append(rep.MissingEvidence, "Prometheus metric history unavailable")
	}
	if collected.RedfishErr != nil {
		rep.Errors = append(rep.Errors, "redfish: "+collected.RedfishErr.Error())
		rep.MissingEvidence = append(rep.MissingEvidence, "Redfish chassis/thermal/power/event log")
	}

	support := []string{}
	switch incident.CorrelationRule {
	case "hardware_cooling":
		rep.PotentialRootCauses = append(rep.PotentialRootCauses,
			"Fan failure or fan-control malfunction",
			"Cooling impairment with secondary thermal rise")
		support = idsMatching(collected.Items, "rackpilot_bmc_fan")
		rep.RecommendedChecks = append(rep.RecommendedChecks,
			"Inspect fan hardware and connectors",
			"Verify fan power and controller logs",
			"Compare chassis thermal trend over a longer Prometheus window")
		rep.MissingEvidence = append(rep.MissingEvidence, missingIfNoMatch(collected.Items, "rackpilot_bmc_fan_speed_rpm", "Historical fan speed trend"))
	case "power_supply":
		rep.PotentialRootCauses = append(rep.PotentialRootCauses, "PSU degradation or power delivery fault")
		support = idsMatching(collected.Items, "rackpilot_bmc_psu")
		rep.RecommendedChecks = append(rep.RecommendedChecks, "Verify PSU redundancy and input power", "Check system event log for power events")
	case "kubernetes_workload":
		rep.PotentialRootCauses = append(rep.PotentialRootCauses, "Workload crash loop or scheduling/availability issue")
		support = idsMatching(collected.Items, "kube_")
		rep.RecommendedChecks = append(rep.RecommendedChecks, "kubectl describe pod / deployment events", "Check image pull and resource limits")
	case "memory_pressure":
		rep.PotentialRootCauses = append(rep.PotentialRootCauses, "Memory errors with container memory pressure (correlated, not confirmed single root)")
		support = idsMatching(collected.Items, "memory")
		rep.RecommendedChecks = append(rep.RecommendedChecks, "Inspect DIMM logs", "Review container memory limits and OOM events")
	default:
		rep.PotentialRootCauses = append(rep.PotentialRootCauses, incident.SuspectedRootCause)
	}

	rep.SupportingEvidenceIDs = support
	if len(support) == 0 {
		rep.Confidence = "low"
		rep.MissingEvidence = append(rep.MissingEvidence, "No supporting metric samples matched rule-specific queries")
	}

	if len(rep.Errors) > 0 && len(collected.Items) > 0 {
		rep.Status = report.StatusPartial
	} else if len(rep.Errors) > 0 && len(collected.Items) == 0 {
		rep.Status = report.StatusFailed
	} else {
		rep.Status = report.StatusCompleted
	}
	// Never confirm root cause without strong multi-source proof
	rep.ConfirmedRootCauses = []string{}
	return rep
}

func redfishClass(mode string) evidence.Provenance {
	if mode == "static" {
		return evidence.ProvenanceDMTFFixture
	}
	return evidence.ProvenanceRedfishLive
}

func factsFromEvidence(items []evidence.Item) []string {
	var out []string
	for _, it := range items {
		if it.Description != "" {
			out = append(out, fmt.Sprintf("[%s] %s", it.EvidenceID, it.Description))
		}
	}
	return uniqueStrings(out)
}

func idsMatching(items []evidence.Item, substr string) []string {
	var ids []string
	for _, it := range items {
		if strings.Contains(it.Name, substr) || strings.Contains(it.Description, substr) {
			ids = append(ids, it.EvidenceID)
		}
	}
	return uniqueStrings(ids)
}

func missingIfNoMatch(items []evidence.Item, name, msg string) string {
	for _, it := range items {
		if strings.Contains(it.Name, name) {
			return ""
		}
	}
	return msg
}

func uniqueAppend(a, b []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range append(a, b...) {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func uniqueStrings(in []string) []string {
	return uniqueAppend(nil, in)
}
