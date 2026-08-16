# Douyin Shop Image Upload Design

## API

Official endpoint: `supplyCenter.material.batchUploadImageSync`

Supports: a public URL (SourceURL) or raw file bytes (Reader + MimeType)

## Idempotency Cache (DouyinImageAsset)

```
Before upload, check:
  SELECT * FROM douyin_image_assets
  WHERE shop_id = ? AND content_hash = ?
  AND status = 'uploaded'
  → on a hit, reuse platform_image_id

After upload:
  → success: INSERT/UPDATE status='uploaded', platform_image_id
  → timeout: status='unknown_result', ManualReviewRequired=true
  → failure: status='failed'
```

Idempotency key: `douyin-image-upload:{shopId}:{storageObjectKey}:{contentHash}`

## Content Hash

SHA256 of the image bytes (computed before storage)

## unknown_result Handling

If the upload times out:
1. `DouyinImageAsset.Status = unknown_result`
2. Before the next upload with the same content_hash, first query the platform via imageId to confirm
3. Manual review path: check the Douyin Shop asset center

## Image Types

| type | Purpose |
|------|------|
| main | Main image |
| desc | Detail image |
| sku | SKU image |
