# 抖店整链路验收清单（E2E Checklist）

> 用于 **真实抖店凭证 + 公网 Storage** 环境下的人工端到端验收。
> 自动化回归由 GitHub Actions 在隔离环境执行；真实平台写链路必须获得外部生产审批，不由仓库脚本自动触发。
> 通用人工签收见 [`PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md`](PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md)。验收结论记录在 PR 或发布工单，不向仓库提交测试报告、截图或日志产物。

---

## 1. 验收前准备

### 1.1 验收边界

- 无 App Key / Secret、未授权店铺或缺少公网 Storage 时，结论必须记录为 **`blocked_by_real_credentials`** 或 **`blocked_by_environment`**，不得伪造通过。
- 只读检查可由维护者在受控环境人工执行；任何真实平台写操作必须先取得明确审批。
- 仓库默认 `L0`。只有发布工单批准的 `L3` 才允许运营任务中心创建 `save_as_platform_draft`；不允许正式发布、上架、库存写入、自动业务重试、无审核执行或多店扩容。
- 验收结果记录在 PR 或发布工单，不向仓库提交 JSON、Markdown 报告、截图或日志产物。

### 1.2 手工检查项

| 项 | 说明 | 如何确认 |
| --- | --- | --- |
| 抖店开放平台 App Key | 在抖店开放平台创建应用后获取 | 设置 → 平台开放配置 → 抖店 |
| 抖店开放平台 App Secret | 加密存库，前端仅脱敏展示 | 保存后重载仍为 `****` 占位 |
| 抖店回调地址 Redirect URI | 须与开放平台登记完全一致 | 与 `GET /api/v1/shops/oauth/douyin/callback` 对外 URL 一致 |
| Service ID | 服务市场自定义授权 URL 必填 | 平台开放配置中已填写 |
| 已授权抖店店铺 | OAuth 完成且连接测试通过 | 店铺管理 → 授权状态 **已授权** |
| Storage `public_base` | **必须是抖店可访问的公网 HTTPS 地址** | 设置 → 存储；本地 `/static` 仅开发代理，生产须公网域名 |
| `order_sync_enabled` | 订单同步开关 | 平台开放配置 → **启用订单同步** = 开 |
| `inventory_sync_enabled` | 库存同步开关 | 平台开放配置 → **启用库存同步** = 开 |
| `product_publish_enabled` | 商品草稿创建开关 | 建议开启 |
| P10 L3 草稿写配置 | Provider/网络/凭据/草稿写/Worker=true，刊登队列和任务回收器=true，自动重试与库存 mutation=false | 后端启动校验通过，`GET /api/v1/p10/status` 与工单一致 |
| 生产控制范围 | 单租户、单白名单已授权店铺、`maxSku<=100` | allowlist 与 active gray 指向同一店铺 |
| 双人灰度审批 | 两名不同管理员分别承担 Owner 与 Technical Lead 职责 | 工单身份核验 + gray revision 审计 |
| 写入 kill switch | provider / tenant / shop / write 默认阻断 | 最终 go/no-go 前保持 active，并完成逐级演练 |
| 测试商品 | 至少 1 个可编辑草稿 | 含标题、价格 |
| 有 SKU 的商品 | 至少 1 个规格 | 采集或手工维护 |
| 主图 + 详情图 | 各至少 1 张 | 发布前检查会通过 |

**不在本 MVP 验收范围：** 直接上架 `publish_online`、售后/退款、财务结算、多仓 WMS、自动补货、复杂 BI。

---

## 2. 验收步骤与预期

每一步均包含：操作入口、预期成功、常见失败、排查位置、失败任务中心、是否可重试。

### 2.1 配置抖店应用

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 设置 → 平台开放配置 → 抖店 / Douyin Shop |
| **预期成功** | App Key、Redirect URI、环境、超时保存成功；App Secret 脱敏；开关项可保存 |
| **常见失败** | 必填项缺失；Redirect URI 与开放平台不一致 |
| **排查位置** | 当前页表单校验；操作日志 `platform.settings.update` |
| **失败任务中心** | 否 |
| **可重试** | 是（补全配置后保存） |

### 2.2 测试连接

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 同上页 → **测试连接** |
| **预期成功** | 提示配置完整性通过（Phase 1 不做商品/订单真实调用） |
| **常见失败** | App Key/Secret 错误；网络超时 |
| **排查位置** | 页面 Toast；操作日志 |
| **失败任务中心** | 否 |
| **可重试** | 是 |

