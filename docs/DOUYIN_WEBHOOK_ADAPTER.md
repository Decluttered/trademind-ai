# Douyin Shop Webhook Adapter Design

## Signature Verification Algorithm

Source: Douyin Shop Open Platform webhook documentation

```
signature = SHA1( appSecret + rawBody )  →  hex lowercase
```

Verification header: prefers `X-Douyin-Signature`, falls back to `X-Sign`

Implementation: `douyinshop/webhook_sign.go` `DouyinSignatureVerifier`

## Event Formats

### Standard Douyin Format
```json
{"event": "order_created", "client_key": "xxx", "content": {...}}
```

### jinritemai Array Format
```json
[{"tag": "100", "msg_id": "xxx", "data": {...}}]
```

### Tag → EventType Mapping

| Tag | EventType |
|-----|-----------|
| 0 | health_check (safe ACK, not processed) |
| 100 | order_created |
| 101 | order_paid |
| 102 | order_shipped |
| 103 | order_completed |
| 104 | order_cancelled |
| 200 | inventory_alert (P3 logs only) |
| 300 | product_status_changed (P3 logs only) |
| Unknown | unknown:{tag} (safe ACK + warning log) |

## Registration

After `webhook.NewRegistry` in `api/router.go`, the secret is loaded from `platform_douyin_shop.app_secret`:

```go
webhookRegistry.Register("douyin_shop", webhook.NewDouyinVerifier(appSecret))
webhookRegistry.Register("douyin", webhook.NewDouyinVerifier(appSecret))
```

If the secret is empty, the verifier is still registered but `Verify` returns `CodeVerifierNotConfigured`.

## Event Handling Path

```
webhook.Handler.Receive
  → webhook.Service.Ingest (persistence + idempotency)
  → webhook.Service.ProcessEvent (async)
    → handlePlatformEvent
      → Service.HandleDouyinPlatformEvent
        → ParseDouyinWebhookEnvelope / ParseJinriteimaiPushEnvelope
          → douyinEventDispatcher.DispatchDouyinEvent
            → OrderEventHandler.HandleDouyinOrderEvent (P3 placeholder)
```

## Unknown Event Handling Principle

Unknown tags/events must always be safely ACKed (return 200) — they must never be silently dropped. A `slog.Warn` log entry must be recorded.

## P3.2 Multi-Shop Routing

Douyin webhook business handling is shop-scoped. After signature verification and JSON validation, the handler extracts `client_key` / app ID, platform shop ID, and optional binding ID, then resolves them through `shops` + `shop_auth_tokens`. The persisted event carries `tenant_id`, `internal_shop_id`, `platform_shop_id`, `app_id`, and `binding_id`.

The resolver rejects missing, ambiguous, mismatched, expired, or revoked bindings. Staging and production reject `DOUYIN_WEBHOOK_TEST_SHOP_BINDING_ID` and `ENABLE_DOUYIN_WEBHOOK_DEMO_FALLBACK` at config validation time.
