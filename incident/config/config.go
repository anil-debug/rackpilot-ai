package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// Config holds incident engine runtime settings.
type Config struct {
	ListenAddr        string
	PrometheusURL     string
	PollInterval      time.Duration
	CorrelationWindow time.Duration
	RecoveryWindow    time.Duration
	ServiceVersion    string

	HealthCritical   float64
	GPUTempWarnC     float64
	CPUTempWarnC     float64
	MemPressureRatio float64
}

// Default returns development defaults.
func Default() Config {
	return Config{
		ListenAddr:        ":8083",
		PrometheusURL:     "http://prometheus:9090",
		PollInterval:      15 * time.Second,
		CorrelationWindow: 5 * time.Minute,
		RecoveryWindow:    2 * time.Minute,
		ServiceVersion:    "0.1.0",
		HealthCritical:    2,
		GPUTempWarnC:      75,
		CPUTempWarnC:      70,
		MemPressureRatio:  0.85,
	}
}

// FromFlags parses env and flags.
func FromFlags() (Config, error) {
	cfg := Default()
	flag.StringVar(&cfg.ListenAddr, "listen", envOr("RACKPILOT_INCIDENT_LISTEN", cfg.ListenAddr), "HTTP listen address")
	flag.StringVar(&cfg.PrometheusURL, "prometheus-url", envOr("RACKPILOT_PROMETHEUS_URL", cfg.PrometheusURL), "Prometheus base URL")
	flag.DurationVar(&cfg.PollInterval, "poll-interval", durationEnv("RACKPILOT_INCIDENT_POLL_MS", cfg.PollInterval), "metric poll interval")
	flag.Parse()
	if cfg.PrometheusURL == "" {
		return cfg, fmt.Errorf("prometheus URL required")
	}
	return cfg, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func durationEnv(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	ms, err := time.ParseDuration(v)
	if err == nil {
		return ms
	}
	// also accept plain milliseconds integer
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
		return time.Duration(n) * time.Millisecond
	}
	return def
}
