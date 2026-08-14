# 批量刊登数据库迁移（Phase A2.1）

## 概述

Phase A2 通过 GORM AutoMigrate 扩展 `product_publish_batches` / `product_publish_tasks`。Phase A2.1 补充**显式 PostgreSQL migration**（Go 启动时执行）；生产维护阶段又增加 tenant ownership 回填，用于：

- 多商品批次 `product_id` 可空（兼容 `single_product` / `multi_product`）
- 批次列表、子任务查询索引
- 活跃批次 `idempotency_key` 部分唯一索引（防并发重复创建）
- 为历史 `product_publish_tasks`、`product_publications`、`product_publish_batches` 回填可确定的 `tenant_id`

实现文件：[`backend/internal/database/migrate_publish_batch_a21.go`](../backend/internal/database/migrate_publish_batch_a21.go)、[`backend/internal/database/migrate_product_publish_tenant.go`](../backend/internal/database/migrate_product_publish_tenant.go)

## 执行时机

服务启动时在 GORM **`AutoMigrate` 完成之后**调用（先由 model 补齐 `idempotency_key`、`batch_id` 等列，再建索引与约束）。可重复执行（`IF NOT EXISTS`）。

## 变更内容

### 1. `product_publish_batches.product_id` 可空

```sql
ALTER TABLE product_publish_batches ALTER COLUMN product_id DROP NOT NULL;
```

仅当 `information_schema` 显示仍为 `NOT NULL` 时执行。历史单商品批次数据不受影响。

### 2. 查询索引

| 索引名 | 表 | 列 | 用途 |
|--------|-----|-----|------|
| `ix_publish_batches_created_at` | product_publish_batches | created_at DESC | 最近批次列表 |
| `ix_publish_batches_status` | product_publish_batches | status | 按状态筛选 |
| `ix_publish_tasks_batch_id` | product_publish_tasks | batch_id | 批次详情子任务 |
| `ix_publish_tasks_target_key` | product_publish_tasks | target_key | 目标维度查询 |
| `ix_publish_tasks_batch_status` | product_publish_tasks | batch_id, status | retry-failed / cancel-pending |

### 3. 幂等键部分唯一索引

**先检查**活跃批次（`status NOT IN ('failed','cancelled')`）中是否存在重复 `idempotency_key`：

```sql
SELECT idempotency_key, COUNT(*) AS cnt
FROM product_publish_batches
WHERE idempotency_key <> ''
  AND status NOT IN ('failed','cancelled')
GROUP BY idempotency_key
HAVING COUNT(*) > 1;
```

- **无重复**：创建

```sql
CREATE UNIQUE INDEX IF NOT EXISTS ux_publish_batches_idempotency_active
 ON product_publish_batches (idempotency_key)
 WHERE idempotency_key <> '' AND status NOT IN ('failed','cancelled');
```

- **有重复**：跳过唯一索引，需人工清理后再重启服务。

#### 重复数据人工清理（示例）

保留 `created_at` 最新的一条，将其余活跃重复批次标记为 `failed` 或合并子任务后删除：

```sql
-- 示例：查看重复组
SELECT id, idempotency_key, status, created_at
FROM product_publish_batches
WHERE idempotency_key = '<duplicate_key>'
ORDER BY created_at DESC;
```

清理完成后重启服务，migration 会自动创建唯一索引。

### 4. 历史 tenant ownership 回填

`migrateProductPublishTenant` 在刊登表完成 AutoMigrate 后执行，并可重复运行：

- task 与 publication 从已存在的 `products.tenant_id` 回填。
- 单商品 batch 优先从其 `product_id` 回填；多商品 batch 从已回填的子任务 tenant 回填。
- 只更新 `tenant_id=0` 且归属可确定的行；孤立商品、孤立任务或无法确定归属的历史行继续保留为 `0`，不会猜测租户。

上线前应先备份并在隔离 PostgreSQL 副本核对 tenant-zero 残留。迁移不会删除或重分配无法确认的历史数据。

## 回滚

按需执行（**回滚 unique 索引不影响数据**）：

```sql
DROP INDEX IF EXISTS ux_publish_batches_idempotency_active;
DROP INDEX IF EXISTS ix_publish_tasks_batch_status;
DROP INDEX IF EXISTS ix_publish_tasks_target_key;
DROP INDEX IF EXISTS ix_publish_tasks_batch_id;
DROP INDEX IF EXISTS ix_publish_batches_status;
DROP INDEX IF EXISTS ix_publish_batches_created_at;
```

恢复 `product_id NOT NULL`（仅当确认无 multi_product 批次且全部有 product_id）：

```sql
-- 谨慎：仅在没有 NULL product_id 行时执行
ALTER TABLE product_publish_batches ALTER COLUMN product_id SET NOT NULL;
```

## 与 GORM AutoMigrate 的关系

列定义（`batch_type`、`name`、`task_count`、`idempotency_key` 等）仍由 GORM model + AutoMigrate 维护。本 migration **只**处理 nullable 约束与索引，不替代 model 演进。

## 变更记录

| 日期 | 说明 |
|------|------|
| 2026-08-14 | 增加刊登任务、publication 与批次 tenant ownership 幂等回填 |
| 2026-06-19 | Phase A2.1 初始 migration |
