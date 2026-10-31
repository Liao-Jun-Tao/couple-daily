# Mock 壳演示

本轮演示优先 **Android**。工程仍是 Flutter 双端。不接真实 API。云端没有 Android 模拟器，下面的图是 widget 渲染，**不是模拟器截图**。

## 截图（非 Android 模拟器）

| 文件 | 画面 |
|------|------|
| `01-auth-widget-not-emulator.png` | 邮箱 + 密码登录 |
| `02-invite-shell-widget-not-emulator.png` | 邀请发送 / 接受 |
| `03-invite-code-widget-not-emulator.png` | 已生成邀请码，空间 1 / 2 |
| `04-timeline-empty-widget-not-emulator.png` | 时间线空态，底栏，写动态入口 |
| `05-compose-empty-widget-not-emulator.png` | 发帖页；文字和图片都空时发布按钮不可用 |

本地重画：把 `NotoSansSC-Regular.otf` 放到 `/tmp/NotoSansSC-Regular.otf`，然后

```bash
cd apps/mobile
flutter test test/demo_shots_test.dart
```

没有这个字体文件时，该测试会跳过，避免 CI 失败。

## Debug APK

APK 不进 git。本机构建：

```bash
cd apps/mobile
flutter pub get
flutter build apk --debug --split-per-abi
```

安装（按设备 ABI 选一个）：

```bash
adb install -r build/app/outputs/flutter-apk/app-arm64-v8a-debug.apk
adb install -r build/app/outputs/flutter-apk/app-x86_64-debug.apk
```

手机和 arm64 模拟器用 `arm64-v8a`。常见桌面模拟器用 `x86_64`。

云端 artifacts（已打好，不必重打）：

- `couple-daily-debug-arm64-v8a.apk`
- `couple-daily-debug-x86_64.apk`

```bash
adb install -r couple-daily-debug-arm64-v8a.apk
adb install -r couple-daily-debug-x86_64.apk
```
