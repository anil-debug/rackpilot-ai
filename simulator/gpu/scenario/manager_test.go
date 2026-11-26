package scenario

import (
	"testing"

	"github.com/anil-debug/rackpilot-ai/simulator/gpu/config"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/engine"
)

func TestScenarioManagerValidation(t *testing.T) {
	cfg := config.Default()
	cfg.GPUCount = 4
	eng := engine.New(cfg)
	m := NewManager(eng)

	if err := m.Overheating(""); err == nil {
		t.Fatal("expected error for empty gpu_id")
	}
	if err := m.Overheating("gpu-9"); err == nil {
		t.Fatal("expected not found")
	}
	if err := m.Overheating("gpu-1"); err != nil {
		t.Fatal(err)
	}
}
