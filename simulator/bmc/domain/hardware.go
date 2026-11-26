package domain

import "time"

// ServerProfile configurable synthetic inventory (not real hardware).
type ServerProfile struct {
	Manufacturer string
	Model        string
	SystemID     string
	ChassisID    string
	ManagerID    string
	CPUSockets   int
	DIMMCount    int
	FanCount     int
	PSUCount     int
}

// DefaultServerProfile returns synthetic Supermicro-class inventory defaults.
func DefaultServerProfile() ServerProfile {
	return ServerProfile{
		Manufacturer: "Supermicro",
		Model:        "SYS-210P-FakeSim",
		SystemID:     "rackpilot-server-001",
		ChassisID:    "rackpilot-chassis-001",
		ManagerID:    "BMC",
		CPUSockets:   2,
		DIMMCount:    8,
		FanCount:     4,
		PSUCount:     2,
	}
}

// HealthThresholds for deriving component health from telemetry.
type HealthThresholds struct {
	CPUTempWarningC   float64
	CPUTempCriticalC  float64
	InletTempWarningC float64
	MemoryUCECritical uint64
	FanRPMMinHealthy  float64
}

func DefaultHealthThresholds() HealthThresholds {
	return HealthThresholds{
		CPUTempWarningC:   75,
		CPUTempCriticalC:  90,
		InletTempWarningC: 35,
		MemoryUCECritical: 1,
		FanRPMMinHealthy:  2000,
	}
}

// Snapshot is the full synthetic server state for APIs and metrics.
type Snapshot struct {
	Synthetic        bool
	ServerID         string
	Manufacturer     string
	Model            string
	SystemID         string
	ChassisID        string
	ManagerID        string
	PowerState       string
	UptimeSeconds    uint64
	SystemHealth     HealthStatus
	SystemPowerWatts float64
	CPUs             []CPUState
	Memory           MemoryState
	Thermal          ThermalState
	Power            PowerState
	Firmware         []FirmwareRecord
	Events           []EventEntry
	UpdatedAt        time.Time
}

type CPUState struct {
	ID             string
	Socket         string
	TempCelsius    float64
	UtilizationPct float64
	Health         HealthStatus
	FaultOverheat  bool
}

type DIMMState struct {
	ID                  string
	CapacityBytes       uint64
	Health              HealthStatus
	CorrectableErrors   uint64
	UncorrectableErrors uint64
}

type MemoryState struct {
	TotalBytes uint64
	UsedBytes  uint64
	DIMMs      []DIMMState
	Health     HealthStatus
}

type FanState struct {
	ID        string
	Name      string
	RPM       float64
	Health    HealthStatus
	FaultDead bool
}

type ThermalState struct {
	InletCelsius   float64
	ExhaustCelsius float64
	Fans           []FanState
	Sensors        []SensorReading
	Health         HealthStatus
}

type SensorReading struct {
	ID          string
	Name        string
	Celsius     float64
	ReadingType string
}

type PSUState struct {
	ID           string
	InputWatts   float64
	OutputWatts  float64
	Health       HealthStatus
	FaultOffline bool
}

type PowerState struct {
	PSUs       []PSUState
	TotalWatts float64
	Health     HealthStatus
}

type FirmwareRecord struct {
	ID      string
	Name    string
	Version string
}

type EventEntry struct {
	ID        string
	Created   time.Time
	Severity  string
	Message   string
	MessageID string
	Origin    string
	Synthetic bool
}

// DeriveSystemHealth aggregates worst component health.
func DeriveSystemHealth(s Snapshot) HealthStatus {
	worst := HealthOK
	for _, c := range s.CPUs {
		worst = maxHealth(worst, c.Health)
	}
	worst = maxHealth(worst, s.Memory.Health)
	worst = maxHealth(worst, s.Thermal.Health)
	worst = maxHealth(worst, s.Power.Health)
	return worst
}
