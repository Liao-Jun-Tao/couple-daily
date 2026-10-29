# couple-daily 服务端技术栈

客户端已定 **Flutter**，目录预留给 `apps/mobile/`。服务端栈已确认，脚手架在 `backend/`。

**Auth 已锁定为邮箱 + 密码（argon2id）。手机号验证码不上首版，不接短信通道。**

## 已确认方案

| 层 | 选型 | 理由 |
|----|------|------|
| 语言 / 框架 | **Go** + [chi](https://github.com/go-chi/chi) | 单二进制、并发稳，和 CRUD + 推送匹配 |
| 模块路径 | `github.com/Liao-Jun-Tao/couple-daily` | 源码放在 `backend/`，避免和 Flutter 抢仓库根目录 |
| API 契约 | `docs/api/openapi.yaml` | 前后端以契约为准；后续可用 oapi-codegen |
| 数据库 | **PostgreSQL 16** | UUID、软删、cursor 分页索引 |
| 迁移 | `backend/migrations/`（golang-migrate 文件名） | `000001_init.up.sql` 含 `users.email` + `password_hash` |
| 缓存 / 队列 | **Redis** | refresh 会话与纪念日提醒锁的预留 |
| 对象存储 | **S3 兼容**（本地 MinIO；生产可用 R2 或 OSS） | 预签名直传，单张 ≤10MB |
| Auth | JWT access（短）+ refresh（长）；邮箱密码，argon2id | 首版无短信 OTP |
| 推送 | **APNs**（iOS）+ **FCM**（Android） | API 只登记 token |
| 可观测 | 当前 stdout 请求日志；OpenTelemetry 以后再加 | |
| 部署 | 单二进制 + Docker；反向代理 Nginx / Caddy | MVP 一台云主机即可 |

### 仓库目录

```
docs/api/openapi.yaml
docs/data-model.md
backend/cmd/api/          # chi，GET /healthz，/v1 契约路由
backend/internal/         # auth, pairing, posts, media, checkins, anniversaries, push, config
backend/migrations/
backend/docker-compose.yml  # postgres + redis + minio
apps/mobile/              # Flutter，由前端 PR 创建
```

## 未采用的备选

| 方案 | 原因 |
|------|------|
| NestJS + Prisma + PostgreSQL | 能做，但本项目已选 Go |
| Dart Shelf / Dart Frog | 与 Flutter 同语言，生态与运维资料更少 |
| Supabase / Firebase 直连 | 双人邀请、软删和作者删除权后期难收 |

不自建 Kafka。不做无服务端的双机同步。

## 本地依赖

`backend/docker-compose.yml` 启动 PostgreSQL 16、Redis 7、MinIO。对象存储 bucket 名为 `couple-daily`。
