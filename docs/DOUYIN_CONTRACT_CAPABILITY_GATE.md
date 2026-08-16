# Douyin Contract Capability Gate (ContractCapabilityGate)

Implementation: `backend/internal/providers/platform/douyinshop/contract_gate.go`

## State Model

| State | Meaning |
|------|------|
| verified | Confirmed in a real environment or via explicit contract confirmation |
| fixture_verified | Verified only against fixtures/transport tests |
| blocked_by_contract_verification | Path/scope/fields not yet confirmed |
| unsupported | Not supported in the current version |
| disabled | Feature flag disabled |

## Capability Keys

- `douyin_im_conversation_list` / `douyin_im_message_list` / `douyin_im_send` → blocked
- `douyin_brand_list` → blocked
- `douyin_webhook_signature_v1` → fixture_verified (production rejects anything not verified)
- `douyin_order_webhook_events` → fixture_verified
- `douyin_inventory_query` / `douyin_product_draft_create` → fixture_verified

Blocking error: `DOUYIN_CONTRACT_VERIFICATION_REQUIRED`
