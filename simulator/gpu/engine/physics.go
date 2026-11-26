package engine

import (
	"math"

	"github.com/anil-debug/rackpilot-ai/simulator/gpu/domain"
)

// baselineDrift returns small deterministic oscillation for utilization (percent).
func baselineDrift(tick int64, phase float64) float64 {
	return 12 * math.Sin(float64(tick)*0.15+phase)
}

// powerFromUtil estimates synthetic power draw from utilization and profile limits.
func powerFromUtil(utilPercent float64, profile domain.GPUProfile) float64 {
	fraction := utilPercent / 100
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	idle := profile.BaselinePowerWatts * 0.35
	max := profile.PowerLimitWatts * 0.85
	return idle + (max-idle)*fraction
}

// tempFromUtil models temperature coupling to utilization for baseline (synthetic).
func tempFromUtil(utilPercent float64, profile domain.GPUProfile, phase float64) float64 {
	base := profile.BaselineTempCelsius + baselineDrift(0, phase)*0.1
	return base + (utilPercent/100)*25
}

// clampMemoryUsed ensures used never exceeds total.
func clampMemoryUsed(used, total uint64) uint64 {
	if used > total {
		return total
	}
	return used
}
