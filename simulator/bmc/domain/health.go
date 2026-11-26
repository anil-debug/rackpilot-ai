package domain

// HealthStatus is derived from telemetry and fault state.
type HealthStatus string

const (
	HealthOK       HealthStatus = "OK"
	HealthWarning  HealthStatus = "Warning"
	HealthCritical HealthStatus = "Critical"
)

// RedfishHealth maps to Redfish Status.Health values.
func RedfishHealth(h HealthStatus) string {
	return string(h)
}

// HealthCode for Prometheus (0=OK, 1=Warning, 2=Critical).
type HealthCode int

const (
	HealthCodeOK       HealthCode = 0
	HealthCodeWarning  HealthCode = 1
	HealthCodeCritical HealthCode = 2
)

func HealthToCode(h HealthStatus) HealthCode {
	switch h {
	case HealthOK:
		return HealthCodeOK
	case HealthWarning:
		return HealthCodeWarning
	case HealthCritical:
		return HealthCodeCritical
	default:
		return HealthCodeOK
	}
}

func maxHealth(a, b HealthStatus) HealthStatus {
	rank := map[HealthStatus]int{HealthOK: 0, HealthWarning: 1, HealthCritical: 2}
	if rank[b] > rank[a] {
		return b
	}
	return a
}
