---
name: api-contract-testing
description: TradeMind API contract, frontend/backend DTO, Admin mocks, response envelope, and drift-detection rules for key endpoints
---

# TradeMind API Contract Testing Standard

## Automatic Scope

Automatically applies to work involving API URLs, HTTP methods, query params, payloads, DTOs, response envelopes, error structures, frontend service/types, backend handler/DTOs, or Admin E2E mocks.

## Current Repo State

There is currently no complete OpenAPI/Swagger file. Do not try to force a full-project OpenAPI spec in one go; for now, cross-check using the key-endpoint contract list, shared fixtures, and backend/frontend tests together.

## Contract Scope

Covers at least:

- `GET /api/v1/auth/profile`
- `GET /api/v1/image/providers`
- `GET /api/v1/products/:id`
- `GET /api/v1/products/:id/readiness`
- `GET /api/v1/products/:id/publications`
- `GET /api/v1/product-publications/:id/douyin/sku-bindings`
- `GET /api/v1/products/:id/publish-targets`
- `POST /api/v1/products/:id/platform-configs/douyin_shop/create-draft`
- `POST /api/v1/products/:id/publish`

## Must Verify

- Method, URL, path params, query, request body.
- Success envelope: `{ code, message, data, traceId? }`.
- Error envelope: `code !== 0`, message, data/null, traceId.
- Data shape, pagination, nullability, enums, and business error codes/messages.
- That Admin E2E mocks match the real backend route's method/data/envelope.

## Implementation Approach

- The contract list lives under `tests/contracts/**`.
- Frontend unit tests verify that the service layer handles the contract's URL/payload/envelope correctly.
- Go tests verify handlers/DTOs/envelopes, or at minimum the route table and key shapes.
- Admin Playwright `@contract` tests continue to verify that browser mocks consume the contract correctly.

## Prohibited

Do not treat "the frontend mock runs" as proof that the real backend contract is correct. Do not verify only HTTP 200. Do not manually duplicate large, drift-prone schemas; when a shared runtime schema isn't available, pin fixtures for key endpoints only.
