package store

import (
	"sort"
	"sync"
	"time"

	"github.com/anil-debug/rackpilot-ai/incident/correlate"
	inc "github.com/anil-debug/rackpilot-ai/incident/incident"
	"github.com/anil-debug/rackpilot-ai/incident/signal"
)

type active struct {
	incident     inc.Incident
	lastActive   time.Time
	recoverSince *time.Time
}

// Memory is in-process incident storage.
type Memory struct {
	mu    sync.RWMutex
	byKey map[string]*active
	all   map[string]inc.Incident
}

func NewMemory() *Memory {
	return &Memory{
		byKey: make(map[string]*active),
		all:   make(map[string]inc.Incident),
	}
}

// ApplyTick merges correlation results and handles recovery.
func (m *Memory) ApplyTick(results []correlate.Result, now time.Time, recovery time.Duration) (opened, updated, resolved int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	activeKeys := map[string]struct{}{}
	for _, r := range results {
		if !r.Active {
			continue
		}
		activeKeys[r.DedupeKey] = struct{}{}
		cur, exists := m.byKey[r.DedupeKey]
		if !exists {
			inst := r.Incident
			inst.Status = inc.StatusOpen
			m.byKey[r.DedupeKey] = &active{incident: inst, lastActive: now}
			m.all[inst.IncidentID] = inst
			opened++
			continue
		}
		prev := cur.incident
		inst := mergeIncident(prev, r.Incident, now)
		inst.Status = inc.StatusUpdated
		cur.incident = inst
		cur.lastActive = now
		cur.recoverSince = nil
		m.all[inst.IncidentID] = inst
		updated++
	}

	for key, cur := range m.byKey {
		if _, ok := activeKeys[key]; ok {
			continue
		}
		if cur.incident.Status == inc.StatusResolved {
			continue
		}
		if cur.recoverSince == nil {
			t := now
			cur.recoverSince = &t
			continue
		}
		if now.Sub(*cur.recoverSince) >= recovery {
			inst := cur.incident
			inst.Status = inc.StatusResolved
			inst.ResolvedAt = &now
			inst.LastSeen = now
			cur.incident = inst
			m.all[inst.IncidentID] = inst
			resolved++
		}
	}
	return opened, updated, resolved
}

func mergeIncident(prev, next inc.Incident, now time.Time) inc.Incident {
	prev.LastSeen = now
	prev.ContributingSignals = mergeSignals(prev.ContributingSignals, next.ContributingSignals)
	prev.AffectedResources = uniqueAppend(prev.AffectedResources, next.AffectedResources)
	prev.ObservedFacts = uniqueAppend(prev.ObservedFacts, next.ObservedFacts)
	prev.CorrelatedSymptoms = uniqueAppend(prev.CorrelatedSymptoms, next.CorrelatedSymptoms)
	prev.Timeline = append(prev.Timeline, inc.TimelineEvent{Timestamp: now, Summary: "Incident updated with new correlated signals"})
	return prev
}

func mergeSignals(a, b []signal.Signal) []signal.Signal {
	seen := map[string]struct{}{}
	var out []signal.Signal
	for _, s := range append(a, b...) {
		if _, ok := seen[s.SignalID]; ok {
			continue
		}
		seen[s.SignalID] = struct{}{}
		out = append(out, s)
	}
	return out
}

func uniqueAppend(a, b []string) []string {
	seen := map[string]struct{}{}
	for _, s := range a {
		seen[s] = struct{}{}
	}
	for _, s := range b {
		seen[s] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func (m *Memory) List(offset, limit int) []inc.Incident {
	m.mu.RLock()
	defer m.mu.RUnlock()
	all := make([]inc.Incident, 0, len(m.all))
	for _, v := range m.all {
		all = append(all, v)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].LastSeen.After(all[j].LastSeen) })
	if offset >= len(all) {
		return nil
	}
	end := offset + limit
	if limit <= 0 || end > len(all) {
		end = len(all)
	}
	return all[offset:end]
}

func (m *Memory) Get(id string) (inc.Incident, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.all[id]
	return v, ok
}

func (m *Memory) Active() []inc.Incident {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []inc.Incident
	for _, cur := range m.byKey {
		if cur.incident.Status == inc.StatusResolved {
			continue
		}
		out = append(out, cur.incident)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeen.After(out[j].LastSeen) })
	return out
}

func (m *Memory) CountActive() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, cur := range m.byKey {
		if cur.incident.Status != inc.StatusResolved {
			n++
		}
	}
	return n
}
