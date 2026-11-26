package engine

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/anil-debug/rackpilot-ai/simulator/gpu/config"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/domain"
)

type gpuState struct {
	id       string
	phase    float64
	tick     int64
	tempC    float64
	utilPct  float64
	memUsed  uint64
	memTotal uint64
	powerW   float64
	scenario domain.ScenarioType
	// Scenario-specific targets accumulated on tick.
	overheatExtraTemp float64
}

// Engine runs synthetic GPU telemetry simulation.
type Engine struct {
	mu         sync.RWMutex
	gpus       map[string]*gpuState
	profile    domain.GPUProfile
	thresholds domain.HealthThresholds
	cfg        config.Config
	rng        *rand.Rand
	now        func() time.Time
}

// New creates an engine and initializes GPUs.
func New(cfg config.Config) *Engine {
	const golden = uint64(0x9e3779b97f4a7c15)
	rng := rand.New(rand.NewPCG(uint64(cfg.SimSeed), uint64(cfg.SimSeed)^golden))
	e := &Engine{
		gpus:       make(map[string]*gpuState),
		profile:    cfg.Profile,
		thresholds: cfg.HealthThresholds,
		cfg:        cfg,
		rng:        rng,
		now:        time.Now,
	}
	for i := 0; i < cfg.GPUCount; i++ {
		id := fmt.Sprintf("gpu-%d", i)
		phase := float64(i) * 1.7
		memTotal := cfg.Profile.MemoryTotalBytes
		memUsed := uint64(float64(memTotal) * cfg.Profile.BaselineMemoryUsedRatio)
		util := cfg.Profile.BaselineUtilizationPercent + baselineDrift(0, phase)
		if util < 5 {
			util = 5
		}
		if util > 95 {
			util = 95
		}
		e.gpus[id] = &gpuState{
			id:       id,
			phase:    phase,
			tempC:    tempFromUtil(util, cfg.Profile, phase),
			utilPct:  util,
			memUsed:  memUsed,
			memTotal: memTotal,
			powerW:   powerFromUtil(util, cfg.Profile),
		}
	}
	return e
}

// SetClock injects time source for tests.
func (e *Engine) SetClock(now func() time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.now = now
}

// Start runs the tick loop until context is canceled.
func (e *Engine) Start(ctx context.Context) {
	ticker := time.NewTicker(e.cfg.TickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.Tick()
		}
	}
}

// Tick advances simulation by one step (public for deterministic tests).
func (e *Engine) Tick() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, g := range e.gpus {
		e.stepGPU(g)
	}
}

func (e *Engine) stepGPU(g *gpuState) {
	g.tick++
	noise := (e.rng.Float64() - 0.5) * 3

	switch g.scenario {
	case domain.ScenarioNone:
		drift := baselineDrift(g.tick, g.phase)
		g.utilPct = e.cfg.Profile.BaselineUtilizationPercent + drift + noise
		if g.utilPct < 5 {
			g.utilPct = 5
		}
		if g.utilPct > 95 {
			g.utilPct = 95
		}
		g.tempC = tempFromUtil(g.utilPct, e.profile, g.phase) + noise*0.2
		g.overheatExtraTemp = 0
		targetUsed := uint64(float64(g.memTotal) * e.profile.BaselineMemoryUsedRatio)
		g.memUsed = blendMemory(g.memUsed, targetUsed, 0.1)
	case domain.ScenarioOverheating:
		g.overheatExtraTemp += e.cfg.OverheatTempDeltaCelsius
		g.utilPct = min(95, g.utilPct+1.5)
		base := tempFromUtil(g.utilPct, e.profile, g.phase)
		g.tempC = base + g.overheatExtraTemp
		if g.tempC > 98 {
			g.tempC = 98
		}
	case domain.ScenarioMemoryExhaustion:
		drift := baselineDrift(g.tick, g.phase)
		g.utilPct = e.cfg.Profile.BaselineUtilizationPercent + drift + 15 + noise
		if g.utilPct > 99 {
			g.utilPct = 99
		}
		g.tempC = tempFromUtil(g.utilPct, e.profile, g.phase) + 5
		target := uint64(float64(g.memTotal) * e.cfg.MemoryExhaustionTargetRatio)
		g.memUsed += e.cfg.MemoryRampBytesPerTick
		if g.memUsed > target {
			g.memUsed = target
		}
	default:
		g.scenario = domain.ScenarioNone
	}

	g.memUsed = clampMemoryUsed(g.memUsed, g.memTotal)
	g.powerW = powerFromUtil(g.utilPct, e.profile)
}

func blendMemory(current, target uint64, alpha float64) uint64 {
	if current == target {
		return current
	}
	delta := float64(target) - float64(current)
	return current + uint64(alpha*delta)
}

// Snapshot returns a copy of all GPU telemetry.
func (e *Engine) Snapshot() []domain.Telemetry {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]domain.Telemetry, 0, len(e.gpus))
	for i := 0; i < e.cfg.GPUCount; i++ {
		id := fmt.Sprintf("gpu-%d", i)
		g, ok := e.gpus[id]
		if !ok {
			continue
		}
		out = append(out, e.toTelemetry(g))
	}
	return out
}

// Get returns telemetry for one GPU.
func (e *Engine) Get(id string) (domain.Telemetry, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	g, ok := e.gpus[id]
	if !ok {
		return domain.Telemetry{}, false
	}
	return e.toTelemetry(g), true
}

func (e *Engine) toTelemetry(g *gpuState) domain.Telemetry {
	health := domain.DeriveHealth(g.tempC, g.memUsed, g.memTotal, e.thresholds)
	ts := e.now()
	return domain.Telemetry{
		ID:                 g.id,
		Model:              e.profile.Model,
		TemperatureCelsius: round2(g.tempC),
		UtilizationPercent: round2(g.utilPct),
		MemoryTotalBytes:   g.memTotal,
		MemoryUsedBytes:    g.memUsed,
		PowerWatts:         round2(g.powerW),
		HealthStatus:       health,
		LastUpdated:        ts,
		SyntheticTelemetry: true,
	}
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// SetScenario applies a scenario to one GPU.
func (e *Engine) SetScenario(gpuID string, scenario domain.ScenarioType) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	g, ok := e.gpus[gpuID]
	if !ok {
		return domain.ErrGPUNotFound
	}
	g.scenario = scenario
	if scenario == domain.ScenarioOverheating {
		g.overheatExtraTemp = 0
	}
	return nil
}

// ResetAll clears scenarios and restores baseline-oriented state.
func (e *Engine) ResetAll() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	count := 0
	for _, g := range e.gpus {
		g.scenario = domain.ScenarioNone
		g.overheatExtraTemp = 0
		memTotal := g.memTotal
		g.memUsed = uint64(float64(memTotal) * e.profile.BaselineMemoryUsedRatio)
		g.utilPct = e.profile.BaselineUtilizationPercent + baselineDrift(g.tick, g.phase)
		g.tempC = tempFromUtil(g.utilPct, e.profile, g.phase)
		g.powerW = powerFromUtil(g.utilPct, e.profile)
		count++
	}
	return count
}

// GPUIDs returns known GPU identifiers in order.
func (e *Engine) GPUIDs() []string {
	ids := make([]string, 0, e.cfg.GPUCount)
	for i := 0; i < e.cfg.GPUCount; i++ {
		ids = append(ids, fmt.Sprintf("gpu-%d", i))
	}
	return ids
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
