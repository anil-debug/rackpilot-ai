# Milestone 7.1 — In-cluster Prometheus & Grafana

## Architecture

All RackPilot workloads and monitoring run in Kubernetes. Compose `prometheus` and `grafana` services remain available as an **alternative dev mode** until you stop them manually.

| Component | Namespace | Service |
|-----------|-----------|---------|
| Prometheus | `rackpilot-monitoring` | `prometheus:9090` |
| Grafana | `rackpilot-monitoring` | `grafana:3000` |
| kube-state-metrics | `rackpilot-monitoring` | `rackpilot-kube-state-metrics:8080` |
| GPU / BMC / incident / investigator | `rackpilot` | existing ClusterIP |

In-cluster Prometheus scrapes application Services by DNS (no API proxy for apps). cAdvisor uses the existing dedicated ServiceAccount and node proxy path.

## Storage

PVs use **hostPath** on the kind control plane:

| PV path (in node) | Host SSD path |
|-------------------|---------------|
| `/var/lib/kubelet/rackpilot-prometheus` | `/mnt/ssd/rackpilot-kind/node-data/rackpilot-prometheus` |
| `/var/lib/kubelet/rackpilot-grafana` | `/mnt/ssd/rackpilot-kind/node-data/rackpilot-grafana` |

Legacy Compose data under `/mnt/ssd/rackpilot-data/{prometheus,grafana}` is **not deleted**. Optional one-time copy:

```bash
# After stopping Compose prom/grafana only
./deployments/kubernetes/monitoring/migrate-historical-data.sh
kubectl rollout restart -n rackpilot-monitoring deployment/rackpilot-prometheus deployment/rackpilot-grafana
```

Without migration, in-cluster stacks start with **fresh** TSDB/Grafana lib dirs on the node-data paths.

## Deploy (Compose monitoring can stay running)

```bash
export KUBECONFIG=/mnt/ssd/rackpilot-kind/kubeconfig
./deployments/kubernetes/monitoring/deploy-monitoring.sh
```

Verify:

```bash
kubectl port-forward -n rackpilot-monitoring svc/prometheus 19090:9090 &
curl -s 'http://127.0.0.1:19090/api/v1/targets' | jq '.data.activeTargets[] | {job: .labels.job, health: .health}'

kubectl port-forward -n rackpilot-monitoring svc/grafana 13000:3000 &
curl -s -u admin:rackpilot-dev http://127.0.0.1:13000/api/health | jq .
```

Switch application tier to in-cluster Prometheus:

```bash
./deployments/kubernetes/monitoring/switch-apps-to-k8s-prometheus.sh
```

## Cut over from Compose (manual — not automated)

Only after validation:

```bash
docker compose -p rackpilot -f deployments/docker/docker-compose.yml stop prometheus grafana
```

**Rollback:**

```bash
docker compose -p rackpilot -f deployments/docker/docker-compose.yml up -d prometheus grafana
# Revert app ConfigMap to compose-prometheus URL and restart incident/investigator
kubectl patch configmap rackpilot-app-config -n rackpilot --type merge -p \
  '{"data":{"RACKPILOT_PROMETHEUS_URL":"http://compose-prometheus.rackpilot.svc.cluster.local:9090"}}'
kubectl rollout restart deployment/incident-engine deployment/investigator -n rackpilot
kubectl delete -k deployments/kubernetes/monitoring   # optional: remove in-cluster monitoring
```

## Remove in-cluster monitoring only

```bash
kubectl delete deployment rackpilot-prometheus rackpilot-grafana -n rackpilot-monitoring --ignore-not-found
# PVCs/PVs retain data on node-data paths unless you delete PVCs explicitly
```