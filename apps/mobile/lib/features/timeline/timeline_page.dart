import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:couple_daily/core/session/session_scope.dart';
import 'package:couple_daily/features/timeline/post_card.dart';
import 'package:couple_daily/shared/widgets/empty_state.dart';

class TimelinePage extends StatelessWidget {
  const TimelinePage({super.key});

  @override
  Widget build(BuildContext context) {
    final session = SessionScope.of(context);
    final posts = session.posts();
    final viewerId = session.userId ?? '';
    return Scaffold(
      appBar: AppBar(title: const Text('时间线')),
      floatingActionButton: FloatingActionButton.extended(
        tooltip: '写动态',
        onPressed: () => context.push('/compose'),
        icon: const Icon(Icons.edit_outlined),
        label: const Text('写动态'),
      ),
      body: posts.isEmpty
          ? const EmptyState(
              title: '还没有动态',
              subtitle: '写一段话，或者发最多 9 张图片。文字和图片不能同时为空。',
            )
          : ListView.builder(
              padding: const EdgeInsets.only(bottom: 88),
              itemCount: posts.length,
              itemBuilder: (context, index) {
                final post = posts[index];
                return PostCard(
                  post: post,
                  viewerId: viewerId,
                  onLike: () => session.like(post.id),
                  onUnlike: () => session.unlike(post.id),
                  onDelete: () => session.deletePost(post.id),
                );
              },
            ),
    );
  }
}
