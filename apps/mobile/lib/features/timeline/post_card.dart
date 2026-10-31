import 'package:flutter/material.dart';

import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/format.dart';
import 'package:couple_daily/core/models/models.dart';
import 'package:couple_daily/features/timeline/comment_sheet.dart';
import 'package:couple_daily/shared/widgets/confirm_delete_dialog.dart';

class PostCard extends StatelessWidget {
  const PostCard({
    required this.post,
    required this.viewerId,
    required this.onLike,
    required this.onUnlike,
    required this.onDelete,
    super.key,
  });

  final Post post;
  final String viewerId;
  final Future<void> Function() onLike;
  final Future<void> Function() onUnlike;
  final Future<void> Function() onDelete;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final mine = post.author.id == viewerId;
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        post.author.displayName,
                        style: theme.textTheme.titleSmall,
                      ),
                      Text(
                        formatTimestamp(post.createdAt),
                        style: theme.textTheme.bodySmall,
                      ),
                    ],
                  ),
                ),
                if (mine)
                  IconButton(
                    tooltip: '删除动态',
                    onPressed: () async {
                      final confirmed = await confirmDelete(
                        context,
                        message: '删除后对方也看不到这条动态。',
                      );
                      if (!confirmed || !context.mounted) return;
                      await _run(context, onDelete);
                    },
                    icon: const Icon(Icons.delete_outline),
                  ),
              ],
            ),
            if (post.body.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(post.body),
            ],
            if (post.media.isNotEmpty) ...[
              const SizedBox(height: 8),
              GridView.builder(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                itemCount: post.media.length,
                gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                  crossAxisCount: 3,
                  crossAxisSpacing: 4,
                  mainAxisSpacing: 4,
                ),
                itemBuilder: (context, index) {
                  return ClipRRect(
                    borderRadius: BorderRadius.circular(8),
                    child: _PostImage(media: post.media[index]),
                  );
                },
              ),
            ],
            if (post.location != null) ...[
              const SizedBox(height: 8),
              Text(
                _locationLabel(post.location!),
                style: theme.textTheme.bodySmall,
              ),
            ],
            const SizedBox(height: 4),
            Row(
              children: [
                IconButton(
                  tooltip: post.likedByMe ? '取消赞' : '点赞',
                  onPressed: () =>
                      _run(context, post.likedByMe ? onUnlike : onLike),
                  icon: Icon(
                    post.likedByMe ? Icons.favorite : Icons.favorite_border,
                  ),
                ),
                Text('${post.likeCount}'),
                IconButton(
                  tooltip: '评论',
                  onPressed: () => showCommentSheet(context, post.id),
                  icon: const Icon(Icons.chat_bubble_outline),
                ),
                Text('${post.commentCount}'),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

Future<void> _run(BuildContext context, Future<void> Function() action) async {
  try {
    await action();
  } catch (error) {
    if (!context.mounted) return;
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(messageOf(error))));
  }
}

class _PostImage extends StatelessWidget {
  const _PostImage({required this.media});

  final MediaRef media;

  @override
  Widget build(BuildContext context) {
    final broken = const ColoredBox(
      color: Color(0xFFEDE6E3),
      child: Icon(Icons.broken_image_outlined),
    );
    final url = media.url;
    if (media.bytes.isNotEmpty) {
      return Image.memory(
        media.bytes,
        fit: BoxFit.cover,
        errorBuilder: (context, error, stackTrace) => broken,
      );
    }
    if (url != null && url.isNotEmpty) {
      return Image.network(
        url,
        fit: BoxFit.cover,
        errorBuilder: (context, error, stackTrace) => broken,
      );
    }
    return broken;
  }
}

String _locationLabel(LocationInput location) {
  return switch (location.visibility) {
    LocationVisibility.hidden => '位置已隐藏',
    LocationVisibility.city => location.name ?? '城市级位置',
    LocationVisibility.exact => [
      if (location.name != null && location.name!.isNotEmpty) location.name!,
      '${location.lat.toStringAsFixed(4)}, ${location.lng.toStringAsFixed(4)}',
    ].join(' · '),
  };
}
