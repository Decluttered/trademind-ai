<h1 align="center">贸灵 TradeMind</h1>

<p align="center">
  <strong>开源 AI 跨境电商增长与运营平台</strong>
</p>

<p align="center">
  从一条商品链接，到可运营、可刊登、可持续协同的商品资产
</p>

<p align="center">
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/badge/license-Apache--2.0-blue.svg"></a>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React-18+-61DAFB?logo=react&logoColor=111">
  <img alt="TypeScript" src="https://img.shields.io/badge/TypeScript-5+-3178C6?logo=typescript&logoColor=white">
  <img alt="Docker" src="https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white">
  <img alt="Self-hosted" src="https://img.shields.io/badge/Self--hosted-supported-2EA043">
</p>

<p align="center">
  简体中文 | <a href="README.en.md">English</a>
</p>

<p align="center">
  <a href="#快速开始">快速开始</a> ·
  <a href="#界面预览">界面预览</a> ·
  <a href="#核心能力">核心能力</a> ·
  <a href="#架构与技术栈">架构与技术栈</a> ·
  <a href="docs/README.md">文档中心</a>
</p>

<p align="center">
  <img src="docs/assets/img/readme-hero-zh.png" alt="TradeMind 产品预览" width="100%" />
</p>

TradeMind 在本仓库中运行 **Amazon.de → eBay.de** 自动运营：采集与评估、AI 内容、eBay 刊登、监控改价、售出后 Amazon 零售履约与利润台账。数据留在自托管环境，能力按 Provider 扩展。

无论是快速搭建自己的跨境运营工作台，还是为现有业务接入 AI 与平台能力，TradeMind 都提供一套可私有化部署、可审计、可二次开发的开放底座：数据掌握在自己手中，能力可以按团队流程自由组合。

## 为什么选择 TradeMind

| 方向 | TradeMind 的做法 |
| --- | --- |
| Amazon → eBay | Amazon.de 采集与快照、Listing Studio、eBay Sell API 刊登、监控改价与利润台账。 |
| AI 商品运营 | 标题、描述、图片处理与发布前检查（含 GPSR）在同一工作台完成。 |
| 开放与可控 | Provider 架构覆盖 AI、存储、图片、eBay 与 Amazon 采集；权限、审计与幂等守住写路径。 |

## 界面预览

下面的界面预览展示 TradeMind 的核心工作流：**商品采集 → 商品草稿 → AI 内容优化**。从发现商品开始，到准备好可发布内容，每一步都清晰可追踪。

<table>
  <tr>
    <td width="50%" align="center">
      <img src="docs/assets/img/2.png" alt="采集中心" width="100%" />
      <br />
      <sub><strong>采集中心</strong>：采集器入口与批量采集</sub>
    </td>
    <td width="50%" align="center">
      <img src="docs/assets/img/3.png" alt="采集任务" width="100%" />
      <br />
      <sub><strong>采集任务</strong>：链接提交、状态追踪与草稿关联</sub>
    </td>
  </tr>
  <tr>
    <td width="50%" align="center">
      <img src="docs/assets/img/4.png" alt="采集监控" width="100%" />
      <br />
      <sub><strong>采集监控</strong>：Worker、任务与批次状态分布</sub>
    </td>
    <td width="50%" align="center">
      <img src="docs/assets/img/1.png" alt="AI 描述生成" width="100%" />
      <br />
      <sub><strong>AI 描述生成</strong>：生成卖点、规格与描述并应用到草稿</sub>
    </td>
  </tr>
</table>

## 核心能力

### Amazon.de → eBay.de

- 采集：Amazon.de 商品页 / ASIN 快照（Playwright；官方读接口在有凭证时使用）。
- Listing Studio：EUR cents、GPSR、AI 内容版本与刊登就绪检查。
- 刊登：eBay OAuth、类目特征缓存、Temporal 编排的 Inventory/Offer 发布（默认沙箱，生产写入需显式环境）。
- 监控：改价决策、Offer 校验、利润台账。
- AI：标题优化、描述生成、Prompt 模板、图片任务。
- 店铺：eBay 授权、敏感配置加密与连接测试。

### 工程化与扩展

- Provider 架构：AI、存储、图片、平台、采集能力均通过 Provider 抽象扩展。
- 自部署友好：默认 PostgreSQL + Redis，支持本地开发和 Docker Compose 完整部署。
- Monorepo 协作：backend、admin、collector 与文档规则统一维护，适合团队协作与持续演进。
- 可靠性地基：关键写路径统一幂等，AI 结果应用/撤销保护，Webhook 快速 ACK，异步 Worker 租约防止陈旧写回。

