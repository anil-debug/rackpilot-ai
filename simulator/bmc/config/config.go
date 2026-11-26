package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/anil-debug/rackpilot-ai/simulator/bmc/domain"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/fixtures"
)

const defaultListen = ":8081"

type Config struct {
	ListenAddr       string
	ServiceVersion   string
	Mode             SimulationMode
	FixtureRoot      string
	Profile          domain.ServerProfile
	HealthThresholds domain.HealthThresholds
	TickInterval     time.Duration
	SimSeed          int64
	ShutdownTimeout  time.Duration
	BaseURL          string // Redfish @odata host prefix
}

func Default() Config {
	return Config{
		ListenAddr:       defaultListen,
		ServiceVersion:   "0.1.0",
		Mode:             ModeDynamic,
		FixtureRoot:      fixtures.DefaultDir(),
		Profile:          domain.DefaultServerProfile(),
		HealthThresholds: domain.DefaultHealthThresholds(),
		TickInterval:     2 * time.Second,
		SimSeed:          42,
		ShutdownTimeout:  10 * time.Second,
		BaseURL:          "",
	}
}

func Load() (Config, error) {
	cfg := Default()
	listen := flag.String("listen", envOr("RACKPILOT_BMC_LISTEN", cfg.ListenAddr), "HTTP listen address")
	tickMs := flag.Int("tick-ms", envIntOr("RACKPILOT_BMC_TICK_MS", int(cfg.TickInterval.Milliseconds())), "tick interval ms")
	seed := flag.Int64("sim-seed", envInt64Or("RACKPILOT_BMC_SIM_SEED", cfg.SimSeed), "RNG seed")
	model := flag.String("server-model", envOr("RACKPILOT_BMC_SERVER_MODEL", cfg.Profile.Model), "synthetic server model")
	base := flag.String("redfish-base-url", envOr("RACKPILOT_BMC_REDFISH_BASE", ""), "optional Redfish URL prefix")
	modeStr := flag.String("mode", envOr("RACKPILOT_BMC_MODE", string(cfg.Mode)), "simulation mode: dynamic or static")
	fixRoot := flag.String("fixture-root", envOr("RACKPILOT_BMC_FIXTURE_ROOT", cfg.FixtureRoot), "DMTF mockup directory")
	flag.Parse()

	cfg.ListenAddr = *listen
	cfg.TickInterval = time.Duration(*tickMs) * time.Millisecond
	cfg.SimSeed = *seed
	cfg.BaseURL = *base
	cfg.Profile.Model = *model
	cfg.FixtureRoot = *fixRoot
	mode, err := ParseMode(*modeStr)
	if err != nil {
		return Config{}, err
	}
	cfg.Mode = mode
	return cfg, nil
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

// Validate ensures config is usable.
func (c Config) Validate() error {
	if c.Mode == ModeDynamic && c.TickInterval < 100*time.Millisecond {
		return fmt.Errorf("tick interval too small")
	}
	if c.Mode == ModeStatic {
		if _, err := fixtures.NewStore(c.FixtureRoot); err != nil {
			return err
		}
	}
	return nil
}
