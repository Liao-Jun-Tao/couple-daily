import 'package:flutter/material.dart';

import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/notifications/notification_gateway.dart';
import 'package:couple_daily/core/notifications/push_platform.dart';
import 'package:couple_daily/core/session/session_scope.dart';

class MePage extends StatefulWidget {
  const MePage({super.key});

  @override
  State<MePage> createState() => _MePageState();
}

class _MePageState extends State<MePage> {
  String? _notice;
  bool _requesting = false;

  Future<void> _requestNotifications() async {
    setState(() => _requesting = true);
    final session = SessionScope.read(context);
    final result = await session.notifications.request();
    if (!mounted) return;
    if (result == NotificationPermissionResult.granted) {
      final userId = session.userId;
      if (userId == null) {
        setState(() => _requesting = false);
        return;
      }
      final platform = currentPushPlatform();
      try {
        await session.registerDevice(
          platform: platform,
          token: 'stub-$platform-$userId',
        );
      } on Object catch (error) {
        if (!mounted) return;
        setState(() {
          _requesting = false;
          _notice = '权限已同意，但设备没有登记成功。应用会继续使用。${messageOf(error)}';
        });
        return;
      }
      if (!mounted) return;
      setState(() {
        _requesting = false;
        _notice = '已同意。已登记 ${pushChannelName(platform)} stub token，尚未连接真实推送。';
      });
      return;
    }
    setState(() {
      _requesting = false;
      _notice = result == NotificationPermissionResult.denied
          ? '通知权限被拒绝。应用会继续使用，不会因此崩溃。'
          : '当前环境没有系统通知权限接口。应用会继续使用。';
    });
  }

  @override
  Widget build(BuildContext context) {
    final session = SessionScope.of(context);
    final user = session.user;
    final space = session.space;
    final devices = user == null ? const [] : session.devices();
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('我的')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
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
