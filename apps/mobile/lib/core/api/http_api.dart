import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import 'package:couple_daily/core/api/api_parse.dart';
import 'package:couple_daily/core/api/token_store.dart';
import 'package:couple_daily/core/constants.dart';
import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/models/models.dart';

const _timeout = Duration(seconds: 20);

/// OpenAPI client for `/v1`. A 401 on an authenticated call refreshes once.
/// A failed refresh calls [onSessionExpired] so the UI returns to auth.
class HttpCoupleApi {
  HttpCoupleApi({
    required this.baseUrl,
    required this.tokens,
    required this.client,
    required this.ownsClient,
    this.onSessionExpired,
  });

  final String baseUrl;
  final TokenStore tokens;
  final http.Client client;
  final bool ownsClient;
  final void Function()? onSessionExpired;

  Future<void>? _refreshing;

  void close() {
    if (ownsClient) client.close();
  }

  Future<AuthResult> register({
    required String email,
    required String password,
    required String displayName,
  }) async {
    final json = await _send(
      'POST',
      '/auth/register',
      auth: false,
      body: {
        'email': email.trim(),
        'password': password,
        'display_name': displayName.trim(),
      },
    );
    return _storeAuth(parseAuth(json));
  }

  Future<AuthResult> login({
    required String email,
    required String password,
  }) async {
    final json = await _send(
      'POST',
      '/auth/login',
      auth: false,
      body: {'email': email.trim(), 'password': password},
    );
    return _storeAuth(parseAuth(json));
  }

  Future<MeSnapshot> me() async {
    return parseMe(await _send('GET', '/auth/me'));
  }

  Future<InviteTicket> createInvite() async {
    return parseInvite(await _send('POST', '/pairing/invites'));
  }

  Future<SpaceSnapshot> acceptInvite(String code) async {
    final json = await _send(
      'POST',
      '/pairing/invites/accept',
      body: {'invite_code': code.trim()},
    );
    return parseSpace(json);
  }

  /// Null when the user is not in a space (404 NOT_PAIRED).
  Future<SpaceSnapshot?> space() async {
    try {
      return parseSpace(await _send('GET', '/pairing/space'));
    } on ApiException catch (error) {
      if (error.code == 'NOT_PAIRED' || error.code == 'NOT_FOUND') return null;
      rethrow;
    }
  }

  Future<List<Post>> listPosts() {
    return _collect('/posts', parsePost);
  }

  Future<String> uploadMedia(MediaDraft draft) async {
    final json = await _send(
      'POST',
      '/media/uploads',
      body: {'content_type': draft.contentType, 'byte_size': draft.byteSize},
    );
    final mediaId = parseApiString(json['media_id']);
    final uploadUrl = parseApiString(json['upload_url']);
    final method = json['upload_method'] is String
        ? json['upload_method'] as String
        : 'PUT';
    final headers = <String, String>{};
    final rawHeaders = json['upload_headers'];
    if (rawHeaders is Map) {
      for (final entry in rawHeaders.entries) {
        headers['${entry.key}'] = '${entry.value}';
      }
    }
    try {
      final request = http.Request(method, Uri.parse(uploadUrl));
      request.bodyBytes = draft.bytes;
      request.headers.addAll(headers);
      final streamed = await client.send(request).timeout(_timeout);
      final response = await http.Response.fromStream(streamed);
      if (response.statusCode < 200 || response.statusCode >= 300) {
        throw ApiException(
          'MEDIA_INVALID',
          chineseApiMessage('MEDIA_INVALID', ''),
        );
      }
    } on ApiException {
      rethrow;
    } on Object {
      throw ApiException('NETWORK', chineseApiMessage('NETWORK', ''));
    }
    return mediaId;
  }

  Future<Post> createPost({
    required String body,
    required List<String> mediaIds,
    LocationInput? location,
  }) async {
    final json = await _send(
      'POST',
      '/posts',
      body: {
        'body': body,
        'media_ids': mediaIds,
        if (location != null) 'location': _locationJson(location),
      },
    );
    return parsePost(json);
  }

  Future<void> deletePost(String postId) {
    return _send('DELETE', '/posts/$postId');
  }

  Future<LikeStatus> like(String postId) async {
    return parseLike(await _send('PUT', '/posts/$postId/likes'));
  }

  Future<LikeStatus> unlike(String postId) async {
    return parseLike(await _send('DELETE', '/posts/$postId/likes'));
  }

