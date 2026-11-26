package engine

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anil-debug/rackpilot-ai/simulator/bmc/config"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/domain"
)

type componentFaults struct {
	fanDead map[string]bool
	psuDead map[string]bool
	cpuHot  map[string]bool
	dimmErr map[string]bool
}

// Engine simulates BMC hardware telemetry.
type Engine struct {
	mu        sync.RWMutex
	cfg       config.Config
	rng       *rand.Rand
	started   time.Time
	tickCount int64
	faults    componentFaults
	events    []domain.EventEntry
	eventSeq  int
	cpuTemp   []float64
	cpuUtil   []float64
	memUsed   uint64
	memTotal  uint64
	dimmCE    []uint64
	dimmUCE   []uint64
	fanRPM    []float64
	psuIn     []float64
	psuOut    []float64
	inletC    float64
	exhaustC  float64
	now       func() time.Time
}

func New(cfg config.Config) *Engine {
	const golden = uint64(0x9e3779b97f4a7c15)
	rng := rand.New(rand.NewPCG(uint64(cfg.SimSeed), uint64(cfg.SimSeed)^golden))
	p := cfg.Profile
	e := &Engine{
		cfg:     cfg,
		rng:     rng,
		started: time.Now(),
		faults: componentFaults{
			fanDead: make(map[string]bool),
			psuDead: make(map[string]bool),
			cpuHot:  make(map[string]bool),
			dimmErr: make(map[string]bool),
		},
		now:      time.Now,
		memTotal: 8 * 64 * 1024 * 1024 * 1024,
		memUsed:  3 * 64 * 1024 * 1024 * 1024,
	}
	e.cpuTemp = make([]float64, p.CPUSockets)
	e.cpuUtil = make([]float64, p.CPUSockets)
	e.dimmCE = make([]uint64, p.DIMMCount)
	e.dimmUCE = make([]uint64, p.DIMMCount)
	e.fanRPM = make([]float64, p.FanCount)
	e.psuIn = make([]float64, p.PSUCount)
	e.psuOut = make([]float64, p.PSUCount)
	for i := range e.cpuTemp {
		e.cpuTemp[i] = 42 + float64(i)*2
		e.cpuUtil[i] = 35 + float64(i)*5
	}
	for i := range e.fanRPM {
		e.fanRPM[i] = 4200 + float64(i)*100
	}
	for i := range e.psuOut {
		e.psuIn[i] = 280
		e.psuOut[i] = 250
	}
	e.inletC = 24
	e.exhaustC = 38
	return e
}

func (e *Engine) SetClock(now func() time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.now = now
}

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

func (e *Engine) Tick() {
	e.mu.Lock()
	defer e.mu.Unlock()
	atomic.AddInt64(&e.tickCount, 1)
	t := e.tickCount
	noise := func() float64 { return (e.rng.Float64() - 0.5) * 2 }

	for i := range e.cpuTemp {
		id := fmt.Sprintf("cpu-%d", i)
		if e.faults.cpuHot[id] {
			e.cpuTemp[i] += 3.5
			if e.cpuTemp[i] > 98 {
				e.cpuTemp[i] = 98
			}
			e.cpuUtil[i] = min(99, e.cpuUtil[i]+2)
		} else {
			e.cpuTemp[i] = 40 + math.Sin(float64(t)*0.1+float64(i))*5 + noise()
			e.cpuUtil[i] = 40 + math.Sin(float64(t)*0.15+float64(i))*20 + noise()*2
		}
	}

	for i := range e.fanRPM {
		id := fmt.Sprintf("fan-%d", i)
		if e.faults.fanDead[id] {
			e.fanRPM[i] = 0
		} else {
			e.fanRPM[i] = 4000 + math.Sin(float64(t)*0.12+float64(i))*300 + noise()*50
		}
	}

	for i := range e.psuOut {
		id := fmt.Sprintf("psu-%d", i)
		if e.faults.psuDead[id] {
			e.psuIn[i] = 0
			e.psuOut[i] = 0
		} else {
			e.psuOut[i] = 240 + math.Sin(float64(t)*0.08+float64(i))*15
			e.psuIn[i] = e.psuOut[i] * 1.08
		}
	}

	for i := range e.dimmCE {
		id := fmt.Sprintf("dimm-%d", i)
		if e.faults.dimmErr[id] {
			if t%3 == 0 {
				e.dimmCE[i]++
			}
			if t%7 == 0 {
				e.dimmUCE[i]++
			}
		}
	}

	e.inletC = 23 + math.Sin(float64(t)*0.05)*2 + noise()*0.2
	e.exhaustC = e.inletC + 12 + avg(e.cpuTemp)*0.15
	targetMem := uint64(float64(e.memTotal) * 0.42)
	e.memUsed = blendU64(e.memUsed, targetMem, 0.05)
}

func (e *Engine) Snapshot() domain.Snapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.buildSnapshot()
}

