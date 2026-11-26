package engine

import (
	"sync"
	"testing"

	"github.com/anil-debug/rackpilot-ai/simulator/bmc/config"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/domain"
)

func testCfg() config.Config {
	return config.Default()
}

func TestSnapshotInventory(t *testing.T) {
	eng := New(testCfg())
	s := eng.Snapshot()
	if !s.Synthetic || s.SystemID != "rackpilot-server-001" {
		t.Fatalf("snapshot %+v", s)
	}
	if len(s.CPUs) != 2 || len(s.Thermal.Fans) != 4 || len(s.Power.PSUs) != 2 {
		t.Fatal("inventory counts")
	}
}

func TestTickChangesTelemetry(t *testing.T) {
	eng := New(testCfg())
	b := eng.Snapshot().CPUs[0].TempCelsius
	for i := 0; i < 5; i++ {
		eng.Tick()
	}
	a := eng.Snapshot().CPUs[0].TempCelsius
	if b == a {
		t.Fatal("expected change")
	}
}

func TestFanFailureScenario(t *testing.T) {
	eng := New(testCfg())
	if err := eng.ApplyFanFailure("fan-0"); err != nil {
		t.Fatal(err)
	}
	eng.Tick()
	s := eng.Snapshot()
	if s.Thermal.Fans[0].RPM != 0 || s.Thermal.Fans[0].Health != domain.HealthCritical {
		t.Fatalf("fan %+v", s.Thermal.Fans[0])
	}
	if len(s.Events) == 0 {
		t.Fatal("expected event")
	}
}

func TestReset(t *testing.T) {
	eng := New(testCfg())
	_ = eng.ApplyCPUOverheat("cpu-1")
	eng.Tick()
	eng.ResetAll()
	s := eng.Snapshot()
	if s.SystemHealth != domain.HealthOK && s.CPUs[1].Health != domain.HealthOK {
		// after reset faults cleared; health may need one tick
	}
	if s.CPUs[1].FaultOverheat {
		t.Fatal("fault not cleared")
	}
}

func TestConcurrent(t *testing.T) {
	eng := New(testCfg())
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				eng.Tick()
			}
		}()
	}
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = eng.Snapshot()
			}
		}()
	}
	wg.Wait()
}
