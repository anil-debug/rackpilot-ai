package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/anil-debug/rackpilot-ai/incident/config"
	"github.com/anil-debug/rackpilot-ai/incident/correlate"
	"github.com/anil-debug/rackpilot-ai/incident/ingest"
	"github.com/anil-debug/rackpilot-ai/incident/metrics"
	"github.com/anil-debug/rackpilot-ai/incident/store"
)

// Runner polls Prometheus, correlates signals, and updates incident storage.
type Runner struct {
	cfg        config.Config
	ingest     *ingest.Collector
	correlator *correlate.Engine
	store      *store.Memory
	reg        *metrics.Registrar

	mu        sync.RWMutex
	lastError string
	lastRun   time.Time
}

func NewRunner(cfg config.Config, coll *ingest.Collector, correlator *correlate.Engine, st *store.Memory, reg *metrics.Registrar) *Runner {
	return &Runner{
		cfg:        cfg,
		ingest:     coll,
		correlator: correlator,
		store:      st,
		reg:        reg,
	}
}

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()
	r.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *Runner) tick(ctx context.Context) {
	now := time.Now().UTC()
	signals, err := r.ingest.Collect(ctx)
	if err != nil {
		r.mu.Lock()
		r.lastError = err.Error()
		r.lastRun = now
		r.mu.Unlock()
		r.reg.Errors.Inc()
		slog.Warn("incident ingest failed", "err", err)
		// Do not run correlation or recovery on failed ingest — avoids false resolution.
		return
	}
	r.reg.Signals.Add(float64(len(signals)))
	results := r.correlator.Evaluate(signals, now)
	r.reg.Evaluations.Inc()
	opened, updated, resolved := r.store.ApplyTick(results, now, r.cfg.RecoveryWindow)
	r.reg.Created.Add(float64(opened))
	r.reg.Resolved.Add(float64(resolved))
	r.reg.Active.Set(float64(r.store.CountActive()))
	r.mu.Lock()
	r.lastError = ""
	r.lastRun = now
	r.mu.Unlock()
	if opened+updated+resolved > 0 {
		slog.Info("incident tick", "signals", len(signals), "opened", opened, "updated", updated, "resolved", resolved)
	}
	_ = updated
}

func (r *Runner) Store() *store.Memory { return r.store }

func (r *Runner) LastRun() (time.Time, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastRun, r.lastError
}
