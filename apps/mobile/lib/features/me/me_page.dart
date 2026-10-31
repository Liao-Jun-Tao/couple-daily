import 'package:flutter/material.dart';

import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/notifications/notification_gateway.dart';
import 'package:couple_daily/core/notifications/notification_permission_copy.dart';
import 'package:couple_daily/core/notifications/push_platform.dart';
import 'package:couple_daily/core/session/session_scope.dart';
import 'package:couple_daily/features/me/notification_permission_banner.dart';

class MePage extends StatefulWidget {
  const MePage({super.key});

  @override
  State<MePage> createState() => _MePageState();
}

class _MePageState extends State<MePage> with WidgetsBindingObserver {
  String? _notice;
  bool _requesting = false;
  int _generation = 0;
  NotificationPermissionResult? _permission;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _loadPermissionStatus();
    });
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _loadPermissionStatus();
    }
  }

  /// Settings entry: read the OS status without showing the system prompt.
  Future<void> _loadPermissionStatus() async {
    if (!mounted || _requesting) return;
    final generation = _generation;
    final session = SessionScope.read(context);
    final result = await session.notifications.status();
    if (!mounted || generation != _generation || _requesting) return;
    setState(() => _permission = result);
  }

  Future<void> _requestNotifications() async {
    final generation = ++_generation;
    setState(() => _requesting = true);
    final session = SessionScope.read(context);
    final result = await session.notifications.request();
    if (!mounted || generation != _generation) return;
    if (result == NotificationPermissionResult.granted) {
      final userId = session.userId;
      if (userId == null) {
        setState(() {
          _requesting = false;
          _permission = result;
        });
        return;
      }
      final platform = currentPushPlatform();
      try {
        await session.registerDevice(
          platform: platform,
          token: 'stub-$platform-$userId',
        );
      } on Object catch (error) {
        if (!mounted || generation != _generation) return;
        setState(() {
          _requesting = false;
          _permission = result;
          _notice = '权限已同意，但设备没有登记成功。应用会继续使用。${messageOf(error)}';
        });
        return;
      }
      if (!mounted || generation != _generation) return;
      setState(() {
        _requesting = false;
        _permission = result;
        _notice = '已同意。已登记 ${pushChannelName(platform)} stub token，尚未连接真实推送。';
      });
      return;
    }
    setState(() {
      _requesting = false;
      _permission = result;
      _notice = result == NotificationPermissionResult.unavailable
          ? '当前环境没有系统通知权限接口。应用会继续使用。'
          : null;
    });
  }

  @override
  Widget build(BuildContext context) {
    final session = SessionScope.of(context);
    final user = session.user;
    final space = session.space;
    final devices = user == null ? const [] : session.devices();
    final theme = Theme.of(context);
    final permission = _permission;
    final showTip =
        permission != null &&
        notificationPermissionIsDenied(permission) &&
        !session.notificationPermissionTipDismissed;
    return Scaffold(
      appBar: AppBar(title: const Text('我的')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          if (showTip) ...[
            NotificationPermissionBanner(
              result: permission,
              onDismiss: () {
                session.dismissNotificationPermissionTip();
                setState(() {});
              },
            ),
            const SizedBox(height: 16),
          ],
          Text(
            user?.displayName ?? '未登录',
            style: theme.textTheme.headlineSmall,
          ),
          const SizedBox(height: 4),
          Text(user?.email ?? ''),
          const SizedBox(height: 16),
          Text('空间', style: theme.textTheme.titleMedium),
          const SizedBox(height: 4),
          Text(space == null ? '尚未配对' : '已配对 · ${space.memberCount} 人'),
          for (final member in space?.members ?? const [])
            Text(member.displayName),
          const SizedBox(height: 24),
          Text('通知', style: theme.textTheme.titleMedium),
          const SizedBox(height: 4),
          const Text(
            'iOS 走 APNs，Android 走 FCM。这里只登记一个 stub token，'
            '拒绝系统权限也不会让应用退出。',
          ),
          const SizedBox(height: 8),
          FilledButton(
            key: const Key('notification-request'),
            onPressed: _requesting ? null : _requestNotifications,
            child: Text(_requesting ? '请求中…' : '请求通知权限'),
          ),
          if (_notice != null) ...[const SizedBox(height: 8), Text(_notice!)],
          for (final device in devices)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(
                '${pushChannelName(device.platform)} · ${device.token}',
                style: theme.textTheme.bodySmall,
              ),
            ),
          const SizedBox(height: 24),
          OutlinedButton(
            onPressed: () async {
              await session.logout();
            },
            child: const Text('退出登录'),
          ),
        ],
      ),
    );
  }
}
