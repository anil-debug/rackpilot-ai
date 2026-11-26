package report

import (
	"time"

	inc "github.com/anil-debug/rackpilot-ai/incident/incident"
	"github.com/anil-debug/rackpilot-ai/investigation/evidence"
)

type Status string

const (
	StatusCompleted Status = "completed"
	StatusPartial   Status = "partial"
	StatusFailed    Status = "failed"
)

// Investigation is the structured root-cause investigation output.
type Investigation struct {
	InvestigationID       string            `json:"investigation_id"`
	IncidentID            string            `json:"incident_id"`
	IncidentSummary       string            `json:"incident_summary"`
	IncidentContext       map[string]any    `json:"incident_context"`
	Status                Status            `json:"status"`
	InvestigationMode     string            `json:"investigation_mode"`
	EvidenceTimeline      []evidence.Item   `json:"evidence_timeline"`
	ObservedFacts         []string          `json:"observed_facts"`
	PotentialRootCauses   []string          `json:"potential_root_causes"`
	ConfirmedRootCauses   []string          `json:"confirmed_root_causes"`
	SupportingEvidenceIDs []string          `json:"supporting_evidence_ids"`
	ContradictingEvidence []string          `json:"contradicting_evidence"`
	Confidence            string            `json:"confidence"`
	MissingEvidence       []string          `json:"missing_evidence"`
	RecommendedChecks     []string          `json:"recommended_next_checks"`
	DataSources           []string          `json:"data_sources_used"`
	DataClassification    map[string]string `json:"data_classification"`
	InvestigatedAt        time.Time         `json:"investigation_timestamp"`
	Errors                []string          `json:"errors,omitempty"`
	CausationDisclaimer   string            `json:"causation_disclaimer"`
}

const CausationDisclaimer = "Potential root causes are hypotheses supported by collected evidence IDs. Correlation does not confirm causation without additional validation."

func ContextFromIncident(i inc.Incident) map[string]any {
	return map[string]any{
		"correlation_rule": i.CorrelationRule,
		"severity":         i.Severity,
		"status":           i.Status,
		"first_seen":       i.FirstSeen,
		"last_seen":        i.LastSeen,
		"confidence":       i.Confidence,
		"affected":         i.AffectedResources,
	}
}
