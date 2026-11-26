package service

import (
	"context"
	"time"

	"github.com/anil-debug/rackpilot-ai/investigation/analyze"
	"github.com/anil-debug/rackpilot-ai/investigation/collector"
	"github.com/anil-debug/rackpilot-ai/investigation/config"
	"github.com/anil-debug/rackpilot-ai/investigation/metrics"
	"github.com/anil-debug/rackpilot-ai/investigation/report"
	"github.com/anil-debug/rackpilot-ai/investigation/store"
)

// Investigator runs evidence collection and analysis (read-only).
type Investigator struct {
	cfg       config.Config
	incidents *collector.IncidentClient
	orch      *collector.Orchestrator
	determ    *analyze.Deterministic
	llm       *analyze.LLMAnalyzer
	st        *store.Memory
	metrics   *metrics.Registrar
}

func New(cfg config.Config, st *store.Memory, m *metrics.Registrar) *Investigator {
	return &Investigator{
		cfg:       cfg,
		incidents: collector.NewIncidentClient(cfg.IncidentEngineURL),
		orch:      collector.NewOrchestrator(cfg),
		determ:    &analyze.Deterministic{},
		llm:       analyze.NewLLMAnalyzer(cfg),
		st:        st,
		metrics:   m,
	}
}

func (i *Investigator) Investigate(ctx context.Context, incidentID string) (report.Investigation, error) {
	start := time.Now()
	i.metrics.Requests.Inc()
	defer func() {
		i.metrics.Duration.Observe(time.Since(start).Seconds())
	}()

	incident, err := i.incidents.Get(ctx, incidentID)
	if err != nil {
		i.metrics.Failures.Inc()
		return report.Investigation{}, err
	}

	collectStart := time.Now()
	collected := i.orch.Collect(ctx, incident)
	i.metrics.EvidenceDuration.Observe(time.Since(collectStart).Seconds())
	if collected.PrometheusErr != nil {
		i.metrics.EvidenceErrors.WithLabelValues("prometheus").Inc()
	}
	if collected.RedfishErr != nil {
		i.metrics.EvidenceErrors.WithLabelValues("redfish").Inc()
	}

	rep := i.determ.Investigate(incident, collected)
	if i.cfg.LLMEnabled {
		enriched, llmErr := i.llm.Enrich(ctx, incident, collected, rep)
		if llmErr != nil {
			i.metrics.LLMErrors.Inc()
			rep.Errors = append(rep.Errors, "llm: "+llmErr.Error())
			rep.InvestigationMode = "deterministic_with_llm_fallback"
		} else {
			rep = enriched
			rep.InvestigationMode = "llm"
		}
	}

	if len(collected.Items) == 0 {
		rep.Status = report.StatusFailed
		rep.Errors = append(rep.Errors, "no evidence collected")
		i.metrics.Failures.Inc()
		i.st.Put(rep)
		return rep, nil
	}

	if rep.Status == report.StatusFailed {
		i.metrics.Failures.Inc()
	} else {
		i.metrics.Successes.Inc()
	}
	i.st.Put(rep)
	return rep, nil
}

func (i *Investigator) Get(id string) (report.Investigation, bool) {
	return i.st.Get(id)
}

func (i *Investigator) List(offset, limit int) []report.Investigation {
	return i.st.List(offset, limit)
}
