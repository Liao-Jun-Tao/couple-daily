import 'dart:io';
import 'dart:ui' as ui;

import 'package:couple_daily/app.dart';
import 'package:couple_daily/core/mock/mock_backend.dart';
import 'package:couple_daily/core/notifications/notification_gateway.dart';
import 'package:couple_daily/core/session/app_session.dart';
import 'package:couple_daily/features/auth_pair/splash_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

/// Widget renders for docs/demo. Not an Android emulator capture.
/// Chinese glyphs come from Noto Sans SC loaded over Roboto for the test font.
final bool _cjkFontReady = File('/tmp/NotoSansSC-Regular.otf').existsSync();

void main() {
  const shotDir = '/workspace/docs/demo';

  setUpAll(() async {
    if (!_cjkFontReady) return;
    final bytes = File('/tmp/NotoSansSC-Regular.otf').readAsBytesSync();
    final data = ByteData.view(
      bytes.buffer,
      bytes.offsetInBytes,
      bytes.lengthInBytes,
    );
    final loader = FontLoader('Roboto')..addFont(Future<ByteData>.value(data));
    await loader.load();
  });

  Future<void> phone(WidgetTester tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
  }

  testWidgets('auth and invite', (tester) async {
    await phone(tester);
    await tester.pumpWidget(
      RepaintBoundary(
        key: const Key('shot'),
        child: CoupleDailyApp(
          key: const Key('auth-app'),
          session: AppSession(useMock: true),
        ),
      ),
    );
    await tester.pump();
    await tester.pump(splashHold);
    await tester.pump(const Duration(milliseconds: 500));
    await _save(tester, '$shotDir/01-auth-widget-not-emulator.png');

    await tester.tap(find.text('注册'));
    await tester.pump(const Duration(milliseconds: 300));
    await tester.enterText(find.byKey(const Key('auth-name')), '甲');
    await tester.enterText(
      find.byKey(const Key('auth-email')),
      'a@example.com',
    );
    await tester.enterText(find.byKey(const Key('auth-password')), 'password1');
    await tester.tap(find.byKey(const Key('auth-submit')));
    await tester.pump(const Duration(milliseconds: 500));
    await _save(tester, '$shotDir/02-invite-shell-widget-not-emulator.png');

    await tester.tap(find.text('生成邀请码'));
    await tester.pump(const Duration(milliseconds: 300));
    await _save(tester, '$shotDir/03-invite-code-widget-not-emulator.png');
  }, skip: !_cjkFontReady);

  testWidgets('timeline and compose', (tester) async {
    await phone(tester);
    await tester.pumpWidget(
      RepaintBoundary(
        key: const Key('shot'),
        child: CoupleDailyApp(
          key: const Key('timeline-app'),
          session: _pairedSession(),
        ),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    await _save(tester, '$shotDir/04-timeline-empty-widget-not-emulator.png');

    await tester.tap(find.text('写动态'));
    await tester.pump(const Duration(milliseconds: 500));
    await _save(tester, '$shotDir/05-compose-empty-widget-not-emulator.png');
  }, skip: !_cjkFontReady);
}

Future<void> _save(WidgetTester tester, String path) async {
  await tester.pump();
  final boundary = tester.renderObject<RenderRepaintBoundary>(
    find.byKey(const Key('shot')),
  );
  await tester.runAsync(() async {
    final image = await boundary.toImage(pixelRatio: 2);
    final bytes = await image.toByteData(format: ui.ImageByteFormat.png);
    final file = File(path);
    await file.parent.create(recursive: true);
    await file.writeAsBytes(bytes!.buffer.asUint8List());
  });
}

AppSession _pairedSession() {
  final session = AppSession(
    backend: MockBackend(),
    notifications: NotificationGateway(client: _DenyClient()),
  );
  session.markReady();
  final author = session.backend.register(
    email: 'a@example.com',
    password: 'password1',
    displayName: '甲',
  );
  final other = session.backend.register(
    email: 'b@example.com',
    password: 'password1',
    displayName: '乙',
  );
  final ticket = session.backend.createInvite(author.user.id);
  session.backend.acceptInvite(userId: other.user.id, code: ticket.inviteCode);
  session.login(email: 'a@example.com', password: 'password1');
  return session;
}

class _DenyClient implements NotificationPermissionClient {
  @override
  Future<NotificationPermissionResult> request() async {
    return NotificationPermissionResult.denied;
  }
}
