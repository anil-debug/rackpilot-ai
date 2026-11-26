package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/anil-debug/rackpilot-ai/simulator/gpu/config"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/engine"
	"github.com/anil-debug/rackpilot-ai/simulator/gpu/server"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		log.Error("configuration error", "error", err)
		os.Exit(1)
	}

	eng := engine.New(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go eng.Start(ctx)

	srv := server.New(cfg, eng, log)

	go func() {
		if err := srv.Start(); err != nil {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Info("shutdown signal received")

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", "error", err)
		os.Exit(1)
	}
	log.Info("shutdown complete")
}
