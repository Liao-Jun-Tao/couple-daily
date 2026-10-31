import 'dart:typed_data';

import 'package:couple_daily/core/constants.dart';
import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/models/models.dart';
import 'package:couple_daily/core/session/session_scope.dart';
import 'package:couple_daily/features/compose/image_content_type.dart';
import 'package:couple_daily/features/compose/location_picker_page.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';

class _DraftImage {
  _DraftImage({required this.bytes, required this.contentType});

  final Uint8List bytes;
  final String contentType;
}

class ComposePage extends StatefulWidget {
  const ComposePage({super.key});

  @override
  State<ComposePage> createState() => _ComposePageState();
}

class _ComposePageState extends State<ComposePage> {
  final _body = TextEditingController();
  final _images = <_DraftImage>[];
  final _picker = ImagePicker();
  LocationInput? _location;
  String? _error;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _body.addListener(_onDraftChanged);
  }

  void _onDraftChanged() {
    setState(() {});
  }

  bool get _canPublish =>
      !_busy && (_body.text.trim().isNotEmpty || _images.isNotEmpty);

  @override
  void dispose() {
    _body.removeListener(_onDraftChanged);
    _body.dispose();
    super.dispose();
  }

  Future<void> _pickImages() async {
    final remaining = maxImagesPerPost - _images.length;
    if (remaining <= 0) {
      setState(() => _error = '最多 9 张图片');
      return;
    }
    try {
      final files = await _picker.pickMultiImage(
        limit: remaining,
        requestFullMetadata: false,
      );
      if (!mounted || files.isEmpty) return;
      final next = <_DraftImage>[];
      String? error;
      for (final file in files) {
        if (_images.length + next.length >= maxImagesPerPost) {
          error = '最多 9 张图片';
          break;
        }
        final contentType = resolveContentType(file);
        if (contentType == null) {
          error = '仅支持 jpeg、png、heic、webp';
          continue;
        }
        final bytes = await file.readAsBytes();
        if (bytes.length > maxImageBytes) {
          error = '单张图片不能超过 10MB';
          continue;
        }
        next.add(_DraftImage(bytes: bytes, contentType: contentType));
      }
      if (!mounted) return;
      setState(() {
        _images.addAll(next);
        _error = error;
      });
    } on Object {
      if (!mounted) return;
      setState(() => _error = '无法读取相册。可以改为只发文字。');
    }
  }

  Future<void> _pickLocation() async {
    final result = await context.push<LocationPickerResult>(
      '/location-picker',
      extra: _location,
    );
    if (!mounted || result == null) return;
    setState(() {
      switch (result) {
        case LocationCleared():
          _location = null;
        case LocationSelected(:final location):
          _location = location;
      }
    });
  }

  Future<void> _publish() async {
    if (_busy) return;
    final session = SessionScope.read(context);
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await session.createPost(
        body: _body.text,
        media: [
          for (final image in _images)
            MediaDraft(bytes: image.bytes, contentType: image.contentType),
        ],
        location: _location,
      );
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _busy = false;
        _error = messageOf(error);
      });
      return;
    }
    if (!mounted) return;
    Navigator.of(context).pop();
  }

  @override
  Widget build(BuildContext context) {
    final location = _location;
    return Scaffold(
      appBar: AppBar(title: const Text('写动态')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(
            key: const Key('compose-body'),
            controller: _body,
            minLines: 4,
            maxLines: 8,
            maxLength: 5000,
            decoration: const InputDecoration(
              labelText: '今天发生了什么',
              alignLabelWithHint: true,
            ),
          ),
          const SizedBox(height: 8),
          Text('图片 ${_images.length} / $maxImagesPerPost，单张不超过 10MB'),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (var index = 0; index < _images.length; index++)
                Stack(
                  children: [
                    ClipRRect(
                      borderRadius: BorderRadius.circular(8),
                      child: Image.memory(
                        _images[index].bytes,
                        width: 88,
                        height: 88,
                        fit: BoxFit.cover,
                        errorBuilder: (context, error, stackTrace) {
                          return const SizedBox(
                            width: 88,
                            height: 88,
                            child: ColoredBox(
                              color: Color(0xFFEDE6E3),
                              child: Icon(Icons.broken_image_outlined),
                            ),
                          );
                        },
                      ),
                    ),
                    Positioned(
                      right: 0,
                      top: 0,
                      child: IconButton(
                        tooltip: '移除图片',
                        onPressed: () {
                          setState(() => _images.removeAt(index));
                        },
                        icon: const Icon(Icons.close),
                      ),
                    ),
                  ],
                ),
              OutlinedButton.icon(
                onPressed: _pickImages,
                icon: const Icon(Icons.photo_outlined),
                label: const Text('添加图片'),
              ),
            ],
          ),
          const SizedBox(height: 16),
          ListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('位置'),
            subtitle: Text(
              location == null
                  ? '不附加位置。打开选择器时默认为完整。'
                  : '${location.visibility.label}'
                        '${location.name == null ? '' : ' · ${location.name}'}',
            ),
            trailing: const Icon(Icons.chevron_right),
            onTap: _pickLocation,
          ),
          if (_error != null) ...[
            const SizedBox(height: 8),
            Text(
              _error!,
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
          ],
          const SizedBox(height: 16),
          FilledButton(
            key: const Key('compose-submit'),
            onPressed: _canPublish
                ? () async {
                    await _publish();
                  }
                : null,
            child: const Text('发布'),
          ),
        ],
      ),
    );
  }
}
