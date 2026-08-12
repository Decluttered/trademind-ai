# AI 客服回复设计

## 流程

默认流程：选择会话 → 生成建议 → 人工编辑 → 二次确认 → 发送（或仅采纳为内部记录）。

可选自动流程：部署级轮询器仅扫描已启用店铺并创建增量消息同步任务 → 写入幂等运行记录 → 独立 Redis 队列 → 生成建议 → 风险、订单上下文、时序、频率、长度与回复内容检查 → 低风险外发或转人工。部署总开关和店铺策略默认均关闭。

## API

- `POST /api/v1/customer/conversations/:id/ai/generate-reply`
- `GET /api/v1/customer/conversations/:id/ai-suggestions`
- `PUT /api/v1/customer/reply-suggestions/:id`
- `POST /api/v1/customer/ai-suggestions/:id/apply`
- `POST /api/v1/customer/ai-suggestions/:id/reject`
- `GET /api/v1/customer/shops/:shopId/auto-reply-policy`
- `PUT /api/v1/customer/shops/:shopId/auto-reply-policy`
- `GET /api/v1/customer/shops/:shopId/auto-reply-runs`

## 状态

`generated` / `edited` / `accepted` / `rejected` / `discarded` / `generate_failed` / `send_failed`

## 上下文

Prompt 使用订单、商品、库存摘要；前端仅展示 `contextSummary`，不含完整 Prompt 与平台 raw。

## 失败

生成/发送失败写入 `customer_failure_events`，聚合至失败任务中心 `customer_failure`。自动回复在 `sending` 阶段租约过期时也必须原子转人工并创建失败事件，避免未知平台结果只停留在最近记录中。

自动回复还写入 `customer_auto_reply_runs`。运行状态为 `pending` / `generating` / `sending` / `sent` / `human_required` / `skipped` / `failed`。数据库运行记录是事实源，Redis ready/processing 列表只负责唤醒；`generating` 租约过期可恢复为 `pending`，`sending` 租约过期或平台结果未知必须进入 `human_required/platform_send_result_unknown`，绝不自动重发。

## 生产边界

- Admin「客服 / AI 自动回复」中的消息同步和自动回复总开关默认关闭，设置持久化到数据库并动态生效。
- 持续自动收件要求页面中的「自动同步客服消息」开启；Redis 或后台 Worker 不可用时仍保持不可用。
- 店铺只能由具备设置管理和客服写权限的账号开启，Admin 必须二次确认。
- 自动发送固定为 `lowRiskOnly=true`，API 不允许放宽。
- 退款、赔付、投诉等敏感输入由风险规则升级；模型输出含退款、赔付、返现等承诺时再次转人工。
- 默认要求会话关联订单；只处理最新且尚未被人工回复的客户文本消息。
- 人工回复、平台同步和自动发送共享会话 mutation lock；模型生成后会在锁内再次检查最新客户消息、人工回复、会话状态与频率限制，再进入 `sending`。
- 多实例轮询使用数据库 `next_poll_at` 原子认领；单店 `pending/running` 同步任务和平台消息外部 ID 由 PostgreSQL 唯一索引兜底。
- Redis 队列或限流不可用时 fail closed 并记录失败，不回退到请求内直接外发。
- `workerAvailable` 代表 Redis、消息同步 Worker、轮询调度器和自动回复消费者整条链路均存活，不仅是 Redis 客户端已创建。
- 平台发送使用持续续期的执行幂等租约；本地消息按 `conversation_id + client_message_id` 唯一保存。平台成功并落库后，幂等完成记录失败不会向调用方返回可重试错误，后续请求先重放本地消息。
- PostgreSQL 平台消息去重迁移在删除重复行前检查建议引用、自动回复来源消息引用和 `sent_message_id` 引用，存在任何引用时失败关闭并要求人工处理。
