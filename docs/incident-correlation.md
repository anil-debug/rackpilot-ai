# Incident correlation engine

Deterministic, event-driven correlation of **synthetic** GPU, BMC, and Kubernetes Prometheus signals. No LLM or automated remediation in this milestone.

## Architecture

```
Prometheus (existing) ──HTTP API──► ingest (normalize) ──► correlate (rules)
                                                         └──► store (in-memory)
                                                         └──► HTTP API + engine metrics
```

- **Ingestion** queries Prometheus instant vectors; missing series produce no signals (no fabrication).
- **Correlation** is rule-based and time-windowed (`RACKPILOT_INCIDENT_POLL_MS`, 5m default window).
- **Storage** is in-memory with deduplication by rule + resource key.

## Signal model

See `incident/signal.Signal` — fields include `source`, `signal_type`, `severity`, labels, `observed_fact`, and `evidence_reference` (PromQL).

## Rules

| Rule ID | Trigger (summary) | Suspected cause (hypothesis only) |
|---------|-------------------|-----------------------------------|
| `hardware_cooling` | Fan critical + (optional) elevated BMC CPU / GPU temps | Cooling impairment |
| `power_supply` | PSU critical (+ system critical, optional K8s availability gap) | Power delivery |
| `kubernetes_workload` | Restarts and/or deployment available < desired | Workload degradation |
| `memory_pressure` | BMC memory errors + container memory pressure ratio | Memory pressure |

Every incident includes `root_cause_disclaimer`: temporal correlation ≠ confirmed causation.

## Severity

| Incident severity | Typical drivers |
|-------------------|-----------------|
| critical | Availability gaps, multiple critical health signals |
| high | Cooling / PSU with strong evidence |
| medium | Partial evidence, workload or memory rules |

## Lifecycle

`OPEN` → `UPDATED` (new signals, same dedupe key) → `RESOLVED` after `RecoveryWindow` (default 2m) with no matching rule.

## API

```bash
curl -s http://127.0.0.1:8083/api/v1/health | jq .
curl -s http://127.0.0.1:8083/api/v1/incidents/active | jq .
curl -s http://127.0.0.1:8083/api/v1/incidents?limit=20 | jq .
curl -s http://127.0.0.1:8083/api/v1/incidents/<id> | jq .
curl -s http://127.0.0.1:8083/api/v1/metrics
```

## Engine metrics

| Metric | Meaning |
|--------|---------|
| `rackpilot_incidents_active` | Open + updated count |
| `rackpilot_incidents_created_total` | Opened incidents |
| `rackpilot_incidents_resolved_total` | Resolved incidents |
| `rackpilot_signals_processed_total` | Normalized signals per tick |
| `rackpilot_correlation_evaluations_total` | Poll ticks |
| `rackpilot_correlation_errors_total` | Prometheus/ingest failures |

## Configuration

| Variable | Default |
|----------|---------|
| `RACKPILOT_INCIDENT_LISTEN` | `:8083` |
| `RACKPILOT_PROMETHEUS_URL` | `http://prometheus:9090` |
| `RACKPILOT_INCIDENT_POLL_MS` | `15000` |

## Limitations

- No Redfish EventLog ingestion yet (model is extensible).
- No persistent incident store.
- Kubernetes signals depend on Prometheus k8s scrape jobs being configured.
- Restart and BMC memory error signals use `increase(...[5m])` instant queries; scrape gaps may yield empty vectors (no signal, no fabricated incident).
- Successful collect with zero signals can still advance incident recovery when rules are inactive (healthy or missing series).

## PromQL references

Defined in `incident/ingest/ingest.go` (`Q*` constants), aligned with metrics present in this repository.