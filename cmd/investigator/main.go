package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anil-debug/rackpilot-ai/investigation/api"
	"github.com/anil-debug/rackpilot-ai/investigation/config"
	"github.com/anil-debug/rackpilot-ai/investigation/metrics"
	"github.com/anil-debug/rackpilot-ai/investigation/service"
	"github.com/anil-debug/rackpilot-ai/investigation/store"
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

	mem := store.NewMemory()
	reg := prom.NewRegistry()
	mreg := metrics.NewRegistrar(reg)
	svc := service.New(cfg, mem, mreg)
	h := api.NewHandler(cfg, svc)

	mux := http.NewServeMux()
	h.Register(mux)
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("investigator listening",
			"addr", cfg.ListenAddr,
			"mode", cfg.InvestigatorMode,
			"incident_engine", cfg.IncidentEngineURL,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