## 架构与技术栈

| 层级 | 技术栈 |
| --- | --- |
| Backend | Go + Gin + GORM |
| Admin | React + TypeScript + Ant Design Pro |
| Collector | Node.js + TypeScript + Playwright |
| Data | PostgreSQL + Redis |
| Deploy | pnpm workspace + Docker Compose |
| Extension Points | AI / Storage / Image / Platform / Collector Providers |

## 快速开始

### 本地开发

```bash
pnpm install
pnpm install:collector:browsers
pnpm dev
```

常用命令：

```bash
pnpm check:dev
pnpm dev:infra
pnpm dev:backend
pnpm dev:admin
pnpm dev:collector
pnpm build:admin
pnpm build:collector
pnpm seed:demo-data
pnpm seed:demo-permissions
```

开发环境默认连接隔离的 PostgreSQL 与 Redis；完整自动化回归由 GitHub Actions 持续执行。

### Docker 部署

```bash
cp .env.example .env
# 在 .env 中设置独立随机的 COLLECTOR_SERVICE_TOKEN（至少 32 字符）
docker compose -f docker-compose.full.yml up -d --build
```

Windows PowerShell：

```powershell
Copy-Item .env.example .env
# 在 .env 中设置独立随机的 COLLECTOR_SERVICE_TOKEN（至少 32 字符）
docker compose -f docker-compose.full.yml up -d --build
```

GitHub Actions 会为 backend、admin 和 collector 自动发布 GHCR 多架构镜像。使用预构建镜像：

```bash
# 在 .env 中设置 COLLECTOR_SERVICE_TOKEN，并覆盖以下镜像引用
TRADEMIND_BACKEND_IMAGE=ghcr.io/lien0219/trademind-backend:dev-v0.2.0
TRADEMIND_ADMIN_IMAGE=ghcr.io/lien0219/trademind-admin:dev-v0.2.0
TRADEMIND_COLLECTOR_IMAGE=ghcr.io/lien0219/trademind-collector:dev-v0.2.0
docker compose -f docker-compose.full.yml pull backend admin collector
docker compose -f docker-compose.full.yml up -d --no-build
```

分支构建会更新分支、分支版本和 `sha-<commit>` 标签，但不会更新 `latest`。正式版本合并到 `main` 后，推送与 `deploy/IMAGE_VERSION` 一致的 `v<version>` Git Tag，工作流才会发布 `v<version>`、`version` 和 `latest`。正式部署应使用工作流输出的 `image@sha256:<manifest-digest>` 不可变引用。完整发布步骤与包地址见 [Docker 部署](docs/docker-deployment.md)。

默认访问地址：

| 服务 | 地址 |
| --- | --- |
| Admin | <http://127.0.0.1:8000> |
| Backend Health | <http://127.0.0.1:8080/health> |

完整 Compose 中 Collector 仅供 backend 通过内部网络访问，不发布宿主机端口；PostgreSQL 与 Redis 仅绑定宿主机回环地址。eBay 写入走 Temporal `publication` 路径；沙箱与生产凭据分离，默认 `EBAY_ENV=sandbox`。

Admin 根路径展示公开产品首页，访客可从首页进入登录页或注册页；登录后再进入运营工作台。

更多说明：

- [本地开发](docs/development.md)
- [Docker 部署](docs/docker-deployment.md)
- [环境变量](docs/env.md)

## 文档导航

- [docs/README.md](docs/README.md)：完整文档入口。
- [docs/development.md](docs/development.md)：本地开发、调试与常用命令。
- [docs/docker-deployment.md](docs/docker-deployment.md)：Docker Compose 完整部署与运维说明。
- [docs/api.md](docs/api.md)：API 契约、统一返回与鉴权说明。
- [docs/provider.md](docs/provider.md)：Provider 扩展机制与安全约束。
- [docs/architecture.md](docs/architecture.md)：系统架构、分层与数据流说明。
- [docs/branching.md](docs/branching.md)：分支策略与 PR 规则。

## 贡献与社区

- 贡献代码或文档前，请先阅读 [CONTRIBUTING.md](CONTRIBUTING.md)。
- 安全问题请参考 [SECURITY.md](SECURITY.md)。
- 如果你愿意补充更好的截图、示例数据或文档，也非常欢迎提交 PR。
- 赞助方式见 [docs/sponsor.md](docs/sponsor.md)。

## License

本项目基于 [Apache License 2.0](LICENSE) 开源。
