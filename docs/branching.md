# 分支管理与 PR 规则

## 分支角色

| 分支 | 用途 |
| --- | --- |
| `main` | 稳定正式分支，只通过 PR 合并 |
| `dev` | 日常开发集成分支 |
| `feat/*` | 从 `dev` 创建的功能分支，PR 到 `dev` |
| `fix/*` | 常规修复 PR 到 `dev`；紧急线上修复可 PR 到 `main` 并回合 `dev` |
| `release/*` | 从 `dev` 创建的发布准备分支，PR 到 `main` |

未经明确授权，不直接 push、commit、打 Tag 或发布 Release。

## 推荐流程

```bash
git switch dev
git pull --ff-only origin dev
git switch -c feat/your-feature-name
```

PR 必须说明变更范围、风险、本地实际检查、交由 CI 的自动化回归和人工验收结果。

## 合并要求

- `main` 不接受直接 push；`dev` 也优先通过 PR。
- GitHub Actions 是自动化门禁的权威执行入口，失败不得通过 skip、降低断言或扩大 baseline 掩盖。
- 功能和业务流程由维护者人工签收。
- Go 改动执行 `gofmt`；前端/Collector 改动至少完成相关静态检查或构建，完整回归可交由 CI。
- 环境变量、Docker、Provider、API、部署或 CI 变更必须同步相关文档。
- 不提交 `.env`、真实凭据、生产数据或本地测试产物。

## CI 隔离资源

PostgreSQL 和 Redis 集成测试由 GitHub Actions service container 提供。CI 中的 `trademind_test` 是作业内临时资源，本地无需创建或维护同名数据库。任何本地集成测试都必须显式指向隔离测试资源，不得回退到开发或生产服务。

## 发布流程

从 `dev` 创建 `release/*`，完成 changelog、版本号、部署文档、CI 和人工验收后 PR 到 `main`。发布修正应回合 `dev`，避免分支漂移。

## 分支保护建议

- `main` 和 `dev` 禁止直接 push并要求 PR 与 CI。
- 按维护者偏好使用线性历史或 squash merge。
- 合并后删除无用远程分支。
