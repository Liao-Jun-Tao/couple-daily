import 'package:permission_handler/permission_handler.dart';

enum NotificationPermissionResult {
  granted,
  denied,
  permanentlyDenied,
  unavailable,
}

/// Asks the OS for notification permission. Denial and missing plugins
/// resolve to a result; they do not throw.
abstract class NotificationPermissionClient {
  /// Shows the system prompt when the OS still allows one.
  Future<NotificationPermissionResult> request();

  /// Reads the current status without prompting. Used when opening 我的.
  Future<NotificationPermissionResult> status();
}

class PluginNotificationPermissionClient
    implements NotificationPermissionClient {
  @override
  Future<NotificationPermissionResult> request() async {
    final status = await Permission.notification.request();
    return notificationResultFromStatus(status);
  }

  @override
  Future<NotificationPermissionResult> status() async {
    final status = await Permission.notification.status;
    return notificationResultFromStatus(status);
  }
}

/// Maps an OS status onto the app result.
///
/// [PermissionStatus.restricted] is treated as permanently denied: the OS
/// will not show the prompt, so the in-app tip points at system settings.
NotificationPermissionResult notificationResultFromStatus(
  PermissionStatus status,
) {
  if (status.isGranted || status.isLimited || status.isProvisional) {
    return NotificationPermissionResult.granted;
  }
  if (status.isPermanentlyDenied || status.isRestricted) {
    return NotificationPermissionResult.permanentlyDenied;
  }
  return NotificationPermissionResult.denied;
}

class NotificationGateway {
  NotificationGateway({NotificationPermissionClient? client})
    : client = client ?? PluginNotificationPermissionClient();

  final NotificationPermissionClient client;
  NotificationPermissionResult? lastResult;

  Future<NotificationPermissionResult> request() {
    return _read(client.request);
  }

  Future<NotificationPermissionResult> status() {
    return _read(client.status);
  }

  Future<NotificationPermissionResult> _read(
    Future<NotificationPermissionResult> Function() read,
  ) async {
    try {
      final result = await read();
      lastResult = result;
      return result;
    } on Object {
      lastResult = NotificationPermissionResult.unavailable;
      return NotificationPermissionResult.unavailable;
    }
  }
}