  Future<List<Comment>> listComments(String postId) {
    return _collect('/posts/$postId/comments', parseComment);
  }

  Future<Comment> addComment(String postId, String body) async {
    final json = await _send(
      'POST',
      '/posts/$postId/comments',
      body: {'body': body},
    );
    return parseComment(json);
  }

  Future<void> deleteComment(String commentId) {
    return _send('DELETE', '/comments/$commentId');
  }

  Future<List<RestaurantCheckin>> listCheckins() {
    return _collect('/checkins', parseCheckin);
  }

  Future<RestaurantCheckin> createCheckin({
    required String name,
    String? address,
    double? lat,
    double? lng,
    String? placeId,
  }) async {
    final json = await _send(
      'POST',
      '/checkins',
      body: {
        'name': name,
        if (address != null && address.isNotEmpty) 'address': address,
        'lat': ?lat,
        'lng': ?lng,
        if (placeId != null && placeId.isNotEmpty) 'place_id': placeId,
      },
    );
    return parseCheckin(json);
  }

  Future<void> deleteCheckin(String checkinId) {
    return _send('DELETE', '/checkins/$checkinId');
  }

  Future<List<Anniversary>> listAnniversaries() {
    return _collect('/anniversaries', parseAnniversary);
  }

  Future<Anniversary> createAnniversary({
    required String title,
    required String eventDate,
    required String recurrence,
    required bool remind,
  }) async {
    final json = await _send(
      'POST',
      '/anniversaries',
      body: _anniversaryBody(
        title: title,
        eventDate: eventDate,
        recurrence: recurrence,
        remind: remind,
      ),
    );
    return parseAnniversary(json);
  }

  Future<Anniversary> updateAnniversary({
    required String anniversaryId,
    required String title,
    required String eventDate,
    required String recurrence,
    required bool remind,
  }) async {
    final json = await _send(
      'PATCH',
      '/anniversaries/$anniversaryId',
      body: _anniversaryBody(
        title: title,
        eventDate: eventDate,
        recurrence: recurrence,
        remind: remind,
      ),
    );
    return parseAnniversary(json);
  }

  Future<void> deleteAnniversary(String anniversaryId) {
    return _send('DELETE', '/anniversaries/$anniversaryId');
  }

  Future<DeviceToken> registerDevice({
    required String platform,
    required String token,
  }) async {
    final json = await _send(
      'POST',
      '/devices',
      body: {'platform': platform, 'token': token},
    );
    return parseDevice(json);
  }

  Future<void> deleteDevice(String token) {
    return _send('DELETE', '/devices', body: {'token': token});
  }

  Map<String, Object?> _anniversaryBody({
    required String title,
    required String eventDate,
    required String recurrence,
    required bool remind,
  }) {
    return {
      'title': title,
      'event_date': eventDate,
      'recurrence': recurrence,
      'remind': remind,
      'remind_days_before': defaultRemindDaysBefore,
      'remind_local_time': defaultRemindLocalTime,
      'timezone': defaultTimezone,
    };
  }

  Map<String, Object?> _locationJson(LocationInput location) {
    return {
      'lat': location.lat,
      'lng': location.lng,
      if (location.name != null) 'name': location.name,
      if (location.placeId != null) 'place_id': location.placeId,
      'visibility': location.visibility.name,
    };
  }

  Future<AuthResult> _storeAuth(AuthResult result) async {
    await tokens.writeTokens(
      StoredTokens(
        accessToken: result.accessToken,
        refreshToken: result.refreshToken,
      ),
    );
    return result;
  }

  Future<List<T>> _collect<T>(
    String path,
    T Function(Map<String, dynamic> json) parse,
  ) async {
    final all = <T>[];
    String? cursor;
    for (var page = 0; page < 20; page++) {
      final json = await _send(
        'GET',
        path,
        query: {'limit': '50', 'cursor': ?cursor},
      );
      for (final item in parseItems(json)) {
        all.add(parse(item));
      }
      final next = parseNextCursor(json);
      if (next == null) break;
      cursor = next;
    }
    return all;
  }

