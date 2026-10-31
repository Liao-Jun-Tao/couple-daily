import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:couple_daily/core/format.dart';
import 'package:couple_daily/core/models/models.dart';
import 'package:couple_daily/core/session/session_scope.dart';
import 'package:couple_daily/shared/widgets/empty_state.dart';
import 'package:couple_daily/shared/widgets/confirm_delete_dialog.dart';

class AnniversaryListPage extends StatelessWidget {
  const AnniversaryListPage({super.key});

  @override
  Widget build(BuildContext context) {
    final session = SessionScope.of(context);
    final items = session.anniversaries();
    final viewerId = session.userId;
    return Scaffold(
      appBar: AppBar(title: const Text('纪念日')),
      floatingActionButton: FloatingActionButton.extended(
        tooltip: '新建纪念日',
        onPressed: () => context.push('/anniversary/new'),
        icon: const Icon(Icons.add),
        label: const Text('新建'),
      ),
      body: items.isEmpty
          ? const EmptyState(
              title: '还没有纪念日',
              subtitle: '保存后会按当天本地 09:00 登记提醒。这是本地 stub，不会真的推送。',
            )
          : ListView.builder(
              padding: const EdgeInsets.only(bottom: 88),
              itemCount: items.length,
              itemBuilder: (context, index) {
                final item = items[index];
                final mine = item.creator.id == viewerId;
                return ListTile(
                  title: Text(item.title),
                  subtitle: Text(
                    '${item.eventDate} · ${_countdown(item)}\n'
                    '${item.recurrence == 'yearly' ? '每年' : '一次'}'
                    '${item.remind ? ' · 当天 ${item.remindLocalTime} 提醒（${item.timezone}，本地 stub）' : ' · 不提醒'}',
                  ),
                  isThreeLine: true,
                  onTap: mine
                      ? () => context.push('/anniversary/edit/${item.id}')
                      : null,
                  trailing: mine
                      ? IconButton(
                          tooltip: '删除纪念日',
                          onPressed: () async {
                            final confirmed = await confirmDelete(
                              context,
                              message: '只有作者可以删除自己的纪念日。',
                            );
                            if (!confirmed || !context.mounted) return;
                            SessionScope.read(context).deleteAnniversary(item.id);
                          },
                          icon: const Icon(Icons.delete_outline),
                        )
                      : null,
                );
              },
            ),
    );
  }
}

String _countdown(Anniversary item) {
  final event = parseDateOnly(item.eventDate);
  if (event == null) return item.eventDate;
  final now = DateTime.now();
  final today = DateTime(now.year, now.month, now.day);
  final DateTime target;
  if (item.recurrence == 'once') {
    target = event;
  } else {
    var next = DateTime(today.year, event.month, event.day);
    if (next.isBefore(today)) {
      next = DateTime(today.year + 1, event.month, event.day);
    }
    target = next;
  }
  final days = target.difference(today).inDays;
  if (days == 0) return '就是今天';
  if (days > 0) return '还有 $days 天';
  return '已过 ${-days} 天';
}
