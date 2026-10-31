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

## 6. 纪念日 + 设备 token（推送调度草案期人工核对）

```bash
curl -sS -X POST "$BASE/devices" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"platform":"ios","token":"TEST_APNS_OR_FCM_TOKEN_A"}'

curl -sS -X POST "$BASE/anniversaries" \
  -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' \
  -d '{"title":"在一起","event_date":"2024-05-20","remind":true,"remind_days_before":0,"remind_local_time":"09:00","timezone":"Asia/Shanghai"}' \
  | tee /tmp/ann.json
ANN=$(jq -r .id /tmp/ann.json)

curl -sS "$BASE/anniversaries" -H "Authorization: Bearer $TOKEN_B"
```

期望：B 列表可见名称与日期；提醒字段默认当天本地 09:00。

草案期推送：确认 worker/cron 文档存在即可；实现后在「纪念日当天 09:00 Asia/Shanghai」查双方 device 是否收到一条推送。

```bash
curl -sS -X PATCH "$BASE/anniversaries/$ANN" \
  -H "Authorization: Bearer $TOKEN_B" -H 'Content-Type: application/json' \
  -d '{"remind_days_before":1}'
curl -sS -X DELETE "$BASE/anniversaries/$ANN" -H "Authorization: Bearer $TOKEN_A"
```

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
| 6 | 纪念日列表；提醒配置可写；设备可登记 | ☐ |
| 7 | P1 收藏双方一致；定位 visibility 双方一致 | ☐ |
