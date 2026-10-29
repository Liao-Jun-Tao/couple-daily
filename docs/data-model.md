# 情侣日常 App（couple-daily）— 数据模型

> 范围：P0 + P1 字段预留。ID 均为 UUID string；时间均为 RFC3339 UTC 存储；业务表一律带 `space_id`，中间件校验当前用户属于该 space。
>
> **Auth 已锁定：邮箱 + 密码。** 手机号验证码不上首版。不建 `phone` 列，不建 `otp_challenges`。
>
> 已锁：仅作者可删、发帖禁止双空、发起邀请时建 space（`member_count=1`）、邀请码 6 位 / 24h / 一码一用、APNs + FCM、多图 ≤9 且单张 ≤10MB、定位默认 `exact`、打卡可手填店名、纪念日默认当天本地 09:00。

迁移初稿：`backend/migrations/000001_init.up.sql`。

## ER 关系概览

```
User ──< RefreshToken
User ──< DeviceToken
User ──< SpaceMember >── CoupleSpace
CoupleSpace ──< InviteCode
CoupleSpace ──< Post ──< PostMedia >── MediaAsset
Post ──< Like
Post ──< Comment
CoupleSpace ──< RestaurantCheckin
CoupleSpace ──< RestaurantFavorite（P1）
CoupleSpace ──< Anniversary
```

唯一双人空间：一个 `CoupleSpace` 最多 2 名 `SpaceMember`；`member_count=2` 后拒绝再接受邀请。

---

## 表定义

### users

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | uuid | PK | |
| email | citext | UNIQUE NOT NULL | 登录账号，大小写不敏感 |
| password_hash | varchar(255) | NOT NULL | argon2id PHC 字符串 |
| display_name | varchar(64) | NOT NULL | 展示名 |
| avatar_media_id | uuid | NULL FK→media_assets | 可选头像 |
| created_at | timestamptz | NOT NULL | |
| updated_at | timestamptz | NOT NULL | |

索引：`UNIQUE(email)`。首版没有 `phone`。

### refresh_tokens

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | uuid | PK | |
| user_id | uuid | FK→users NOT NULL | |
| token_hash | varchar(64) | UNIQUE NOT NULL | 存 hash，不存明文 |
| expires_at | timestamptz | NOT NULL | |
| revoked_at | timestamptz | NULL | |
| created_at | timestamptz | NOT NULL | |

索引：`(user_id)`，`UNIQUE(token_hash)`。

### couple_spaces

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | uuid | PK | space_id |
| member_count | smallint | NOT NULL DEFAULT 0 | 0–2；=2 后拒绝再加入 |
| created_at | timestamptz | NOT NULL | |
| updated_at | timestamptz | NOT NULL | |

约束：`CHECK (member_count >= 0 AND member_count <= 2)`。

发起邀请时写入 `member_count=1`。列默认值仍为 0，避免未完成的插入被误看成已配对。

### space_members

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | uuid | PK | |
| space_id | uuid | FK→couple_spaces NOT NULL | |
| user_id | uuid | FK→users NOT NULL | |
| role | varchar(16) | NOT NULL DEFAULT 'member' | 预留 owner/member |
| joined_at | timestamptz | NOT NULL | |

约束：

- `UNIQUE(space_id, user_id)`
- `UNIQUE(user_id)` — **一用户同时只属于一个 space**（MVP）
- 应用层与 `member_count` 约束保证同一 space 成员数 ≤ 2

索引：`(space_id)`，`UNIQUE(user_id)`。

### invite_codes

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | uuid | PK | |
| code | char(6) | UNIQUE NOT NULL | 6 位字母数字，存储为大写，`^[A-Z0-9]{6}$` |
| inviter_user_id | uuid | FK→users NOT NULL | |
| space_id | uuid | FK→couple_spaces NOT NULL | 发起邀请时即绑定刚创建的 space |
| status | varchar(16) | NOT NULL | `pending` \| `accepted` \| `expired` \| `revoked` |
| expires_at | timestamptz | NOT NULL | 创建 + 24h |
| accepted_by_user_id | uuid | FK→users NULL | |
| accepted_at | timestamptz | NULL | |
| created_at | timestamptz | NOT NULL | |

