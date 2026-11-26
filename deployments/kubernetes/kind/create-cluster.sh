#!/usr/bin/env bash
# Create RackPilot kind cluster (does not touch minikube or RackPilot Compose).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
KIND_BIN="${KIND_BIN:-/mnt/ssd/rackpilot-kind/bin/kind}"
KUBECONFIG_OUT="${KUBECONFIG_OUT:-/mnt/ssd/rackpilot-kind/kubeconfig}"
CONFIG="${ROOT}/deployments/kubernetes/kind/cluster-config.yaml"

if [[ ! -x "${KIND_BIN}" ]]; then
  echo "kind not found at ${KIND_BIN}; install per deployments/kubernetes/README.md" >&2
  exit 1
fi

mkdir -p /mnt/ssd/rackpilot-kind/node-data

if "${KIND_BIN}" get clusters 2>/dev/null | grep -qx rackpilot-kind; then
  echo "Cluster rackpilot-kind already exists"
else
  "${KIND_BIN}" create cluster \
    --name rackpilot-kind \
    --config "${CONFIG}" \
    --kubeconfig "${KUBECONFIG_OUT}"
fi

echo "Kubeconfig: ${KUBECONFIG_OUT}"
echo "Context: kind-rackpilot-kind"
echo "API: https://127.0.0.1:16443"