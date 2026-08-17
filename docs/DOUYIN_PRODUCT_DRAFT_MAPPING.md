# Douyin Shop Product Draft Field Mapping

## Creation API

API: `product.addV2` (commit=false, start_sale_type=1)

## Field Mapping Table

| Internal Field | Douyin Field | Type | Notes |
|---------|-----------|------|------|
| Title | product_name | string(max 100) | AI-optimized title |
| Description | detail_text | string(max 5000) | AI-generated description, HTML or text |
| CategoryID | category_id | int64 | Must be a leaf category |
| Images[].PlatformImageID | image_list[].material_id | string | Obtained by uploading first |
| SKU.Price | spec_prices[].price | int64 | Unit: cents |
| SKU.Stock | spec_prices[].stock_num | int32 | Stock quantity |
| SKU.ExternalID | spec_prices[].sku_id | string | Internal SKU ID |
| Attributes | product_properties | []PropertyItem | Category attributes |
| Weight | delivery_weight | int32 | Unit: grams |

## Unsupported Fields

| Field | Reason |
|------|------|
| Brand | blocked_by_contract_verification |
| Video | Not implemented in P3; planned for P6 |
| Shipping lead time | Not implemented in P3 |

## outer_product_id

The internal product_draft_id is passed on every draft creation and used for `tryRecoverDouyinDraftFromPlatform` lookups.

## Notes

- `commit=false`: saves the draft only, does not trigger platform review
- The draft product_id differs from the ID of an already-published product
