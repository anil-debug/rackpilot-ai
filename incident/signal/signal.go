package signal

import "time"

// Source identifies telemetry origin.
type Source string

const (
	SourceGPU          Source = "gpu_simulator"
	SourceBMC          Source = "bmc_simulator"
	SourceKubernetes   Source = "kubernetes"
	SourceUnclassified Source = "unknown"
)

// Severity classifies signal urgency for correlation input (not incident severity).
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// SignalType describes what kind of observation this is.
type SignalType string

const (
	TypeHealthStatus     SignalType = "health_status"
	TypeThreshold        SignalType = "threshold"
	TypeRateIncrease     SignalType = "rate_increase"
	TypeAvailabilityGap  SignalType = "availability_gap"
	TypeCounterIncrease  SignalType = "counter_increase"
	TypeResourcePressure SignalType = "resource_pressure"
)

// Signal is a normalized infrastructure observation.
type Signal struct {
	SignalID          string            `json:"signal_id"`
	Source            Source            `json:"source"`
	SignalType        SignalType        `json:"signal_type"`
	Severity          Severity          `json:"severity"`
	Timestamp         time.Time         `json:"timestamp"`
	Cluster           string            `json:"cluster,omitempty"`
	Node              string            `json:"node,omitempty"`
	Namespace         string            `json:"namespace,omitempty"`
	Pod               string            `json:"pod,omitempty"`
	Component         string            `json:"component,omitempty"`
	MetricName        string            `json:"metric_name"`
	MetricValue       float64           `json:"metric_value"`
	Labels            map[string]string `json:"labels,omitempty"`
	EvidenceReference string            `json:"evidence_reference"`
	ObservedFact      string            `json:"observed_fact"`
}
