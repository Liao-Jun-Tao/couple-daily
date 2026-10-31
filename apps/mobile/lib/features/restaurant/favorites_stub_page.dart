import 'package:flutter/material.dart';

/// P1 favorites are intentionally not implemented past this route.
class FavoritesStubPage extends StatelessWidget {
  const FavoritesStubPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('收藏')),
      body: const Center(
        child: Padding(
          padding: EdgeInsets.all(24),
          child: Text(
            '餐厅收藏是 P1，这条路由只是占位，还不能保存收藏。',
            textAlign: TextAlign.center,
          ),
        ),
      ),
    );
  }
}
