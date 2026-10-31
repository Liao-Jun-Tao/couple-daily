import 'package:flutter/material.dart';

import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/format.dart';
import 'package:couple_daily/core/session/session_scope.dart';
import 'package:couple_daily/shared/widgets/confirm_delete_dialog.dart';

Future<void> showCommentSheet(BuildContext context, String postId) {
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    showDragHandle: true,
    builder: (context) {
      return FractionallySizedBox(
        heightFactor: 0.72,
        child: CommentSheet(postId: postId),
      );
    },
  );
}

class CommentSheet extends StatefulWidget {
  const CommentSheet({required this.postId, super.key});

  final String postId;

  @override
  State<CommentSheet> createState() => _CommentSheetState();
}

class _CommentSheetState extends State<CommentSheet> {
  final _body = TextEditingController();
  String? _error;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      _reload();
    });
  }

  Future<void> _reload() async {
    try {
      await SessionScope.read(context).refreshComments(widget.postId);
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = messageOf(error));
    }
  }

  @override
  void dispose() {
    _body.dispose();
    super.dispose();
  }

  Future<void> _send() async {
    try {
      await SessionScope.read(context).addComment(widget.postId, _body.text);
      _body.clear();
      if (!mounted) return;
      setState(() => _error = null);
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = messageOf(error));
    }
  }

  @override
  Widget build(BuildContext context) {
    final session = SessionScope.of(context);
    final viewerId = session.userId;
    final comments = session.comments(widget.postId);
    final bottom = MediaQuery.viewInsetsOf(context).bottom;
    return Padding(
      padding: EdgeInsets.only(bottom: bottom),
      child: Column(
        children: [
          Text('评论', style: Theme.of(context).textTheme.titleMedium),
          Expanded(
            child: comments.isEmpty
                ? const Center(child: Text('还没有评论'))
                : ListView.builder(
                    itemCount: comments.length,
                    itemBuilder: (context, index) {
                      final comment = comments[index];
                      final mine = comment.author.id == viewerId;
                      return ListTile(
                        title: Text(comment.author.displayName),
                        subtitle: Text(
                          '${comment.body}\n${formatTimestamp(comment.createdAt)}',
                        ),
                        isThreeLine: true,
                        trailing: mine
                            ? IconButton(
                                tooltip: '删除评论',
                                onPressed: () async {
                                  final confirmed = await confirmDelete(
                                    context,
                                    message: '只有作者可以删除自己的评论。',
                                  );
                                  if (!confirmed || !context.mounted) return;
                                  try {
                                    await SessionScope.read(context)
                                        .deleteComment(comment.id);
                                  } catch (error) {
                                    if (!context.mounted) return;
                                    setState(() => _error = messageOf(error));
                                  }
                                },
                                icon: const Icon(Icons.delete_outline),
                              )
                            : null,
                      );
                    },
                  ),
          ),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Text(
                _error!,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
            ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    key: const Key('comment-body'),
                    controller: _body,
                    decoration: const InputDecoration(hintText: '写一条评论'),
                    onSubmitted: (_) => _send(),
                  ),
                ),
                const SizedBox(width: 8),
                FilledButton(onPressed: _send, child: const Text('发送')),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
