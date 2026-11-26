package domain

import "time"

// HealthStatus represents derived GPU health from telemetry thresholds.
type HealthStatus string

const (
	HealthHealthy  HealthStatus = "healthy"
	HealthWarning  HealthStatus = "warning"
	HealthCritical HealthStatus = "critical"
)

// HealthCode is the numeric encoding exported to Prometheus (0/1/2).
type HealthCode int

const (
	HealthCodeHealthy  HealthCode = 0
	HealthCodeWarning  HealthCode = 1
	HealthCodeCritical HealthCode = 2
)

// GPUProfile describes hardware characteristics and simulation defaults for a model.
type GPUProfile struct {
	Model            string
	MemoryTotalBytes uint64
	PowerLimitWatts  float64
	// Baseline simulation bands (synthetic, not hardware specs).
	BaselineTempCelsius        float64
	BaselineUtilizationPercent float64
	BaselineMemoryUsedRatio    float64
	BaselinePowerWatts         float64
}

// HealthThresholds defines how telemetry maps to health status (consistent derivation).
type HealthThresholds struct {
	TempWarningCelsius  float64
	TempCriticalCelsius float64
	MemoryWarningRatio  float64
	MemoryCriticalRatio float64
}

// DefaultH100Profile is the default simulated NVIDIA H100 80GB profile.
func DefaultH100Profile() GPUProfile {
	const eightyGB = 80 * 1024 * 1024 * 1024
	return GPUProfile{
		Model:                      "NVIDIA H100 80GB",
		MemoryTotalBytes:           eightyGB,
		PowerLimitWatts:            700,
		BaselineTempCelsius:        45,
		BaselineUtilizationPercent: 55,
		BaselineMemoryUsedRatio:    0.45,
		BaselinePowerWatts:         280,
	}
}

// DefaultHealthThresholds returns standard threshold values for health derivation.
func DefaultHealthThresholds() HealthThresholds {
	return HealthThresholds{
		TempWarningCelsius:  80,
		TempCriticalCelsius: 90,
		MemoryWarningRatio:  0.90,
		MemoryCriticalRatio: 0.95,
	}
}

// Telemetry holds synthetic GPU telemetry exposed via API and metrics.
type Telemetry struct {
	ID                 string       `json:"id"`
	Model              string       `json:"model"`
	TemperatureCelsius float64      `json:"temperature_celsius"`
	UtilizationPercent float64      `json:"utilization_percent"`
	MemoryTotalBytes   uint64       `json:"memory_total_bytes"`
	MemoryUsedBytes    uint64       `json:"memory_used_bytes"`
	PowerWatts         float64      `json:"power_watts"`
	HealthStatus       HealthStatus `json:"health_status"`
	LastUpdated        time.Time    `json:"last_updated"`
	SyntheticTelemetry bool         `json:"synthetic_telemetry"`
}

// DeriveHealth computes health from current telemetry and thresholds (single source of truth).
func DeriveHealth(tempC float64, memoryUsed, memoryTotal uint64, t HealthThresholds) HealthStatus {
	var worst HealthStatus = HealthHealthy

	if tempC >= t.TempCriticalCelsius {
		worst = maxHealth(worst, HealthCritical)
	} else if tempC >= t.TempWarningCelsius {
		worst = maxHealth(worst, HealthWarning)
	}

	if memoryTotal > 0 {
		ratio := float64(memoryUsed) / float64(memoryTotal)
		if ratio >= t.MemoryCriticalRatio {
			worst = maxHealth(worst, HealthCritical)
		} else if ratio >= t.MemoryWarningRatio {
			worst = maxHealth(worst, HealthWarning)
		}
	}

	return worst
}

func maxHealth(a, b HealthStatus) HealthStatus {
	severity := map[HealthStatus]int{
		HealthHealthy:  0,
		HealthWarning:  1,
		HealthCritical: 2,
	}
	if severity[b] > severity[a] {
		return b
	}
	return a
}

// HealthToCode converts health status to Prometheus gauge value.
func HealthToCode(h HealthStatus) HealthCode {
	switch h {
	case HealthHealthy:
		return HealthCodeHealthy
	case HealthWarning:
		return HealthCodeWarning
	case HealthCritical:
		return HealthCodeCritical
	default:
		return HealthCodeHealthy
	}
}

// ScenarioType identifies an active fault simulation on a GPU.
type ScenarioType string

const (
	ScenarioNone             ScenarioType = ""
	ScenarioOverheating      ScenarioType = "overheating"
	ScenarioMemoryExhaustion ScenarioType = "memory_exhaustion"
)