func (e *Engine) buildSnapshot() domain.Snapshot {
	p := e.cfg.Profile
	th := e.cfg.HealthThresholds
	ts := e.now()

	cpus := make([]domain.CPUState, p.CPUSockets)
	for i := range cpus {
		id := fmt.Sprintf("cpu-%d", i)
		h := deriveCPUHealth(e.cpuTemp[i], e.faults.cpuHot[id], th)
		cpus[i] = domain.CPUState{
			ID:             id,
			Socket:         fmt.Sprintf("CPU%d", i),
			TempCelsius:    round2(e.cpuTemp[i]),
			UtilizationPct: round2(e.cpuUtil[i]),
			Health:         h,
			FaultOverheat:  e.faults.cpuHot[id],
		}
	}

	dimms := make([]domain.DIMMState, p.DIMMCount)
	memHealth := domain.HealthOK
	for i := range dimms {
		id := fmt.Sprintf("dimm-%d", i)
		cap := uint64(64 * 1024 * 1024 * 1024)
		h := domain.HealthOK
		if e.dimmUCE[i] >= th.MemoryUCECritical {
			h = domain.HealthCritical
		} else if e.dimmCE[i] > 10 {
			h = domain.HealthWarning
		}
		if e.faults.dimmErr[id] && h == domain.HealthOK {
			h = domain.HealthWarning
		}
		memHealth = maxHealth(memHealth, h)
		dimms[i] = domain.DIMMState{
			ID:                  id,
			CapacityBytes:       cap,
			Health:              h,
			CorrectableErrors:   e.dimmCE[i],
			UncorrectableErrors: e.dimmUCE[i],
		}
	}

	fans := make([]domain.FanState, p.FanCount)
	thermHealth := domain.HealthOK
	for i := range fans {
		id := fmt.Sprintf("fan-%d", i)
		h := domain.HealthOK
		if e.faults.fanDead[id] || e.fanRPM[i] < th.FanRPMMinHealthy {
			h = domain.HealthCritical
		}
		thermHealth = maxHealth(thermHealth, h)
		fans[i] = domain.FanState{
			ID:        id,
			Name:      fmt.Sprintf("System Fan %d", i+1),
			RPM:       round2(e.fanRPM[i]),
			Health:    h,
			FaultDead: e.faults.fanDead[id],
		}
	}

	sensors := []domain.SensorReading{
		{ID: "inlet-temp", Name: "Inlet Temp", Celsius: round2(e.inletC), ReadingType: "Temperature"},
		{ID: "exhaust-temp", Name: "Exhaust Temp", Celsius: round2(e.exhaustC), ReadingType: "Temperature"},
	}
	for i := range cpus {
		sensors = append(sensors, domain.SensorReading{
			ID:          fmt.Sprintf("cpu-%d-temp", i),
			Name:        fmt.Sprintf("CPU %d Temp", i),
			Celsius:     cpus[i].TempCelsius,
			ReadingType: "Temperature",
		})
	}
	if e.inletC >= th.InletTempWarningC {
		thermHealth = maxHealth(thermHealth, domain.HealthWarning)
	}
	for _, c := range cpus {
		thermHealth = maxHealth(thermHealth, c.Health)
	}

	psus := make([]domain.PSUState, p.PSUCount)
	pwrHealth := domain.HealthOK
	totalW := 0.0
	for i := range psus {
		id := fmt.Sprintf("psu-%d", i)
		h := domain.HealthOK
		if e.faults.psuDead[id] {
			h = domain.HealthCritical
		}
		pwrHealth = maxHealth(pwrHealth, h)
		totalW += e.psuOut[i]
		psus[i] = domain.PSUState{
			ID:           id,
			InputWatts:   round2(e.psuIn[i]),
			OutputWatts:  round2(e.psuOut[i]),
			Health:       h,
			FaultOffline: e.faults.psuDead[id],
		}
	}

	sys := domain.Snapshot{
		Synthetic:        true,
		ServerID:         p.SystemID,
		Manufacturer:     p.Manufacturer,
		Model:            p.Model,
		SystemID:         p.SystemID,
		ChassisID:        p.ChassisID,
		ManagerID:        p.ManagerID,
		PowerState:       "On",
		UptimeSeconds:    uint64(e.now().Sub(e.started).Seconds()),
		SystemPowerWatts: round2(totalW),
		CPUs:             cpus,
		Memory: domain.MemoryState{
			TotalBytes: e.memTotal,
			UsedBytes:  e.memUsed,
			DIMMs:      dimms,
			Health:     memHealth,
		},
		Thermal: domain.ThermalState{
			InletCelsius:   round2(e.inletC),
			ExhaustCelsius: round2(e.exhaustC),
			Fans:           fans,
			Sensors:        sensors,
			Health:         thermHealth,
		},
		Power: domain.PowerState{
			PSUs:       psus,
			TotalWatts: round2(totalW),
			Health:     pwrHealth,
		},
		Firmware:  defaultFirmware(),
		Events:    append([]domain.EventEntry(nil), e.events...),
		UpdatedAt: ts,
	}
	sys.SystemHealth = domain.DeriveSystemHealth(sys)
	return sys
}

