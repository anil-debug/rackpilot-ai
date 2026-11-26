package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/anil-debug/rackpilot-ai/simulator/bmc/config"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/engine"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/fixtures"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/metrics"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/redfish"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/scenario"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	cfg        config.Config
	httpServer *http.Server
}

func New(cfg config.Config, eng *engine.Engine, log *slog.Logger) (*Server, error) {
	reg := prometheus.NewRegistry()
	mux := http.NewServeMux()

	var scn *scenario.Manager
	if cfg.Mode == config.ModeDynamic {
		if eng == nil {
			return nil, fmt.Errorf("dynamic mode requires engine")
		}
		scn = scenario.NewManager(eng)
		metricsReg := metrics.NewRegistrar(reg, eng)
		rf := redfish.NewRouter(cfg.BaseURL, eng.Snapshot)
		rf.Register(mux)
		prom := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
		mux.Handle("GET /metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			metricsReg.CollectFromEngine()
			prom.ServeHTTP(w, r)
		}))
	} else {
		store, err := fixtures.NewStore(cfg.FixtureRoot)
		if err != nil {
			return nil, err
		}
		metrics.RegisterStaticMode(reg)
		redfish.NewStaticRouter(store).Register(mux)
		mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	}

	h := NewHandler(cfg, eng, scn, cfg.ServiceVersion)
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /api/v1/scenarios/fan-failure", h.ScenarioFan)
	mux.HandleFunc("POST /api/v1/scenarios/psu-failure", h.ScenarioPSU)
	mux.HandleFunc("POST /api/v1/scenarios/cpu-overheating", h.ScenarioCPU)
	mux.HandleFunc("POST /api/v1/scenarios/memory-error", h.ScenarioMemory)
	mux.HandleFunc("POST /api/v1/scenarios/reset", h.ScenarioReset)

	return &Server{
		cfg: cfg,
		httpServer: &http.Server{
			Addr:              cfg.ListenAddr,
			Handler:           loggingMiddleware(log, mux),
			ReadHeaderTimeout: 5 * time.Second,
		},
	}, nil
}

func (s *Server) Start() error {
	slog.Info("bmc simulator listening", "addr", s.cfg.ListenAddr, "mode", s.cfg.Mode)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) HTTPHandler() http.Handler {
	return s.httpServer.Handler
}
