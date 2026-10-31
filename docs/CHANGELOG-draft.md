# couple-daily 契约草案摘要（0.1.0-draft）

## 范围（本轮交付）

- 项目名：**couple-daily**（仓：https://github.com/Liao-Jun-Tao/couple-daily ；契约在 `docs/`）
- P0：邮箱密码 Auth + 邀请配对、动态 CRUD、可选定位、赞/评、餐厅打卡、纪念日 + device token、媒体预签名
- P1 预留：餐厅收藏；定位 `city|hidden`；**手机号验证码登录（首版不做）**
- 不做：语音/视频动态、心愿单、记账、外部 POI 库

## Auth（已确认，已写死）

| 项 | 值 |
|----|-----|
| MVP 登录 | **邮箱 + 密码**（`/auth/register` + `/auth/login` + `/auth/refresh` + `/auth/me`） |
| 配对 | 邀请码配对（与登录方式独立） |
| 手机号 OTP | **不上首版**（P1 备注一行即可；契约不含短信发码路径） |

## 已写死参数（骏涛已确认）

| 项 | 值 | 状态 |
|----|-----|------|
| 登录 | 邮箱 + 密码 | **已锁** |
| 邀请码 | 6 位字母数字，TTL 24h，一码一用，接受后失效 | 已锁 |
| space | **发起邀请时建**（member_count=1）；接受时加第二人；满员=2；第三人拒绝 | 已锁 |
| 删除权限 | **仅作者可删**（帖/评/打卡/纪念日/收藏）；赞仅本人可取消 | 已锁 |
| 多图 | ≤9；单张 ≤10MB；jpg/png/heic/webp | 已锁 |
| 发帖内容 | 仅图或仅文均可；**禁止双空**（实现层硬校验） | 已锁 |
| 提醒 | 当天本地 **09:00** 推一次；首版无提前提醒档位 | 已锁 |
| 定位 | lat/lng + 展示名/POI；默认 **exact（完整）**，客户端可改 | 已锁 |
| 餐厅打卡 | 搜索无结果可**手填店名**（name 必填，place_id 可空） | 已锁 |
| 推送 | iOS **APNs** + Android **FCM**；未开通知不崩溃，进 App 可 in-app 提示（API 只登记 token） | 已锁 |
| ID / 时间 | UUID；RFC3339 UTC | 已锁 |

## 服务端栈推荐

**Go + chi + PostgreSQL + Redis + S3 兼容 + APNs/FCM**；Auth = 邮箱密码 + JWT（argon2id/bcrypt）。短信 OTP 为 P1，首版不做。

## 文件

- `docs/data-model.md`
- `docs/api/openapi.yaml`
- `docs/self-test.md`
- `docs/tech-stack-recommendation.md`
- `docs/CHANGELOG-draft.md`（本页）

## 下一步

1. 脚手架已落在 `backend/`（Go + chi）。`apps/mobile/` 留给 Flutter，本变更不创建该目录。
2. 实现 P0：Auth → Pairing → Posts/Media → Likes/Comments → Checkins → Anniversaries/Devices。`/v1` handler 目前返回 `501`。  
