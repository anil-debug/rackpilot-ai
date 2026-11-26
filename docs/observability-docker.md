# Milestone 2: Docker Compose + Observability

Run the **RackPilot** stack in isolation (project name `rackpilot`). This does not modify or share networks/volumes with other Docker projects on the host.

## Prerequisites

- Docker Engine + Docker Compose v2
- SSD paths created. **Permissions:** Prometheus needs write access to its TSDB directory; Grafana needs write access for plugins/state. On a typical single-user dev host:

  ```bash
  chmod 755 /mnt/ssd/rackpilot-data/prometheus   # host UID 1000 matches Compose user
  chmod 777 /mnt/ssd/rackpilot-data/grafana        # Grafana runs as 1000:1000 in Compose
  ```

  Alternatives: `chown 472:472` for Grafana only (container default) or match Compose `user` to your host UID.
  - `/mnt/ssd/rackpilot-data/prometheus` — owned by host user `1000` (Prometheus `user: "1000:1000"`)
  - `/mnt/ssd/rackpilot-data/grafana` — writable by Grafana container user (`1000:1000` in Compose)
  - `/mnt/ssd/rackpilot-data/prometheus`
  - `/mnt/ssd/rackpilot-data/grafana`
- Host ports **8080**, **9090**, **3000** — publish address set by `RACKPILOT_PUBLISH_HOST` in `deployments/docker/.env` (default `127.0.0.1`)

## Configuration

```bash
cd /path/to/rackpilot-ai
cp deployments/docker/.env.example deployments/docker/.env
# Edit GRAFANA_ADMIN_PASSWORD before sharing the machine
```

## Start

```bash
docker compose -p rackpilot -f deployments/docker/docker-compose.yml up -d --build
```

Watch health:

```bash
docker compose -p rackpilot -f deployments/docker/docker-compose.yml ps
```

## Stop (keep data on SSD)

```bash
docker compose -p rackpilot -f deployments/docker/docker-compose.yml down
```

**Do not run** `docker compose down -v` for this project. Do not run global `docker system prune` or `docker volume prune`.

## Remote development — recommended access (SSH port forwarding)

When Docker runs on a **remote** host and you use a **local** browser, do **not** use the server’s IP with the default setup. Services bind to `127.0.0.1` on the remote host only. That is intentional.

**Most robust approach:** keep that bind and use **SSH port forwarding** (what Cursor’s **Ports** tab uses).

| Why this beats `0.0.0.0` + LAN IP |
|-----------------------------------|
| No extra firewall or ufw rules |
| No exposing Grafana/Prometheus on the LAN |
| Stable URLs on your laptop: always `http://localhost:3000` |
| Works the same over VPN or home NAT |

### Cursor

1. Start the stack on the remote machine (`docker compose … up -d`).
2. Open the **Ports** view. Forward **3000**, **8080**, **9090** (workspace `.vscode/settings.json` can auto-notify).
3. On your laptop: **http://localhost:3000** (Grafana), **:9090** (Prometheus), **:8080** (simulator).

### SSH without Cursor

On your laptop:

```bash
ssh -N \
  -L 3000:127.0.0.1:3000 \
  -L 8080:127.0.0.1:8080 \
  -L 9090:127.0.0.1:9090 \
  <user>@<remote-host>
```

Then browse **http://localhost:3000** on the laptop.

### LAN access via host IP (`0.0.0.0` publish)

In `deployments/docker/.env`:

```bash
RACKPILOT_PUBLISH_HOST=0.0.0.0
GRAFANA_ROOT_URL=http://<host-ip>:3000
```

Recreate RackPilot only:

```bash
docker compose -p rackpilot -f deployments/docker/docker-compose.yml up -d --force-recreate
```

From your laptop: `http://<host-ip>:3000`, `:9090`, `:8080`.

**Firewall:** if `ufw` is active, Docker may publish on `0.0.0.0` but packets can still be blocked. Allow only your subnet, for example:

```bash
sudo ufw allow from 10.184.0.0/16 to any port 3000 proto tcp
sudo ufw allow from 10.184.0.0/16 to any port 8080 proto tcp
sudo ufw allow from 10.184.0.0/16 to any port 9090 proto tcp
```

Do not open these globally without reviewing `sudo ufw status`.

## Verify

```bash
curl -s http://127.0.0.1:8080/health | jq .
curl -s 'http://127.0.0.1:9090/api/v1/targets' | jq '.data.activeTargets[] | {job: .labels.job, health: .health}'
```

Grafana: http://127.0.0.1:3000 — login with `GRAFANA_ADMIN_USER` / `GRAFANA_ADMIN_PASSWORD` from `.env`.

Open dashboard: **RackPilot → RackPilot Synthetic GPU Telemetry**.

## Trigger scenarios (observe in Grafana)

Overheating on one GPU (others unchanged):

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/scenarios/overheating \
  -H 'Content-Type: application/json' \
  -d '{"gpu_id":"gpu-0"}'
```

Memory pressure:

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/scenarios/memory-exhaustion \
  -H 'Content-Type: application/json' \
  -d '{"gpu_id":"gpu-2"}'
```

Reset all GPUs:

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/scenarios/reset \
  -H 'Content-Type: application/json' \
  -d '{}'
```

Within ~15–30 seconds (scrape interval), temperature/memory/health panels should move on the affected GPU only.

## Architecture

| Service | Image / build | Host port | Persistent data |
|---------|----------------|-----------|-----------------|
| `gpu-simulator` | `simulator/gpu/Dockerfile` | host :8080 | none |
| `bmc-simulator` | `simulator/bmc/Dockerfile` | host :8082 → container :8081 | none |
| `prometheus` | `prom/prometheus:v2.55.1` | host :9090 | `/mnt/ssd/rackpilot-data/prometheus` |
| `grafana` | `grafana/grafana:11.3.0` | host :3000 | `/mnt/ssd/rackpilot-data/grafana` |

Network: `rackpilot_net` (bridge). Prometheus scrapes `http://gpu-simulator:8080/metrics`.