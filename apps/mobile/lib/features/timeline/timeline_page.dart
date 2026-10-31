import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:couple_daily/core/session/session_scope.dart';
import 'package:couple_daily/features/timeline/post_card.dart';
import 'package:couple_daily/shared/widgets/empty_state.dart';

class TimelinePage extends StatefulWidget {
  const TimelinePage({super.key});

  @override
  State<TimelinePage> createState() => _TimelinePageState();
}

class _TimelinePageState extends State<TimelinePage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      SessionScope.read(context).refreshPosts();
    });
  }

  @override
  Widget build(BuildContext context) {
    final session = SessionScope.of(context);
    final posts = session.posts();
    final viewerId = session.userId ?? '';
    final loadError = session.loadError;
    return Scaffold(
      appBar: AppBar(title: const Text('时间线')),
      floatingActionButton: FloatingActionButton.extended(
        tooltip: '写动态',
        onPressed: () => context.push('/compose'),
        icon: const Icon(Icons.edit_outlined),
        label: const Text('写动态'),
      ),
      body: RefreshIndicator(
        onRefresh: session.refreshPosts,
        child: posts.isEmpty
            ? ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                children: [
                  if (!session.postsReady)
                    const Padding(
                      padding: EdgeInsets.all(48),
                      child: Center(child: CircularProgressIndicator()),
                    )
                  else
                    EmptyState(
                      title: '还没有动态',
                      subtitle: loadError ?? '写一段话，或者发最多 9 张图片。文字和图片不能同时为空。',
                    ),
                ],
              )
            : ListView.builder(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.only(bottom: 88),
                itemCount: posts.length + (loadError == null ? 0 : 1),
                itemBuilder: (context, index) {
                  if (loadError != null && index == 0) {
                    return Padding(
                      padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
                      child: Text(
                        loadError,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.error,
                        ),
                      ),
                    );
                  }
                  final post = posts[loadError == null ? index : index - 1];
                  return PostCard(
                    post: post,
                    viewerId: viewerId,
                    onLike: () => session.like(post.id),
                    onUnlike: () => session.unlike(post.id),
                    onDelete: () => session.deletePost(post.id),
                  );
                },
              ),
      ),
    );
  }
}
