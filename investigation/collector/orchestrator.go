package collector

import (
	"context"
	"fmt"
	"sort"
	"time"

	inc "github.com/anil-debug/rackpilot-ai/incident/incident"
	"github.com/anil-debug/rackpilot-ai/investigation/config"
	"github.com/anil-debug/rackpilot-ai/investigation/evidence"
)

// Result holds evidence and per-provider errors.
type Result struct {
	Items         []evidence.Item
	PrometheusErr error
	RedfishErr    error
	Sources       []string
	BMCMode       string
}

// Orchestrator collects evidence for an incident.
type Orchestrator struct {
	cfg     config.Config
	prom    *PromClient
	redfish *RedfishCollector
}

func NewOrchestrator(cfg config.Config) *Orchestrator {
	return &Orchestrator{
		cfg:     cfg,
		prom:    NewPromClient(cfg.PrometheusURL),
		redfish: NewRedfishCollector(cfg.BMCBaseURL),
	}
}

var promQueriesForRule = map[string][]string{
	"hardware_cooling": {
		`rackpilot_bmc_fan_health_status`,
		`rackpilot_bmc_fan_speed_rpm`,
		`rackpilot_bmc_cpu_temperature_celsius`,
		`rackpilot_gpu_temperature_celsius`,
	},
	"power_supply": {
		`rackpilot_bmc_psu_health_status`,
		`rackpilot_bmc_system_health_status`,
		`kube_deployment_status_replicas_available{namespace="rackpilot"}`,
	},
	"kubernetes_workload": {
		`kube_pod_container_status_restarts_total{namespace="rackpilot"}`,
		`kube_deployment_status_replicas_available{namespace="rackpilot"}`,
		`kube_deployment_spec_replicas{namespace="rackpilot"}`,
	},
	"memory_pressure": {
		`rackpilot_bmc_memory_errors_total`,
		`container_memory_working_set_bytes{namespace="rackpilot",container!="",image!=""}`,
	},
}

func (o *Orchestrator) Collect(ctx context.Context, incident inc.Incident) Result {
	start := incident.FirstSeen.Add(-o.cfg.EvidenceBefore)
	end := incident.LastSeen.Add(o.cfg.EvidenceAfter)
	if end.Before(start) {
		end = time.Now().UTC()
	}
	queries := promQueriesForRule[incident.CorrelationRule]
	if len(queries) == 0 {
		queries = promQueriesForRule["hardware_cooling"]
	}

	var all []evidence.Item
	var promErr error
	promOK := 0
	sources := []string{}
	for _, q := range queries {
		items, err := o.prom.QueryRange(ctx, q, start, end, 30*time.Second)
		if err != nil {
			if promErr == nil {
				promErr = err
			}
			continue
		}
		promOK++
		all = append(all, items...)
	}
	if promOK > 0 {
		sources = append(sources, "prometheus")
	}

	rfItems, mode, rfErr := o.redfish.Collect(ctx)
	if rfErr == nil {
		all = append(all, rfItems...)
		sources = append(sources, "redfish")
	}

	// Seed from incident contributing signals
	for _, s := range incident.ContributingSignals {
		all = append(all, evidence.Item{
			Source:      string(s.Source),
			Timestamp:   s.Timestamp,
			Resource:    s.Component,
			Name:        s.MetricName,
			Value:       fmt.Sprintf("%g", s.MetricValue),
			Description: s.ObservedFact,
			Provenance:  evidence.ProvenanceSyntheticLive,
			Reference:   s.EvidenceReference,
			Labels:      s.Labels,
		})
	}
	all = renumberEvidence(all)
	sort.Slice(all, func(i, j int) bool { return all[i].Timestamp.Before(all[j].Timestamp) })

	return Result{
		Items:         all,
		PrometheusErr: promErr,
		RedfishErr:    rfErr,
		Sources:       sources,
		BMCMode:       mode,
	}
}

func renumberEvidence(items []evidence.Item) []evidence.Item {
	for i := range items {
		items[i].EvidenceID = fmt.Sprintf("E%03d", i+1)
	}
	return items
}
