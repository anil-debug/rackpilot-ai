#!/usr/bin/env bash
# Point incident-engine and investigator at in-cluster Prometheus and roll out.
set -euo pipefail
export KUBECONFIG="${KUBECONFIG:-/mnt/ssd/rackpilot-kind/kubeconfig}"
URL="http://prometheus.rackpilot-monitoring.svc.cluster.local:9090"

kubectl get cm rackpilot-app-config -n rackpilot -o yaml | \
  sed "s|RACKPILOT_PROMETHEUS_URL: .*|RACKPILOT_PROMETHEUS_URL: \"${URL}\"|" | \
  kubectl apply -f -

kubectl rollout restart deployment/incident-engine deployment/investigator -n rackpilot
kubectl rollout status deployment/incident-engine -n rackpilot --timeout=120s
kubectl rollout status deployment/investigator -n rackpilot --timeout=120s
echo "Apps now use in-cluster Prometheus at ${URL}"