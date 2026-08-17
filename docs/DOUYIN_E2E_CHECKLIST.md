# Douyin Shop Full-Chain Acceptance Checklist (E2E Checklist)

> For manual end-to-end acceptance in an environment with **real Douyin Shop credentials + public-facing Storage**.
> Automated regression runs in GitHub Actions in an isolated environment; real platform write paths must obtain external production approval and are never auto-triggered by repository scripts.
> For the general manual sign-off process, see [`PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md`](PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md). Acceptance conclusions are recorded in the PR or release ticket; do not commit test reports, screenshots, or log artifacts to the repository.

---

## 1. Pre-Acceptance Preparation

### 1.1 Acceptance Scope

- Without an App Key/Secret, an authorized shop, or public-facing Storage, the conclusion must be recorded as **`blocked_by_real_credentials`** or **`blocked_by_environment`** — never fake a pass.
- Read-only checks may be performed manually by a maintainer in a controlled environment; any real platform write operation requires explicit approval beforehand.
- The repository defaults to `L0`. Only an `L3` approved via a release ticket allows the Ops Task Center to create `save_as_platform_draft` tasks; formal publishing, listing, inventory writes, automatic business retries, unreviewed execution, and multi-shop scale-out are not allowed.
- Acceptance results are recorded in the PR or release ticket; do not commit JSON, Markdown reports, screenshots, or log artifacts to the repository.

### 1.2 Manual Checks

| Item | Description | How to Confirm |
| --- | --- | --- |
| Douyin Shop Open Platform App Key | Obtained after creating an app on the Douyin Shop Open Platform | Settings → Platform Configuration → Douyin Shop |
| Douyin Shop Open Platform App Secret | Stored encrypted; the frontend only shows a masked value | Still shows `****` placeholder after saving and reloading |
| Douyin Shop callback Redirect URI | Must exactly match what's registered on the Open Platform | Matches the public URL of `GET /api/v1/shops/oauth/douyin/callback` |
| Service ID | Required for the Service Marketplace custom authorization URL | Filled in under Platform Configuration |
| Authorized Douyin Shop | OAuth completed and the connection test passes | Shop Management → authorization status **Authorized** |
| Storage `public_base` | **Must be a publicly reachable HTTPS URL for Douyin Shop** | Settings → Storage; the local `/static` path is dev-proxy only — production requires a public domain |
| `order_sync_enabled` | Order sync toggle | Platform Configuration → **Enable Order Sync** = on |
| `inventory_sync_enabled` | Inventory sync toggle | Platform Configuration → **Enable Inventory Sync** = on |
| `product_publish_enabled` | Product draft creation toggle | Recommended to enable |
| P10 L3 draft-write configuration | Provider/network/credentials/draft-write/worker = true; publish queue and task reaper = true; auto-retry and inventory mutation = false | Backend startup validation passes, and `GET /api/v1/p10/status` matches the ticket |
| Production control scope | Single tenant, single allowlisted authorized shop, `maxSku<=100` | The allowlist and active gray target the same shop |
| Two-person gray approval | Two different admins hold Owner and Technical Lead roles respectively | Ticket identity verification + gray revision audit |
| Write kill switch | provider / tenant / shop / write default to blocked | Stays active until the final go/no-go, with staged drills completed |
| Test product | At least 1 editable draft | Includes title and price |
| Product with SKUs | At least 1 variant | Collected or manually maintained |
| Main image + detail images | At least 1 of each | Pre-publish check passes |

**Not in this MVP acceptance scope:** direct listing (`publish_online`), after-sales/refunds, financial settlement, multi-warehouse WMS, auto-replenishment, complex BI.

---

## 2. Acceptance Steps and Expectations

Each step covers: entry point, expected success, common failures, where to investigate, whether it appears in the Failed Task Center, and whether it's retryable.

### 2.1 Configure the Douyin Shop App

