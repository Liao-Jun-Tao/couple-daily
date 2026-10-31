# couple-daily 自测说明（P0 + P1 字段）

> 基址默认 `http://localhost:8080/v1`。将 `$BASE`、`$TOKEN_A`、`$TOKEN_B`、`$TOKEN_C` 替换为实际值。  
> 验收对齐：配对 → 发帖 → 赞评 → 打卡 → 纪念日（+ P1 收藏/定位粒度）。
> **Auth（已确认）**：邮箱+密码注册/登录；邀请码配对。手机号验证码不上首版（P1）。项 2–5 已锁。

```bash
BASE=http://localhost:8080/v1
```

---

## 0. 准备：邮箱注册拿到三人 token

> Auth 已确认：邮箱 + 密码。P1 备注：手机号验证码不上首版。

```bash
curl -sS -X POST "$BASE/auth/register" -H 'Content-Type: application/json' \
  -d '{"email":"a@example.com","password":"password123","display_name":"Alice"}' | tee /tmp/reg_a.json
TOKEN_A=$(jq -r .access_token /tmp/reg_a.json)

curl -sS -X POST "$BASE/auth/register" -H 'Content-Type: application/json' \
  -d '{"email":"b@example.com","password":"password123","display_name":"Bob"}' | tee /tmp/reg_b.json
TOKEN_B=$(jq -r .access_token /tmp/reg_b.json)

curl -sS -X POST "$BASE/auth/register" -H 'Content-Type: application/json' \
  -d '{"email":"c@example.com","password":"password123","display_name":"Carol"}' | tee /tmp/reg_c.json
TOKEN_C=$(jq -r .access_token /tmp/reg_c.json)
```

期望：三人各 `201` + `access_token` / `refresh_token`。

登录回归：

```bash
curl -sS -X POST "$BASE/auth/login" -H 'Content-Type: application/json' \
  -d '{"email":"a@example.com","password":"password123"}'
```

期望：`200` + token；错误密码 → `401`。

---

## 1. 配对：邀请 → 接受 → 第三人拒绝 → 重登回同一 space

### 1.1 A 发起邀请

```bash
curl -sS -X POST "$BASE/pairing/invites" -H "Authorization: Bearer $TOKEN_A" | tee /tmp/invite.json
CODE=$(jq -r .invite_code /tmp/invite.json)
SPACE_ID=$(jq -r .space.id /tmp/invite.json)
```

期望：`invite_code` 为 6 位 `[A-Z0-9]`；`expires_at` ≈ now+24h；`space.member_count=1`。

### 1.2 B 接受

```bash
curl -sS -X POST "$BASE/pairing/invites/accept" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d "{\"invite_code\":\"$CODE\"}" | tee /tmp/space_ab.json
```

期望：`member_count=2`，`members` 含 A、B；`id` 等于 `$SPACE_ID`。

### 1.3 C 再接受同一码 / 任意新码进该 space

```bash
# 一码一用：同一码应失败（已 accepted）
curl -sS -o /tmp/c_reuse.json -w '%{http_code}\n' -X POST "$BASE/pairing/invites/accept" \
  -H "Authorization: Bearer $TOKEN_C" -H 'Content-Type: application/json' \
  -d "{\"invite_code\":\"$CODE\"}"
cat /tmp/c_reuse.json
```

期望：`409`，`code` 为 `SPACE_FULL`（OpenAPI：满员后再接受）。码已作废但 space 仍只有 1 人时为 `INVITE_USED`；过期码为 `INVITE_EXPIRED`。

若 A 在已满 space 再发邀请（产品：已配对不可再邀）：

```bash
curl -sS -o /tmp/a_reinvite.json -w '%{http_code}\n' -X POST "$BASE/pairing/invites" \
  -H "Authorization: Bearer $TOKEN_A"
cat /tmp/a_reinvite.json
```

期望：`409 ALREADY_PAIRED`（或等价）。

### 1.4 重登仍回同一 space

```bash
# refresh 或重新 login 后
curl -sS "$BASE/pairing/space" -H "Authorization: Bearer $TOKEN_A"
curl -sS "$BASE/auth/me" -H "Authorization: Bearer $TOKEN_B"
```

期望：双方 `space.id` 相同；`me.pairing.status=paired`。

---

## 2. 发动态：文案 + 多图，双方时间线倒序

### 2.1 上传意图（可重复最多 9 次）

