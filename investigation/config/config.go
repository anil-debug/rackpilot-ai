package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

type Config struct {
	ListenAddr        string
	PrometheusURL     string
	IncidentEngineURL string
	BMCBaseURL        string
	ServiceVersion    string
	EvidenceBefore    time.Duration
	EvidenceAfter     time.Duration
	LLMEnabled        bool
	LLMBaseURL        string
	LLMModel          string
	LLMAPIKey         string
	LLMTimeout        time.Duration
	InvestigatorMode  string
}

func Default() Config {
	return Config{
		ListenAddr:        ":8084",
		PrometheusURL:     "http://prometheus:9090",
		IncidentEngineURL: "http://incident-engine:8083",
		BMCBaseURL:        "http://bmc-simulator:8081",
		ServiceVersion:    "0.1.0",
		EvidenceBefore:    10 * time.Minute,
		EvidenceAfter:     5 * time.Minute,
		LLMEnabled:        false,
		LLMBaseURL:        "https://api.openai.com/v1",
		LLMModel:          "gpt-4o-mini",
		LLMTimeout:        45 * time.Second,
		InvestigatorMode:  "deterministic",
	}
}

func FromFlags() (Config, error) {
	cfg := Default()
	flag.StringVar(&cfg.ListenAddr, "listen", envOr("RACKPILOT_INVESTIGATOR_LISTEN", cfg.ListenAddr), "HTTP listen")
	flag.StringVar(&cfg.PrometheusURL, "prometheus-url", envOr("RACKPILOT_PROMETHEUS_URL", cfg.PrometheusURL), "Prometheus URL")
	flag.StringVar(&cfg.IncidentEngineURL, "incident-url", envOr("RACKPILOT_INCIDENT_ENGINE_URL", cfg.IncidentEngineURL), "Incident engine URL")
	flag.StringVar(&cfg.BMCBaseURL, "bmc-url", envOr("RACKPILOT_BMC_URL", cfg.BMCBaseURL), "BMC simulator URL")
	flag.StringVar(&cfg.InvestigatorMode, "mode", envOr("RACKPILOT_INVESTIGATOR_MODE", cfg.InvestigatorMode), "deterministic or llm")
	flag.Parse()
	if v := os.Getenv("RACKPILOT_EVIDENCE_BEFORE"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.EvidenceBefore = d
		}
	}
	if v := os.Getenv("RACKPILOT_EVIDENCE_AFTER"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.EvidenceAfter = d
		}
	}
	if os.Getenv("RACKPILOT_LLM_API_KEY") != "" {
		cfg.LLMAPIKey = os.Getenv("RACKPILOT_LLM_API_KEY")
		cfg.LLMEnabled = cfg.InvestigatorMode == "llm" && cfg.LLMAPIKey != ""
	}
	if cfg.PrometheusURL == "" || cfg.IncidentEngineURL == "" {
		return cfg, fmt.Errorf("prometheus and incident engine URLs required")
	}
	return cfg, nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
