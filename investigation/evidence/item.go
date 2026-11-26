package evidence

import "time"

// Provenance classifies how evidence was obtained.
type Provenance string

const (
	ProvenanceSyntheticLive Provenance = "synthetic_live"
	ProvenanceK8sMetrics    Provenance = "kubernetes_metrics"
	ProvenanceRedfishLive   Provenance = "redfish_synthetic"
	ProvenanceDMTFFixture   Provenance = "redfish_dmtf_fixture"
)

// Item is normalized investigation evidence.
type Item struct {
	EvidenceID  string            `json:"evidence_id"`
	Source      string            `json:"source"`
	Timestamp   time.Time         `json:"timestamp"`
	Resource    string            `json:"resource"`
	Name        string            `json:"metric_or_event_name"`
	Value       string            `json:"observed_value"`
	Unit        string            `json:"unit,omitempty"`
	Severity    string            `json:"severity,omitempty"`
	Description string            `json:"description"`
	Provenance  Provenance        `json:"provenance"`
	Reference   string            `json:"source_reference"`
	Labels      map[string]string `json:"labels,omitempty"`
}