```bash
curl -sS -X POST "$BASE/media/uploads" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"content_type":"image/jpeg","byte_size":12345}' | tee /tmp/up1.json
MEDIA1=$(jq -r .media_id /tmp/up1.json)
UPLOAD_URL=$(jq -r .upload_url /tmp/up1.json)

# 直传对象存储（本地 mock 可跳过或 PUT 空文件）
# curl -sS -X PUT "$UPLOAD_URL" -H 'Content-Type: image/jpeg' --data-binary @./photo1.jpg
```

再拿 `MEDIA2`…（≤9）。拒绝：`byte_size>10485760` 或非法 `content_type` → `400`。

### 2.2 发帖（带图）

```bash
curl -sS -X POST "$BASE/posts" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d "{\"body\":\"今天天气真好\",\"media_ids\":[\"$MEDIA1\"]}" | tee /tmp/post1.json
POST1=$(jq -r .id /tmp/post1.json)
```

### 2.3 再发一条（稍后，用于倒序）

```bash
sleep 1
curl -sS -X POST "$BASE/posts" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d '{"body":"第二条，应排在前面"}' | tee /tmp/post2.json
```

### 2.4 双方时间线

```bash
curl -sS "$BASE/posts?limit=20" -H "Authorization: Bearer $TOKEN_A" | tee /tmp/tl_a.json
curl -sS "$BASE/posts?limit=20" -H "Authorization: Bearer $TOKEN_B" | tee /tmp/tl_b.json
```

期望：两侧 `items` 一致；第一条为「第二条…」；均含对方内容。

未配对用户访问：

```bash
curl -sS -o /tmp/c_posts.json -w '%{http_code}\n' "$BASE/posts" -H "Authorization: Bearer $TOKEN_C"
```

期望：`403`。

---

## 3. 定位可选

```bash
# 无定位
curl -sS -X POST "$BASE/posts" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"body":"无定位也可以发"}'

# 有定位 + P1 visibility
curl -sS -X POST "$BASE/posts" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"body":"外滩散步","location":{"lat":31.240,\"lng\":121.490,\"name\":\"外滩\",\"visibility\":\"city\"}}'
```

期望：两条均 `201`；带定位的帖双方读到相同 `location.visibility`。

---

## 4. 赞 / 取消 / 评论

```bash
# A 赞
curl -sS -X PUT "$BASE/posts/$POST1/likes" -H "Authorization: Bearer $TOKEN_A"
# B 侧可见 like_count
curl -sS "$BASE/posts/$POST1" -H "Authorization: Bearer $TOKEN_B"
# A 取消
curl -sS -X DELETE "$BASE/posts/$POST1/likes" -H "Authorization: Bearer $TOKEN_A"

# B 评论
curl -sS -X POST "$BASE/posts/$POST1/comments" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d '{"body":"拍得真好看"}' | tee /tmp/cmt.json
CMT=$(jq -r .id /tmp/cmt.json)

curl -sS "$BASE/posts/$POST1/comments" -H "Authorization: Bearer $TOKEN_A"
```

期望：赞状态双方一致；评论列表含 `body` 与 `author.display_name`（Bob）。

```bash
curl -sS -X DELETE "$BASE/comments/$CMT" -H "Authorization: Bearer $TOKEN_B"
```

---

## 5. 餐厅打卡

```bash
curl -sS -X POST "$BASE/checkins" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"name":"一乐拉面","checked_in_at":"2026-10-29T12:00:00Z","address":"上海市…"}' | tee /tmp/ck.json
CK=$(jq -r .id /tmp/ck.json)

curl -sS "$BASE/checkins" -H "Authorization: Bearer $TOKEN_B"
```

期望：B 可见 `name` + `checked_in_at`。

```bash
curl -sS -X DELETE "$BASE/checkins/$CK" -H "Authorization: Bearer $TOKEN_A"
```

---

## 6. 纪念日 + 设备 token + 本地 09:00 推送（APNs 与 FCM）

推送没有公开 HTTP 发送接口，也没有 `POST /v1/dev/push/test`。API 进程内每分钟扫一次 `remind=true` 的纪念日，在纪念日 `timezone` 的 `remind_local_time` 起 2 分钟窗口内，向该 space 两名成员名下的**每一个**已登记 token 发一条。`platform=ios` 走 **APNs**，`platform=android` 走 **FCM**。两条通道都要测。时刻用纪念日上的时区（默认 `Asia/Shanghai` 的 09:00），不是每台手机自己的系统时区。

系统通知关闭：请求体没有权限字段。客户端只要提交合法 `platform`+`token`，登记必须 `200`。拿不到 token 时不要调用登记。进 App 的权限文案是客户端的事，本接口不失败、进程不崩溃。

