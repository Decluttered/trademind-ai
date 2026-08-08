# P10 Credential, Read-only Provider and Safety Control API

P10 endpoints reuse the authenticated `/api/v1` group, tenant context, store scope and the standard `{code,message,data,traceId?}` envelope. Runtime is fixed at L0, so real provider calls fail closed. Credential responses contain metadata only; they never contain access token, refresh token, ciphertext, app secret or client secret. All request bodies use strict JSON with size limits.

| Method | Path | Permission | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/p10/status` | `inventory_sync.read` | L0 boundary and provider flags; shop-scoped allowlist, gray and last-read details are filtered by the principal's store grants |
| `GET` | `/api/v1/p10/credentials` | `inventory_sync.read` | Tenant- and store-scoped credential metadata list |
| `POST` | `/api/v1/p10/credentials/offline` | `config.manage` | Development/test-only fixture credential creation |
| `POST` | `/api/v1/p10/credentials/:credentialId/rotate` | `config.manage` | Atomic offline credential rotation with `expectedRevision` |
| `POST` | `/api/v1/p10/credentials/:credentialId/revoke` | `config.manage` | Local revocation with `expectedRevision` |
| `POST` | `/api/v1/p10/credentials/oauth/offline/start` | `config.manage` | Create single-use fixture OAuth state for an exact allowlisted redirect |
| `POST` | `/api/v1/p10/credentials/oauth/offline/complete` | `config.manage` | Consume fixture OAuth state once and create credential metadata |
| `PUT` | `/api/v1/p10/controls/kill-switches` | `config.manage` | Update provider/tenant/shop/read switches; write always stays active |
| `PUT` | `/api/v1/p10/controls/allowlist` | `config.manage` | Save one-tenant/one-shop allowlist with revision control |
| `PUT` | `/api/v1/p10/gray` | `config.manage` | Save a gray draft for at most 100 SKUs and reset approval fields |
| `POST` | `/api/v1/p10/gray/pause` | `config.manage` | Pause gray by expected revision |
| `POST` | `/api/v1/p10/gray/stop` | `config.manage` | Stop gray by expected revision |
| `POST` | `/api/v1/p10/inventory-read/runs` | `inventory_sync.run` | Manual read-only run; requires `Idempotency-Key` and an allowed store |
| `POST` | `/api/v1/p10/inventory-read/runs/:runId/rerun` | `inventory_sync.run` | Manual rerun of a tenant-scoped failed/cancelled run; requires revision and a new `Idempotency-Key` |

The P10 read adapter paginates locally bound Douyin publications and calls only the repository-confirmed `product.detail` operation. The repository has no confirmed all-shop inventory endpoint, so no endpoint was invented. Historical `sku.syncStock` write paths remain separate and are not reachable through P10.