| Field | Content |
| --- | --- |
| **Entry point** | Settings → Platform Configuration → Douyin Shop |
| **Expected success** | App Key, Redirect URI, environment, and timeout save successfully; App Secret is masked; toggles can be saved |
| **Common failures** | Required fields missing; Redirect URI doesn't match the Open Platform |
| **Where to investigate** | Form validation on the current page; operation log `platform.settings.update` |
| **Failed Task Center** | No |
| **Retryable** | Yes (save again after completing the config) |

### 2.2 Test Connection

| Field | Content |
| --- | --- |
| **Entry point** | Same page → **Test Connection** |
| **Expected success** | Confirms configuration completeness (Phase 1 does not make real product/order calls) |
| **Common failures** | Incorrect App Key/Secret; network timeout |
| **Where to investigate** | Page toast; operation log |
| **Failed Task Center** | No |
| **Retryable** | Yes |

### 2.3 Authorize a Shop

| Field | Content |
| --- | --- |
| **Entry point** | Platform Configuration → **Connect Shop**, or Shop Management → Douyin Shop → Authorize |
| **Expected success** | Redirects to Douyin Shop OAuth → after the callback, shop status is **Authorized**; no plaintext token is returned |
| **Common failures** | Service ID not filled in; callback URL unreachable; user cancels authorization |
| **Where to investigate** | Shop Management details; operation log `douyin.auth.*` |
| **Failed Task Center** | No (failed authorization URL carries `auth=failed&reason=`) |
| **Retryable** | Yes (restart authorization) |

### 2.4 Sync Categories

| Field | Content |
| --- | --- |
| **Entry point** | Platform Configuration → **Sync Categories** (requires an authorized shop) |
| **Expected success** | Category cache alert shows count and last sync time |
| **Common failures** | Shop not authorized; token expired; insufficient permissions |
| **Where to investigate** | Category alert on Platform Configuration page; `GET /api/v1/platform/douyin/categories/stats` |
| **Failed Task Center** | No |
| **Retryable** | Yes (retry after refreshing authorization) |

### 2.5 Sync Attributes

| Field | Content |
| --- | --- |
| **Entry point** | Product Details → Publish tab → Douyin Shop category & attributes → **Refresh Attributes** |
| **Expected success** | After selecting a leaf category, the required attribute form loads |
| **Common failures** | Categories not synced; not a leaf category; token expired |
| **Where to investigate** | Attribute section alert on the Publish tab; API `.../categories/:id/attributes/sync` |
| **Failed Task Center** | No |
| **Retryable** | Yes |

### 2.6 Collect Products (1688 / Pinduoduo / Taobao / Tmall)

| Field | Content |
| --- | --- |
| **Entry point** | Collection Center → corresponding collector → single or batch collection |
| **Expected success** | Task succeeds → generates a `status=draft` product draft |
| **Common failures** | Not logged in; CAPTCHA; invalid link; missing main image |
| **Where to investigate** | Collection task page; Failed Task Center (collection category) |
| **Failed Task Center** | Yes |
| **Retryable** | Yes (retry individual item) |

### 2.7 AI Title Optimization

| Field | Content |
| --- | --- |
| **Entry point** | Product Details → Basic Info → AI Optimize Title |
| **Expected success** | Generates candidate titles → **Apply to Draft** writes `aiTitle` |
| **Common failures** | AI provider not configured; timeout |
| **Where to investigate** | AI task page; Failed Task Center (AI category) |
| **Failed Task Center** | Yes |
| **Retryable** | Yes |

### 2.8 AI Description Generation

| Field | Content |
| --- | --- |
| **Entry point** | Product Details → Basic Info → AI Generate Description |
| **Expected success** | Generates a description → applied to `aiDescription` |
| **Common failures** | Same as AI Title |
| **Where to investigate** | AI task page |
| **Failed Task Center** | Yes |
| **Retryable** | Yes |

### 2.9 Apply Pricing Rule

| Field | Content |
| --- | --- |
| **Entry point** | Product Details → Publish tab → **Apply Pricing Rule** |
| **Expected success** | SKU selling price updated; operation log `pricing.apply` |
| **Common failures** | Cost price missing; rule not configured |
| **Where to investigate** | SKU table on Publish tab; Settings → Pricing |
| **Failed Task Center** | No |
| **Retryable** | Yes |

