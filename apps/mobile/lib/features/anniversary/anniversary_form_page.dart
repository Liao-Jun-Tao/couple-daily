import 'package:flutter/material.dart';

import 'package:couple_daily/core/constants.dart';
import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/format.dart';
import 'package:couple_daily/core/session/session_scope.dart';

class AnniversaryFormPage extends StatefulWidget {
  const AnniversaryFormPage({this.anniversaryId, super.key});

  final String? anniversaryId;

  @override
  State<AnniversaryFormPage> createState() => _AnniversaryFormPageState();
}

class _AnniversaryFormPageState extends State<AnniversaryFormPage> {
  final _title = TextEditingController();
  DateTime _date = DateTime.now();
  String _recurrence = 'yearly';
  bool _remind = true;
  String? _error;
  bool _loaded = false;
  bool _missing = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_loaded) return;
    _loaded = true;
    final id = widget.anniversaryId;
    if (id == null) return;
    final existing = SessionScope.of(context)
        .anniversaries()
        .where((item) => item.id == id)
        .firstOrNull;
    if (existing == null) {
      _missing = true;
      return;
    }
    _title.text = existing.title;
    _date = parseDateOnly(existing.eventDate) ?? _date;
    _recurrence = existing.recurrence;
    _remind = existing.remind;
  }

  @override
  void dispose() {
    _title.dispose();
    super.dispose();
  }

  Future<void> _pickDate() async {
    final picked = await showDatePicker(
      context: context,
      initialDate: _date,
      firstDate: DateTime(1970),
      lastDate: DateTime(2100),
    );
    if (picked == null || !mounted) return;
    setState(() => _date = picked);
  }

  Future<void> _save() async {
    final session = SessionScope.read(context);
    try {
      final saved = await session.saveAnniversary(
        id: widget.anniversaryId,
        title: _title.text,
        eventDate: formatDateOnly(_date),
        recurrence: _recurrence,
        remind: _remind,
      );
      if (!mounted) return;
      final message = saved.remind
          ? '已保存。提醒登记为当天本地 ${saved.remindLocalTime}（${saved.timezone}）。推送仍是 stub，未连接 APNs/FCM。'
          : '已保存，未开启提醒。';
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(message)));
      Navigator.of(context).pop();
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = messageOf(error));
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_missing) {
      return Scaffold(
        appBar: AppBar(title: const Text('纪念日')),
        body: const Center(child: Text('纪念日不存在')),
      );
    }
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.anniversaryId == null ? '新建纪念日' : '编辑纪念日'),
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(
            key: const Key('anniversary-title'),
            controller: _title,
            decoration: const InputDecoration(labelText: '名称'),
          ),
          const SizedBox(height: 12),
          ListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('日期'),
            subtitle: Text(formatDateOnly(_date)),
            trailing: const Icon(Icons.calendar_today_outlined),
            onTap: _pickDate,
          ),
          const SizedBox(height: 8),
          SegmentedButton<String>(
            segments: const [
              ButtonSegment(value: 'yearly', label: Text('每年')),
              ButtonSegment(value: 'once', label: Text('一次')),
            ],
            selected: {_recurrence},
            onSelectionChanged: (value) {
              setState(() => _recurrence = value.first);
            },
          ),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('当天提醒'),
            value: _remind,
            onChanged: (value) => setState(() => _remind = value),
          ),
          Text(
            '提醒时间 $defaultRemindLocalTime（$defaultTimezone）。'
            '首版只在当天提醒一次，不提前。推送仍是 stub，未连接 APNs/FCM。',
          ),
          if (_error != null) ...[
            const SizedBox(height: 12),
            Text(
              _error!,
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
          ],
          const SizedBox(height: 16),
          FilledButton(
            key: const Key('anniversary-save'),
            onPressed: () async {
              await _save();
            },
            child: const Text('保存'),
          ),
        ],
      ),
    );
  }
}
