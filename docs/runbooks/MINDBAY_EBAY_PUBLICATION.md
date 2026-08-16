# MindBay eBay publication runbook

## Safety baseline

- Keep `AUTOMATION_MODE=DRY_RUN`, `EBAY_ENV=sandbox`, and `TEMPORAL_ENABLED=false` until the Sandbox account, business policies, inventory location, OAuth scopes, and GPSR data have been reviewed.
- eBay Client Secret and user tokens are stored through encrypted settings/shop auth paths. Never place them in source, fixtures, logs, or request artifacts.
- Admin, extension, MCP, and Temporal code do not call eBay directly. Only the Go eBay adapter may mutate Inventory Item and Offer resources.

## Local workflow

1. Set a random `TEMPORAL_SERVICE_TOKEN` in the ignored `.env`.
2. Start `docker compose -f docker-compose.full.yml --profile mindbay up --build`.
3. Configure `platform_ebay` and the eBay publish policy IDs in Admin.
4. Complete eBay Sandbox OAuth for an eBay shop.
5. For categories that require product-safety labels, enter Metadata API statement IDs such as `EBPSS102`; free-form GPSR safety text is never guessed into an eBay code.
6. Create a Planner preview. Confirm that preview created no `calendar_slot` or `publication_job` rows.
7. Apply the preview with a stable Idempotency-Key. A repeat must return the same slots/jobs.
8. In DRY_RUN, verify the job ends as `DRY_RUN`, the slot as `HELD`, the listing returns to `READY`, and no eBay mutating request was sent.

## Sandbox acceptance

For an explicitly approved Sandbox run, use `EBAY_ENV=sandbox` and `AUTOMATION_MODE=SIMULATED_CHECKOUT`. Verify exactly one Inventory Item, one Offer, one published Sandbox listing, one local `marketplace_listing`, and redacted artifacts. A timeout after createOffer must reconcile by SKU before any retry.

## Failure recovery

- Validation/GPSR errors: fix the listing and validate it back to READY. Only a persisted workspace-enabled, reasoned and audited GPSR override may publish without a complete profile; request payloads cannot assert an override themselves.
- OAuth 401/403: reauthorize the shop; never copy tokens into job payloads.
- 429/5xx: Temporal performs at most three bounded activity attempts. Provider unknown results reconcile before retry.
- `UNKNOWN_RESULT`: inspect eBay by stable SKU/Offer ID and the local publication job. Do not manually create another offer.
- Production publishing requires both `EBAY_ENV=production` and explicit `AUTOMATION_MODE=LIVE`; the provider rejects every other combination.

## Browser fallback decision

Phase 2 uses the eBay Inventory and Taxonomy APIs for the implemented happy path. No concrete API coverage gap was found, so no browser writer is enabled. A future fallback requires a documented API gap plus encrypted workspace session storage and the mandatory action-log/screenshot/trace artifacts from the phase specification; it must never become a second writer for the same publication.
