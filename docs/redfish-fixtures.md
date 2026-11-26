# Redfish fixture mode (DMTF mockup data)

RackPilot can serve **official DMTF mockup JSON** (static mode) or run the **synthetic dynamic BMC engine** (default).

## Source

| Field | Value |
|-------|--------|
| Dataset | `public-rackmount1` |
| Repository | [DMTF/Redfish-Mockup-Server](https://github.com/DMTF/Redfish-Mockup-Server) |
| License | BSD 3-Clause (`testdata/redfish/dmtf/LICENSE.BSD`) |
| Retrieved | 2026-11-26 |
| Notices | `testdata/redfish/dmtf/NOTICE.md` |

Files are copied **without content changes**. RackPilot is **not** DMTF-certified.

## Modes

| Mode | Env | Redfish | Scenarios | Prometheus |
|------|-----|---------|-----------|------------|
| **dynamic** (default) | `RACKPILOT_BMC_MODE=dynamic` | Synthetic RackPilot resources (`rackpilot-server-001`) | Supported | Full `rackpilot_bmc_*` telemetry |
| **static** | `RACKPILOT_BMC_MODE=static` | DMTF files (`437XR1138R2`, chassis `1U`, …) | **409 Conflict** | `rackpilot_bmc_fixture_mode_active` only |

Static mode does **not** present mockup readings as live telemetry.

## Run locally

```bash
# Dynamic (unchanged)
go run ./cmd/bmc-simulator -listen :8081 -mode dynamic

# Static fixtures
go run ./cmd/bmc-simulator -listen :8081 -mode static \
  -fixture-root testdata/redfish/dmtf/public-rackmount1
```

Example static Redfish URLs (DMTF IDs):

- `/redfish/v1/`
- `/redfish/v1/Systems/437XR1138R2`
- `/redfish/v1/Chassis/1U/Thermal`
- `/redfish/v1/Systems/437XR1138R2/LogServices/Log1/Entries`

## Docker static mode

```bash
RACKPILOT_BMC_MODE=static docker compose -p rackpilot -f deployments/docker/docker-compose.yml up -d --force-recreate bmc-simulator
```

Fixtures are baked into the BMC image at `/fixtures/dmtf/public-rackmount1`.

## Validation tests

```bash
go test ./simulator/bmc/fixtures/... ./simulator/bmc/redfish/...
```

Tests parse every bundled `index.json`, verify `@odata.id` matches HTTP paths where present, resolve all `Members` links inside the subset, check `Members@odata.count` (with documented DMTF exceptions), and assert static HTTP responses are byte-identical to fixture files.

Full DMTF JSON Schema validation is **not** run in CI (mockup schema versions may differ from latest CSDL). Document any intentional version gaps in this file when updating fixtures.

## Adding your own captures

Only add data you are **authorized** to redistribute. Do **not** commit proprietary Supermicro or production BMC dumps. Prefer official DMTF mockups or your own sanitized exports with license files beside them.