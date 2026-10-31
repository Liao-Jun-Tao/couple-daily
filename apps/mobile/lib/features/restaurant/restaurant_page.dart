import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/format.dart';
import 'package:couple_daily/core/session/session_scope.dart';
import 'package:couple_daily/features/restaurant/restaurant_catalog.dart';
import 'package:couple_daily/shared/widgets/confirm_delete_dialog.dart';

class RestaurantPage extends StatefulWidget {
  const RestaurantPage({super.key});

  @override
  State<RestaurantPage> createState() => _RestaurantPageState();
}

class _RestaurantPageState extends State<RestaurantPage> {
  final _query = TextEditingController();
  final _manualName = TextEditingController();
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
      await SessionScope.read(context).refreshCheckins();
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = messageOf(error));
    }
  }

  @override
  void dispose() {
    _query.dispose();
    _manualName.dispose();
    super.dispose();
  }

  Future<void> _checkInPlace(RestaurantPlace place) async {
    try {
      await SessionScope.read(context).createCheckin(
        name: place.name,
        address: place.address,
        lat: place.lat,
        lng: place.lng,
        placeId: place.placeId,
      );
      if (!mounted) return;
      setState(() => _error = null);
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = messageOf(error));
    }
  }

  Future<void> _checkInManual() async {
    try {
      await SessionScope.read(context).createCheckin(name: _manualName.text);
      _manualName.clear();
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
    final results = searchRestaurants(_query.text);
    final queryEmpty = _query.text.trim().isEmpty;
    final showManual = !queryEmpty && results.isEmpty;
    final checkins = session.checkins();

    return Scaffold(
      appBar: AppBar(
        title: const Text('餐厅'),
        actions: [
          IconButton(
            tooltip: '收藏',
            onPressed: () => context.push('/favorites'),
            icon: const Icon(Icons.bookmark_outline),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(
            key: const Key('restaurant-search'),
            controller: _query,
            decoration: const InputDecoration(
              hintText: '搜索餐厅',
              prefixIcon: Icon(Icons.search),
            ),
            onChanged: (value) {
              final trimmed = value.trim();
              final matches = searchRestaurants(trimmed);
              if (trimmed.isNotEmpty && matches.isEmpty) {
                _manualName.text = trimmed;
              }
              setState(() {});
            },
          ),
          const SizedBox(height: 12),
          if (queryEmpty)
            Text('试试搜索，或从下面的店打卡', style: Theme.of(context).textTheme.bodySmall),
          if (showManual) ...[
            Text(
              '没有搜到这家店，可以手填店名打卡。店名必填，可以不填 place id。',
              style: Theme.of(context).textTheme.bodyMedium,
            ),
            const SizedBox(height: 8),
            TextField(
              key: const Key('manual-store-name'),
              controller: _manualName,
              decoration: const InputDecoration(labelText: '店名'),
            ),
            const SizedBox(height: 8),
            FilledButton(
              key: const Key('manual-checkin'),
              onPressed: () async {
                await _checkInManual();
              },
              child: const Text('用手填店名打卡'),
            ),
          ] else
            for (final place in results)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      place.name,
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    Text(place.address),
                    const SizedBox(height: 8),
                    FilledButton(
                      onPressed: () async {
                        await _checkInPlace(place);
                      },
                      child: const Text('打卡'),
                    ),
                  ],
                ),
              ),
          if (_error != null) ...[
            const SizedBox(height: 8),
            Text(
              _error!,
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
          ],
          const SizedBox(height: 24),
          Text('打卡记录', style: Theme.of(context).textTheme.titleMedium),
          if (checkins.isEmpty)
            const Padding(
              padding: EdgeInsets.only(top: 12),
              child: Text('还没有打卡'),
            )
          else
            for (final checkin in checkins)
              ListTile(
                contentPadding: EdgeInsets.zero,
                title: Text(checkin.name),
                subtitle: Text(
                  [
                    if (checkin.placeId == null) '手填店名',
                    if (checkin.address != null) checkin.address!,
                    formatTimestamp(checkin.checkedInAt),
                  ].join(' · '),
                ),
                trailing: checkin.creator.id == viewerId
                    ? IconButton(
                        tooltip: '删除打卡',
                        onPressed: () async {
                          final confirmed = await confirmDelete(
                            context,
                            message: '只有作者可以删除自己的打卡。',
                          );
                          if (!confirmed || !context.mounted) return;
                          try {
                            await SessionScope.read(context)
                                .deleteCheckin(checkin.id);
                          } catch (error) {
                            if (!context.mounted) return;
                            setState(() => _error = messageOf(error));
                          }
                        },
                        icon: const Icon(Icons.delete_outline),
                      )
                    : null,
              ),
        ],
      ),
    );
  }
}
