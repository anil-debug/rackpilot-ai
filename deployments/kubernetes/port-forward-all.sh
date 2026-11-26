#!/usr/bin/env bash
# Start RackPilot port-forwards in the background (one shell command).
set -euo pipefail
export KUBECONFIG="${KUBECONFIG:-/mnt/ssd/rackpilot-kind/kubeconfig}"
# 127.0.0.1 = only this machine. Use 0.0.0.0 for LAN browser access (dev only).
BIND="${RACKPILOT_PORT_FORWARD_BIND:-127.0.0.1}"
PIDFILE="${PIDFILE:-/tmp/rackpilot-k8s-portforward.pids}"
LOGDIR="${LOGDIR:-/tmp/rackpilot-k8s-portforward-logs}"
mkdir -p "${LOGDIR}"

stop_existing() {
  if [[ -f "${PIDFILE}" ]]; then
    while read -r pid; do
      kill "${pid}" 2>/dev/null || true
    done < "${PIDFILE}"
    rm -f "${PIDFILE}"
  fi
}

case "${1:-start}" in
  start)
    stop_existing
    forwards=(
      "rackpilot-monitoring:prometheus:9090:9090"
      "rackpilot-monitoring:grafana:3000:3000"
      "rackpilot:gpu-simulator:8080:8080"
      "rackpilot:bmc-simulator:8082:8081"
      "rackpilot:incident-engine:8083:8083"
      "rackpilot:investigator:8084:8084"
    )
    for spec in "${forwards[@]}"; do
      IFS=: read -r ns svc local remote <<< "${spec}"
      log="${LOGDIR}/${svc}-${local}.log"
      kubectl port-forward --address "${BIND}" -n "${ns}" "svc/${svc}" "${local}:${remote}" >"${log}" 2>&1 &
      echo $! >> "${PIDFILE}"
      echo "  ${svc} -> http://${BIND}:${local} (service :${remote})  (log: ${log})"
    done
    sleep 2
    echo ""
    echo "Port-forwards running (bind=${BIND}). Test on server:"
    echo "  curl -s http://${BIND}:8082/health | jq ."
    echo "  curl -s http://${BIND}:9090/-/healthy"
    if [[ "${BIND}" == "127.0.0.1" ]]; then
      echo ""
      echo "From another PC, use SSH tunnel (recommended):"
      echo "  ssh -L 3000:127.0.0.1:3000 -L 9090:127.0.0.1:9090 user@<server-ip>"
      echo "Or LAN bind (dev only): RACKPILOT_PORT_FORWARD_BIND=0.0.0.0 $0 start"
    fi
    echo ""
    echo "Stop: $0 stop"
    ;;
  stop)
    stop_existing
    echo "Port-forwards stopped."
    ;;
  status)
    if [[ ! -f "${PIDFILE}" ]]; then
      echo "No port-forwards recorded (run: $0 start)"
      exit 0
    fi
    while read -r pid; do
      if kill -0 "${pid}" 2>/dev/null; then
        echo "running pid ${pid}"
      else
        echo "dead pid ${pid}"
      fi
    done < "${PIDFILE}"
    ;;
  *)
    echo "Usage: $0 {start|stop|status}" >&2
    exit 1
    ;;
esac