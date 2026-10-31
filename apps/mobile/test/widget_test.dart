import 'package:couple_daily/app.dart';
import 'package:couple_daily/core/mock/mock_backend.dart';
import 'package:couple_daily/core/models/models.dart';
import 'package:couple_daily/core/notifications/notification_gateway.dart';
import 'package:couple_daily/core/session/app_session.dart';
import 'package:couple_daily/core/session/session_scope.dart';
import 'package:couple_daily/features/auth_pair/splash_page.dart';
import 'package:couple_daily/features/compose/location_picker_page.dart';
import 'package:couple_daily/features/timeline/comment_sheet.dart';
import 'package:couple_daily/features/timeline/timeline_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('auth is email and password', (tester) async {
    await tester.pumpWidget(CoupleDailyApp(session: AppSession(useMock: true)));
    await tester.pump(splashHold);
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('auth-email')), findsOneWidget);
    expect(find.byKey(const Key('auth-password')), findsOneWidget);
    expect(find.text('手机号'), findsNothing);
    expect(find.text('验证码'), findsNothing);

    await tester.tap(find.text('注册'));
    await tester.pump();
    expect(find.byKey(const Key('auth-name')), findsOneWidget);
    await tester.enterText(find.byKey(const Key('auth-name')), '甲');
    await tester.enterText(
      find.byKey(const Key('auth-email')),
      'a@example.com',
    );
    await tester.enterText(find.byKey(const Key('auth-password')), 'password1');
    await tester.tap(find.byKey(const Key('auth-submit')));
    await tester.pumpAndSettle();

    expect(find.text('发起邀请'), findsOneWidget);
    expect(find.text('接受邀请'), findsWidgets);
    expect(find.text('手机号'), findsNothing);
    expect(find.text('验证码'), findsNothing);
  });

  testWidgets('location picker defaults to 完整', (tester) async {
    await tester.pumpWidget(const MaterialApp(home: LocationPickerPage()));
    final button = tester.widget<SegmentedButton<LocationVisibility>>(
      find.byType(SegmentedButton<LocationVisibility>),
    );
    expect(button.selected, {LocationVisibility.exact});
    expect(find.text('完整'), findsOneWidget);
  });

  testWidgets('delete controls are author only', (tester) async {
    final session = AppSession(useMock: true);
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
    session.backend.acceptInvite(
      userId: other.user.id,
      code: ticket.inviteCode,
    );
    final post = session.backend.createPost(
      userId: author.user.id,
      body: '只有甲能删',
      media: const [],
    );
    session.backend.addComment(
      userId: author.user.id,
      postId: post.id,
      body: '作者评论',
    );
    session.login(email: 'b@example.com', password: 'password1');

    await tester.pumpWidget(
      SessionScope(
        session: session,
        child: const MaterialApp(home: TimelinePage()),
      ),
    );
    expect(find.text('只有甲能删'), findsOneWidget);
    expect(find.byTooltip('删除动态'), findsNothing);

    session.logout();
    session.login(email: 'a@example.com', password: 'password1');
    await tester.pump();
    expect(find.byTooltip('删除动态'), findsOneWidget);

    await tester.pumpWidget(
      SessionScope(
        session: session,
        child: MaterialApp(
          home: Scaffold(body: CommentSheet(postId: post.id)),
        ),
      ),
    );
    expect(find.byTooltip('删除评论'), findsOneWidget);

    session.logout();
    session.login(email: 'b@example.com', password: 'password1');
    await tester.pump();
    expect(find.byTooltip('删除评论'), findsNothing);
  });

  testWidgets('shell, manual check-in, and denied notifications', (
    tester,
  ) async {
    final session = _pairedSession(client: _DenyClient());
    await tester.pumpWidget(CoupleDailyApp(session: session));
    await tester.pumpAndSettle();

    expect(find.text('时间线'), findsWidgets);
    await tester.tap(find.byTooltip('写动态'));
    await tester.pumpAndSettle();
    FilledButton submit() =>
        tester.widget<FilledButton>(find.byKey(const Key('compose-submit')));
    expect(submit().onPressed, isNull);
    await tester.enterText(find.byKey(const Key('compose-body')), '今天');
    await tester.pump();
    expect(submit().onPressed, isNotNull);
    await tester.tap(find.byType(BackButton));
    await tester.pumpAndSettle();

    await tester.tap(find.text('餐厅'));
    await tester.pumpAndSettle();
    expect(find.text('搜索餐厅'), findsOneWidget);
    await tester.enterText(
      find.byKey(const Key('restaurant-search')),
      '没有这家店xyz',
    );
    await tester.pump();
    expect(find.text('用手填店名打卡'), findsOneWidget);
    await tester.tap(find.byKey(const Key('manual-checkin')));
    await tester.pump();
    expect(find.text('没有这家店xyz'), findsWidgets);

    await tester.tap(find.byTooltip('收藏'));
    await tester.pumpAndSettle();
    expect(find.textContaining('P1'), findsOneWidget);
    await tester.tap(find.byType(BackButton));
    await tester.pumpAndSettle();

    await tester.tap(find.text('纪念日'));
    await tester.pumpAndSettle();
    expect(find.text('还没有纪念日'), findsOneWidget);

    await tester.tap(find.text('我的'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('notification-request')));
    await tester.pumpAndSettle();
    expect(find.textContaining('通知权限被拒绝'), findsOneWidget);
    expect(find.text('我的'), findsWidgets);
  });
}

class _DenyClient implements NotificationPermissionClient {
  @override
  Future<NotificationPermissionResult> request() async {
    return NotificationPermissionResult.denied;
  }
}

AppSession _pairedSession({required NotificationPermissionClient client}) {
  final session = AppSession(
    backend: MockBackend(),
    notifications: NotificationGateway(client: client),
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