### 2.10 Complete Douyin Shop Category Attributes

| Field | Content |
| --- | --- |
| **Entry point** | Publish tab → select Douyin Shop + leaf category + required attributes → **Save Douyin Shop Publish Config** |
| **Expected success** | Config saved; pre-publish check for category/attribute items passes |
| **Common failures** | Shop not selected; leaf category not selected; required attribute left empty |
| **Where to investigate** | Pre-publish check on Publish tab; `product_platform_publish_configs` |
| **Failed Task Center** | No |
| **Retryable** | Yes |

### 2.11 Generate Douyin Shop Publish Draft

| Field | Content |
| --- | --- |
| **Entry point** | Publish tab → **Generate Douyin Shop Publish Draft** |
| **Expected success** | Preview of title/description/main image/detail images/SKU/price/inventory; can be created only when errors is empty |
| **Common failures** | Invalid title/main image/category/attributes/SKU price |
| **Where to investigate** | Mapping errors/warnings on the Publish tab |
| **Failed Task Center** | No (mapping validation is not written to the failure center) |
| **Retryable** | Yes |

### 2.12 Upload Images to Douyin Shop

| Field | Content |
| --- | --- |
| **Entry point** | Publish tab → **Upload Images to Douyin Shop** / retry single image |
| **Expected success** | Main/detail image status becomes **Uploaded**, with a `platformImageId` |
| **Common failures** | External link not synced to Storage; `public_base` not public; main image upload failure; SSRF check blocks a private-network URL |
| **Where to investigate** | Image status column on the Publish tab; operation log `douyin.image.*` |
| **Failed Task Center** | In some scenarios (associated with a publish task) |
| **Retryable** | Yes (retry single image / all) |

### 2.13 Create a Douyin Shop Product Draft

| Field | Content |
| --- | --- |
| **Entry point** | From Product Details, the Douyin Shop action navigates to the **Ops Task Center** → create a platform draft task → verify the frozen snapshot → manual review → execute (`save_as_platform_draft`, not listed) |
| **Expected success** | The creation stage only freezes the product/mapping/request and does not call the platform; after approving the exact version/hash it executes once; returns `platformProductId`, the publish task succeeds, the ops task reaches `draft_written`, and `product_publications` plus the SKU mapping are written transactionally |
| **Common failures** | Blocked by L0/kill switch; allowlist or active gray mismatch; missing two-person approval; not authorized; invalid category/attributes/images/SKU; queue unavailable; platform API error or unknown result |
| **Where to investigate** | Draft/Review/Execution/Audit tabs on the ops task detail page; publish task detail; P10 status; Failure Center. The legacy `.../create-draft` always returns 409 `DOUYIN_OPERATION_TASK_REQUIRED` |
| **Failed Task Center** | Yes |
| **Retryable** | Only for known failures where the ops task returns `retryable=true`, and only via manual retry from the Ops Task Center; `result_unknown` may never be retried or recreated — only manual, read-only recovery reconciliation is allowed |

**Manual recovery verification:** using an account with `operationtask.execute` and shop operation permissions, call the publish task `recover-douyin-draft` (or the `douyin/recover` alias) for records where the downstream task, execution attempt, and ops task are all `result_unknown`, confirming the request only performs `product.detail`. Queued, executing, known-failed, or ordinary tasks always return 409 `DOUYIN_RECOVERY_NOT_ALLOWED` with no state change; when the same `outer_product_id` is found, both task centers converge to success, and when not found, the task remains non-retryable and is handed to manual investigation. Douyin Shop drafts must never be recreated from the legacy publish, multi-target, batch, single-task-retry, or batch-retry entry points.

### 2.14 Calibrate SKU Bindings

| Field | Content |
| --- | --- |
| **Entry point** | Publish tab → **Calibrate Douyin Shop SKU Bindings** |
| **Expected success** | Pulls the platform SKUs via `product.detail`; matching variants auto-bind as `bound` |
| **Common failures** | No `platformProductId`; insufficient permissions; multiple candidates → `ambiguous`; no match → `unmatched` |
| **Where to investigate** | Douyin Shop SKU binding management table; `GET .../douyin/sku-bindings` |
| **Failed Task Center** | No |
| **Retryable** | Yes (recalibrate) |

