# apps/mobile

情侣日常 Flutter 客户端。默认请求本机 Go API（`/v1`）。`USE_MOCK=true` 时仍使用进程内 Mock，不访问网络。

## 配置

| dart-define | 默认 | 含义 |
|-------------|------|------|
| `USE_MOCK` | `false` | `true` 走 `MockBackend`，不发请求 |
| `API_BASE_URL` | `http://10.0.2.2:8080/v1` | 必须带 `/v1`。这是 Android 模拟器访问宿主机的地址 |

Token 存在系统安全存储（Android 加密偏好 / iOS Keychain）。access 过期会用 refresh 换一对新 token。refresh 失效会清掉登录态，回到登录页。

### 模拟器和真机

| 运行位置 | API 基址 |
|----------|----------|
| Android 模拟器 | `http://10.0.2.2:8080/v1`（默认）。`10.0.2.2` 是模拟器里的宿主机 |
| iOS 模拟器 | `http://127.0.0.1:8080/v1`。模拟器与电脑共享网络 |
| 真机 | `http://<电脑局域网 IP>:8080/v1`。手机和电脑在同一网段，API 监听 `0.0.0.0` 或该网卡 |
| `adb reverse` 之后 | `http://127.0.0.1:8080/v1`。模拟器的 `127.0.0.1:8080` 被转到电脑 |

带图发帖会先 `POST /media/uploads`，再把字节 PUT 到返回的预签名 URL。本地 MinIO 的 URL 主机是 `localhost:9000`，签名覆盖了主机名，客户端不能改成 `10.0.2.2`。只发文字不经过 MinIO。要在 Android 模拟器上传图片：

```bash
adb reverse tcp:8080 tcp:8080
adb reverse tcp:9000 tcp:9000
cd apps/mobile
flutter run --dart-define=API_BASE_URL=http://127.0.0.1:8080/v1
```

明文 HTTP 已在 Android `usesCleartextTraffic` 和 iOS ATS 里放开，方便打本机 API。

## 先起 API

见仓库根 README 和 `docs/self-test.md`。摘要：

```bash
cd backend
cp .env.example .env
docker compose up -d
set -a && . ./.env && set +a
go run ./cmd/api
```

另一个终端确认 `curl -sS http://localhost:8080/healthz` 返回 `{"status":"ok"}`。

## 真 API 自测（USE_MOCK=false）

```bash
cd apps/mobile
flutter pub get
flutter devices
flutter run -d <emulator-id> --dart-define=API_BASE_URL=http://10.0.2.2:8080/v1
```

不写 `USE_MOCK` 就是 false。主路径：

1. 注册邮箱 A（密码至少 8 位）并登录。没有手机号，没有短信验证码。
2. 生成 6 位邀请码。空间人数应为 1。复制邀请码。
3. 退出，注册邮箱 B，填入邀请码并接受。短暂看到「配对成功」，进入空时间线。
4. 写一条只有文字的动态并发布。时间线出现这条。作者自己能看到删除；对方的动态没有删除按钮。
5. 杀掉应用再打开。应仍是登录态并回到时间线（token 在安全存储里）。
6. 可选：餐厅里搜一个不存在的店名，用手填店名打卡。纪念日保存后提醒时刻是当天 `09:00` `Asia/Shanghai`。我的页请求通知权限；拒绝不会退出。同意后登记的是 stub token，不是真实 APNs/FCM。

错误码按服务端原样展示。接受邀请时，空间已满返回 `SPACE_FULL`（「这个空间已经有两个人」），即使这枚码也已经用过。码已作废但空间仍只有 1 人时才是 `INVITE_USED`。

Mock 自测：

```bash
flutter run --dart-define=USE_MOCK=true
flutter test
```

`flutter analyze` 与 `flutter test` 不需要 API。Mock 单测覆盖邀请、双空禁发和作者删除。真 API 客户端用假 HTTP 覆盖登录、邀请、发帖、打卡、纪念日、设备登记，以及 refresh 失败后退出登录。

## 结构

```
lib/
  core/            会话、路由、Mock、HTTP 客户端、安全存储、通知 stub
  features/
    auth_pair/     Splash、邮箱密码、邀请、配对成功
    timeline/      时间线、帖子卡片、评论
    compose/       发帖、位置选择
    restaurant/    搜索、手填打卡、收藏路由占位
    anniversary/   列表与表单
    me/            我的、通知权限
  shared/widgets/
```

底部导航：时间线 / 餐厅 / 纪念日 / 我的。路由仍是 Splash → Auth → 邀请 → 主壳。

## 行为

| 项 | 行为 |
|----|------|
| 登录 | 邮箱 + 密码。密码至少 8 位。 |
| 邀请码 | 6 位 `[A-Z0-9]`，24 小时，一码一用。发起时服务端建 space，`member_count = 1`。 |
| 邀请码再次获取 | 真 API 只在创建时返回码。客户端记在安全存储里。清掉存储后又还在这个未满空间时，不能再要一个新码（`ALREADY_PAIRED`）。 |
| 配对成功 | 接受成功后短暂进入 PairingDone，然后进入时间线。 |
| 发帖 | 正文去掉空白后为空且没有图片，则拒绝。最多 9 张，单张不超过 10MB，jpeg / png / heic / webp。 |
| 位置 | 可选。选择器默认完整（`exact`）。 |
| 删除 | 动态、评论、打卡、纪念日的删除入口只出现在作者自己的内容上。 |
| 餐厅 | 本地名单搜索。没有结果时手填店名，`place_id` 可空。收藏仍是 `/favorites` 占位（P1）。 |
| 纪念日 | 默认当天 `09:00`、`Asia/Shanghai`、不提前。 |
| 通知 | 我的页请求系统权限。拒绝或插件不可用时给出提示，不抛错。登记的是 stub token。 |

## 已知限制

- 没有短信 OTP。
- 推送只登记 token。没有 APNs / FCM 凭证时服务端发送走 stub，客户端也不会收系统推送。
- 收藏是 P1 占位，不挡主路径。
- 带图发帖依赖 MinIO 预签名 URL 能从设备访问（见上面的 `adb reverse`）。只发文字不依赖 MinIO。
- 服务端在「已经在空间里」时不会重发邀请码。过期后如果仍是 1 人，客户端不能补发新码。
- 设备列表没有 GET。我的页只显示这次运行里登记成功的 token。
