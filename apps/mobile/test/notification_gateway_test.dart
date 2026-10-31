import 'package:couple_daily/core/notifications/notification_gateway.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('denied permission resolves and does not throw', () async {
    final gateway = NotificationGateway(client: _Fixed(NotificationPermissionResult.denied));
    expect(await gateway.request(), NotificationPermissionResult.denied);
    expect(gateway.lastResult, NotificationPermissionResult.denied);
  });

  test('a missing plugin becomes unavailable instead of crashing', () async {
    final gateway = NotificationGateway(client: _Throwing());
    expect(await gateway.request(), NotificationPermissionResult.unavailable);
  });
}

class _Fixed implements NotificationPermissionClient {
  _Fixed(this.result);

  final NotificationPermissionResult result;

  @override
  Future<NotificationPermissionResult> request() async => result;
}

class _Throwing implements NotificationPermissionClient {
  @override
  Future<NotificationPermissionResult> request() async {
    throw StateError('plugin missing');
  }
}
