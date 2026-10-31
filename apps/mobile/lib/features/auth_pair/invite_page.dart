import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';

import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/format.dart';
import 'package:couple_daily/core/models/models.dart';
import 'package:couple_daily/core/session/session_scope.dart';

/// Invite send (creates a 1-member space) and invite accept (joins as member 2).
class InvitePage extends StatefulWidget {
  const InvitePage({super.key});

  @override
  State<InvitePage> createState() => _InvitePageState();
}

class _InvitePageState extends State<InvitePage> {
  final _code = TextEditingController();
  String? _error;

  @override
  void dispose() {
    _code.dispose();
    super.dispose();
  }

  bool _expired(DateTime? expiresAt) {
    if (expiresAt == null) return false;
    return !DateTime.now().toUtc().isBefore(expiresAt.toUtc());
  }

  void _createInvite() {
    try {
      SessionScope.read(context).createInvite();
      if (!mounted) return;
      setState(() => _error = null);
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = messageOf(error));
    }
  }

  Future<void> _accept() async {
    final session = SessionScope.read(context);
    try {
      session.acceptInvite(_code.text);
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = messageOf(error));
      return;
    }
    if (!mounted) return;
    context.go('/pairing-done');
  }

  Future<void> _copy(String code) async {
    await Clipboard.setData(ClipboardData(text: code));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('邀请码已复制')),
    );
  }

  @override
  Widget build(BuildContext context) {
    final session = SessionScope.of(context);
    final pairing = session.pairing;
    final pending = pairing.status == PairingStatus.pendingInvite;
    final code = pairing.inviteCode;
    final expired = _expired(pairing.inviteExpiresAt);
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('配对'),
        actions: [
          TextButton(
            onPressed: session.logout,
            child: const Text('退出登录'),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Text(
            '邀请码是 6 位字母或数字，24 小时内有效，只能使用一次。'
            '发起邀请会立刻建一个空间，此时只有你一个人。',
            style: theme.textTheme.bodyMedium,
          ),
          const SizedBox(height: 8),
          Text(
            '同一台手机演示：生成邀请码后退出，换一个邮箱注册再接受。数据只留在这次运行里。',
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 24),
          if (pending && code != null) ...[
            Text('你的邀请码', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            SelectableText(
              code,
              style: theme.textTheme.displaySmall?.copyWith(
                letterSpacing: 8,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 8),
            Text('空间人数 ${pairing.memberCount ?? 1} / 2'),
            if (pairing.inviteExpiresAt != null)
              Text(
                expired
                    ? '已过期'
                    : '有效期至 ${formatTimestamp(pairing.inviteExpiresAt!)}',
              ),
            const SizedBox(height: 12),
            OutlinedButton(
              onPressed: () => _copy(code),
              child: const Text('复制邀请码'),
            ),
            if (expired) ...[
              const SizedBox(height: 8),
              FilledButton(
                onPressed: _createInvite,
                child: const Text('重新生成'),
              ),
            ],
          ] else ...[
            Text('发起邀请', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            FilledButton(
              onPressed: _createInvite,
              child: const Text('生成邀请码'),
            ),
            const SizedBox(height: 28),
            Text('接受邀请', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            TextField(
              key: const Key('invite-code'),
              controller: _code,
              textCapitalization: TextCapitalization.characters,
              maxLength: 6,
              decoration: const InputDecoration(
                labelText: '6 位邀请码',
                counterText: '',
              ),
            ),
            const SizedBox(height: 12),
            FilledButton(
              key: const Key('invite-accept'),
              onPressed: _accept,
              child: const Text('接受邀请'),
            ),
          ],
          if (_error != null) ...[
            const SizedBox(height: 16),
            Text(_error!, style: TextStyle(color: theme.colorScheme.error)),
          ],
        ],
      ),
    );
  }
}
