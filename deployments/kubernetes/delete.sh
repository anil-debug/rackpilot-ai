#!/usr/bin/env bash
# Remove RackPilot K8s application tier only (not namespaces, demo workloads, or PVCs).
set -euo pipefail
export KUBECONFIG="${KUBECONFIG:-/mnt/ssd/rackpilot-kind/kubeconfig}"

echo "This deletes Deployments/Services/ConfigMaps labeled rackpilot.io/app-tier=rackpilot-core in namespace rackpilot."
echo "It does NOT delete namespaces, demo API/worker, kube-state-metrics, or Compose monitoring."
read -r -p "Continue? [y/N] " ans
if [[ "${ans}" != [yY] ]]; then
  echo "Aborted."
  exit 0
fi

kubectl delete deployment,service,configmap -n rackpilot -l rackpilot.io/app-tier=rackpilot-core --ignore-not-found
kubectl delete endpoints compose-prometheus -n rackpilot --ignore-not-found
echo "RackPilot core application resources removed from Kubernetes."