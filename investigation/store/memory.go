package store

import (
	"sync"

	"github.com/anil-debug/rackpilot-ai/investigation/report"
)

// Memory holds investigations in process memory (not durable across restarts).
type Memory struct {
	mu   sync.RWMutex
	byID map[string]report.Investigation
	ord  []string
}

func NewMemory() *Memory {
	return &Memory{byID: map[string]report.Investigation{}}
}

func (m *Memory) Put(inv report.Investigation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[inv.InvestigationID]; !ok {
		m.ord = append(m.ord, inv.InvestigationID)
	}
	m.byID[inv.InvestigationID] = inv
}

func (m *Memory) Get(id string) (report.Investigation, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inv, ok := m.byID[id]
	return inv, ok
}

func (m *Memory) List(offset, limit int) []report.Investigation {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if offset >= len(m.ord) {
		return nil
	}
	end := offset + limit
	if end > len(m.ord) {
		end = len(m.ord)
	}
	out := make([]report.Investigation, 0, end-offset)
	for _, id := range m.ord[offset:end] {
		out = append(out, m.byID[id])
	}
	return out
}
