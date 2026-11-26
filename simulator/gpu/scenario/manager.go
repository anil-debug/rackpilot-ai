package scenario

import (
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/domain"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/engine"
)

// Manager handles fault scenario API operations.
type Manager struct {
	eng *engine.Engine
}

// NewManager creates a scenario manager.
func NewManager(eng *engine.Engine) *Manager {
	return &Manager{eng: eng}
}

// Overheating starts overheating simulation on gpuID.
func (m *Manager) Overheating(gpuID string) error {
	if gpuID == "" {
		return domain.ErrInvalidGPUID
	}
	return m.eng.SetScenario(gpuID, domain.ScenarioOverheating)
}

// MemoryExhaustion starts memory pressure simulation on gpuID.
func (m *Manager) MemoryExhaustion(gpuID string) error {
	if gpuID == "" {
		return domain.ErrInvalidGPUID
	}
	return m.eng.SetScenario(gpuID, domain.ScenarioMemoryExhaustion)
}

// Reset restores all GPUs to baseline simulation.
func (m *Manager) Reset() int {
	return m.eng.ResetAll()
}
