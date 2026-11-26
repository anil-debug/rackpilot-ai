#!/usr/bin/env bash
# Build RackPilot app images and load into the existing kind cluster (no prune).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-rackpilot-kind}"
KIND_BIN="${KIND_BIN:-/mnt/ssd/rackpilot-kind/bin/kind}"
TAG="${TAG:-0.1.0}"
COMPOSE_FILE="${ROOT}/deployments/docker/docker-compose.yml"

IMAGES=(
  "rackpilot/gpu-simulator:${TAG}"
  "rackpilot/bmc-simulator:${TAG}"
  "rackpilot/incident-engine:${TAG}"
  "rackpilot/investigator:${TAG}"
)

echo "Building images (tag ${TAG})..."
docker compose -p rackpilot -f "${COMPOSE_FILE}" build gpu-simulator bmc-simulator incident-engine investigator

if ! [[ -x "${KIND_BIN}" ]]; then
  echo "ERROR: kind binary not found at KIND_BIN=${KIND_BIN}" >&2
  exit 1
fi
if ! docker ps --format '{{.Names}}' | grep -qx "${CLUSTER_NAME}-control-plane"; then
  echo "ERROR: kind cluster ${CLUSTER_NAME} control plane container not found." >&2
  exit 1
fi

for img in "${IMAGES[@]}"; do
  echo "Loading ${img} into kind..."
  "${KIND_BIN}" load docker-image "${img}" --name "${CLUSTER_NAME}"
done

echo "Done. Images loaded into ${CLUSTER_NAME}."