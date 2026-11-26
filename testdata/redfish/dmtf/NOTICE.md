# DMTF Redfish mockup fixtures

## Dataset

- **Name:** `public-rackmount1` (Simple Rack Mount server)
- **Source repository:** [DMTF/Redfish-Mockup-Server](https://github.com/DMTF/Redfish-Mockup-Server)
- **Explorer reference:** [redfish.dmtf.org/redfish/v1](https://redfish.dmtf.org/redfish/v1)
- **Retrieved:** 2026-11-26
- **License:** BSD 3-Clause (see `LICENSE.BSD` in this directory)

## Files included (subset)

ServiceRoot, Systems, Chassis (1U Thermal/Power), Managers (BMC), UpdateService (+ FirmwareInventory members), System LogServices/EventLog entries, and resolved collection members.

## Link validation scope

Automated tests require every `Members[].@odata.id` in bundled files to resolve within this subset (19 `index.json` resources). ServiceRoot may advertise other DMTF services (e.g. `AccountService`) that are **not** bundled; those URLs correctly return **HTTP 404** in static mode.

Known upstream quirk (unmodified): `UpdateService/FirmwareInventory` has `Members@odata.count` 2 with 3 `Members` entries — documented exception in `fixtures/membersCountExceptions`.

## Modifications

- Subset of JSON files copied unchanged for RackPilot static fixture mode.
- Added Oem metadata is **not** applied to DMTF files; RackPilot serves them as-is in static mode.
- Paths in JSON retain original `@odata.id` values (`/redfish/v1/...`).

## Disclaimer

These files are **official DMTF mockup data** for interoperability testing. They are **not** live telemetry from your hardware and **not** evidence of DMTF product certification for RackPilot AI.