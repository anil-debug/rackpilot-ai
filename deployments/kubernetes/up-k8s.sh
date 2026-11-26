#!/usr/bin/env bash
# One-command RackPilot on Kubernetes (apps + in-cluster monitoring). No Compose required.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export KUBECONFIG="${KUBECONFIG:-/mnt/ssd/rackpilot-kind/kubeconfig}"

cd "${ROOT}"
./deployments/kubernetes/build-and-load-images.sh
./deployments/kubernetes/monitoring/deploy-monitoring.sh
USE_K8S_PROMETHEUS=1 ./deployments/kubernetes/deploy.sh

echo ""
echo "=== RackPilot on Kubernetes is ready ==="
echo "Start ALL host ports (background):"
echo "  ./deployments/kubernetes/port-forward-all.sh start"