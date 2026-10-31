# couple-daily

情侣日常 monorepo：Flutter 双端客户端 + Go API。一个 CoupleSpace 最多两人，用邀请码配对。

## 目录

| 路径 | 内容 |
|------|------|
| `backend/` | Go API（chi）、PostgreSQL 迁移、本地 postgres / redis / minio |
| `docs/` | 已锁定的接口契约、数据模型、自测与技术栈 |
| `apps/mobile/` | Flutter 客户端。默认接 `/v1`；`USE_MOCK=true` 仍走进程内 Mock |

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
set -a && . ./.env && set +a
go run ./cmd/api
```

进程启动时会执行 `migrations/*.up.sql`。如果这份初始表已经用 psql 建过，启动只记下版本，不会重复执行。

`.env.example` 里 `APP_ENV=local`，因此可以用示例 `JWT_SECRET=change-me` 和 MinIO 账号在本机启动。`APP_ENV` 不是 `local` / `dev` / `development`（含未设置）时，进程会直接退出：`JWT_SECRET` 不能为空、不能是 `change-me`、长度至少 32；对象存储一旦配置了 endpoint 或 bucket，必须带上访问密钥，且不能用示例的 `minio` / `minio12345`。

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

## 客户端

`apps/mobile` 默认请求真 API（`USE_MOCK` 默认 false）。先按上文把 API 跑在本机 `http://localhost:8080`，验收步骤见 `docs/self-test.md`。

Android 模拟器访问宿主机用 `10.0.2.2`，这是默认基址 `http://10.0.2.2:8080/v1`。iOS 模拟器、以及 `adb reverse` 之后的 Android，用 `http://127.0.0.1:8080/v1`。真机用电脑的局域网地址。带图发帖时，预签名 URL 的主机是 MinIO 的 `localhost:9000`，签名里包含主机名，客户端不会改写；Android 模拟器需要 `adb reverse tcp:9000 tcp:9000`（API 同样 reverse 时把基址改成 `127.0.0.1`）。

```bash
cd apps/mobile
flutter pub get
flutter run --dart-define=API_BASE_URL=http://10.0.2.2:8080/v1
```

离线 Mock：`flutter run --dart-define=USE_MOCK=true`。细节、自测点击路径和已知限制见 `apps/mobile/README.md`。
