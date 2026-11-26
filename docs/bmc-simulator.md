# BMC / Redfish Simulator (Milestone 3)

Synthetic BMC and **subset** Redfish API — **not** certified Redfish, **not** real Supermicro hardware.

## Simulation modes

- **dynamic** (default) — synthetic telemetry + scenarios + Prometheus.
- **static** — official DMTF `public-rackmount1` JSON — see [redfish-fixtures.md](redfish-fixtures.md).

## Run locally

```bash
go build -o bin/bmc-simulator ./cmd/bmc-simulator
./bin/bmc-simulator -listen :8081 -mode dynamic
```

## Redfish (subset)

| Method | Path |
|--------|------|
| GET | `/redfish/v1/` |
| GET | `/redfish/v1/Systems` |
| GET | `/redfish/v1/Systems/rackpilot-server-001` |
| GET | `/redfish/v1/Chassis/rackpilot-chassis-001` |
| GET | `/redfish/v1/Chassis/{id}/Thermal` |
| GET | `/redfish/v1/Chassis/{id}/Power` |
| GET | `/redfish/v1/Managers/BMC` |
| GET | `/redfish/v1/UpdateService/FirmwareInventory` |
| GET | `/redfish/v1/Systems/{id}/LogServices/EventLog/Entries` |

## Scenarios (RackPilot API)

| POST | Body |
|------|------|
| `/api/v1/scenarios/fan-failure` | `{"component_id":"fan-0"}` |
| `/api/v1/scenarios/psu-failure` | `{"component_id":"psu-0"}` |
| `/api/v1/scenarios/cpu-overheating` | `{"component_id":"cpu-0"}` |
| `/api/v1/scenarios/memory-error` | `{"component_id":"dimm-0"}` |
| `/api/v1/scenarios/reset` | `{}` |

## Prometheus metrics

Health encoding: **0=OK, 1=Warning, 2=Critical** (`rackpilot_bmc_*_health_status`).

All metrics use bounded labels: `server_id`, `cpu_id`, `dimm_id`, `fan_id`, `psu_id`, `sensor_id`.

## Docker

Container port **8081**, host **8082** (host **8081** reserved by other stacks):

```bash
docker compose -p rackpilot -f deployments/docker/docker-compose.yml up -d bmc-simulator --build
```

## Default inventory

- Manufacturer label: Supermicro **(synthetic)**  
- System: `rackpilot-server-001`  
- 2× CPU, 8× DIMM, 4× fans, 2× PSU  