func defaultFirmware() []domain.FirmwareRecord {
	return []domain.FirmwareRecord{
		{ID: "bios", Name: "BIOS", Version: "3.4-sim"},
		{ID: "bmc", Name: "BMC", Version: "01.02.03-sim"},
		{ID: "cpld", Name: "CPLD", Version: "1.0-sim"},
	}
}

func deriveCPUHealth(tempC float64, hot bool, th domain.HealthThresholds) domain.HealthStatus {
	if hot {
		if tempC >= th.CPUTempCriticalC {
			return domain.HealthCritical
		}
		return domain.HealthWarning
	}
	if tempC >= th.CPUTempCriticalC {
		return domain.HealthCritical
	}
	if tempC >= th.CPUTempWarningC {
		return domain.HealthWarning
	}
	return domain.HealthOK
}

func maxHealth(a, b domain.HealthStatus) domain.HealthStatus {
	r := map[domain.HealthStatus]int{domain.HealthOK: 0, domain.HealthWarning: 1, domain.HealthCritical: 2}
	if r[b] > r[a] {
		return b
	}
	return a
}

func (e *Engine) appendEvent(severity, msg, origin string) {
	e.eventSeq++
	e.events = append(e.events, domain.EventEntry{
		ID:        fmt.Sprintf("evt-%d", e.eventSeq),
		Created:   e.now(),
		Severity:  severity,
		Message:   msg,
		MessageID: "RackPilot.Synthetic.1.0.Event",
		Origin:    origin,
		Synthetic: true,
	})
	if len(e.events) > 200 {
		e.events = e.events[len(e.events)-200:]
	}
}

// ApplyFanFailure simulates a failed fan.
func (e *Engine) ApplyFanFailure(fanID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.validFan(fanID) {
		return domain.ErrNotFound
	}
	e.faults.fanDead[fanID] = true
	e.appendEvent("Warning", fmt.Sprintf("Synthetic fan failure on %s", fanID), "ThermalSubsystem")
	return nil
}

func (e *Engine) ApplyPSUFailure(psuID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.validPSU(psuID) {
		return domain.ErrNotFound
	}
	e.faults.psuDead[psuID] = true
	e.appendEvent("Critical", fmt.Sprintf("Synthetic PSU failure on %s", psuID), "PowerSubsystem")
	return nil
}

func (e *Engine) ApplyCPUOverheat(cpuID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.validCPU(cpuID) {
		return domain.ErrNotFound
	}
	e.faults.cpuHot[cpuID] = true
	e.appendEvent("Warning", fmt.Sprintf("Synthetic CPU overheating on %s", cpuID), "ProcessorSubsystem")
	return nil
}

func (e *Engine) ApplyMemoryError(dimmID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.validDIMM(dimmID) {
		return domain.ErrNotFound
	}
	e.faults.dimmErr[dimmID] = true
	e.dimmCE[dimmIndex(dimmID)]++
	e.appendEvent("Warning", fmt.Sprintf("Synthetic memory errors on %s", dimmID), "MemorySubsystem")
	return nil
}

func (e *Engine) ResetAll() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.faults.fanDead = map[string]bool{}
	e.faults.psuDead = map[string]bool{}
	e.faults.cpuHot = map[string]bool{}
	e.faults.dimmErr = map[string]bool{}
	for i := range e.dimmCE {
		e.dimmCE[i] = 0
		e.dimmUCE[i] = 0
	}
	e.appendEvent("OK", "Synthetic hardware reset to baseline", "BMC")
}

func (e *Engine) validFan(id string) bool {
	for i := 0; i < e.cfg.Profile.FanCount; i++ {
		if id == fmt.Sprintf("fan-%d", i) {
			return true
		}
	}
	return false
}

func (e *Engine) validPSU(id string) bool {
	for i := 0; i < e.cfg.Profile.PSUCount; i++ {
		if id == fmt.Sprintf("psu-%d", i) {
			return true
		}
	}
	return false
}

func (e *Engine) validCPU(id string) bool {
	for i := 0; i < e.cfg.Profile.CPUSockets; i++ {
		if id == fmt.Sprintf("cpu-%d", i) {
			return true
		}
	}
	return false
}

func (e *Engine) validDIMM(id string) bool {
	for i := 0; i < e.cfg.Profile.DIMMCount; i++ {
		if id == fmt.Sprintf("dimm-%d", i) {
			return true
		}
	}
	return false
}

func dimmIndex(id string) int {
	var i int
	_, _ = fmt.Sscanf(id, "dimm-%d", &i)
	return i
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func avg(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}

func blendU64(cur, target uint64, alpha float64) uint64 {
	return cur + uint64(alpha*float64(int64(target)-int64(cur)))
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