凭证为空时不访问 Apple / Google，日志类似（不打印 token 明文）：

```text
push: apns configured=false fcm configured=false
push stub: provider=apns token_len=17 title="今天也要在一起"
push stub: provider=fcm token_len=16 title="今天也要在一起"
```

`provider=apns` 表示会发给 iOS token，`provider=fcm` 表示会发给 Android token。`token_len` 是登记字符串的字节长度（下面两个测试 token 分别是 17 和 16）。

### 6.1 登记设备（含非法 platform / token）

先完成第 1 节配对。未配对也可以登记，但下面的推送只投给 space 成员。

```bash
curl -sS -X POST "$BASE/devices" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"platform":"ios","token":"TEST_APNS_TOKEN_A"}' | tee /tmp/dev_a.json

curl -sS -X POST "$BASE/devices" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d '{"platform":"android","token":"TEST_FCM_TOKEN_B"}' | tee /tmp/dev_b.json

# 大小写与空白会被规范化
curl -sS -X POST "$BASE/devices" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"platform":" IOS ","token":" TEST_APNS_TOKEN_A "}'
```

期望：两条均为 `200`；`platform` 分别为 `ios`、`android`；同一 token 再登记仍是 `200`（upsert）。这与系统通知开或关无关。

```bash
curl -sS -o /tmp/dev_bad_platform.json -w '%{http_code}\n' -X POST "$BASE/devices" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"platform":"apns","token":"TEST_APNS_TOKEN_A"}'
cat /tmp/dev_bad_platform.json

curl -sS -o /tmp/dev_bad_token.json -w '%{http_code}\n' -X POST "$BASE/devices" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"platform":"ios","token":"  "}'
cat /tmp/dev_bad_token.json
```

期望：两处都是 `400`，`code=VALIDATION_ERROR`。`platform` 不是 `ios`/`android` 时 message 为 `platform must be ios or android`（不要传 `apns`/`fcm`）。token 空或超过 512 字符时 message 为 `token must be 1 to 512 characters`。

### 6.2 默认提醒是当天本地 09:00；双方可见；只有创建者能改、能删

```bash
curl -sS -X POST "$BASE/anniversaries" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"title":"在一起","event_date":"2024-05-20"}' | tee /tmp/ann.json
ANN=$(jq -r .id /tmp/ann.json)

curl -sS "$BASE/anniversaries" -H "Authorization: Bearer $TOKEN_B" | tee /tmp/ann_b.json
```

期望：`201`。省略提醒字段时响应仍是 `remind=true`、`remind_days_before=0`、`remind_local_time="09:00"`、`timezone="Asia/Shanghai"`、`recurrence=yearly`。B 的列表含同一 `id`、标题与日期。

A 创建后，B（已配对、非创建者）改提醒或删除都是 `403`；A 自己可以改，也可以删。

```bash
# 非创建者改提醒字段 → 403 FORBIDDEN，行保持原值
curl -sS -o /tmp/ann_patch_forbid.json -w '%{http_code}\n' -X PATCH "$BASE/anniversaries/$ANN" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d '{"title":"被改掉","remind":false,"remind_days_before":1,"remind_local_time":"10:00","timezone":"Asia/Tokyo"}'
cat /tmp/ann_patch_forbid.json

# 非创建者删除 → 403 FORBIDDEN
curl -sS -o /tmp/ann_forbid.json -w '%{http_code}\n' -X DELETE "$BASE/anniversaries/$ANN" \
  -H "Authorization: Bearer $TOKEN_B"
cat /tmp/ann_forbid.json

curl -sS "$BASE/anniversaries" -H "Authorization: Bearer $TOKEN_B" | tee /tmp/ann_b_after_forbid.json

# 创建者可改提醒
curl -sS -X PATCH "$BASE/anniversaries/$ANN" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"remind_days_before":1}' | tee /tmp/ann_patch.json

# 创建者可删
curl -sS -o /dev/null -w '%{http_code}\n' -X DELETE "$BASE/anniversaries/$ANN" \
  -H "Authorization: Bearer $TOKEN_A"
```

期望：B 的 PATCH 与 DELETE 都是 `403`，`code=FORBIDDEN`（message 为 `only the author can do that`）。拒绝之后 B 的列表仍含 `$ANN`，且仍是 `title=在一起`、`remind=true`、`remind_days_before=0`、`remind_local_time="09:00"`、`timezone="Asia/Shanghai"`。A 的 PATCH 为 `200` 且 `remind_days_before=1`；A 的 DELETE 为 `204`。删除后 B 的列表不再包含 `$ANN`。