### 2.3 授权店铺

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 平台开放配置 → **连接店铺**，或 店铺管理 → 抖店 → 授权 |
| **预期成功** | 跳转抖店 OAuth → 回调后店铺状态 **已授权**；不返回 token 明文 |
| **常见失败** | 未填 Service ID；回调地址不可达；用户取消授权 |
| **排查位置** | 店铺管理详情；操作日志 `douyin.auth.*` |
| **失败任务中心** | 否（授权失败 URL 带 `auth=failed&reason=`） |
| **可重试** | 是（重新发起授权） |

### 2.4 同步类目

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 平台开放配置 → **同步类目**（需已授权店铺） |
| **预期成功** | 类目缓存 Alert 显示数量与最近同步时间 |
| **常见失败** | 未授权店铺；token 过期；权限不足 |
| **排查位置** | 平台开放配置页类目 Alert；`GET /api/v1/platform/douyin/categories/stats` |
| **失败任务中心** | 否 |
| **可重试** | 是（刷新授权后重试） |

### 2.5 同步属性

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 商品详情 → 刊登 Tab → 抖店类目与属性 → **刷新属性** |
| **预期成功** | 选中叶子类目后可加载必填属性表单 |
| **常见失败** | 类目未同步；非叶子类目；token 过期 |
| **排查位置** | 刊登 Tab 属性区 Alert；API `.../categories/:id/attributes/sync` |
| **失败任务中心** | 否 |
| **可重试** | 是 |

### 2.6 采集商品（1688 / 拼多多 / 淘宝天猫）

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 采集中心 → 对应采集器 → 单采或批量 |
| **预期成功** | 任务 success → 生成 `status=draft` 商品草稿 |
| **常见失败** | 未登录；验证码；链接无效；主图缺失 |
| **排查位置** | 采集任务页；失败任务中心（采集类） |
| **失败任务中心** | 是 |
| **可重试** | 是（单条重试） |

### 2.7 AI 标题优化

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 商品详情 → 基础信息 → AI 优化标题 |
| **预期成功** | 生成候选标题 → **应用到草稿** 写入 `aiTitle` |
| **常见失败** | AI Provider 未配置；超时 |
| **排查位置** | AI 任务页；失败任务中心（ai 类） |
| **失败任务中心** | 是 |
| **可重试** | 是 |

### 2.8 AI 描述生成

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 商品详情 → 基础信息 → AI 生成描述 |
| **预期成功** | 生成描述 → 应用到 `aiDescription` |
| **常见失败** | 同 AI 标题 |
| **排查位置** | AI 任务页 |
| **失败任务中心** | 是 |
| **可重试** | 是 |

### 2.9 应用定价规则

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 商品详情 → 刊登 Tab → **应用定价规则** |
| **预期成功** | SKU 销售价更新；操作日志 `pricing.apply` |
| **常见失败** | 成本价缺失；规则未配置 |
| **排查位置** | 刊登 Tab SKU 表；设置 → 定价 |
| **失败任务中心** | 否 |
| **可重试** | 是 |

### 2.10 补全抖店类目属性

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 刊登 Tab → 选择抖店店铺 + 叶子类目 + 必填属性 → **保存抖店刊登配置** |
| **预期成功** | 配置保存；发布前检查类目/属性项 passed |
| **常见失败** | 未选店铺；未选叶子类目；必填属性空 |
| **排查位置** | 刊登 Tab 发布前检查；`product_platform_publish_configs` |
| **失败任务中心** | 否 |
| **可重试** | 是 |

### 2.11 生成抖店刊登草稿

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 刊登 Tab → **生成抖店刊登草稿** |
| **预期成功** | 预览标题/描述/主图/详情图/SKU/价格/库存；errors 为空才可创建 |
| **常见失败** | 标题/主图/类目/属性/SKU 价格无效 |
| **排查位置** | 刊登 Tab 映射 errors/warnings |
| **失败任务中心** | 否（映射校验不写失败中心） |
| **可重试** | 是 |

### 2.12 上传图片到抖店

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 刊登 Tab → **上传图片到抖店** / 单张重试 |
| **预期成功** | 主图/详情图状态 **已上传**，有 `platformImageId` |
| **常见失败** | 外链未同步 Storage；`public_base` 非公网；主图上传失败；SSRF 拦截内网 URL |
| **排查位置** | 刊登 Tab 图片状态列；操作日志 `douyin.image.*` |
| **失败任务中心** | 部分场景（刊登任务关联） |
| **可重试** | 是（单张/全部重试） |

