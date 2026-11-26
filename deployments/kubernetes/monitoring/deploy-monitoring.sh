#!/usr/bin/env bash
# Deploy in-cluster Prometheus + Grafana (does not stop Compose monitoring).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
export KUBECONFIG="${KUBECONFIG:-/mnt/ssd/rackpilot-kind/kubeconfig}"
EXPECTED_CTX="${EXPECTED_CTX:-kind-rackpilot-kind}"
MON_DIR="${ROOT}/deployments/kubernetes/monitoring"

ctx="$(kubectl config current-context 2>/dev/null || true)"
if [[ "${ctx}" != "${EXPECTED_CTX}" ]]; then
  echo "ERROR: expected context ${EXPECTED_CTX}, got '${ctx}'" >&2
  exit 1
fi

if ! kubectl get secret rackpilot-grafana-admin -n rackpilot-monitoring >/dev/null 2>&1; then
  echo "Creating Grafana admin secret from example (dev password)..."
  kubectl apply -f "${MON_DIR}/grafana/secret.example.yaml"
fi

mkdir -p /mnt/ssd/rackpilot-kind/node-data/rackpilot-prometheus \
  /mnt/ssd/rackpilot-kind/node-data/rackpilot-grafana

kubectl apply -k "${MON_DIR}"

kubectl rollout status deployment/rackpilot-prometheus -n rackpilot-monitoring --timeout=180s
kubectl rollout status deployment/rackpilot-grafana -n rackpilot-monitoring --timeout=180s
kubectl rollout status deployment/rackpilot-kube-state-metrics -n rackpilot-monitoring --timeout=180s

echo ""
echo "In-cluster monitoring is up (Compose prom/grafana unchanged)."
echo "  kubectl port-forward -n rackpilot-monitoring svc/prometheus 19090:9090"
echo "  kubectl port-forward -n rackpilot-monitoring svc/grafana 13000:3000"
echo ""
echo "Point apps at K8s Prometheus:"
echo "  ${ROOT}/deployments/kubernetes/monitoring/switch-apps-to-k8s-prometheus.sh"