### 6.3 本地 dry-run：对准「现在」触发 stub，不必等到早上 9 点

保持 `APNS_KEY_ID`、`APNS_TEAM_ID`、`APNS_BUNDLE_ID`、`APNS_KEY_PATH`、`FCM_PROJECT_ID`、`FCM_CREDENTIALS_PATH` 为空，重启或确认 API 已在跑，启动日志含 `push: apns configured=false fcm configured=false`。

worker 约每 60 秒跳一次，**不会在进程启动的那一秒立刻扫**。窗口是当地 `remind_local_time` 起的 2 分钟。API 先跑起来，再由创建者把提醒时刻设成 Asia/Shanghai 的当前 `HH:MM`，然后盯日志最多约 2 分钟。没出现就由创建者再 PATCH 成新的当前分钟（新时刻是另一次触发，不受上一次 48 小时去重影响）。非创建者 PATCH 为 `403`。

```bash
TODAY=$(TZ=Asia/Shanghai date +%F)
NOW=$(TZ=Asia/Shanghai date +%H:%M)

curl -sS -X POST "$BASE/anniversaries" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d "{\"title\":\"今天也要在一起\",\"event_date\":\"$TODAY\",\"remind\":true,\"remind_days_before\":0,\"remind_local_time\":\"$NOW\",\"timezone\":\"Asia/Shanghai\"}" \
  | tee /tmp/ann_today.json
ANN_TODAY=$(jq -r .id /tmp/ann_today.json)
echo "wait for stub log; fire minute=$NOW date=$TODAY id=$ANN_TODAY"
```

期望（API 标准输出，顺序不限）：

- `push stub: provider=apns token_len=17 title="今天也要在一起"`（A 的 iOS token `TEST_APNS_TOKEN_A` 长度为 17）
- `push stub: provider=fcm token_len=16 title="今天也要在一起"`（B 的 Android token `TEST_FCM_TOKEN_B` 长度为 16）

同一轮只应出现一次 apns、一次 fcm（每人一个 token 时）。没有登记 token 的成员不会有对应行，也不报错。`remind=false` 的纪念日没有 stub 行。

客户端应解析的正文与 data（OpenAPI `AnniversaryPush` / `ApnsAnniversaryPush` / `FcmAnniversaryPush`）：

| 字段 | 当天（remind_days_before=0） |
|------|------------------------------|
| title | 纪念日标题，例如 `今天也要在一起` |
| body | `今天是「今天也要在一起」` |
| data.type | `anniversary_reminder` |
| data.anniversary_id | 纪念日 id |
| data.space_id | space id |
| data.event_date | `YYYY-MM-DD` |
| data.remind_days_before | 字符串 `"0"`（提前 n 天则为 `"n"`，body 为 `「标题」还有 n 天`） |

APNs：`aps.alert.title/body`，`aps.sound=default`，data 键与 `aps` 同级。FCM HTTP v1：`message.notification.title/body`，`message.data` 为上表，`message.android.priority=HIGH`。

本地 stub **只打印 title**，不打印 body/data，也不发网。本步通过标准是两条 `push stub` 日志。data 键已在契约里锁定，供客户端先写解析；sender 还没附带 data 时，缺 data 只展示 title/body，App 不崩溃。真机在凭证配齐后按该 schema 核对报文。系统通知关闭时 OS 可能不展示，服务端仍会按 token 尝试（stub 下就是上面的日志）。

去重：同一次触发（纪念日 id + 该次本地时刻）48 小时内只扇出一轮。窗口错过不会自动补发。由创建者（`$TOKEN_A`）把 `remind_local_time` 改到新的当前分钟再等下一跳。非创建者 PATCH 为 `403`，时刻不变。

```bash
NOW=$(TZ=Asia/Shanghai date +%H:%M)
curl -sS -o /tmp/ann_retry.json -w '%{http_code}\n' -X PATCH "$BASE/anniversaries/$ANN_TODAY" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d "{\"remind_local_time\":\"$NOW\"}"
```

期望：创建者这次 PATCH 为 `200`。用 `$TOKEN_B` 做同样的 PATCH 应为 `403`。

提前一天的可选核对：`event_date` 设为 Asia/Shanghai 的明天、`remind_days_before=1`、`remind_local_time` 设为当前分钟。stub 的 title 仍是标题；约定 body 为 `「标题」还有 1 天`。

