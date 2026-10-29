# couple-daily 自测说明（P0 + P1 字段）

> 业务基址默认 `http://localhost:8080/v1`。健康检查在版本前缀之外：`http://localhost:8080/healthz`。
> 将 `$BASE`、`$TOKEN_A`、`$TOKEN_B`、`$TOKEN_C` 替换为实际值。
> 验收对齐：邮箱注册 → 配对 → 发帖 → 赞评 → 打卡 → 纪念日（+ P1 收藏 / 定位粒度）。
>
> **Auth 已锁定为邮箱 + 密码。** 手机号验证码不上首版，不要调用 `/auth/otp/*`。
>
> 脚手架阶段：`GET /healthz` 返回 200；`/v1` 下契约路由返回 `501` 与 `{"code":"NOT_IMPLEMENTED"}`。下面的业务步骤在对应 handler 落地后执行。

```bash
BASE=http://localhost:8080/v1
curl -sS http://localhost:8080/healthz
# 期望 {"status":"ok"}
```

---

## 0. 准备：邮箱密码注册，拿到三人 token

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

期望：三人各 `201`，响应含 `access_token` 与 `refresh_token`。重复注册同一邮箱 → `409`。

登录回归：

```bash
curl -sS -X POST "$BASE/auth/login" -H 'Content-Type: application/json' \
  -d '{"email":"a@example.com","password":"password123"}'
```

错误密码 → `401`。

---

## 1. 配对：邀请 → 接受 → 第三人拒绝 → 重登回同一 space

### 1.1 A 发起邀请

```bash
curl -sS -X POST "$BASE/pairing/invites" -H "Authorization: Bearer $TOKEN_A" | tee /tmp/invite.json
CODE=$(jq -r .invite_code /tmp/invite.json)
SPACE_ID=$(jq -r .space.id /tmp/invite.json)
```

期望：`invite_code` 为 6 位 `[A-Z0-9]`；`expires_at` ≈ now+24h；`space.member_count=1`。发起时已创建 space，A 已是成员。

### 1.2 B 接受

```bash
curl -sS -X POST "$BASE/pairing/invites/accept" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d "{\"invite_code\":\"$CODE\"}" | tee /tmp/space_ab.json
```

期望：`member_count=2`，`members` 含 A、B；`id` 等于 `$SPACE_ID`。

### 1.3 C 再接受同一码

```bash
curl -sS -o /tmp/c_reuse.json -w '%{http_code}\n' -X POST "$BASE/pairing/invites/accept" \
  -H "Authorization: Bearer $TOKEN_C" -H 'Content-Type: application/json' \
  -d "{\"invite_code\":\"$CODE\"}"
cat /tmp/c_reuse.json
```

期望：`4xx`，`code` 为 `INVITE_USED` 或 `INVITE_INVALID`。一码一用。

已配对用户再发邀请：

```bash
curl -sS -o /tmp/a_reinvite.json -w '%{http_code}\n' -X POST "$BASE/pairing/invites" \
  -H "Authorization: Bearer $TOKEN_A"
cat /tmp/a_reinvite.json
```

期望：`409 ALREADY_PAIRED`。

### 1.4 重登仍回同一 space

```bash
curl -sS "$BASE/pairing/space" -H "Authorization: Bearer $TOKEN_A"
curl -sS "$BASE/auth/me" -H "Authorization: Bearer $TOKEN_B"
```

期望：双方 `space.id` 相同；`me.pairing.status=paired`；`me.user.email` 为登录邮箱。

---

## 2. 发动态：文案 + 多图，双方时间线倒序

### 2.1 上传意图（最多 9 次）

```bash
curl -sS -X POST "$BASE/media/uploads" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"content_type":"image/jpeg","byte_size":12345}' | tee /tmp/up1.json
MEDIA1=$(jq -r .media_id /tmp/up1.json)
UPLOAD_URL=$(jq -r .upload_url /tmp/up1.json)

# curl -sS -X PUT "$UPLOAD_URL" -H 'Content-Type: image/jpeg' --data-binary @./photo1.jpg
```

拒绝：`byte_size>10485760` 或非法 `content_type` → `400`。

### 2.2 禁止双空

```bash
curl -sS -o /tmp/empty_post.json -w '%{http_code}\n' -X POST "$BASE/posts" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"body":"   ","media_ids":[]}'
```

期望：`400`。仅文或仅图均可。

### 2.3 发帖（带图）

```bash
curl -sS -X POST "$BASE/posts" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d "{\"body\":\"今天天气真好\",\"media_ids\":[\"$MEDIA1\"]}" | tee /tmp/post1.json
POST1=$(jq -r .id /tmp/post1.json)
```

### 2.4 再发一条（用于倒序）

```bash
sleep 1
curl -sS -X POST "$BASE/posts" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d '{"body":"第二条，应排在前面"}' | tee /tmp/post2.json
```

### 2.5 双方时间线

```bash
curl -sS "$BASE/posts?limit=20" -H "Authorization: Bearer $TOKEN_A" | tee /tmp/tl_a.json
curl -sS "$BASE/posts?limit=20" -H "Authorization: Bearer $TOKEN_B" | tee /tmp/tl_b.json
curl -sS -o /tmp/c_posts.json -w '%{http_code}\n' "$BASE/posts" -H "Authorization: Bearer $TOKEN_C"
```

期望：A、B 的 `items` 一致，第一条为「第二条…」；C 为 `403`。

---

## 3. 定位可选，默认 exact

```bash
curl -sS -X POST "$BASE/posts" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"body":"无定位也可以发"}'

curl -sS -X POST "$BASE/posts" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"body":"外滩散步","location":{"lat":31.240,"lng":121.490,"name":"外滩","visibility":"city"}}'
```

