import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';

import 'package:couple_daily/core/router/app_router.dart';
import 'package:couple_daily/core/session/app_session.dart';
import 'package:couple_daily/core/session/session_scope.dart';
import 'package:couple_daily/core/theme/app_theme.dart';

class CoupleDailyApp extends StatefulWidget {
  const CoupleDailyApp({super.key, this.session});

  final AppSession? session;

  @override
  State<CoupleDailyApp> createState() => _CoupleDailyAppState();
}

class _CoupleDailyAppState extends State<CoupleDailyApp> {
  late final AppSession _session = widget.session ?? AppSession();
  late final _router = buildRouter(_session);

  @override
  void dispose() {
    _router.dispose();
    if (widget.session == null) {
      _session.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return SessionScope(
      session: _session,
      child: MaterialApp.router(
        title: '情侣日常',
        locale: const Locale('zh', 'CN'),
        supportedLocales: const [Locale('zh', 'CN'), Locale('en')],
        localizationsDelegates: const [
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        theme: buildAppTheme(),
        routerConfig: _router,
      ),
    );
  }
}
