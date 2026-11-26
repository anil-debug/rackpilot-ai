#!/usr/bin/env bash
# Deploy RackPilot application tier to the existing kind cluster.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export KUBECONFIG="${KUBECONFIG:-/mnt/ssd/rackpilot-kind/kubeconfig}"
EXPECTED_CTX="${EXPECTED_CTX:-kind-rackpilot-kind}"
KUSTOMIZE_DIR="${ROOT}/deployments/kubernetes"
PROM_CONTAINER="${PROM_CONTAINER:-rackpilot-prometheus}"
USE_K8S_PROMETHEUS="${USE_K8S_PROMETHEUS:-0}"

ctx="$(kubectl config current-context 2>/dev/null || true)"
if [[ "${ctx}" != "${EXPECTED_CTX}" ]]; then
  echo "ERROR: kubectl context is '${ctx}', expected '${EXPECTED_CTX}'." >&2
  echo "  export KUBECONFIG=/mnt/ssd/rackpilot-kind/kubeconfig" >&2
  exit 1
fi

if ! kubectl get ns rackpilot rackpilot-monitoring >/dev/null 2>&1; then
  echo "ERROR: namespaces rackpilot / rackpilot-monitoring missing." >&2
  exit 1
fi

PROM_IP=""
compose_prom_running=false
if docker ps --format '{{.Names}}' | grep -qx "${PROM_CONTAINER}"; then
  compose_prom_running=true
fi
k8s_prom_exists=false
if kubectl get svc prometheus -n rackpilot-monitoring >/dev/null 2>&1; then
  k8s_prom_exists=true
fi

if [[ "${USE_K8S_PROMETHEUS}" == "1" ]] || { [[ "${USE_K8S_PROMETHEUS}" == "0" ]] && ! $compose_prom_running && $k8s_prom_exists; }; then
  if ! $k8s_prom_exists; then
    echo "ERROR: in-cluster Prometheus not found. Run:" >&2
    echo "  ./deployments/kubernetes/monitoring/deploy-monitoring.sh" >&2
    exit 1
  fi
  USE_K8S_PROMETHEUS=1
  echo "Using in-cluster Prometheus (no Compose bridge)."
elif $compose_prom_running; then
  PROM_IP="$(docker inspect -f '{{(index .NetworkSettings.Networks "kind").IPAddress}}' "${PROM_CONTAINER}" 2>/dev/null || true)"
  if [[ -z "${PROM_IP}" ]]; then
    echo "ERROR: Could not resolve ${PROM_CONTAINER} IP on Docker network 'kind'." >&2
    exit 1
  fi
  echo "Using Compose Prometheus bridge at ${PROM_IP}:9090"
else
  echo "ERROR: No Prometheus available." >&2
  echo "  Option A: ./deployments/kubernetes/monitoring/deploy-monitoring.sh" >&2
  echo "  Option B: docker compose -p rackpilot -f deployments/docker/docker-compose.yml up -d prometheus" >&2
  exit 1
fi

echo "Applying RackPilot Kubernetes manifests..."
kubectl apply -k "${KUSTOMIZE_DIR}"

if [[ -n "${PROM_IP}" ]]; then
  echo "Creating compose-prometheus Endpoints → ${PROM_IP}:9090"
  kubectl apply -f - <<EOF
apiVersion: v1
kind: Endpoints
metadata:
  name: compose-prometheus
  namespace: rackpilot
  labels:
    app.kubernetes.io/part-of: rackpilot
    rackpilot.io/app-tier: rackpilot-core
subsets:
  - addresses:
      - ip: ${PROM_IP}
    ports:
      - name: http
        port: 9090
        protocol: TCP
EOF
fi

echo "Waiting for deployments..."
kubectl rollout status deployment/gpu-simulator -n rackpilot --timeout=180s
kubectl rollout status deployment/bmc-simulator -n rackpilot --timeout=180s
kubectl rollout status deployment/incident-engine -n rackpilot --timeout=180s
kubectl rollout status deployment/investigator -n rackpilot --timeout=180s

if [[ "${USE_K8S_PROMETHEUS}" != "1" ]]; then
  echo "Refreshing Compose Prometheus Kubernetes scrape config..."
  "${ROOT}/deployments/kubernetes/monitoring/sync-prometheus-k8s-creds.sh"
  if curl -sf -X POST "http://127.0.0.1:9090/-/reload" >/dev/null 2>&1; then
    echo "Compose Prometheus config reloaded."
  else
    echo "WARN: Compose Prometheus reload failed." >&2
  fi
fi

kubectl get deploy,svc -n rackpilot -l rackpilot.io/app-tier=rackpilot-core
echo "Deploy complete."