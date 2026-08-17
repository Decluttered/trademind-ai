# Douyin Shop Customer Messaging Adapter Design

## Current Status

**blocked_by_contract_verification** — The Douyin Shop IM messaging API requires contract approval before it can be used.

## What's Implemented

- `CustomerCapability` interface (`douyinshop/customer.go`)
- `PullMessages` / `SendMessage`: return `CodeDouyinContractMismatch`
- `CustomerMessageEnvelope` DTO (used for fixture-driven parsing tests)
- `ParseCustomerMessageEnvelope`: for synthetic tests only, not applicable to the real API

## API Shape (Pending Contract Confirmation)

The following field names are speculative and **must not be used for real API calls**:

```json
{
  "messages": [
    {
      "message_id": "...",
      "conversation_id": "...",
      "content": "...",
      "content_type": "text",
      "sender_type": "buyer"
    }
  ],
  "next_token": "..."
}
```

## Activation Conditions

1. Apply for IM API access through the Douyin Shop Open Platform
2. Set `platform_douyin_shop.customer_message_api_enabled = true` in settings
3. Configure the message pull/push API paths
4. Remove the `contractMismatchError` guard

## MVP Constraint

Customer messages must be manually confirmed before sending; no automatic outbound messaging (per the rule in `.cursorrules`).
