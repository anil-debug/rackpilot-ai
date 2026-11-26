package scenario

import "github.com/anil-debug/rackpilot-ai/simulator/bmc/engine"

type Manager struct {
	eng *engine.Engine
}

func NewManager(eng *engine.Engine) *Manager {
	return &Manager{eng: eng}
}

func (m *Manager) FanFailure(id string) error {
	if id == "" {
		return errInvalid()
	}
	return m.eng.ApplyFanFailure(id)
}

func (m *Manager) PSUFailure(id string) error {
	if id == "" {
		return errInvalid()
	}
	return m.eng.ApplyPSUFailure(id)
}

func (m *Manager) CPUOverheat(id string) error {
	if id == "" {
		return errInvalid()
	}
	return m.eng.ApplyCPUOverheat(id)
}

func (m *Manager) MemoryError(id string) error {
	if id == "" {
		return errInvalid()
	}
	return m.eng.ApplyMemoryError(id)
}

func (m *Manager) Reset() {
	m.eng.ResetAll()
}
