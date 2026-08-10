# 抖店人工验收前置检查指南

> **抖店 Release Candidate** · **Douyin Production Adapter Implemented** · 真实平台验收必须在授权环境人工执行 · 非 Production Ready

## 边界说明

- 当前适配器代码覆盖 OAuth、Token、类目、图片、平台草稿、订单、库存、Webhook 签名与事件路由
- 仓库脚本和 CI **不执行**真实抖店写链路、不自动写不可逆接口、不直接上架
- 凭证缺失状态：`blocked_by_real_credentials` / `environment_required`（**不是**系统 P0/P1 失败）
- 创建本地草稿 ≠ 平台真实草稿；创建抖店草稿 ≠ 商品上架
- **代码实现完成 ≠ 真实 E2E 已通过**

## 前置条件清单

| 检查项 | 通过标准 | 缺失时状态 |
| --- | --- | --- |
| App Key / App Secret | 已配置（脱敏展示） | 待真实凭证 |
| OAuth 授权 | ≥1 已授权店铺 | 未授权 |
| Token 有效性 | 结构检查 / 可选 liveTest | 需人工确认 |
| 类目 / 属性同步 | 有缓存数据 | 当前跳过 |
| Storage `public_base` | 公网 HTTPS 可访问 | 待公网 Storage |
| 抖店图片上传 | 前置检查通过 | 待公网 Storage |
| Release Candidate 标识 | 保留 | — |

## 页面入口

| 页面 | 用途 |
| --- | --- |
| `/settings/platforms?platform=douyin_shop` | 抖店接入 + 生产预检面板 |
| `/settings/config-status` | 凭证 / Storage / 发布能力总览 |
| `/product/drafts` | 刊登前商品准备 |
| `/product/publish-batches` | 批量刊登边界提示 |
| `/ops/task-center/failures?platform=douyin_shop` | 抖店相关失败 |

## 生产维护验收方式

- 自动化回归由 GitHub Actions 在隔离环境执行。
- 维护者按 [`DOUYIN_E2E_CHECKLIST.md`](DOUYIN_E2E_CHECKLIST.md) 完成页面、只读接口和受控业务流程检查。
- 真实平台写操作必须先取得外部生产审批；无凭证或环境时记录 `blocked_by_real_credentials` / `blocked_by_environment`。
- 结论记录在 PR 或发布工单，不提交一次性 JSON、Markdown 报告、截图或日志。

## 凭证缺失文案（统一）

> 当前未配置抖店真实凭证，系统只能完成本地 Demo 与前置检查，不能执行真实抖店 E2E。

按钮：去配置平台凭证 · 查看 E2E 准备清单 · 查看失败任务

## 相关文档

- [`DOUYIN_E2E_CHECKLIST.md`](DOUYIN_E2E_CHECKLIST.md)
- [`P10_MANUAL_ACCEPTANCE_CHECKLIST.md`](P10_MANUAL_ACCEPTANCE_CHECKLIST.md)
- [`STORAGE_PUBLIC_URL_GUIDE.md`](STORAGE_PUBLIC_URL_GUIDE.md)
