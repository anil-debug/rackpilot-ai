package config

import "fmt"

// SimulationMode selects dynamic synthetic engine vs static DMTF fixtures.
type SimulationMode string

const (
	ModeDynamic SimulationMode = "dynamic"
	ModeStatic  SimulationMode = "static"
)

func ParseMode(s string) (SimulationMode, error) {
	switch SimulationMode(s) {
	case ModeDynamic, ModeStatic, "":
		if s == "" {
			return ModeDynamic, nil
		}
		return SimulationMode(s), nil
	default:
		return "", fmt.Errorf("invalid simulation mode %q (use dynamic or static)", s)
	}
}
