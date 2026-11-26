#!/usr/bin/env bash
# Sync scrape token from dedicated ServiceAccount (not admin kubeconfig).
set -euo pipefail
export KUBECONFIG="${KUBECONFIG:-/mnt/ssd/rackpilot-kind/kubeconfig}"
OUT_DIR="${OUT_DIR:-/mnt/ssd/rackpilot-data/prometheus/k8s}"
API_HOST="${API_HOST:-rackpilot-kind-control-plane}"
API_PORT="${API_PORT:-6443}"

mkdir -p "${OUT_DIR}"
docker exec rackpilot-kind-control-plane cat /etc/kubernetes/pki/ca.crt >"${OUT_DIR}/ca.crt"
kubectl get secret rackpilot-prometheus-scrape-token -n rackpilot-monitoring -o jsonpath='{.data.token}' | base64 -d >"${OUT_DIR}/token"
chmod 600 "${OUT_DIR}/token" "${OUT_DIR}/ca.crt"

# Stable target for Prometheus on the kind Docker network (control-plane container).
API_IP="$(docker inspect -f '{{(index .NetworkSettings.Networks "kind").IPAddress}}' rackpilot-kind-control-plane)"
API_ADDR="${API_IP}:${API_PORT}"
echo "${API_ADDR}" >"${OUT_DIR}/api_address"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
TEMPLATE="${ROOT}/monitoring/prometheus/k8s-scrape.template.yml"
SCRAPE_OUT="${OUT_DIR}/scrape.yml"
sed "s/__API_ADDRESS__/${API_ADDR}/g" "${TEMPLATE}" >"${SCRAPE_OUT}"

echo "Wrote ${OUT_DIR}/{ca.crt,token,api_address,scrape.yml} (API ${API_ADDR})"