# apps/mobile

情侣日常 Flutter 客户端脚手架。数据在进程内 Mock，不调用 `backend/`。

## 运行（本轮演示优先 Android）

工程仍是 Flutter 双端。本轮可安装包和演示以 **Android** 为准。iOS 模拟器后置，本轮不做 iOS 装包。

```bash
cd apps/mobile
flutter pub get
```

打开 Android 模拟器后：

```bash
flutter devices
flutter run
```

有多台设备时用模拟器 id：

```bash
flutter run -d <emulator-id>
```

Debug APK（不提交进 git）：

```bash
cd apps/mobile
flutter build apk --debug
adb install -r build/app/outputs/flutter-apk/app-debug.apk
```

`flutter analyze` 和 `flutter test` 用来检查 Mock 规则。本脚手架在 Flutter stable 3.47 / Dart 3.13 上分析通过。

### iOS（后置）

Xcode 打开 Simulator 后，同样在 `apps/mobile` 里 `flutter pub get`，再 `flutter run`。多设备时先 `flutter devices`，然后 `flutter run -d <iphone-sim-id>`。本轮不验收。

## 结构

```
lib/
  core/            会话、路由、Mock、通知 stub
  features/
    auth_pair/     Splash、邮箱密码、邀请、配对成功
    timeline/      时间线、帖子卡片、评论
    compose/       发帖、位置选择
    restaurant/    搜索、手填打卡、收藏路由占位
    anniversary/   列表与表单
    me/            我的、通知权限
  shared/widgets/
```

底部导航：时间线 / 餐厅 / 纪念日 / 我的。

## Mock 规则

与已锁定的产品规则一致。实现在 `lib/core/mock/mock_backend.dart`，界面只走 `AppSession`。

| 项 | 行为 |
|----|------|
| 登录 | **邮箱 + 密码**。密码至少 8 位。没有手机号，没有短信验证码。 |
| 邀请码 | 6 位 `[A-Z0-9]`，创建后 24 小时，一码一用。大小写不敏感。 |
| 空间 | 发起邀请时创建，`member_count = 1`。接受后变为 2。第三人 `SPACE_FULL`。已在空间里的人不能再接受。过期后可在同一空间重发。 |
| 配对成功 | 接受成功后短暂进入 PairingDone，然后进入时间线。 |
| 发帖 | 正文去掉空白后为空，且没有图片，则拒绝。仅文字或仅图片都可以。最多 9 张，单张不超过 10MB，类型为 jpeg / png / heic / webp。 |
| 位置 | 可选。选择器默认 **完整**（`exact`）。也可改为城市或隐藏。 |
| 删除 | 动态、评论、打卡、纪念日的删除入口只出现在作者自己的内容上。Mock 也会拒绝别人删除。赞只取消自己的。 |
| 餐厅 | 本地名单搜索。没有结果时手填店名打卡，`place_id` 可空。收藏只有 `/favorites` 占位（P1）。 |
| 纪念日 | 默认当天本地 **09:00**、`Asia/Shanghai`、不提前。提醒写入 `ReminderStub`，不连接推送。 |
| 通知 | 我的页请求系统通知权限。iOS 记为 APNs stub，Android 记为 FCM stub。拒绝或插件不可用时给出提示，不抛错。 |

同一进程里可以注册两个邮箱：一方生成邀请码并退出，另一方注册后接受。杀掉应用后 Mock 清空。
