# 架构设计

TradeMind 采用 monorepo 组织，核心由 Go backend、React admin、Node collector、PostgreSQL 与 Redis 组成。

## 总体架构

```text
React + Ant Design Pro Admin
        ↓
Go Gin API
        ↓
PostgreSQL + Redis（短任务）+ Temporal（eBay 刊登）
        ↓
Node Playwright Collector（Amazon.de）
```

## 目录职责

```text
backend/    Go Gin API、业务模块、Provider、数据库迁移、队列 Worker
admin/      React + TypeScript + Ant Design Pro 管理后台
collector/  Node.js + TypeScript + Playwright 商品采集服务
docs/       项目文档
scripts/    本地开发编排脚本
```

## 后端分层

- `handler`：处理请求参数、鉴权上下文与响应。
- `service`：编排业务流程与事务。
- `providers`：封装 AI、存储、图片、平台、采集等外部能力。
- `modules`：按业务域组织认证、设置、商品、采集、AI、图片、店铺、订单、客服等模块。
- `queue` / worker：处理采集、图片、短异步任务。eBay 刊登由 Temporal + `modules/publication` 编排，不走 Redis `productpublish`。

## 管理端分层

- 页面使用 React + TypeScript。
- 表格优先使用 ProTable。
- 表单优先使用 ProForm。
- API 请求统一放在 services。
- 敏感配置展示必须脱敏。
- 前端不直接调用第三方 AI、平台或存储 API。

## 采集服务

Collector 是独立 Node.js 服务：

- 当前货源是 Amazon.de（`sourceAmazon`）。
- 不直接操作主业务数据库。
- 通过 HTTP 与 Go backend 通信。
- 每个采集源以 Collector Provider 形式接入；不要再注册 1688 / 淘宝 / 拼多多 / AliExpress。

## 数据与队列

- PostgreSQL 是默认数据库。
- Redis 用于异步任务队列、租约与部分状态协调。
- 敏感配置通过后端 AES-GCM 加密存储。

### 数据库对象命名

持久化表、索引、约束和触发器使用业务领域命名，不使用开发阶段、批次或验收编号。库存读取与 SKU 绑定使用 `inventory_*`、`sku_*`、`manual_binding_*`；平台凭据与 OAuth 使用 `platform_*`；生产保护控制使用 `production_*`。

旧版 `p9_*` / `p10_*` 表、历史图像任务子表 `ai_image_task_items` 以及带 `p7` 阶段号的性能索引，会在 `AutoMigrate` 创建当前模型前执行事务内原地重命名；图像任务子表的当前名称为 `image_task_items`。数据、主键和外键关系保持不变，关联索引、约束、触发器及不可变保护函数同步改名。旧名和新名同时存在时默认拒绝启动；唯一例外是同一表上的旧名与当前名索引经 `pg_get_indexdef` 确认除索引名外完全等价，此时只删除旧名重复索引。唯一性、表、访问方法、列、表达式或条件不同的索引仍 fail closed，禁止自动合并或覆盖数据。

该重命名不兼容仍引用旧表名的历史后端二进制。生产部署必须先备份并使用维护窗口或协调发布，确保迁移期间不混跑新旧后端；回滚应用版本前必须先执行经审核的反向表名迁移。

## 安全原则

- 不在代码中写死 API Key、Token、Secret、平台凭证。
- 日志禁止输出完整密钥、密码、Cookie、Token。
- AI、存储、平台能力必须由后端 Provider 调用。
- AI 客服默认只生成建议，外发必须人工确认。

## 扩展方向

TradeMind 的产品路径是 Amazon.de → eBay.de。扩展点包括：

- AI Provider
- Storage Provider
- Image Provider
- Platform Provider（`ebay`）
- Collector Provider（`amazon.de`）
- Prompt 模板
- Temporal workflow（刊登）
- Redis Worker（短任务）
