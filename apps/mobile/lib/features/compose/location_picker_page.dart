import 'package:flutter/material.dart';

import 'package:couple_daily/core/models/models.dart';

sealed class LocationPickerResult {
  const LocationPickerResult();
}

class LocationCleared extends LocationPickerResult {
  const LocationCleared();
}

class LocationSelected extends LocationPickerResult {
  const LocationSelected(this.location);

  final LocationInput location;
}

class _Preset {
  const _Preset({
    required this.name,
    required this.lat,
    required this.lng,
    required this.placeId,
  });

  final String name;
  final double lat;
  final double lng;
  final String placeId;
}

const _presets = [
  _Preset(name: '外滩', lat: 31.2400, lng: 121.4900, placeId: 'preset-waitan'),
  _Preset(name: '西湖', lat: 30.2480, lng: 120.1450, placeId: 'preset-xihu'),
  _Preset(name: '宽窄巷子', lat: 30.6690, lng: 104.0530, placeId: 'preset-kuanzhai'),
];

/// Picks an optional post location. Visibility defaults to 完整 (exact).
class LocationPickerPage extends StatefulWidget {
  const LocationPickerPage({this.initial, super.key});

  final LocationInput? initial;

  @override
  State<LocationPickerPage> createState() => _LocationPickerPageState();
}

class _LocationPickerPageState extends State<LocationPickerPage> {
  late final TextEditingController _name;
  late final TextEditingController _lat;
  late final TextEditingController _lng;
  late LocationVisibility _visibility;
  String? _placeId;
  String? _error;

  @override
  void initState() {
    super.initState();
    final initial = widget.initial;
    _name = TextEditingController(text: initial?.name ?? '');
    _lat = TextEditingController(
      text: initial == null ? '' : initial.lat.toString(),
    );
    _lng = TextEditingController(
      text: initial == null ? '' : initial.lng.toString(),
    );
    _visibility = initial?.visibility ?? LocationVisibility.exact;
    _placeId = initial?.placeId;
  }

  @override
  void dispose() {
    _name.dispose();
    _lat.dispose();
    _lng.dispose();
    super.dispose();
  }

  void _applyPreset(_Preset preset) {
    setState(() {
      _name.text = preset.name;
      _lat.text = preset.lat.toString();
      _lng.text = preset.lng.toString();
      _placeId = preset.placeId;
      _error = null;
    });
  }

  void _save() {
    final lat = double.tryParse(_lat.text.trim());
    final lng = double.tryParse(_lng.text.trim());
    if (lat == null || lng == null) {
      setState(() => _error = '请选择地点，或填写经纬度');
      return;
    }
    if (lat < -90 || lat > 90 || lng < -180 || lng > 180) {
      setState(() => _error = '经纬度超出范围');
      return;
    }
    Navigator.pop(
      context,
      LocationSelected(
        LocationInput(
          lat: lat,
          lng: lng,
          name: _name.text.trim().isEmpty ? null : _name.text.trim(),
          placeId: _placeId,
          visibility: _visibility,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('位置')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Text(
            '默认「完整」：保存坐标和地点名称。城市只展示名称，隐藏则时间线不显示具体位置。',
            style: Theme.of(context).textTheme.bodyMedium,
          ),
          const SizedBox(height: 12),
          SegmentedButton<LocationVisibility>(
            segments: const [
              ButtonSegment(
                value: LocationVisibility.exact,
                label: Text('完整'),
              ),
              ButtonSegment(
                value: LocationVisibility.city,
                label: Text('城市'),
              ),
              ButtonSegment(
                value: LocationVisibility.hidden,
                label: Text('隐藏'),
              ),
            ],
            selected: {_visibility},
            onSelectionChanged: (value) {
              setState(() => _visibility = value.first);
            },
          ),
          const SizedBox(height: 16),
          Text('常用地点', style: Theme.of(context).textTheme.titleSmall),
          for (final preset in _presets)
            ListTile(
              contentPadding: EdgeInsets.zero,
              title: Text(preset.name),
              subtitle: Text('${preset.lat}, ${preset.lng}'),
              onTap: () => _applyPreset(preset),
            ),
          const SizedBox(height: 8),
          TextField(
            controller: _name,
            decoration: const InputDecoration(labelText: '地点名称'),
          ),
          const SizedBox(height: 12),
          TextField(
            key: const Key('location-lat'),
            controller: _lat,
            keyboardType: const TextInputType.numberWithOptions(
              decimal: true,
              signed: true,
            ),
            decoration: const InputDecoration(labelText: '纬度'),
            onChanged: (_) => _placeId = null,
          ),
          const SizedBox(height: 12),
          TextField(
            key: const Key('location-lng'),
            controller: _lng,
            keyboardType: const TextInputType.numberWithOptions(
              decimal: true,
              signed: true,
            ),
            decoration: const InputDecoration(labelText: '经度'),
            onChanged: (_) => _placeId = null,
          ),
          if (_error != null) ...[
            const SizedBox(height: 12),
            Text(
              _error!,
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
          ],
          const SizedBox(height: 16),
          FilledButton(onPressed: _save, child: const Text('使用这个位置')),
          TextButton(
            onPressed: () => Navigator.pop(context, const LocationCleared()),
            child: const Text('不附加位置'),
          ),
        ],
      ),
    );
  }
}
