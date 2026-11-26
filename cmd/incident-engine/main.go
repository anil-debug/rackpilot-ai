package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anil-debug/rackpilot-ai/incident/api"
	"github.com/anil-debug/rackpilot-ai/incident/collector/prometheus"
	"github.com/anil-debug/rackpilot-ai/incident/config"
	"github.com/anil-debug/rackpilot-ai/incident/correlate"
	eng "github.com/anil-debug/rackpilot-ai/incident/engine"
	"github.com/anil-debug/rackpilot-ai/incident/ingest"
	"github.com/anil-debug/rackpilot-ai/incident/metrics"
	"github.com/anil-debug/rackpilot-ai/incident/store"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg, err := config.FromFlags()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	promClient := prometheus.NewHTTPClient(cfg.PrometheusURL)
	collector := ingest.New(promClient, cfg)
	correlator := correlate.New(cfg.CorrelationWindow)
	mem := store.NewMemory()
	reg := prom.NewRegistry()
	mreg := metrics.NewRegistrar(reg)
	runner := eng.NewRunner(cfg, collector, correlator, mem, mreg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.Run(ctx)

	h := api.NewHandler(cfg, mem, func() (time.Time, string) {
		return runner.LastRun()
	})
	mux := http.NewServeMux()
	h.Register(mux)
	mux.Handle("GET /api/v1/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("incident engine listening", "addr", cfg.ListenAddr, "prometheus", cfg.PrometheusURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	cancel()
	_ = srv.Shutdown(shutdownCtx)
}
