import 'package:flutter/widgets.dart';

import 'package:couple_daily/core/session/app_session.dart';

class SessionScope extends InheritedNotifier<AppSession> {
  const SessionScope({
    required AppSession session,
    required super.child,
    super.key,
  }) : super(notifier: session);

  static AppSession of(BuildContext context) {
    final scope = context.dependOnInheritedWidgetOfExactType<SessionScope>();
    assert(scope != null, 'SessionScope is missing above this widget');
    return scope!.notifier!;
  }

  /// Does not subscribe. Use this from callbacks and initState timers.
  static AppSession read(BuildContext context) {
    final scope = context.findAncestorWidgetOfExactType<SessionScope>();
    assert(scope != null, 'SessionScope is missing above this widget');
    return scope!.notifier!;
  }
}
