package engine

import (
	"sync"
	"testing"

	"github.com/anil-debug/rackpilot-ai/simulator/gpu/config"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/domain"
)

func testConfig() config.Config {
	cfg := config.Default()
	cfg.GPUCount = 4
	cfg.SimSeed = 1
	return cfg
}

func TestGPUInitialization(t *testing.T) {
	eng := New(testConfig())
	snap := eng.Snapshot()
	if len(snap) != 4 {
		t.Fatalf("want 4 gpus got %d", len(snap))
	}
	for _, g := range snap {
		if g.ID == "" || !g.SyntheticTelemetry {
			t.Fatalf("invalid gpu %+v", g)
		}
		if g.MemoryUsedBytes > g.MemoryTotalBytes {
			t.Fatal("memory used exceeds total")
		}
	}
}

func TestGetGPU(t *testing.T) {
	eng := New(testConfig())
	_, ok := eng.Get("gpu-0")
	if !ok {
		t.Fatal("expected gpu-0")
	}
	_, ok = eng.Get("gpu-99")
	if ok {
		t.Fatal("expected missing gpu")
	}
}

func TestTelemetryUpdatesOnTick(t *testing.T) {
	eng := New(testConfig())
	before := eng.Snapshot()[0].UtilizationPercent
	for i := 0; i < 5; i++ {
		eng.Tick()
	}
	after := eng.Snapshot()[0].UtilizationPercent
	if before == after {
		t.Fatalf("expected utilization change before=%v after=%v", before, after)
	}
}

func TestOverheatingScenario(t *testing.T) {
	eng := New(testConfig())
	otherBefore := eng.Snapshot()[1].TemperatureCelsius
	if err := eng.SetScenario("gpu-0", domain.ScenarioOverheating); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		eng.Tick()
	}
	snap := eng.Snapshot()
	if snap[0].TemperatureCelsius < domain.DefaultHealthThresholds().TempWarningCelsius {
		t.Fatalf("expected elevated temp got %v", snap[0].TemperatureCelsius)
	}
	if snap[0].HealthStatus == domain.HealthHealthy {
		t.Fatal("expected non-healthy")
	}
	if snap[1].TemperatureCelsius == otherBefore && snap[1].HealthStatus != domain.HealthHealthy {
		// other gpu should remain in healthy band; temp may drift slightly but not scenario-driven
	}
	if snap[1].HealthStatus != domain.HealthHealthy {
		t.Fatalf("other gpu should stay healthy, got %v", snap[1].HealthStatus)
	}
}

func TestMemoryExhaustionScenario(t *testing.T) {
	eng := New(testConfig())
	if err := eng.SetScenario("gpu-2", domain.ScenarioMemoryExhaustion); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		eng.Tick()
	}
	g, _ := eng.Get("gpu-2")
	if g.MemoryUsedBytes > g.MemoryTotalBytes {
		t.Fatal("memory used must not exceed total")
	}
	ratio := float64(g.MemoryUsedBytes) / float64(g.MemoryTotalBytes)
	if ratio < 0.9 {
		t.Fatalf("expected high memory ratio got %v", ratio)
	}
	if g.HealthStatus == domain.HealthHealthy {
		t.Fatal("expected warning or critical")
	}
}

func TestReset(t *testing.T) {
	eng := New(testConfig())
	_ = eng.SetScenario("gpu-0", domain.ScenarioOverheating)
	for i := 0; i < 15; i++ {
		eng.Tick()
	}
	n := eng.ResetAll()
	if n != 4 {
		t.Fatalf("reset count %d", n)
	}
	snap := eng.Snapshot()
	for _, g := range snap {
		if g.HealthStatus != domain.HealthHealthy {
			t.Fatalf("%s health %v", g.ID, g.HealthStatus)
		}
	}
}

func TestConcurrentSnapshotAndTick(t *testing.T) {
	eng := New(testConfig())
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				eng.Tick()
			}
		}()
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = eng.Snapshot()
			}
		}()
	}
	wg.Wait()
}
