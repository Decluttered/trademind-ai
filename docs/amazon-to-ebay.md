# Amazon → eBay Workflow

This document is the product overlay on TradeMind's Provider architecture. It **extends** the accepted MindBay ADRs and [mindbay-module-matrix.md](mindbay-module-matrix.md). It does not start a parallel Douyin-style publish domain.

Canonical decisions:

- [ADR-0001](adr/0001-ebay-go-client.md) — native Go eBay Sell client
- [ADR-0002](adr/0002-temporal-hybrid-orchestration.md) — Temporal for durable eBay publish; Redis for short jobs
- [ADR-0003](adr/0003-amazon-source-collector.md) — Amazon.de as Collector source
- [ADR-0004](adr/0004-wxt-companion-extension.md) — companion extension capture

## Position in the platform

```
Amazon.de (Collector Provider: sourceAmazon)
    ↓  collect (Collector must not write the domain DB)
Catalog snapshots (modules/catalog, immutable)
    ↓
Listing Studio draft (modules/listingstudio, cents, GPSR)
    ↓  calendar slot
eBay publish (modules/publication + Temporal + providers/platform/ebay)
    ↓
Monitor / reprice / sale detect (modules/monitoring)
    ↓
Amazon retail checkout → tracking → profit ledger
```

Nothing in the Amazon collector imports the eBay provider. Nothing in the eBay provider imports Amazon source code. They meet at the neutral draft. Routing eBay writes through Redis `productpublish` is a design error (matrix: **Leave**).

Product scope is **Amazon.de source → eBay.de target only**. Do not add TikTok, Shopee, Lazada, Shopify, Douyin, 1688, Taobao, Pinduoduo, or AliExpress as sources or publish targets. Shared infrastructure (auth, settings, AI, storage, shops, task center) stays and is reused.

## Amazon acquisition mode

Recorded from ADR-0003 (do not re-litigate in feature PRs):

| Mode | When it applies | Status |
| --- | --- | --- |
| Official read APIs (SP-API / PA-API) | Operator has the matching read credentials | use when available |
| Playwright (`sourceAmazon`) | Equal fallback for public Amazon.de product pages | implemented |
| Companion Extension | Controlled capture with short-lived tokens | implemented |
| Operator-supplied (CSV / ASIN / paste) | Always available fallback | supported |

`providers/platform/amazon/` remains an Amazon **selling-channel** adapter. Do not turn it into the source collector. A later `AmazonRetailProvider` for checkout is a separate fulfillment/browser adapter.

**Marketplace is a required field.** `.de`, `.com`, and `.co.uk` differ in currency, locale, category semantics, and compliance text. Never infer it silently in new code.

## Content reuse boundary

Amazon product photography, A+ content, and brand imagery belong to the seller or brand that published them. The pipeline stores them as references with source attribution and flags them on the draft. It does not silently republish them to eBay. The operator decides; the AI content step exists so listings can be rewritten rather than copied.

This is a product requirement: republished listings are a common cause of eBay VeRO takedowns.

## Mapping gaps to expect

Amazon and eBay do not share a category or attribute model. Degrade honestly rather than guess.

| Field | Behaviour |
| --- | --- |
| Category | Amazon browse node does not map to an eBay leaf category. Operator selects, optionally AI-suggested. Never auto-committed. |
| Item aspects | Map confidently-known values only. Unmapped **required** aspects become mapping errors and block publishing. Never invent a value. |
| Title | eBay enforces a length limit. Truncation emits a warning and routes to the AI title step. Never silent. |
| Identifiers (EAN / GTIN / MPN / Brand) | If the source has no valid identifier, use eBay's documented "does not apply" value. Never fabricate. |
| Description | Bullet points and A+ content go through AI rewrite plus operator confirmation, not direct copy. |
| Price | Integer cents + ISO currency. Conversion, fees, and margin are operator/rule decisions in Listing Studio / monitoring, not a hidden float transform. |
| Variations | Amazon variation families map to the neutral product + SKU structure, then to eBay SKU-based offers. Never flatten. |

Listing preparation for eBay lives on MindBay drafts / publication jobs, not on Douyin `product_platform_publish_configs`.

## eBay publishing preconditions

An offer cannot be published unless all of these hold. The readiness check validates each and returns a distinct code.

- Seller has valid fulfillment, payment, and return business policies, referenced by the offer
- Leaf category selected and cached fresh
- All required item aspects for that category populated
- Condition, quantity, price, currency, and merchant location set
- Images satisfy eBay's hosting, count, and dimension rules
- SKU unique within the seller's inventory
- GPSR fields as required for the marketplace

## Environment separation

Sandbox and production have separate credentials and hosts. Environment is an explicit setting (`EBAY_ENV` / `platform_ebay.environment`). A publish action in sandbox must be visually distinguishable in the admin UI from production.

OAuth `redirect_uri` is the Developer Portal **RuName**, not the registered https callback. Consent requests `sell.account.readonly` for Account API getPrivileges. Existing refresh tokens do not gain new scopes; shops must re-authorize after this change. Shop connection tests call getPrivileges; app-settings tests use a client-credentials application token only.

## Out of scope

- Other marketplaces or suppliers
- Physical warehouse / WMS
- Returns, cases, disputes, or refund handling (until explicitly scheduled)
- Tax, VAT, OSS, or DAC7 as a tax engine
- A second domain model or a Douyin-mirror publish stack for eBay
