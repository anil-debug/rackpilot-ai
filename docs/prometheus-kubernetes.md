# RackPilot Prometheus ↔ kind integration

## Network architecture

| Network | Members |
|---------|---------|
| `rackpilot_net` | Prometheus, Grafana, GPU/BMC simulators |
| `kind` | `rackpilot-kind-control-plane`, Prometheus (attached) |

Pod IPs (`10.244.0.0/16`) are **not** routable from Docker Compose. Scrapes use the **Kubernetes API proxy** over `https://<kind-control-plane-address>:6443` on the Docker `kind` network (discovered by `sync-prometheus-k8s-creds.sh`; not a public endpoint).

The API on the host (`127.0.0.1:16443`) is **not** used inside the Prometheus container.

## Authentication

- ServiceAccount: `rackpilot-prometheus-scrape` in `rackpilot-monitoring`
- Credentials on host: `/mnt/ssd/rackpilot-data/prometheus/k8s/{ca.crt,token,scrape.yml}`
- **Administrator kubeconfig is never mounted** into Prometheus.

Refresh after RBAC changes or token rotation:

```bash
./deployments/kubernetes/monitoring/sync-prometheus-k8s-creds.sh
curl -X POST http://127.0.0.1:9090/-/reload
```

If the kind control-plane IP changes, re-run the sync script and reload Prometheus.

## Scrape jobs

| Job | Source |
|-----|--------|
| `rackpilot-kube-state-metrics` | Service proxy → existing headless Service |
| `rackpilot-kubernetes-cadvisor` | Per-node `/proxy/metrics/cadvisor` via API |

Labels on all Kubernetes targets: `cluster=rackpilot-kind`, `rackpilot_telemetry=kubernetes`.

Cardinality: `metric_relabel_configs` keep `kube_*` for `rackpilot` / `rackpilot-monitoring` and cAdvisor for `rackpilot` only.

## TLS notes (kind lab)

- **API**: TLS verified with cluster CA (`ca.crt`).
- **Direct kubelet :10250** is not scraped and not published publicly.
- Do **not** reuse `insecure_skip_verify` patterns from generic lab guides in production.

## Connectivity checklist

1. `docker network inspect kind` — Prometheus has `kind` + `rackpilot_net`.
2. `kubectl get endpoints -n rackpilot-monitoring rackpilot-kube-state-metrics`.
3. `sync-prometheus-k8s-creds.sh` then Prometheus **Targets** UI.