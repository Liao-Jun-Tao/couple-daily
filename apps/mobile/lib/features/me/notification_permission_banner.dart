import 'package:flutter/material.dart';

import 'package:couple_daily/core/notifications/notification_gateway.dart';
import 'package:couple_daily/core/notifications/notification_permission_copy.dart';

/// Dismissible in-app banner. It is not a dialog, so the rest of 我的
/// stays usable.
class NotificationPermissionBanner extends StatelessWidget {
  const NotificationPermissionBanner({
    required this.result,
    required this.onDismiss,
    super.key,
  });

  final NotificationPermissionResult result;
  final VoidCallback onDismiss;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Material(
      key: const Key('notification-permission-tip'),
      color: scheme.errorContainer,
      borderRadius: BorderRadius.circular(12),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 12, 4, 12),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Icon(
                Icons.notifications_off_outlined,
                color: scheme.onErrorContainer,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Padding(
                padding: const EdgeInsets.only(top: 10),
                child: Text(
                  notificationPermissionTipText(result),
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: scheme.onErrorContainer,
                  ),
                ),
              ),
            ),
            IconButton(
              key: const Key('notification-permission-tip-dismiss'),
              tooltip: '关闭通知权限提示',
              onPressed: onDismiss,
              color: scheme.onErrorContainer,
              icon: const Icon(Icons.close),
            ),
          ],
        ),
      ),
    );
  }
}
