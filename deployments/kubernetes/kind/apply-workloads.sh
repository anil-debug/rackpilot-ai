#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
export KUBECONFIG="${KUBECONFIG:-/mnt/ssd/rackpilot-kind/kubeconfig}"

kubectl config use-context kind-rackpilot-kind
kubectl apply -f "${ROOT}/deployments/kubernetes/namespaces/namespaces.yaml"
kubectl apply -f "${ROOT}/deployments/kubernetes/workloads/demo-api.yaml"
kubectl apply -f "${ROOT}/deployments/kubernetes/workloads/demo-worker.yaml"
kubectl apply -f "${ROOT}/deployments/kubernetes/monitoring/kube-state-metrics.yaml"

kubectl rollout status deployment/rackpilot-demo-api -n rackpilot --timeout=120s
kubectl rollout status deployment/rackpilot-demo-worker -n rackpilot --timeout=120s
kubectl rollout status deployment/rackpilot-kube-state-metrics -n rackpilot-monitoring --timeout=120s

echo "RackPilot Kubernetes workloads applied."