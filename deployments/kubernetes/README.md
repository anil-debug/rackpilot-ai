# RackPilot Kubernetes (Milestone 4)

Isolated **kind** lab for workload and cluster monitoring. The **only** observability UI remains Docker Compose **Prometheus** and **Grafana** (integration is a later phase).

## Layout

| Path | Purpose |
|------|---------|
| `kind/` | Cluster config, create script |
| `namespaces/` | `rackpilot`, `rackpilot-monitoring` |
| `workloads/` | Demo API (2 replicas) and worker (1 replica) |
| `monitoring/` | `kube-state-metrics` only |

## Prerequisites

- Docker (unchanged daemon config)
- `kubectl` client
- **kind** binary (default: `/mnt/ssd/rackpilot-kind/bin/kind` v0.26.0+)

Host paths (SSD):

| Path | Content |
|------|---------|
| `/mnt/ssd/rackpilot-kind/kubeconfig` | Dedicated kubeconfig |
| `/mnt/ssd/rackpilot-kind/node-data` | Control-plane `kubelet` mount |
| `/mnt/ssd/rackpilot-kind/bin/kind` | kind CLI (optional install location) |

## Resource requirements (approximate)

| Component | CPU | Memory |
|-----------|-----|--------|
| kind control plane | 2 vCPU | 2–4 GiB |
| Demo workloads | 150m req | ~192 Mi req |
| kube-state-metrics | 50m req | 128 Mi req |

## Ports

| Port | Binding | Use |
|------|---------|-----|
| **16443** | `127.0.0.1` only | Kubernetes API |

Does not use RackPilot ports **8080**, **8082**, **9090**, or **3000**.

## Cluster creation

```bash
chmod +x deployments/kubernetes/kind/create-cluster.sh
./deployments/kubernetes/kind/create-cluster.sh
```

Or manually:

```bash
export KUBECONFIG=/mnt/ssd/rackpilot-kind/kubeconfig
/mnt/ssd/rackpilot-kind/bin/kind create cluster \
  --name rackpilot-kind \
  --config deployments/kubernetes/kind/cluster-config.yaml \
  --kubeconfig "$KUBECONFIG"
```

**Do not** run `minikube start`. Use context `kind-rackpilot-kind` only:

```bash
export KUBECONFIG=/mnt/ssd/rackpilot-kind/kubeconfig
kubectl config use-context kind-rackpilot-kind
kubectl cluster-info
```

## Deploy workloads and monitoring

```bash
./deployments/kubernetes/kind/apply-workloads.sh
```

## Verify cluster health

```bash
export KUBECONFIG=/mnt/ssd/rackpilot-kind/kubeconfig
kubectl get nodes
kubectl get ns rackpilot rackpilot-monitoring
kubectl get deploy,pods -n rackpilot
kubectl get deploy,pods -n rackpilot-monitoring
kubectl get --raw=/readyz?verbose
```

Expected:

- Node `Ready`
- `rackpilot-demo-api` **2/2** ready
- `rackpilot-demo-worker` **1/1** ready
- `rackpilot-kube-state-metrics` **1/1** ready

Verify API is loopback-only:

```bash
ss -tln | grep 16443
# LISTEN 127.0.0.1:16443
```

## Cluster deletion

```bash
/mnt/ssd/rackpilot-kind/bin/kind delete cluster --name rackpilot-kind
# Optional: retain kubeconfig backup; remove node-data only when sure
```

This removes kind containers only—not RackPilot Compose services or unrelated stacks.

## Troubleshooting

| Symptom | Check |
|---------|--------|
| Wrong cluster | `echo $KUBECONFIG` → must be `/mnt/ssd/rackpilot-kind/kubeconfig` |
| API connection refused | `docker ps \| grep rackpilot-kind` — control plane running? |
| Pods Pending | `kubectl describe pod -n rackpilot` — single-node should schedule quickly |
| Image pull errors | Node needs outbound network to `registry.k8s.io` |
| minikube confusion | `kubectl config get-contexts` — use `kind-rackpilot-kind`, not `minikube` |

## Prometheus integration (Phase 2)

```bash
kubectl apply -f deployments/kubernetes/monitoring/prometheus-scrape-rbac.yaml
./deployments/kubernetes/monitoring/sync-prometheus-k8s-creds.sh
docker compose -p rackpilot -f deployments/docker/docker-compose.yml up -d --no-deps prometheus
```

See [docs/prometheus-kubernetes.md](../../docs/prometheus-kubernetes.md).

## Safety

- No changes to `/etc/docker/daemon.json`, firewall, or profile/SAA/Traefik containers
- No in-cluster Prometheus, Grafana, GPU Operator, or kube-prometheus-stack
- Labels: `app.kubernetes.io/part-of=rackpilot`, `rackpilot.io/telemetry` on demo workloads