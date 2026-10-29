# couple-daily 契约（0.1.0）

## 已锁定

- 项目：**couple-daily** monorepo
- **Auth：邮箱 + 密码注册登录（MVP）。手机号验证码不上首版。** 不再保留手机 OTP 候选，OpenAPI / 数据模型 / 自测只描述邮箱密码。
- 服务端栈：**Go + chi + PostgreSQL + Redis + S3 兼容 + APNs/FCM**。源码在 `backend/`，模块路径 `github.com/Liao-Jun-Tao/couple-daily`。
- 客户端：**Flutter**，目录留给 `apps/mobile/`（本后端变更不创建该目录）。
- 契约在 `docs/`。`/v1` 业务 handler 仍是脚手架（`501 NOT_IMPLEMENTED`）；`GET /healthz` 已可用。

## 范围

- P0：邮箱密码注册登录、刷新、邀请配对、动态 CRUD、可选定位、赞 / 评、餐厅打卡、纪念日 + device token、媒体预签名
- P1 预留：餐厅收藏；定位 `city` / `hidden`；`remind_days_before` 可设 0–30（默认 0）
- 不做：手机号验证码、语音 / 视频动态、心愿单、记账、外部 POI 库

## 已写死参数

| 项 | 值 |
|----|-----|
| 登录 | 邮箱 + 密码。`POST /auth/register`、`POST /auth/login`、`POST /auth/refresh`、`GET /auth/me`。密码 8–128，哈希算法 argon2id。无手机号字段 |
| 邀请码 | 6 位 `[A-Z0-9]`，TTL 24h，一码一用 |
| space | **发起邀请时建**（`member_count=1`）；接受时加第二人；满员 = 2；第三人拒绝 |
| 删除权限 | **仅作者可删**（帖 / 评 / 打卡 / 纪念日 / 收藏）；赞仅本人可取消 |
| 多图 | ≤9；单张 ≤10MB；jpeg / png / heic / webp |
| 发帖内容 | 仅图或仅文均可；**禁止双空** |
| 提醒 | 默认纪念日**当天本地 09:00** 推一次（`Asia/Shanghai`，`remind_days_before=0`） |
| 定位 | lat/lng + 展示名 / POI；默认 **exact** |
| 餐厅打卡 | 搜索无结果可**手填店名**（`name` 必填，`place_id` 可空） |
| 推送 | iOS **APNs** + Android **FCM**。API 只登记 token；未开通知不导致登记失败 |
| ID / 时间 | UUID；RFC3339 UTC |

## 布局

```
docs/                         契约
backend/                      Go API、migrations、docker-compose
apps/mobile/                  Flutter（前端 PR，本变更不占用）
```

## 文件

- `docs/data-model.md`
- `docs/api/openapi.yaml`
- `docs/self-test.md`
- `docs/tech-stack-recommendation.md`
- `docs/CHANGELOG-draft.md`（本页）

## 下一步

按已锁契约实现 `/v1` handler（注册登录、配对、发帖、媒体预签名、打卡、纪念日、设备登记），替换当前的 `501`。
