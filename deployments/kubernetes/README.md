# RackPilot Kubernetes

Isolated **kind** lab plus **Kubernetes-native application deployment** (Milestone 7). Compose **Prometheus** and **Grafana** remain the monitoring UI; apps run as Deployments in namespace `rackpilot`.

See [docs/kubernetes-deployment.md](../../docs/kubernetes-deployment.md) for architecture, E2E, and port-forward access.

### Application tier (Milestone 7)

```bash
export KUBECONFIG=/mnt/ssd/rackpilot-kind/kubeconfig
./deployments/kubernetes/build-and-load-images.sh
./deployments/kubernetes/deploy.sh
./deployments/kubernetes/status.sh
```

| Script | Purpose |
|--------|---------|
| `build-and-load-images.sh` | Build four app images, `kind load` |
| `deploy.sh` | Apply kustomize, patch Prometheus bridge, reload scrape |
| `delete.sh` | Remove core app resources only (interactive) |
| `status.sh` | Deployments, services, diagnostics |

### Milestone 4 lab (demo workloads)

| Path | Purpose |
|------|---------|
| `kind/` | Cluster config, create script |
| `namespaces/` | `rackpilot`, `rackpilot-monitoring` |
| `workloads/demo-*.yaml` | Demo API and worker |
| `workloads/gpu-simulator/` … | **M7** application services |
| `config/` | ConfigMaps and Compose Prometheus bridge |
| `monitoring/` | `kube-state-metrics`, scrape RBAC, sync script |
| `kustomization.yaml` | Kustomize root for M7 apps |

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