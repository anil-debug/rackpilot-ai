package correlate

import (
	"fmt"
	"strings"
	"time"

	inc "github.com/anil-debug/rackpilot-ai/incident/incident"
	"github.com/anil-debug/rackpilot-ai/incident/signal"
)

const disclaimer = "Suspected root cause is a hypothesis based on co-occurring symptoms within the correlation window; temporal correlation alone is not confirmed causation."

// Result is a correlation outcome for one evaluation tick.
type Result struct {
	RuleID    string
	DedupeKey string
	Incident  inc.Incident
	Active    bool
}

// Engine applies deterministic rules to normalized signals.
type Engine struct {
	Window time.Duration
}

func New(window time.Duration) *Engine {
	return &Engine{Window: window}
}

// Evaluate returns active rule matches (may be multiple rules).
func (e *Engine) Evaluate(signals []signal.Signal, now time.Time) []Result {
	windowStart := now.Add(-e.Window)
	inWindow := filterSince(signals, windowStart)

	var results []Result
	if r := e.ruleCooling(inWindow, now); r.Active {
		results = append(results, r)
	}
	if r := e.rulePSU(inWindow, now); r.Active {
		results = append(results, r)
	}
	if r := e.ruleK8sWorkload(inWindow, now); r.Active {
		results = append(results, r)
	}
	if r := e.ruleMemoryPressure(inWindow, now); r.Active {
		results = append(results, r)
	}
	return results
}

func (e *Engine) ruleCooling(signals []signal.Signal, now time.Time) Result {
	fanCrit := filterMetric(signals, "rackpilot_bmc_fan_health_status", signal.SeverityCritical)
	cpuHot := filterMetric(signals, "rackpilot_bmc_cpu_temperature_celsius", signal.SeverityWarning)
	gpuHot := filterMetric(signals, "rackpilot_gpu_temperature_celsius", signal.SeverityWarning)

	if len(fanCrit) == 0 {
		return Result{RuleID: "hardware_cooling", Active: false}
	}
	key := "hardware_cooling:" + firstComponent(fanCrit)
	var contrib []signal.Signal
	var facts, symptoms []string
	contrib = append(contrib, fanCrit...)
	for _, s := range fanCrit {
		facts = append(facts, s.ObservedFact)
	}
	symptoms = append(symptoms, "BMC fan health reported critical")
	if len(cpuHot) > 0 {
		contrib = append(contrib, cpuHot...)
		symptoms = append(symptoms, "BMC CPU temperature elevated in same window")
	}
	if len(gpuHot) > 0 {
		contrib = append(contrib, gpuHot...)
		symptoms = append(symptoms, "GPU temperature elevated in same window")
	}
	conf := "medium"
	if len(cpuHot) > 0 && len(gpuHot) > 0 {
		conf = "high"
	}
	cause := "Cooling impairment is suspected because fan critical status coincides with elevated CPU/GPU thermal readings."
	if len(cpuHot) == 0 && len(gpuHot) == 0 {
		cause = "Fan critical status observed; thermal rise not yet confirmed in window."
		conf = "low"
	}
	return Result{
		RuleID:    "hardware_cooling",
		DedupeKey: key,
		Active:    true,
		Incident:  buildIncident("hardware_cooling", key, "Hardware cooling incident", cause, inc.SeverityHigh, conf, contrib, facts, symptoms, cause, now),
	}
}

func (e *Engine) rulePSU(signals []signal.Signal, now time.Time) Result {
	psuCrit := filterMetric(signals, "rackpilot_bmc_psu_health_status", signal.SeverityCritical)
	sysCrit := filterMetric(signals, "rackpilot_bmc_system_health_status", signal.SeverityCritical)
	avail := filterType(signals, signal.TypeAvailabilityGap)

	if len(psuCrit) == 0 {
		return Result{RuleID: "power_supply", Active: false}
	}
	key := "power_supply:" + firstComponent(psuCrit)
	var contrib []signal.Signal
	var facts, symptoms []string
	contrib = append(contrib, psuCrit...)
	for _, s := range psuCrit {
		facts = append(facts, s.ObservedFact)
	}
	symptoms = append(symptoms, "PSU health critical")
	if len(sysCrit) > 0 {
		contrib = append(contrib, sysCrit...)
		symptoms = append(symptoms, "System health critical")
	}
	if len(avail) > 0 {
		contrib = append(contrib, avail...)
		symptoms = append(symptoms, "Kubernetes deployment availability below desired")
	}
	sev := inc.SeverityHigh
	conf := "medium"
	cause := "Power delivery impairment is suspected due to critical PSU health"
	if len(sysCrit) > 0 {
		conf = "high"
		cause += " with critical system health"
	}
	if len(avail) > 0 {
		cause += " and reduced workload availability"
	}
	cause += "."
	return Result{
		RuleID:    "power_supply",
		DedupeKey: key,
		Active:    true,
		Incident:  buildIncident("power_supply", key, "Power supply incident", cause, sev, conf, contrib, facts, symptoms, cause, now),
	}
}

