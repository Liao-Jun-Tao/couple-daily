import 'package:permission_handler/permission_handler.dart';

enum NotificationPermissionResult { granted, denied, unavailable }

/// Asks the OS for notification permission. Denial and missing plugins
/// resolve to a result; they do not throw.
abstract class NotificationPermissionClient {
  Future<NotificationPermissionResult> request();
}

class PluginNotificationPermissionClient implements NotificationPermissionClient {
  @override
  Future<NotificationPermissionResult> request() async {
    final status = await Permission.notification.request();
    if (status.isGranted || status.isLimited || status.isProvisional) {
      return NotificationPermissionResult.granted;
    }
    return NotificationPermissionResult.denied;
  }
}

class NotificationGateway {
  NotificationGateway({NotificationPermissionClient? client})
    : client = client ?? PluginNotificationPermissionClient();

  final NotificationPermissionClient client;
  NotificationPermissionResult? lastResult;

  Future<NotificationPermissionResult> request() async {
    try {
      final result = await client.request();
      lastResult = result;
      return result;
    } on Object {
      lastResult = NotificationPermissionResult.unavailable;
      return NotificationPermissionResult.unavailable;
    }
  }
}
