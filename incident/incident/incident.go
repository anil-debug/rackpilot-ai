package incident

import (
	"time"

	"github.com/anil-debug/rackpilot-ai/incident/signal"
)

type Status string

const (
	StatusOpen     Status = "OPEN"
	StatusUpdated  Status = "UPDATED"
	StatusResolved Status = "RESOLVED"
)

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Incident is a correlated infrastructure event.
type Incident struct {
	IncidentID          string          `json:"incident_id"`
	Title               string          `json:"title"`
	Description         string          `json:"description"`
	Severity            Severity        `json:"severity"`
	Status              Status          `json:"status"`
	FirstSeen           time.Time       `json:"first_seen"`
	LastSeen            time.Time       `json:"last_seen"`
	ResolvedAt          *time.Time      `json:"resolved_at,omitempty"`
	AffectedResources   []string        `json:"affected_resources"`
	ContributingSignals []signal.Signal `json:"contributing_signals"`
	CorrelationRule     string          `json:"correlation_rule"`
	Confidence          string          `json:"confidence"`
	ObservedFacts       []string        `json:"observed_facts"`
	CorrelatedSymptoms  []string        `json:"correlated_symptoms"`
	SuspectedRootCause  string          `json:"suspected_root_cause"`
	RootCauseDisclaimer string          `json:"root_cause_disclaimer"`
	Timeline            []TimelineEvent `json:"timeline"`
}

type TimelineEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Summary   string    `json:"summary"`
}
