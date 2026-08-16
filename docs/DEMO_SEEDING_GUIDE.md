# Demo Data Seeding Guide (Phase F7)

> **Purpose**: Import end-to-end demo data into a local or staging environment to support the 16-step MVP main-path walkthrough.
> **Status**: Post-F9 Enhancement · MVP Demo Ready · Tag deferred · Not Production Ready · Douyin Release Candidate

## Prerequisites

1. PostgreSQL + Redis are running (`docker compose up -d` or equivalent)
2. The backend API is reachable (default `http://127.0.0.1:8080`)
3. The root `.env` contains `ADMIN_BOOTSTRAP_EMAIL` / `ADMIN_BOOTSTRAP_PASSWORD`
4. (Optional) An AI provider is configured — customer service AI suggestion samples are best-effort

## One-Command Seeding

```powershell
# Repository root
.\scripts\seed-demo-data.ps1 -ApiBase http://127.0.0.1:8080 -OutFile docs/demo-dataset.json
.\scripts\seed-demo-permissions.ps1 -ApiBase http://127.0.0.1:8080
```

Linux / macOS:

```bash
./scripts/seed-demo-data.sh
./scripts/seed-demo-permissions.ps1   # requires PowerShell
```

## Script Behavior

### seed-demo-data

1. Log in as the bootstrap admin
2. Call `a1-prepare-samples.ps1` to fill out the 20 product slots
3. Create **F2 order**, **F3 inventory**, and **F4 customer service** samples
4. Probe the **F6/F7 dashboard** KPI API
5. Aggregate AI / listing / failed task / workbench to-dos
6. Write the following temporary output locally (not under version control):
   - `docs/demo-dataset.json`
   - `docs/demo-dataset.orders.json`
   - `docs/demo-dataset.inventory.json`
   - `docs/demo-dataset.customer.json`
   - `docs/demo-dataset.dashboard.json`

### seed-demo-permissions

Creates demo accounts and writes `docs/demo-dataset.permissions.json`:

| Account | Role | Purpose |
| --- | --- | --- |
| `demo_admin@trademind.local` | admin | Full-permission demo |
| `demo_operator@trademind.local` | operator | Shop isolation demo |
| `demo_readonly@trademind.local` | readonly | Read-only blocking demo |

Default passwords are in the script output or the locally generated `demo-dataset.permissions.json` (development environments only).

## Verification

```powershell
# Read the validation section
Get-Content docs/demo-dataset.json | ConvertFrom-Json | Select-Object -ExpandProperty validation
```

Expect `passed: true` (at least 20 slots, 7 task samples, and at least 3 each for orders/inventory/customer service).

Automated regression is executed by GitHub Actions; product pages and business flows are checked against the manual acceptance checklist.

## F8 Dev-Only Edge-Case Samples

In **non-production** environments, an admin can call:

```http
POST /api/v1/dev/demo-seed/full-project-edge-cases
Authorization: Bearer <admin token>
```

This writes (**without calling any real external platform**):

- Order sync `partial_success` + page-level error
- Inventory sync `failed` (SKU unbound)
- Customer service send failure + failed task center record
- Sample shop with platform not authorized

The operation is written to **operationlog** (`dev.demo_seed.full_project_edge_cases`).

`seed-demo-data.ps1` automatically probes this endpoint when the API is online.

## Notes

- **Does not write real platform data**; the Douyin step is expected to return `blocked_by_real_credentials` or `local_draft_only`
- Repeated runs append/update samples; you can clear the dev database or accept incremental data before a demo
- Product titles include prefixes like `R1 demo` / `F3 demo` for easy searching

## Related Documents

- [`DEMO_DATASET.md`](DEMO_DATASET.md) — slot and sample details
- [`PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md`](PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md) — manual acceptance checklist
