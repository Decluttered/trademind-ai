# Operation Audit Design (Phase F5)

## Table

`operation_logs` extended fields: `adminRole`, `shopId`, `platform`

## Sensitive Operations That Must Be Logged

User role/shop permission changes, AI application, publish drafts, SKU binding, inventory/customer-service/task retries, system configuration and storage tests, Douyin authorization, etc.

## Must Not Be Logged

Full secret keys, tokens, full prompts, buyer plaintext sensitive information.

## View Permissions

- admin: all records
- operator/readonly: filtered by authorized shop (records with a non-null `shop_id`)

## API

`GET /api/v1/operation-logs` — requires `operationlog.view`
