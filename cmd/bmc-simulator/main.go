package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/anil-debug/rackpilot-ai/simulator/bmc/config"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/engine"
	"github.com/anil-debug/rackpilot-ai/simulator/bmc/server"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "error", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		log.Error("config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var eng *engine.Engine
	if cfg.Mode == config.ModeDynamic {
		eng = engine.New(cfg)
		go eng.Start(ctx)
	}

	srv, err := server.New(cfg, eng, log)
	if err != nil {
		log.Error("server", "error", err)
		os.Exit(1)
	}
	go func() {
		if err := srv.Start(); err != nil {
			log.Error("server", "error", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()
	shCtx, shCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shCancel()
	if err := srv.Shutdown(shCtx); err != nil {
		log.Error("shutdown", "error", err)
		os.Exit(1)
	}
}