期望：两条均 `201`。未传 `visibility` 时存储并返回 `exact`。带定位的帖双方读到相同 `location.visibility`。

---

## 4. 赞 / 取消 / 评论 / 仅作者可删

```bash
curl -sS -X PUT "$BASE/posts/$POST1/likes" -H "Authorization: Bearer $TOKEN_A"
curl -sS "$BASE/posts/$POST1" -H "Authorization: Bearer $TOKEN_B"
curl -sS -X DELETE "$BASE/posts/$POST1/likes" -H "Authorization: Bearer $TOKEN_A"

curl -sS -X POST "$BASE/posts/$POST1/comments" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d '{"body":"拍得真好看"}' | tee /tmp/cmt.json
CMT=$(jq -r .id /tmp/cmt.json)
curl -sS "$BASE/posts/$POST1/comments" -H "Authorization: Bearer $TOKEN_A"
```

期望：赞状态双方一致；评论列表含 `body` 与 `author.display_name`（Bob）。

```bash
# 非作者删帖 → 403；作者删评论 → 204
curl -sS -o /tmp/del_post.json -w '%{http_code}\n' -X DELETE "$BASE/posts/$POST1" \
  -H "Authorization: Bearer $TOKEN_B"
curl -sS -o /dev/null -w '%{http_code}\n' -X DELETE "$BASE/comments/$CMT" \
  -H "Authorization: Bearer $TOKEN_B"
```

期望：删帖 `403`，删自己的评论 `204`。

---

## 5. 餐厅打卡（可手填店名）

```bash
curl -sS -X POST "$BASE/checkins" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"name":"一乐拉面","checked_in_at":"2026-10-29T12:00:00Z","address":"上海市…"}' | tee /tmp/ck.json
CK=$(jq -r .id /tmp/ck.json)

curl -sS -X POST "$BASE/checkins" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d '{"name":"巷口小店"}'

curl -sS "$BASE/checkins" -H "Authorization: Bearer $TOKEN_A"
curl -sS -o /dev/null -w '%{http_code}\n' -X DELETE "$BASE/checkins/$CK" \
  -H "Authorization: Bearer $TOKEN_B"
curl -sS -o /dev/null -w '%{http_code}\n' -X DELETE "$BASE/checkins/$CK" \
  -H "Authorization: Bearer $TOKEN_A"
```

期望：手填店名（无 `place_id`）成功；B 能看见 A 的打卡；B 删 A 的打卡为 `403`，A 自己删除为 `204`。

---

## 6. 纪念日 + 设备 token

```bash
curl -sS -X POST "$BASE/devices" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"platform":"ios","token":"TEST_APNS_TOKEN_A"}'
curl -sS -X POST "$BASE/devices" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d '{"platform":"android","token":"TEST_FCM_TOKEN_B"}'

curl -sS -X POST "$BASE/anniversaries" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"title":"在一起","event_date":"2024-05-20","remind":true}' | tee /tmp/ann.json
ANN=$(jq -r .id /tmp/ann.json)
curl -sS "$BASE/anniversaries" -H "Authorization: Bearer $TOKEN_B"
```

期望：省略提醒字段时，`remind_local_time=09:00`、`timezone=Asia/Shanghai`、`remind_days_before=0`。B 的列表可见该纪念日。未开系统通知不影响登记接口。

```bash
curl -sS -X PATCH "$BASE/anniversaries/$ANN" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d '{"remind_days_before":1}'
curl -sS -o /dev/null -w '%{http_code}\n' -X DELETE "$BASE/anniversaries/$ANN" \
  -H "Authorization: Bearer $TOKEN_A"
```

实现推送后，在纪念日当天本地 09:00 核对双方设备是否各收到一条（APNs / FCM）。脚手架不发送推送。

---

## 7. P1：餐厅收藏双方一致

```bash
curl -sS -X POST "$BASE/favorites" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d '{"name":"海底捞","place_id":"poi_demo_1"}' | tee /tmp/fav.json
FAV=$(jq -r .id /tmp/fav.json)

curl -sS "$BASE/favorites" -H "Authorization: Bearer $TOKEN_A"
curl -sS -o /dev/null -w '%{http_code}\n' -X DELETE "$BASE/favorites/$FAV" \
  -H "Authorization: Bearer $TOKEN_A"
```

期望：添加后双方列表一致。A 不是作者，删除为 `403`，B 的列表仍在。

```bash
curl -sS -o /dev/null -w '%{http_code}\n' -X DELETE "$BASE/favorites/$FAV" \
  -H "Authorization: Bearer $TOKEN_B"
```

期望：`204`，双方列表都没有该条。

---

## 8. Token 刷新

```bash
REFRESH_A=$(jq -r .refresh_token /tmp/reg_a.json)
curl -sS -X POST "$BASE/auth/refresh" -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$REFRESH_A\"}"
```

期望：新的 `access_token`；用新 token 仍可 `GET /pairing/space`。

---

## 验收勾选

| # | 场景 | 通过 |
|---|------|------|
| 0 | 邮箱密码注册 / 登录；无手机号 OTP | ☐ |
| 1 | 邀请时 space.member_count=1；接受后为 2；一码一用；第三人不可入；重登仍同 space | ☐ |
| 2 | 禁止双空；文 + 多图（≤9，单张 ≤10MB）；双方倒序可见 | ☐ |
| 3 | 定位可选；默认 exact | ☐ |
| 4 | 赞 / 取消一致；评论见内容与作者；非作者删除 403 | ☐ |
| 5 | 打卡名可手填；双方可见；仅作者可删 | ☐ |
| 6 | 纪念日默认当天本地 09:00；iOS APNs / Android FCM token 可登记 | ☐ |
| 7 | P1 收藏双方一致；非作者不能删 | ☐ |
