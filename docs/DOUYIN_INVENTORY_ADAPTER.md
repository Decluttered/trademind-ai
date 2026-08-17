# Douyin Shop Inventory Adapter Design

## Write Path (Inventory Push)

API: `sku.syncStock`

```go
params = {
  product_id:  platformProductID,
  sku_id:      platformSKUID,
  stock_num:   stock,  // >= 0
  incremental: false,  // full overwrite
}
```

Idempotency key: `inventory-push:{platform}:{shopId}:{skuId}:{stockVersion}`

## Read Path (Inventory Query)

**No dedicated stock query API** (design decision): the standard Douyin OpenAPI contract has no dedicated inventory query endpoint.

Read approach: parse the SKU stock field from the `spec_prices` field returned by `product.detail`.

Function: `GetSKUStockFromDetail(ctx, client, shopID, platformProductID, platformSKUID)`

If platformSKUID is empty, returns the sum of stock across all SKUs.

## Error Handling

| Error | Handling |
|------|------|
| DOUYIN_INVENTORY_RATE_LIMITED | Wait for Retry-After |
| DOUYIN_SKU_NOT_BOUND | Validation failure, no retry |
| DOUYIN_PRODUCT_NOT_BOUND | Validation failure, no retry |
| DOUYIN_UNKNOWN_RESULT (push timeout) | ManualReviewRequired=true |