测完删除，避免下一分钟重复刷日志：

```bash
curl -sS -o /dev/null -w '%{http_code}\n' -X DELETE "$BASE/anniversaries/$ANN_TODAY" \
  -H "Authorization: Bearer $TOKEN_A"
curl -sS -o /dev/null -w '%{http_code}\n' -X DELETE "$BASE/devices" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"token":"TEST_APNS_TOKEN_A"}'
curl -sS -o /dev/null -w '%{http_code}\n' -X DELETE "$BASE/devices" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d '{"token":"TEST_FCM_TOKEN_B"}'
# 再删一次同一 token：仍然 204
curl -sS -o /dev/null -w '%{http_code}\n' -X DELETE "$BASE/devices" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"token":"TEST_APNS_TOKEN_A"}'
```

期望：纪念日删除 `204`；注销 `204`；token 不存在也是 `204`。空 token 注销为 `400 VALIDATION_ERROR`。

### 6.4 以后接真 APNs / FCM 时的环境变量（只写名字）

留空即 stub。不要把密钥、p8、服务账号 JSON 写进仓库或本文。

| 变量 | 用途 |
|------|------|
| `APNS_KEY_ID` | APNs Auth Key 的 Key ID |
| `APNS_TEAM_ID` | Apple Team ID |
| `APNS_BUNDLE_ID` | iOS bundle id，作为 `apns-topic` |
| `APNS_KEY_PATH` | 本机 `.p8` 路径（文件不入库） |
| `APNS_USE_SANDBOX` | `true` 时走 `api.sandbox.push.apple.com`，否则走生产 |
| `FCM_PROJECT_ID` | Firebase 项目 id |
| `FCM_CREDENTIALS_PATH` | 服务账号 JSON 路径（文件不入库） |
| `APP_ENV` | 本地用 `local` 或 `dev`。与推送开关无关；空凭证即 stub |
| `REDIS_URL` | 可选。可用时去重键为 `reminder:{anniversaryId}:{fireAt RFC3339}`，TTL 48h；Redis 不可用时退回进程内内存 |

四项 APNs 都非空才发 APNs；`FCM_PROJECT_ID` 与 `FCM_CREDENTIALS_PATH` 都非空才发 FCM。只配一边时，另一边仍 stub。本 PR 不要求配真证书。

---

## 7. P1：餐厅收藏双方一致

```bash
curl -sS -X POST "$BASE/favorites" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d '{"name":"海底捞","place_id":"poi_demo_1"}' | tee /tmp/fav.json
FAV=$(jq -r .id /tmp/fav.json)

curl -sS "$BASE/favorites" -H "Authorization: Bearer $TOKEN_A"
# 非作者删除 → 403 FORBIDDEN（OpenAPI：仅作者可删）
curl -sS -o /tmp/fav_forbid.json -w '%{http_code}\n' -X DELETE "$BASE/favorites/$FAV" \
  -H "Authorization: Bearer $TOKEN_A"
# 作者删除
curl -sS -X DELETE "$BASE/favorites/$FAV" -H "Authorization: Bearer $TOKEN_B"
curl -sS "$BASE/favorites" -H "Authorization: Bearer $TOKEN_A"
curl -sS "$BASE/favorites" -H "Authorization: Bearer $TOKEN_B"
```

期望：添加后双方列表一致；非作者删除 `403`；作者删除后双方均无该条。

---

## 8. Token 刷新

```bash
REFRESH_A=$(jq -r .refresh_token /tmp/reg_a.json)
curl -sS -X POST "$BASE/auth/refresh" -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$REFRESH_A\"}"
```

期望：新 `access_token`；用新 token 仍可 `GET /pairing/space`。

---

## 验收勾选

| # | 场景 | 通过 |
|---|------|------|
| 1 | 邀请→接受→同 space；第三人不可入；重登仍同 space | ☐ |
| 2 | 文+多图，双方倒序可见 | ☐ |
| 3 | 定位可选；不选可发 | ☐ |
| 4 | 赞/取消一致；评论见内容与作者 | ☐ |
| 5 | 打卡名+时间双方可见 | ☐ |
| 6 | 纪念日双方可见；默认当天 09:00；非创建者 PATCH/DELETE 为 403；创建者可改可删；ios→APNs、android→FCM；本地 stub 日志各一条；非法 token 400；关系统通知不影响登记 | ☐ |
| 7 | P1 收藏双方一致；定位 visibility 双方一致 | ☐ |