约束：**一码一用** — `accepted` 后不可再 accept；过期在接受时校验，也可由 worker 标记。

索引：`UNIQUE(code)`，`(inviter_user_id, status)`，`(expires_at) WHERE status='pending'`。

已锁流程：未配对用户 `POST /pairing/invites` → 创建 `couple_spaces`（`member_count=1`）+ 把自己写入 `space_members` + 生成 invite。对方 accept → 校验 pending、未过期、`space.member_count=1`、接受方不在其他 space → 加入第二人、`member_count=2`、`invite.status=accepted`。第三人返回 `SPACE_FULL`。

### media_assets

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | uuid | PK | |
| uploader_user_id | uuid | FK→users NOT NULL | |
| space_id | uuid | FK→couple_spaces NULL | 上传意图时可先空，绑定帖子时写入 |
| content_type | varchar(64) | NOT NULL | `image/jpeg` \| `image/png` \| `image/heic` \| `image/webp` |
| byte_size | int | NOT NULL | ≤ 10_485_760（10MB） |
| storage_key | varchar(512) | UNIQUE NOT NULL | 对象存储路径 |
| status | varchar(16) | NOT NULL | `pending_upload` \| `ready` \| `failed` |
| created_at | timestamptz | NOT NULL | |

索引：`(uploader_user_id)`，`(space_id)`。

### posts

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | uuid | PK | |
| space_id | uuid | FK→couple_spaces NOT NULL | |
| author_user_id | uuid | FK→users NOT NULL | |
| body | text | NOT NULL | 文案；允许空串，但须与 media **禁止双空**（仅图 / 仅文均可） |
| location_lat | double precision | NULL | 可选 |
| location_lng | double precision | NULL | 与 lat 同有或同无 |
| location_name | varchar(256) | NULL | 展示名 / POI 名 |
| location_place_id | varchar(128) | NULL | 可选外部 POI id |
| location_visibility | varchar(16) | NOT NULL DEFAULT 'exact' | 默认 `exact`；P1 可用 `city` / `hidden` |
| created_at | timestamptz | NOT NULL | |
| updated_at | timestamptz | NOT NULL | |
| deleted_at | timestamptz | NULL | 软删 |

约束：`CHECK ((location_lat IS NULL) = (location_lng IS NULL))`；`location_visibility IN ('exact','city','hidden')`。

禁止双空由服务层硬校验：`trim(body) != "" OR 关联 media ≥ 1`。

索引：`(space_id, created_at DESC, id DESC) WHERE deleted_at IS NULL`。

### post_media

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| post_id | uuid | FK→posts NOT NULL | |
| media_id | uuid | FK→media_assets NOT NULL | |
| sort_order | smallint | NOT NULL | 0–8 |

约束：`PRIMARY KEY (post_id, media_id)`；`UNIQUE(post_id, sort_order)`；每帖 media ≤ 9（`sort_order` 0–8，应用层再校验张数）。

### likes

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | uuid | PK | |
| space_id | uuid | FK→couple_spaces NOT NULL | 冗余便于隔离校验 |
| post_id | uuid | FK→posts NOT NULL | |
| user_id | uuid | FK→users NOT NULL | |
| created_at | timestamptz | NOT NULL | |

约束：`UNIQUE(post_id, user_id)` — 同一用户对同一帖只能赞一次。取消赞仅本人。

索引：`(post_id)`，`(space_id, post_id)`。

### comments

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | uuid | PK | |
| space_id | uuid | FK→couple_spaces NOT NULL | |
| post_id | uuid | FK→posts NOT NULL | |
| author_user_id | uuid | FK→users NOT NULL | |
| body | text | NOT NULL | 1–2000 字符 |
| created_at | timestamptz | NOT NULL | |
| deleted_at | timestamptz | NULL | |

索引：`(post_id, created_at ASC, id ASC) WHERE deleted_at IS NULL`。仅作者可删。

### restaurant_checkins

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | uuid | PK | |
| space_id | uuid | FK→couple_spaces NOT NULL | |
| creator_user_id | uuid | FK→users NOT NULL | |
| name | varchar(128) | NOT NULL | 餐厅名；允许手填，不要求 place_id |
| address | varchar(256) | NULL | |
| lat | double precision | NULL | |
| lng | double precision | NULL | |
| place_id | varchar(128) | NULL | 手填店名时为空 |
| checked_in_at | timestamptz | NOT NULL | 客户端可传，默认 now |
| created_at | timestamptz | NOT NULL | |
| deleted_at | timestamptz | NULL | |

