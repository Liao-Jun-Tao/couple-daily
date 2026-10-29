# couple-daily 服务端技术栈推荐

客户端已定 **Flutter**，目录留给 `apps/mobile/`。服务端栈已确认，脚手架在 `backend/`。

> Auth（已确认）：邮箱 + 密码 + JWT。短信 OTP 为 P1，首版不做。

## 推荐方案（默认）

| 层 | 选型 | 理由 |
|----|------|------|
| 语言/框架 | **Go** + [chi](https://github.com/go-chi/chi)（或 Gin） | 部署简单、并发稳、OpenAPI 易生成/校验；与「双人实时感不强、CRUD+推送」匹配 |
| API 契约 | 本仓 `docs/api/openapi.yaml` 为源；可用 oapi-codegen | 前后端以契约为准 |
| 数据库 | **PostgreSQL 16** | UUID、软删、JSON、cursor 分页索引友好 |
| 迁移 | golang-migrate 或 goose | 可复现 |
| 缓存/队列 | **Redis** | refresh 黑名单可选；纪念日提醒延迟任务/分布式锁 |
| 对象存储 | **S3 兼容**（优先 Cloudflare R2 或阿里云 OSS；本地 MinIO） | 预签名直传多图；成本可控 |
| Auth | **邮箱密码** + JWT access（短）+ refresh（长）；密码用 argon2id/bcrypt | 已确认 MVP；**短信 OTP 标 P1 不做** |
| 推送 | **APNs**（iOS）+ **FCM**（Android） | 工程推荐；未开通知时 API 仍 200，客户端 in-app 提示 |
| 可观测 | structured JSON log + OpenTelemetry（可选） | 前期 stdout 即可 |
| 部署 | 单二进制 + Docker；反向代理 Nginx/Caddy | MVP 一台云主机或 Fly/Render |

### 仓库目录

Go 模块路径是 `github.com/Liao-Jun-Tao/couple-daily`，源码放在 `backend/`，避免和 Flutter 抢仓库根目录。

```
docs/                         # 契约
backend/cmd/api/              # chi，GET /healthz，/v1 契约路由
backend/internal/             # auth, pairing, posts, media, checkins, anniversaries, push, config
backend/migrations/
backend/docker-compose.yml    # postgres + redis + minio
apps/mobile/                  # Flutter，由前端 PR 创建
```

## 备选方案

| 方案 | 适用 | 代价 |
|------|------|------|
| **NestJS (TypeScript) + Prisma + PG** | 团队更熟 Node/TS、希望与脚本同语言 | 内存与冷启动略重；同样能对接 OpenAPI |
| **Dart Shelf / Dart Frog** | 想与 Flutter 同语言 | 生态与运维资料少于 Go/Node，不建议作首版默认 |
| **Supabase/Firebase 直连** | 极速原型 | 双人邀请/软删/细权难控，后期易绑死；**不推荐**作正式后端 |

## 不推荐（本项目）

- 无服务端的纯本地双机同步（产品要邀请配对与推送）
- 自建 Kafka 等重中间件（MVP 过重）

## 已确认

1. **服务端栈**：Go + chi + PostgreSQL + Redis + S3 兼容 + APNs/FCM。脚手架在 `backend/`。
2. **Auth**：邮箱密码 + JWT。短信 OTP 为 P1，首版不做，契约没有发码路径。