### 2.15 Manually Bind Ambiguous / Unmatched SKUs

| Field | Content |
| --- | --- |
| **Entry point** | Publish tab → SKU Binding table → **Manual Bind** / **Unbind** |
| **Expected success** | `bindStatus=bound`, `external_sku_id` is written; conflicts are blocked |
| **Common failures** | The same Douyin Shop SKU bound to multiple local variants; platform SKU ID missing |
| **Where to investigate** | SKU binding table; operation log `douyin.sku.binding.*` |
| **Failed Task Center** | No |
| **Retryable** | Yes |

### 2.16 Sync Orders

| Field | Content |
| --- | --- |
| **Entry point** | Shop Management → Douyin Shop → **Sync Orders**; or Orders → Order Sync Tasks |
| **Expected success** | Task succeeds / partial_success; orders written; SKU match summary |
| **Common failures** | `order_sync_enabled=false`; not authorized; token expired; partial pagination failure |
| **Where to investigate** | Order sync task page; order list; Failed Task Center `DOUYIN_ORDER_*` |
| **Failed Task Center** | Yes |
| **Retryable** | Yes |

### 2.17 Local Inventory Deduction

| Field | Content |
| --- | --- |
| **Entry point** | Automatic after a successful order sync (when the policy allows it); can also be manually deducted from the Order Exceptions Workbench |
| **Expected success** | Matched line items deduct local inventory; repeated syncs don't double-deduct |
| **Common failures** | SKU not matched; insufficient inventory; deduction policy disabled |
| **Where to investigate** | Order Exceptions Workbench; inventory ledger; `inventory/effects` |
| **Failed Task Center** | Yes (deduction failure category) |
| **Retryable** | Yes (**Retry Inventory Deduction** in the Exceptions Workbench) |

### 2.18 Sync Inventory to Douyin Shop

| Field | Content |
| --- | --- |
| **Entry point** | Product Details → Inventory tab → **Sync Inventory to Douyin Shop**; or batch sync from the Inventory Alerts page |
| **Expected success** | Inventory sync task succeeds; Douyin Shop `sku.syncStock` succeeds |
| **Common failures** | `inventory_sync_enabled=false`; SKU not bound; ambiguous/unmatched entries exist; invalid inventory |
| **Where to investigate** | Inventory sync task page; Failed Task Center `DOUYIN_INVENTORY_*` / `DOUYIN_SKU_*` |
| **Failed Task Center** | Yes |
| **Retryable** | Yes |

---

## 3. Security Checks (Must Pass for Acceptance)

| # | Check | Expected | Verification Method |
| --- | --- | --- | --- |
| 1 | App Secret is not returned in plaintext to the frontend | API `GET platform/settings/douyin_shop` value is `****` | Browser Network tab |
| 2 | accessToken is not returned in plaintext to the frontend | `auth.accessToken` masked in shop details | Shop Management Network tab |
| 3 | refreshToken is not returned in plaintext to the frontend | Same as above | Same as above |
| 4 | Logs do not print tokens / secrets | No complete secrets in backend logs | Search for `access_token` / `app_secret` |
| 5 | Order shipping info is masked | Buyer name/phone/address partially masked | Order detail UI |
| 6 | Raw error text is masked | `SanitizeErrorText` masks tokens | Failed task detail |
| 7 | Image download blocks private-network addresses | Private-network URL returns `private network` error | Pre-check before Douyin Shop image upload |
| 8 | Frontend never calls the Douyin Shop API directly | No requests to Douyin Shop domains | Browser Network tab |
| 9 | Douyin Shop calls go through the backend client | All go through `/api/v1/*` | Code review of `douyinshop.Client` |
| 10 | Douyin Shop draft writes only via the Ops Task Center | Legacy direct-create always returns 409; legacy/multi-target/batch/retry entry points make zero writes | Browser Network + before/after DB counts |
| 11 | Freeze is bound to approval | Worker validates the exact task/draft/approval/attempt/downstream/hash before accessing the platform | Audit timeline + tamper negative test |
| 12 | Unknown results are never auto-recreated | `result_unknown` is non-retryable; only manual, read-only reconciliation is allowed | Publish task, ops task, and platform draft box |
| 13 | Runtime controls take priority | Any of provider/tenant/shop/write kill switches blocks platform calls | Staged kill-switch drills |

