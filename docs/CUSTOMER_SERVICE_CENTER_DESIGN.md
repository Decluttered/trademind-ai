# 客服中心设计

## 入口

- 客服中心首页：`/customer/hub`（KPI + 快捷入口）
- 会话列表：`/customer/conversations`
- 会话详情：`/customer/conversations/:id`
- 消息同步任务：`/customer/message-sync-tasks`
- AI 自动回复策略：`/customer/auto-reply-settings`
- API 别名：`/api/v1/customer/*` 与 `/api/v1/customer-service/*`

## 列表字段

平台、店铺、买家（脱敏）、状态、最近消息、关联订单/商品、AI 建议状态、发送状态、更新时间。

## 权限

- `admin` / `operator`：可生成建议、发送、重试
- `readonly`：仅查看（后端 `CanWriteCustomer` + 前端 `canWrite`）

## 原则

默认人工确认发送；自动回复必须由部署总开关和店铺策略双重显式开启，且仅允许低风险消息。系统不展示平台 raw，技术详情默认折叠；数据库租约与可靠 Redis processing 队列负责崩溃恢复，发送结果未知时转人工且不自动重试。
