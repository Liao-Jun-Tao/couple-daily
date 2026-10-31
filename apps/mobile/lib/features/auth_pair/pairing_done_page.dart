import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:couple_daily/core/session/session_scope.dart';

const Duration pairingDoneHold = Duration(milliseconds: 1100);

/// Shown once after a successful accept, then the shell takes over.
class PairingDonePage extends StatefulWidget {
  const PairingDonePage({super.key});

  @override
  State<PairingDonePage> createState() => _PairingDonePageState();
}

class _PairingDonePageState extends State<PairingDonePage> {
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _timer = Timer(pairingDoneHold, () {
      if (!mounted) return;
      SessionScope.read(context).clearPairingDoneFlag();
      context.go('/timeline');
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
            Icon(Icons.favorite, size: 64, color: theme.colorScheme.primary),
            const SizedBox(height: 16),
            Text('配对成功', style: theme.textTheme.headlineMedium),
            const SizedBox(height: 8),
            Text(
              '你们的空间现在有两个人',
              style: theme.textTheme.bodyLarge?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