func (e *Engine) ruleK8sWorkload(signals []signal.Signal, now time.Time) Result {
	restarts := filterType(signals, signal.TypeRateIncrease)
	avail := filterType(signals, signal.TypeAvailabilityGap)
	if len(restarts) == 0 && len(avail) == 0 {
		return Result{RuleID: "kubernetes_workload", Active: false}
	}
	dep := "rackpilot"
	if len(avail) > 0 {
		dep = avail[0].Component
	} else if len(restarts) > 0 {
		dep = restarts[0].Namespace
	}
	key := "kubernetes_workload:" + dep
	var contrib []signal.Signal
	var facts, symptoms []string
	contrib = append(contrib, restarts...)
	contrib = append(contrib, avail...)
	for _, s := range contrib {
		facts = append(facts, s.ObservedFact)
	}
	if len(restarts) > 0 {
		symptoms = append(symptoms, "Pod restart counters elevated")
	}
	if len(avail) > 0 {
		symptoms = append(symptoms, "Deployment replicas below desired")
	}
	conf := "medium"
	sev := inc.SeverityMedium
	if len(restarts) > 0 && len(avail) > 0 {
		conf = "high"
		sev = inc.SeverityHigh
	}
	cause := "Workload degradation correlated from Kubernetes availability and restart signals within the window (not confirmed root cause)."
	return Result{
		RuleID:    "kubernetes_workload",
		DedupeKey: key,
		Active:    true,
		Incident:  buildIncident("kubernetes_workload", key, "Kubernetes workload degradation", cause, sev, conf, contrib, facts, symptoms, cause, now),
	}
}

func (e *Engine) ruleMemoryPressure(signals []signal.Signal, now time.Time) Result {
	memErr := filterMetric(signals, "rackpilot_bmc_memory_errors_total", signal.SeverityWarning)
	memPress := filterType(signals, signal.TypeResourcePressure)
	if len(memErr) == 0 || len(memPress) == 0 {
		return Result{RuleID: "memory_pressure", Active: false}
	}
	key := "memory_pressure:" + firstComponent(memErr)
	var contrib []signal.Signal
	var facts, symptoms []string
	contrib = append(contrib, memErr...)
	contrib = append(contrib, memPress...)
	for _, s := range contrib {
		facts = append(facts, s.ObservedFact)
	}
	symptoms = append(symptoms, "BMC memory errors present with container memory pressure in rackpilot namespace")
	cause := "Memory pressure is suspected from co-occurring BMC memory errors and high container memory use (correlation only)."
	return Result{
		RuleID:    "memory_pressure",
		DedupeKey: key,
		Active:    true,
		Incident:  buildIncident("memory_pressure", key, "Memory pressure incident", cause, inc.SeverityMedium, "medium", contrib, facts, symptoms, cause, now),
	}
}

func buildIncident(rule, dedupeKey, title, desc string, sev inc.Severity, conf string, contrib []signal.Signal, facts, symptoms []string, suspected string, now time.Time) inc.Incident {
	resources := affectedFromSignals(contrib)
	return inc.Incident{
		IncidentID:          stableID(rule, dedupeKey),
		Title:               title,
		Description:         desc,
		Severity:            sev,
		Status:              inc.StatusOpen,
		FirstSeen:           now,
		LastSeen:            now,
		AffectedResources:   resources,
		ContributingSignals: contrib,
		CorrelationRule:     rule,
		Confidence:          conf,
		ObservedFacts:       uniqueStrings(facts),
		CorrelatedSymptoms:  uniqueStrings(symptoms),
		SuspectedRootCause:  suspected,
		RootCauseDisclaimer: disclaimer,
		Timeline: []inc.TimelineEvent{
			{Timestamp: now, Summary: title + " correlated"},
		},
	}
}

func stableID(rule, dedupeKey string) string {
	return fmt.Sprintf("inc-%s-%s", rule, strings.ReplaceAll(dedupeKey, ":", "-"))
}

func filterSince(signals []signal.Signal, since time.Time) []signal.Signal {
	var out []signal.Signal
	for _, s := range signals {
		if !s.Timestamp.Before(since) {
			out = append(out, s)
		}
	}
	return out
}

func filterMetric(signals []signal.Signal, metric string, min signal.Severity) []signal.Signal {
	var out []signal.Signal
	for _, s := range signals {
		if s.MetricName != metric {
			continue
		}
		if severityRank(s.Severity) >= severityRank(min) {
			out = append(out, s)
		}
	}
	return out
}

func filterType(signals []signal.Signal, t signal.SignalType) []signal.Signal {
	var out []signal.Signal
	for _, s := range signals {
		if s.SignalType == t {
			out = append(out, s)
		}
	}
	return out
}

func severityRank(s signal.Severity) int {
	switch s {
	case signal.SeverityCritical:
		return 3
	case signal.SeverityWarning:
		return 2
	case signal.SeverityInfo:
		return 1
	default:
		return 0
	}
}

func firstComponent(signals []signal.Signal) string {
	if len(signals) == 0 {
		return "unknown"
	}
	if signals[0].Component != "" {
		return signals[0].Component
	}
	return "unknown"
}

func affectedFromSignals(signals []signal.Signal) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range signals {
		ref := s.Component
		if s.Namespace != "" {
			ref = s.Namespace + "/" + ref
		}
		if ref == "" || ref == "/" {
			ref = string(s.Source)
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
