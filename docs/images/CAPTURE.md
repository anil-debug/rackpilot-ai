# Capturing documentation screenshots

Prometheus screenshots in this folder were captured from the local RackPilot stack at `http://127.0.0.1:9090`.

## Prometheus (automated)

With the stack running:

```bash
# Targets and example queries — use browser or Playwright against 127.0.0.1:9090
open http://127.0.0.1:9090/targets
```

Saved files: `prometheus-*.png`

## Grafana dashboards

Automated headless capture may fail when Grafana requires interactive login. Capture manually:

1. Open `http://127.0.0.1:3000` (development credentials in `deployments/docker/.env.example` — change defaults before any shared deployment).
2. Log in as admin.
3. Open each dashboard under folder **RackPilot**:
   - RackPilot GPU Synthetic (`rackpilot-gpu-synthetic`)
   - RackPilot BMC Hardware Health (`rackpilot-bmc-synthetic`)
   - RackPilot Kubernetes Overview (`rackpilot-kubernetes-overview`)
4. Use browser screenshot or **Share → Export → Save image** (if available).
5. Save as:

| File | Dashboard |
|------|-----------|
| `rackpilot-gpu-dashboard.png` | GPU Synthetic |
| `rackpilot-bmc-dashboard.png` | BMC Synthetic |
| `rackpilot-kubernetes-dashboard.png` | Kubernetes Overview |

Optional: `node docs/images/capture-grafana.mjs` after `npm install playwright && npx playwright install chromium` in this directory.

## Incident engine

- `prometheus-incident-metrics.png` — query `rackpilot_incidents_active` on Prometheus graph UI.
- API JSON: `curl http://127.0.0.1:8083/api/v1/incidents/active`