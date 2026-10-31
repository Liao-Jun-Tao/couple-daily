import 'package:couple_daily/core/notifications/notification_gateway.dart';
import 'package:couple_daily/core/notifications/notification_permission_copy.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:permission_handler/permission_handler.dart';

void main() {
  test('denied permission resolves and does not throw', () async {
    final gateway = NotificationGateway(
      client: _Fixed(NotificationPermissionResult.denied),
    );
    expect(await gateway.request(), NotificationPermissionResult.denied);
    expect(gateway.lastResult, NotificationPermissionResult.denied);
  });

  test('permanently denied status is kept and does not throw', () async {
    final gateway = NotificationGateway(
      client: _Fixed(NotificationPermissionResult.permanentlyDenied),
    );
    expect(
      await gateway.status(),
      NotificationPermissionResult.permanentlyDenied,
    );
    expect(gateway.lastResult, NotificationPermissionResult.permanentlyDenied);
    expect(
      await gateway.request(),
      NotificationPermissionResult.permanentlyDenied,
    );
  });

  test('a missing plugin becomes unavailable instead of crashing', () async {
    final gateway = NotificationGateway(client: _Throwing());
    expect(await gateway.request(), NotificationPermissionResult.unavailable);
    expect(await gateway.status(), NotificationPermissionResult.unavailable);
  });

  test('OS statuses map denied and permanently denied separately', () {
    expect(
      notificationResultFromStatus(PermissionStatus.granted),
      NotificationPermissionResult.granted,
    );
    expect(
      notificationResultFromStatus(PermissionStatus.limited),
      NotificationPermissionResult.granted,
    );
    expect(
      notificationResultFromStatus(PermissionStatus.provisional),
      NotificationPermissionResult.granted,
    );
    expect(
      notificationResultFromStatus(PermissionStatus.denied),
      NotificationPermissionResult.denied,
    );
    expect(
      notificationResultFromStatus(PermissionStatus.permanentlyDenied),
      NotificationPermissionResult.permanentlyDenied,
    );
    expect(
      notificationResultFromStatus(PermissionStatus.restricted),
      NotificationPermissionResult.permanentlyDenied,
    );
  });

  test(
    'denied tips name anniversary reminders and can be dismissed in copy',
    () {
      final denied = notificationPermissionTipText(
        NotificationPermissionResult.denied,
      );
      final permanent = notificationPermissionTipText(
        NotificationPermissionResult.permanentlyDenied,
      );
      expect(denied, contains(notificationPermissionTipLead));
      expect(denied, contains('可以关闭这条提示'));
      expect(permanent, contains('系统设置'));
      expect(
        notificationPermissionIsDenied(NotificationPermissionResult.denied),
        isTrue,
      );
      expect(
        notificationPermissionIsDenied(
          NotificationPermissionResult.permanentlyDenied,
        ),
        isTrue,
      );
      expect(
        notificationPermissionIsDenied(NotificationPermissionResult.granted),
        isFalse,
      );
    },
  );
}

class _Fixed implements NotificationPermissionClient {
  _Fixed(this.result);

  final NotificationPermissionResult result;

  @override
  Future<NotificationPermissionResult> request() async => result;

  @override
  Future<NotificationPermissionResult> status() async => result;
}

class _Throwing implements NotificationPermissionClient {
  @override
  Future<NotificationPermissionResult> request() async {
    throw StateError('plugin missing');
  }

  @override
  Future<NotificationPermissionResult> status() async {
    throw StateError('plugin missing');
  }
}
