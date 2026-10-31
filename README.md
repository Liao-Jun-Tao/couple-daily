# couple-daily

情侣日常 monorepo：Flutter 双端客户端 + Go API。一个 CoupleSpace 最多两人，用邀请码配对。

## 目录

| 路径 | 内容 |
|------|------|
| `backend/` | Go API（chi）、PostgreSQL 迁移、本地 postgres / redis / minio |
| `docs/` | 已锁定的接口契约、数据模型、自测与技术栈 |
| `apps/mobile/` | Flutter 客户端。留给前端脚手架，本仓库的后端变更不创建该目录 |

模块路径是 `github.com/Liao-Jun-Tao/couple-daily`。Go 源码在 `backend/`，不在仓库根目录。

## 已锁定

- **Auth：** 邮箱 + 密码注册登录。手机号验证码不上首版。
- **邀请码：** 6 位 `[A-Z0-9]`，TTL 24 小时，一码一用。发起邀请时创建 space，`member_count=1`；接受时加入第二人；满员 2。
- **删除：** 仅作者可删帖、评论、打卡、纪念日、收藏。赞只由本人取消。
- **发帖：** 仅图或仅文均可。正文去掉空白后为空，且没有图片，则拒绝。多图最多 9 张，单张不超过 10MB（jpeg / png / heic / webp）。
- **定位：** 可选，默认 `exact`。
- **打卡：** 店名必填，可以手填；`place_id` 可以为空。
- **纪念日：** 默认当天本地 `09:00`（`Asia/Shanghai`）提醒。
- **推送：** iOS 走 APNs，Android 走 FCM。API 只登记 token。

## 本地启动 API

需要 Go 1.22+ 和 Docker。

```bash
cd backend
cp .env.example .env
docker compose up -d
go run ./cmd/api
```

进程启动时会执行 `migrations/*.up.sql`。如果这份初始表已经用 psql 建过，启动只记下版本，不会重复执行。

另一个终端：

```bash
curl -sS http://localhost:8080/healthz
```

期望返回 `{"status":"ok"}`。

业务路由在 `/v1`，路径与 `docs/api/openapi.yaml` 一致。Redis 或 MinIO 没起来时，注册、登录和邀请码仍然可用。没有 APNs / FCM 凭证时，`POST /devices` 照常登记 iOS 和 Android token，发送走 stub。

```bash
cd backend
go build ./...
go test ./...
```

依赖端口：PostgreSQL `localhost:5432`，Redis `localhost:6379`，MinIO API `localhost:9000`（控制台 `localhost:9001`）。compose 会创建 bucket `couple-daily`。

## 契约

- `docs/api/openapi.yaml`
- `docs/data-model.md`
- `docs/self-test.md`
- `docs/tech-stack-recommendation.md`
- `docs/CHANGELOG-draft.md`

业务验收步骤见 `docs/self-test.md`。
