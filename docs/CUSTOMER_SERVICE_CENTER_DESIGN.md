# Customer Service Center Design

## Entry Points

- Customer service center home: `/customer/hub` (KPIs + quick-access entries)
- Conversation list: `/customer/conversations`
- Conversation detail: `/customer/conversations/:id`
- Message sync tasks: `/customer/message-sync-tasks`
- AI auto-reply policy: `/customer/auto-reply-settings`
- API aliases: `/api/v1/customer/*` and `/api/v1/customer-service/*`

## List Fields

Platform, shop, buyer (masked), status, latest message, related order/product, AI suggestion status, send status, updated time.

## Permissions

- `admin` / `operator`: can generate suggestions, send, and retry
- `readonly`: view only (backend `CanWriteCustomer` + frontend `canWrite`)

## Principles

Sending defaults to human confirmation; auto-reply must be explicitly enabled through both the global deployment switch and the per-shop policy, and is only permitted for low-risk messages. The system does not display raw platform payloads; technical details are collapsed by default. Database leases and a reliable Redis processing queue handle crash recovery; when the send result is unknown, the conversation is handed off to a human and not retried automatically.
