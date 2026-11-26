# GPU Telemetry Simulator (Milestone 1)

The RackPilot GPU Telemetry Simulator exposes **synthetic** GPU telemetry through REST APIs and Prometheus metrics. It does **not** require physical NVIDIA GPUs and must **not** be interpreted as real hardware measurements.

## Quick start

```bash
go build -o bin/gpu-simulator ./cmd/gpu-simulator
./bin/gpu-simulator -listen :8080
```

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `RACKPILOT_LISTEN` | `:8080` | HTTP listen address |
| `RACKPILOT_GPU_COUNT` | `4` | Number of simulated GPUs |
| `RACKPILOT_GPU_MODEL` | `NVIDIA H100 80GB` | Profile selector |
| `RACKPILOT_TICK_MS` | `2000` | Simulation tick interval |
| `RACKPILOT_SIM_SEED` | `42` | RNG seed for reproducibility |

## REST API

### `GET /health`

Service liveness. Response includes `"synthetic_telemetry": true`.

### `GET /api/v1/gpus`

List all simulated GPUs with telemetry and a disclaimer.

### `GET /api/v1/gpus/{id}`

Return one GPU (`gpu-0` … `gpu-N`). `404` if unknown.

### `POST /api/v1/scenarios/overheating`

Body: `{"gpu_id":"gpu-0"}`. Ramps temperature on the selected GPU only.

### `POST /api/v1/scenarios/memory-exhaustion`

Body: `{"gpu_id":"gpu-0"}`. Ramps memory usage toward a configurable cap (≤ total memory).

### `POST /api/v1/scenarios/reset`

Body: `{}`. Clears scenarios and restores baseline simulation for all GPUs.

### `GET /metrics`

Prometheus exposition format.

## Health derivation

Health is **derived** on each read from telemetry thresholds (not stored separately):

| Signal | Warning | Critical |
|--------|---------|----------|
| Temperature (°C) | ≥ 80 | ≥ 90 |
| Memory used / total | ≥ 0.90 | ≥ 0.95 |

## Prometheus metrics

| Metric | Labels | Notes |
|--------|--------|-------|
| `rackpilot_gpu_temperature_celsius` | `gpu_id`, `model` | Synthetic °C |
| `rackpilot_gpu_utilization_percent` | `gpu_id`, `model` | 0–100 |
| `rackpilot_gpu_memory_used_bytes` | `gpu_id`, `model` | bytes |
| `rackpilot_gpu_memory_total_bytes` | `gpu_id`, `model` | bytes |
| `rackpilot_gpu_power_watts` | `gpu_id`, `model` | watts |
| `rackpilot_gpu_health_status` | `gpu_id`, `model` | 0=healthy, 1=warning, 2=critical |

## GPU profiles

Hardware characteristics live in `domain.GPUProfile` and `config.ProfileByModelName`. Built-in names:

- `NVIDIA H100 80GB` (default)
- `NVIDIA A100 80GB PCIe`
- `NVIDIA B200`

Unknown model strings use H100-class defaults with the supplied model label.

## Future integration

Telemetry JSON field names and metric semantics are intended to align with future real GPU collectors. The simulator remains independent: no NVML, no shared binaries with production monitoring.