---

## 4. Risk List (MVP Demo Release)

| Level | Risk | Impact | Mitigation |
| --- | --- | --- | --- |
| P0 | E2E not completed without real Douyin Shop credentials | Cannot prove field alignment with production | Run this checklist with a real app + test shop |
| P0 | `public_base` not public-facing | Image upload / Douyin Shop fetch fails | Configure a public HTTPS prefix in production |
| P0 | `product.addV2` / `product.detail` field consistency with production | Draft creation or SKU binding failure | Calibrate the payload in a real environment; log the requestId |
| P0 | Process interruption between platform success and local commit/write-back | The two task centers' states may temporarily diverge | Transactional outbox + task lease/reaper + manual read-only reconciliation, never auto-recreate |
| P1 | `order.searchList` pagination/time field differences | Missed orders or partial_success | Cross-check against the official docs and `pageErrors` |
| P1 | `sku.syncStock` parameter differences | Inventory sync failure | Retry via Failed Task Center + logs |
| P1 | High rate of ambiguous auto-matched SKUs | Requires manual binding before inventory can sync | Manual binding flow on the Publish tab |
| P2 | Scheduled order/inventory polling is disabled by default | Sync must be triggered manually | Documented behavior; not enabled by default |
| — | Direct listing | Bypasses review | **Not in MVP** — platform draft only |
| — | After-sales / finance / multi-warehouse | Scope creep | Explicitly out of scope for this release |

---

## 5. Manual Sign-off

**Pre-sign-off checks:**

- [ ] All of sections 2.1–2.18 in this checklist pass, or known issues are logged in PROGRESS as carryover items
- [ ] Affected GitHub Actions workflows — backend, contract, Admin build, and Admin E2E — pass
- [ ] PostgreSQL migration/unique constraint and Redis queue/outbox/reaper regressions pass in an isolated CI service container
- [ ] Production pre-checks, read-only checks, and controlled write paths are all performed manually by a maintainer in an authorized environment
- [ ] Backup, isolated recovery, application rollback, the four-tier write kill switch, and gray pause/stop drills are recorded in the release ticket
- [ ] Two different admins hold the Owner/Technical Lead approval roles respectively, and the single-tenant/single-shop/max-100-SKU scope matches the ticket
- [ ] The default L0 configuration has not been committed; the target environment's L3 allows only platform draft writes, with auto-retry and inventory mutation kept off
- [ ] Masking conclusions are recorded in the PR or release ticket; no JSON, Markdown reports, screenshots, or logs are committed to the repository
- [ ] `git diff --check` shows no conflict markers
- [ ] The relevant process in `PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md` has been manually signed off
- [ ] `docs/PROGRESS.md` has been updated with the phase status

Until CI, real credentials, backup/recovery, gray rollout, rollback, manual acceptance, and the release ticket are all signed off, the conclusion may only be "the code is a candidate capable of controlled production execution" — never recorded as "live in production."

---

## 6. Related Documents

- [`PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md`](PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md) — current manual acceptance checklist
- [`DOUYIN_E2E_PRECHECK_GUIDE.md`](DOUYIN_E2E_PRECHECK_GUIDE.md) — pre-checks before manual acceptance
- [`DOUYIN_PRODUCTION_RUNBOOK.md`](DOUYIN_PRODUCTION_RUNBOOK.md) — production operations and gray-rollout observation
- [`docs/PROGRESS.md`](PROGRESS.md) — phase progress and carryover items
- [`docs/api.md`](api.md) — API contract (including Douyin Shop observability)
- [`docs/provider.md`](provider.md) — Platform Provider notes
