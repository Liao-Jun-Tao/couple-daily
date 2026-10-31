import 'dart:async';

import 'package:flutter/material.dart';

import 'package:couple_daily/core/session/session_scope.dart';

const Duration splashHold = Duration(milliseconds: 600);

class SplashPage extends StatefulWidget {
  const SplashPage({super.key});

  @override
  State<SplashPage> createState() => _SplashPageState();
}

class _SplashPageState extends State<SplashPage> {
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _timer = Timer(splashHold, () {
      if (!mounted) return;
      unawaited(SessionScope.read(context).bootstrap());
    });
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.favorite, size: 56, color: theme.colorScheme.primary),
            const SizedBox(height: 16),
            Text('情侣日常', style: theme.textTheme.headlineMedium),
            const SizedBox(height: 8),
            Text(
              '两个人的时间线',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
