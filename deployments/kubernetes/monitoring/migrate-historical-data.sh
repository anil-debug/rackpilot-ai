#!/usr/bin/env bash
# Optional one-time copy from Compose bind mounts to kind node host paths.
# STOP Compose Prometheus/Grafana before copying TSDB (see rollback/migration doc).
set -euo pipefail
SRC_PROM="${SRC_PROM:-/mnt/ssd/rackpilot-data/prometheus}"
SRC_GRAF="${SRC_GRAF:-/mnt/ssd/rackpilot-data/grafana}"
DST_PROM="${DST_PROM:-/mnt/ssd/rackpilot-kind/node-data/rackpilot-prometheus}"
DST_GRAF="${DST_GRAF:-/mnt/ssd/rackpilot-kind/node-data/rackpilot-grafana}"

echo "This copies data WITHOUT deleting sources."
echo "  Prometheus: ${SRC_PROM} -> ${DST_PROM}"
echo "  Grafana:    ${SRC_GRAF} -> ${DST_GRAF}"
echo "Stop Compose prometheus/grafana first to avoid TSDB corruption."
read -r -p "Continue? [y/N] " ans
if [[ "${ans}" != [yY] ]]; then
  exit 0
fi

mkdir -p "${DST_PROM}" "${DST_GRAF}"
if [[ -d "${SRC_PROM}" ]]; then
  rsync -a --exclude 'k8s' "${SRC_PROM}/" "${DST_PROM}/"
fi
if [[ -d "${SRC_GRAF}" ]]; then
  rsync -a "${SRC_GRAF}/" "${DST_GRAF}/"
fi
echo "Done. Restart in-cluster Prometheus/Grafana pods to pick up copied Grafana state."