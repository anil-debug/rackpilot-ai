package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/anil-debug/rackpilot-ai/simulator/gpu/config"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/engine"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/metrics"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/scenario"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server is the HTTP server for the GPU simulator.
type Server struct {
	cfg        config.Config
	httpServer *http.Server
}

// New builds the HTTP server and routes.
func New(cfg config.Config, eng *engine.Engine, log *slog.Logger) *Server {
	reg := prometheus.NewRegistry()
	metricsReg := metrics.NewRegistrar(reg, eng)
	scn := scenario.NewManager(eng)
	h := NewHandler(eng, scn, cfg.ServiceVersion)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /api/v1/gpus", h.ListGPUs)
	mux.HandleFunc("GET /api/v1/gpus/{id}", h.GetGPU)
	mux.HandleFunc("POST /api/v1/scenarios/overheating", h.ScenarioOverheating)
	mux.HandleFunc("POST /api/v1/scenarios/memory-exhaustion", h.ScenarioMemoryExhaustion)
	mux.HandleFunc("POST /api/v1/scenarios/reset", h.ScenarioReset)

	prom := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	mux.Handle("GET /metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metricsReg.CollectFromEngine()
		prom.ServeHTTP(w, r)
	}))

	wrapped := loggingMiddleware(log, mux)

	return &Server{
		cfg: cfg,
		httpServer: &http.Server{
			Addr:              cfg.ListenAddr,
			Handler:           wrapped,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Start listens and serves HTTP until shutdown.
func (s *Server) Start() error {
	slog.Info("gpu simulator listening", "addr", s.cfg.ListenAddr, "synthetic_telemetry", true)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