  Future<Map<String, dynamic>> _send(
    String method,
    String path, {
    Map<String, Object?>? body,
    Map<String, String>? query,
    bool auth = true,
    bool allowRefresh = true,
  }) async {
    var response = await _raw(
      method,
      path,
      body: body,
      query: query,
      auth: auth,
    );
    if (response.statusCode == 401 && auth && allowRefresh) {
      await _refreshOnce();
      response = await _raw(method, path, body: body, query: query, auth: true);
      return _decode(response, expireOn401: true);
    }
    return _decode(response, expireOn401: false);
  }

  Future<void> _refreshOnce() {
    final existing = _refreshing;
    if (existing != null) return existing;
    final future = _refresh();
    _refreshing = future;
    return future.whenComplete(() {
      if (identical(_refreshing, future)) _refreshing = null;
    });
  }

  Future<void> _refresh() async {
    final stored = await tokens.readTokens();
    if (stored == null || stored.refreshToken.isEmpty) {
      onSessionExpired?.call();
      throw ApiException(
        'TOKEN_INVALID',
        chineseApiMessage('TOKEN_INVALID', ''),
      );
    }
    final response = await _raw(
      'POST',
      '/auth/refresh',
      auth: false,
      body: {'refresh_token': stored.refreshToken},
    );
    if (response.statusCode == 401 || response.statusCode >= 400) {
      final error = _errorFrom(response);
      if (response.statusCode == 401) onSessionExpired?.call();
      throw error;
    }
    final result = parseAuth(_decode(response));
    await tokens.writeTokens(
      StoredTokens(
        accessToken: result.accessToken,
        refreshToken: result.refreshToken,
      ),
    );
  }

  Future<http.Response> _raw(
    String method,
    String path, {
    Map<String, Object?>? body,
    Map<String, String>? query,
    required bool auth,
  }) async {
    try {
      final headers = <String, String>{'Accept': 'application/json'};
      if (body != null) headers['Content-Type'] = 'application/json';
      if (auth) {
        final stored = await tokens.readTokens();
        final access = stored?.accessToken;
        if (access == null || access.isEmpty) {
          throw ApiException(
            'UNAUTHORIZED',
            chineseApiMessage('UNAUTHORIZED', ''),
          );
        }
        headers['Authorization'] = 'Bearer $access';
      }
      final request = http.Request(method, _uri(path, query));
      request.headers.addAll(headers);
      if (body != null) request.body = jsonEncode(body);
      final streamed = await client.send(request).timeout(_timeout);
      return await http.Response.fromStream(streamed);
    } on ApiException {
      rethrow;
    } on Object {
      throw ApiException('NETWORK', chineseApiMessage('NETWORK', ''));
    }
  }

  Uri _uri(String path, Map<String, String>? query) {
    final suffix = path.startsWith('/') ? path : '/$path';
    final uri = Uri.parse('$baseUrl$suffix');
    if (query == null || query.isEmpty) return uri;
    return uri.replace(queryParameters: {...uri.queryParameters, ...query});
  }

  Map<String, dynamic> _decode(
    http.Response response, {
    bool expireOn401 = false,
  }) {
    if (response.statusCode >= 200 && response.statusCode < 300) {
      if (response.statusCode == 204 || response.body.trim().isEmpty) {
        return {};
      }
      final decoded = _jsonObject(response.body);
      if (decoded == null) {
        throw ApiException('INTERNAL', '服务器返回了无法识别的数据');
      }
      return decoded;
    }
    final error = _errorFrom(response);
    if (response.statusCode == 401 && expireOn401) {
      onSessionExpired?.call();
    }
    throw error;
  }

  ApiException _errorFrom(http.Response response) {
    final decoded = _jsonObject(response.body);
    final code = decoded?['code'] is String
        ? decoded!['code'] as String
        : _fallbackCode(response.statusCode);
    final serverMessage = decoded?['message'] is String
        ? decoded!['message'] as String
        : '';
    return ApiException(code, chineseApiMessage(code, serverMessage));
  }

  Map<String, dynamic>? _jsonObject(String body) {
    if (body.trim().isEmpty) return null;
    try {
      final decoded = jsonDecode(body);
      if (decoded is Map) return Map<String, dynamic>.from(decoded);
    } on FormatException {
      return null;
    }
    return null;
  }

  String _fallbackCode(int status) {
    return switch (status) {
      400 => 'VALIDATION_ERROR',
      401 => 'UNAUTHORIZED',
      403 => 'FORBIDDEN',
      404 => 'NOT_FOUND',
      409 => 'CONFLICT',
      429 => 'RATE_LIMITED',
      _ => 'INTERNAL',
    };
  }
}