### 2.13 创建抖店商品草稿

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 商品详情的抖店操作跳转 **运营任务中心** → 创建平台草稿任务 → 核对冻结快照 → 人工审核 → 执行（`save_as_platform_draft`，不上架） |
| **预期成功** | 创建阶段只冻结商品/映射/请求且不访问平台；批准 exact version/hash 后执行一次；返回 `platformProductId`，刊登任务 success，运营任务 `draft_written`，`product_publications` 与 SKU 映射事务写入 |
| **常见失败** | L0/kill switch 阻断；白名单或 active gray 不一致；双人审批缺失；未授权；类目/属性/图片/SKU 无效；队列不可用；平台 API 错误或结果未知 |
| **排查位置** | 运营任务详情的草稿/审核/执行/审计 Tab；刊登任务详情；P10 status；失败中心。旧 `.../create-draft` 固定 409 `DOUYIN_OPERATION_TASK_REQUIRED` |
| **失败任务中心** | 是 |
| **可重试** | 仅已知失败且运营任务返回 `retryable=true` 时由运营任务中心人工重试；`result_unknown` 禁止重试/重建，只能人工只读恢复对账 |

**人工恢复验证：** 使用具备 `operationtask.execute` 与店铺操作权限的账号，对下游任务、执行尝试和运营任务三层均为 `result_unknown` 的记录调用刊登任务 `recover-douyin-draft`（或 `douyin/recover` 别名），确认请求只执行 `product.detail`。排队中、执行中、已知失败或普通任务固定 409 `DOUYIN_RECOVERY_NOT_ALLOWED` 且状态不变；找到相同 `outer_product_id` 时两个任务中心收敛成功，未找到时保持不可重试并交人工调查。不得从传统刊登、多目标、批量、单任务重试或批次重试入口重建抖店草稿。

### 2.14 校准 SKU 绑定

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 刊登 Tab → **校准抖店 SKU 绑定** |
| **预期成功** | `product.detail` 拉取平台 SKU；规格一致自动 `bound` |
| **常见失败** | 无 platformProductId；权限不足；多候选 `ambiguous`；无匹配 `unmatched` |
| **排查位置** | 抖店 SKU 绑定管理表；`GET .../douyin/sku-bindings` |
| **失败任务中心** | 否 |
| **可重试** | 是（重新校准） |

### 2.15 手动绑定 ambiguous / unmatched SKU

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 刊登 Tab → SKU 绑定表 → **手动绑定** / **解除绑定** |
| **预期成功** | `bindStatus=bound`，`external_sku_id` 写入；冲突被拦截 |
| **常见失败** | 同一抖店 SKU 绑定到多个本地规格；平台 SKU ID 缺失 |
| **排查位置** | SKU 绑定表；操作日志 `douyin.sku.binding.*` |
| **失败任务中心** | 否 |
| **可重试** | 是 |

### 2.16 同步订单

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 店铺管理 → 抖店店铺 → **同步订单**；或 订单 → 订单同步任务 |
| **预期成功** | 任务 success / partial_success；订单写入；SKU 匹配摘要 |
| **常见失败** | `order_sync_enabled=false`；未授权；token 过期；分页部分失败 |
| **排查位置** | 订单同步任务页；订单列表；失败任务中心 `DOUYIN_ORDER_*` |
| **失败任务中心** | 是 |
| **可重试** | 是 |

### 2.17 本地库存扣减

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 订单同步成功后自动（策略允许时）；订单异常工作台可补扣 |
| **预期成功** | 匹配成功的明细扣减本地库存；重复同步不重复扣 |
| **常见失败** | SKU 未匹配；库存不足；扣减策略关闭 |
| **排查位置** | 订单异常工作台；库存流水；`inventory/effects` |
| **失败任务中心** | 是（扣减失败类） |
| **可重试** | 是（异常工作台 **重试扣库存**） |

### 2.18 同步库存到抖店

| 字段 | 内容 |
| --- | --- |
| **操作入口** | 商品详情 → 库存 Tab → **同步库存到抖店**；或 库存预警页批量同步 |
| **预期成功** | 库存同步任务 success；抖店 `sku.syncStock` 成功 |
| **常见失败** | `inventory_sync_enabled=false`；SKU 未绑定；存在 ambiguous/unmatched；库存无效 |
| **排查位置** | 库存同步任务页；失败任务中心 `DOUYIN_INVENTORY_*` / `DOUYIN_SKU_*` |
| **失败任务中心** | 是 |
| **可重试** | 是 |

---

## 3. 安全检查（验收必过）

