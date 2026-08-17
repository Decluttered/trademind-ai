# Douyin Category Sync Design

## API

| Function | Method | Parameters |
|------|------|------|
| Get category tree | `shop.getShopCategory` | shop_id |
| Get category attributes | `product.getCatePropertyV2` | category_id |

## Data Flow

```
GetCategories()
  → shop.getShopCategory
  → returns []Category{ID, ParentID, Name, IsLeaf}

GetCategoryAttributes(categoryID)
  → product.getCatePropertyV2
  → returns []CategoryAttribute{PropertyID, Name, Required, Options}
```

## Caching Strategy

- The category tree changes infrequently; cache in Redis for 1 hour
- Cache attribute lists by category_id for 30 minutes

## Field Mapping

| Douyin field | Internal field | Description |
|------------|---------|------|
| category_id | PlatformCategoryID | |
| leaf | IsLeaf | true means a leaf category that can host products |
| property_id | AttributeID | |
| option_value_id | OptionID | |
| required | Required | Whether the attribute is required |

## Notes

- Brand list `blocked_by_contract_verification`: `brand.go` returns an explicit unsupported error
- The `standard_brand_id` field is written via product mapping, not matched from the brand list
