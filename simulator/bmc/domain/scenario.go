package domain

type ScenarioType string

const (
	ScenarioNone        ScenarioType = ""
	ScenarioFanFailure  ScenarioType = "fan_failure"
	ScenarioPSUFailure  ScenarioType = "psu_failure"
	ScenarioCPUOverheat ScenarioType = "cpu_overheating"
	ScenarioMemoryError ScenarioType = "memory_error"
)
