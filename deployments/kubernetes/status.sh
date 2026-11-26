#!/usr/bin/env bash
set -euo pipefail
export KUBECONFIG="${KUBECONFIG:-/mnt/ssd/rackpilot-kind/kubeconfig}"

echo "=== Context ==="
kubectl config current-context

echo ""
echo "=== RackPilot core (rackpilot namespace) ==="
kubectl get deploy,svc,pod -n rackpilot -l rackpilot.io/app-tier=rackpilot-core -o wide 2>/dev/null || echo "(none)"

echo ""
echo "=== Demo workloads ==="
kubectl get deploy,pod -n rackpilot -l 'rackpilot.io/telemetry=kubernetes-workload' 2>/dev/null || true

echo ""
echo "=== Monitoring (rackpilot-monitoring) ==="
kubectl get deploy,pod,svc -n rackpilot-monitoring 2>/dev/null || true

echo ""
echo "=== Unhealthy pods (rackpilot*) ==="
kubectl get pods -A -l app.kubernetes.io/part-of=rackpilot --field-selector=status.phase!=Running 2>/dev/null || true

echo ""
echo "=== compose-prometheus bridge ==="
kubectl get endpoints compose-prometheus -n rackpilot -o yaml 2>/dev/null | sed -n '/addresses:/,/ports:/p' || true