索引：`(space_id, checked_in_at DESC, id DESC) WHERE deleted_at IS NULL`。仅作者可删。

### restaurant_favorites（P1）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | uuid | PK | |
| space_id | uuid | FK→couple_spaces NOT NULL | |
| creator_user_id | uuid | FK→users NOT NULL | |
| name | varchar(128) | NOT NULL | 可手填 |
| address | varchar(256) | NULL | |
| lat | double precision | NULL | |
| lng | double precision | NULL | |
| place_id | varchar(128) | NULL | |
| note | varchar(512) | NULL | |
| created_at | timestamptz | NOT NULL | |
| deleted_at | timestamptz | NULL | |

索引：`(space_id, created_at DESC, id DESC)`；`UNIQUE(space_id, place_id) WHERE place_id IS NOT NULL AND deleted_at IS NULL`。仅作者可删。

### anniversaries

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | uuid | PK | |
| space_id | uuid | FK→couple_spaces NOT NULL | |
| creator_user_id | uuid | FK→users NOT NULL | |
| title | varchar(128) | NOT NULL | 名称 |
| event_date | date | NOT NULL | 纪念日日期（按日，无时区） |
| recurrence | varchar(16) | NOT NULL DEFAULT 'yearly' | `yearly` \| `once` |
| remind | boolean | NOT NULL DEFAULT true | 是否提醒 |
| remind_days_before | smallint | NOT NULL DEFAULT 0 | 提前天数；0 = 当天 |
| remind_local_time | time | NOT NULL DEFAULT '09:00' | 本地提醒时刻；默认当天 09:00 |
| timezone | varchar(64) | NOT NULL DEFAULT 'Asia/Shanghai' | IANA，用于换算推送 UTC |
| created_at | timestamptz | NOT NULL | |
| updated_at | timestamptz | NOT NULL | |
| deleted_at | timestamptz | NULL | |

索引：`(space_id, event_date)`；`(remind) WHERE remind=true AND deleted_at IS NULL`。仅作者可删。

推送调度：服务端按 `timezone` + `remind_local_time` + `remind_days_before` 换算下次触发 UTC。默认是纪念日当天 09:00（`Asia/Shanghai`）。向该 space 双方已登记的 `device_tokens` 发推送：iOS 走 APNs，Android 走 FCM。本契约只登记设备与提醒配置；worker 落地前不发推送。

### device_tokens

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | uuid | PK | |
| user_id | uuid | FK→users NOT NULL | |
| platform | varchar(16) | NOT NULL | `ios`（APNs）\| `android`（FCM） |
| token | varchar(512) | NOT NULL | 推送 token |
| created_at | timestamptz | NOT NULL | |
| updated_at | timestamptz | NOT NULL | |
| last_seen_at | timestamptz | NOT NULL | |

约束：`UNIQUE(user_id, token)`。索引：`(user_id)`。

API 只登记 / 注销 token。用户未开系统通知时登记仍成功；客户端不崩溃，并可做 in-app 提示。

---

## 资源隔离规则

1. 除 Auth / Pairing（create 与 accept）/ Devices 外，业务 API 从 JWT 解析 `user_id`，经 `space_members` 得到 `space_id`。
2. 列表与读写一律 `WHERE space_id = :current_space_id`。
3. 删除：**仅作者**可删帖 / 评 / 打卡 / 纪念日 / 收藏；点赞仅本人可取消。
4. `member_count=2` 后，第三人 `POST /pairing/invites/accept` 返回 `SPACE_FULL`。已在 space 中的用户再发邀请返回 `ALREADY_PAIRED`。

## Cursor 分页

请求：`?limit=20&cursor=<opaque>`。

Cursor 编码：base64url(`created_at|id`)，列表按 `(created_at DESC, id DESC)`；下一页条件 `(created_at, id) < (cursor_created_at, cursor_id)`。

## 不做（表不建）

手机号、短信验证码、语音 / 视频动态、心愿单、记账、外部 POI 同步库。