| # | 检查项 | 预期 | 验证方式 |
| --- | --- | --- | --- |
| 1 | App Secret 不返回前端明文 | API `GET platform/settings/douyin_shop` 值为 `****` | 浏览器 Network |
| 2 | accessToken 不返回前端明文 | 店铺详情 `auth.accessToken` 脱敏 | 店铺管理 Network |
| 3 | refreshToken 不返回前端明文 | 同上 | 同上 |
| 4 | 日志不打印 token / secret | 后端日志无完整密钥 | 检索 `access_token` / `app_secret` |
| 5 | 订单收货信息脱敏 | 买家姓名/电话/地址部分掩码 | 订单详情 UI |
| 6 | raw error 脱敏 | `SanitizeErrorText` 掩码 token | 失败任务详情 |
| 7 | 图片下载禁止内网地址 | 内网 URL 报 `private network` | 抖店图片上传前校验 |
| 8 | 前端不直连抖店 API | 无抖店域名请求 | 浏览器 Network |
| 9 | 抖店调用走后端 Client | 全部经 `/api/v1/*` | 代码审查 `douyinshop.Client` |
| 10 | 抖店草稿写只有运营任务入口 | 旧直接创建固定 409；传统/多目标/批量/重试入口零写 | Browser Network + DB 前后计数 |
| 11 | 冻结与审批绑定 | Worker 平台访问前校验 exact task/draft/approval/attempt/downstream/hash | 审计时间线 + 篡改负例 |
| 12 | 未知结果不自动重建 | `result_unknown` 为不可重试，仅人工只读对账 | 刊登任务、运营任务与平台草稿箱 |
| 13 | 运行时控制优先 | provider/tenant/shop/write 任一 kill switch 阻断平台调用 | 逐级 kill switch 演练 |

---

## 4. 风险清单（MVP Demo Release）

| 级别 | 风险 | 影响 | 缓解 |
| --- | --- | --- | --- |
| P0 | 无真实抖店凭证未完成 E2E | 无法证明线上字段对齐 | 使用真实 App + 测试店跑本清单 |
| P0 | `public_base` 非公网 | 图片上传/抖店拉取失败 | 生产配置 HTTPS 公网前缀 |
| P0 | `product.addV2` / `product.detail` 字段与线上一致性 | 创建草稿或 SKU 绑定失败 | 真实环境校准 payload，记录 requestId |
| P0 | 平台成功与本地提交/回写之间进程中断 | 两个任务中心状态暂时分裂 | 事务 outbox + 任务租约/reaper + 人工只读对账，不自动重建 |
| P1 | `order.searchList` 分页/时间字段差异 | 漏单或 partial_success | 对照官方文档与 `pageErrors` |
| P1 | `sku.syncStock` 参数差异 | 库存同步失败 | 失败任务中心重试 + 日志 |
| P1 | SKU 自动匹配 ambiguous 率高 | 需人工绑定才能同步库存 | 刊登 Tab 手动绑定流程 |
| P2 | 定时订单/库存轮询默认关闭 | 需手动触发同步 | 文档说明，不默认开启 |
| — | 直接上架 | 绕过审核风险 | **MVP 不做**，仅平台草稿 |
| — | 售后/财务/多仓 | 范围蔓延 | 明确不在本 Release |

---

## 5. 人工签收

**签收前检查：**

- [ ] 本清单 2.1–2.18 全部通过或已知问题记入 PROGRESS 遗留
- [ ] GitHub Actions 中 backend、contract、Admin build 与 Admin E2E 等受影响工作流通过
- [ ] PostgreSQL migration/唯一约束与 Redis queue/outbox/reaper 回归在隔离 CI service container 通过
- [ ] 生产预检、只读检查和受控写链路均由维护者在授权环境人工执行
- [ ] 备份、隔离恢复、应用回滚、四级写 kill switch 与灰度暂停/停止演练已在发布工单记录
- [ ] 两名不同管理员分别承担 Owner/Technical Lead 审批职责，且单租户/单店/最多 100 SKU 范围与工单一致
- [ ] 默认 L0 配置未被提交；目标环境 L3 只允许平台草稿写，自动重试和库存 mutation 保持关闭
- [ ] 脱敏结论已记录在 PR 或发布工单，未向仓库提交 JSON、Markdown 报告、截图或日志
- [ ] `git diff --check` 无冲突标记
- [ ] `PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md` 中相关流程已人工签收
- [ ] `docs/PROGRESS.md` 已更新阶段状态

在 CI、真实凭据、备份恢复、灰度、回滚、人工验收和发布工单全部签收前，结论只能是“代码具备受控生产执行候选能力”，不得记录为“已上线”。

---

## 6. 相关文档

- [`PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md`](PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md) — 当前人工验收清单
- [`DOUYIN_E2E_PRECHECK_GUIDE.md`](DOUYIN_E2E_PRECHECK_GUIDE.md) — 人工验收前置检查
- [`DOUYIN_PRODUCTION_RUNBOOK.md`](DOUYIN_PRODUCTION_RUNBOOK.md) — 生产操作与灰度观察
- [`docs/PROGRESS.md`](PROGRESS.md) — 阶段进度与遗留
- [`docs/api.md`](api.md) — API 契约（含抖店可观测性）
- [`docs/provider.md`](provider.md) — Platform Provider 说明
