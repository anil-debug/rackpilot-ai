package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/anil-debug/rackpilot-ai/simulator/gpu/domain"
)

const defaultListen = ":8080"

// Config holds runtime configuration for the GPU telemetry simulator.
type Config struct {
	ListenAddr       string
	GPUCount         int
	Profile          domain.GPUProfile
	HealthThresholds domain.HealthThresholds
	TickInterval     time.Duration
	SimSeed          int64
	ShutdownTimeout  time.Duration
	ServiceVersion   string
	// Scenario ramp rates (per simulation step / tick).
	OverheatTempDeltaCelsius    float64
	MemoryExhaustionTargetRatio float64
	MemoryRampBytesPerTick      uint64
}

// Default returns configuration with H100 profile and four GPUs.
func Default() Config {
	return Config{
		ListenAddr:                  defaultListen,
		GPUCount:                    4,
		Profile:                     domain.DefaultH100Profile(),
		HealthThresholds:            domain.DefaultHealthThresholds(),
		TickInterval:                2 * time.Second,
		SimSeed:                     42,
		ShutdownTimeout:             10 * time.Second,
		ServiceVersion:              "0.1.0",
		OverheatTempDeltaCelsius:    2.5,
		MemoryExhaustionTargetRatio: 0.98,
		MemoryRampBytesPerTick:      2 * 1024 * 1024 * 1024, // 2 GiB per tick toward cap
	}
}

// Load parses flags and environment variables (env overrides defaults; flags override env where set).
func Load() (Config, error) {
	cfg := Default()

	listen := flag.String("listen", envOr("RACKPILOT_LISTEN", cfg.ListenAddr), "HTTP listen address")
	gpuCount := flag.Int("gpu-count", envIntOr("RACKPILOT_GPU_COUNT", cfg.GPUCount), "number of simulated GPUs")
	tickMs := flag.Int("tick-ms", envIntOr("RACKPILOT_TICK_MS", int(cfg.TickInterval.Milliseconds())), "simulation tick interval in milliseconds")
	seed := flag.Int64("sim-seed", envInt64Or("RACKPILOT_SIM_SEED", cfg.SimSeed), "simulation RNG seed")
	model := flag.String("gpu-model", envOr("RACKPILOT_GPU_MODEL", cfg.Profile.Model), "simulated GPU model name (profile selector)")
	flag.Parse()

	cfg.ListenAddr = *listen
	cfg.GPUCount = *gpuCount
	cfg.TickInterval = time.Duration(*tickMs) * time.Millisecond
	cfg.SimSeed = *seed

	profile, err := ProfileByModelName(*model)
	if err != nil {
		return Config{}, err
	}
	cfg.Profile = profile

	if cfg.GPUCount < 1 || cfg.GPUCount > 64 {
		return Config{}, fmt.Errorf("gpu-count must be between 1 and 64, got %d", cfg.GPUCount)
	}
	if cfg.TickInterval < 100*time.Millisecond {
		return Config{}, fmt.Errorf("tick interval too small")
	}

	return cfg, nil
}

// ProfileByModelName selects a known profile or builds a custom profile from the model string with H100-like defaults.
func ProfileByModelName(name string) (domain.GPUProfile, error) {
	switch name {
	case "", domain.DefaultH100Profile().Model:
		return domain.DefaultH100Profile(), nil
	case "NVIDIA A100 80GB PCIe":
		p := domain.DefaultH100Profile()
		p.Model = name
		p.PowerLimitWatts = 400
		p.BaselinePowerWatts = 250
		p.MemoryTotalBytes = 80 * 1024 * 1024 * 1024
		return p, nil
	case "NVIDIA B200":
		p := domain.DefaultH100Profile()
		p.Model = name
		p.PowerLimitWatts = 1000
		p.BaselinePowerWatts = 350
		p.MemoryTotalBytes = 192 * 1024 * 1024 * 1024
		return p, nil
	default:
		// Unknown model: use H100 characteristics but honor the model label for future profiles.
		p := domain.DefaultH100Profile()
		p.Model = name
		return p, nil
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envInt64Or(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}
