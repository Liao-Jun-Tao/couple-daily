import 'dart:convert';
import 'dart:typed_data';

import 'package:couple_daily/core/api/http_api.dart';
import 'package:couple_daily/core/api/token_store.dart';
import 'package:couple_daily/core/config/api_config.dart';
import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/models/models.dart';
import 'package:couple_daily/core/session/app_session.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  test('real API is the default and the emulator host is the base URL', () {
    expect(ApiConfig.fromEnvironment.useMock, isFalse);
    expect(ApiConfig.fromEnvironment.baseUrl, 'http://10.0.2.2:8080/v1');
    expect(
      const ApiConfig(
        useMock: false,
        baseUrl: 'http://10.0.2.2:8080/v1/',
      ).normalizedBaseUrl,
      'http://10.0.2.2:8080/v1',
    );
  });

  test(
    'SPACE_FULL is shown as returned, not rewritten to INVITE_USED',
    () async {
      final store = MemoryTokenStore()
        ..tokens = const StoredTokens(
          accessToken: 'access',
          refreshToken: 'refresh',
        );
      final api = HttpCoupleApi(
        baseUrl: 'http://example.test/v1',
        tokens: store,
        client: MockClient((request) async {
          expect(request.url.path, '/v1/pairing/invites/accept');
          final body = jsonDecode(request.body) as Map<String, dynamic>;
          expect(body['invite_code'], 'A3K9M2');
          return _json(409, {
            'code': 'SPACE_FULL',
            'message': 'couple space already has two members',
          });
        }),
        ownsClient: true,
      );
      await expectLater(
        api.acceptInvite('A3K9M2'),
        throwsA(
          isA<ApiException>()
              .having((error) => error.code, 'code', 'SPACE_FULL')
              .having((error) => error.message, 'message', '这个空间已经有两个人'),
        ),
      );
    },
  );

  test(
    'expired access token refreshes once and retries with the new token',
    () async {
      final store = MemoryTokenStore()
        ..tokens = const StoredTokens(
          accessToken: 'old-access',
          refreshToken: 'old-refresh',
        );
      var posts = 0;
      final api = HttpCoupleApi(
        baseUrl: 'http://example.test/v1',
        tokens: store,
        client: MockClient((request) async {
          if (request.url.path == '/v1/posts') {
            posts += 1;
            if (posts == 1) {
              expect(request.headers['authorization'], 'Bearer old-access');
              return _json(401, {'code': 'UNAUTHORIZED', 'message': 'expired'});
            }
            expect(request.headers['authorization'], 'Bearer new-access');
            return _json(200, {
              'items': [_postMap()],
              'next_cursor': null,
            });
          }
          expect(request.url.path, '/v1/auth/refresh');
          expect(jsonDecode(request.body)['refresh_token'], 'old-refresh');
          return _json(
            200,
            _authMap(access: 'new-access', refresh: 'new-refresh'),
          );
        }),
        ownsClient: true,
      );

      final postsPage = await api.listPosts();
      expect(postsPage.single.body, '今天天气真好');
      expect(store.tokens?.accessToken, 'new-access');
      expect(store.tokens?.refreshToken, 'new-refresh');
      expect(posts, 2);
    },
  );

  test('a bad refresh clears the session and reports TOKEN_INVALID', () async {
    var expired = 0;
    final store = MemoryTokenStore()
      ..tokens = const StoredTokens(
        accessToken: 'old-access',
        refreshToken: 'old-refresh',
      );
    final session = AppSession(
      useMock: false,
      config: const ApiConfig(
        useMock: false,
        baseUrl: 'http://example.test/v1',
      ),
      tokenStore: store,
      httpClient: MockClient((request) async {
        if (request.url.path == '/v1/auth/refresh') {
          return _json(401, {
            'code': 'TOKEN_INVALID',
            'message': 'refresh token is invalid or expired',
          });
        }
        return _json(401, {'code': 'UNAUTHORIZED', 'message': 'expired'});
      }),
    );
    session.addListener(() => expired += 1);
    await session.bootstrap();
    expect(session.ready, isTrue);
    expect(session.isLoggedIn, isFalse);
    expect(store.tokens, isNull);
    expect(expired, greaterThan(0));
    session.dispose();
  });

  test(
    'login, invite, timeline post, check-in, anniversary, and device',
    () async {
      final fake = _FlowApi();
      final session = AppSession(
        useMock: false,
        config: const ApiConfig(
          useMock: false,
          baseUrl: 'http://example.test/v1',
        ),
        tokenStore: fake.store,
        httpClient: fake.client,
      );

      await session.login(email: 'a@example.com', password: 'password1');
      expect(session.isLoggedIn, isTrue);
      expect(session.isPaired, isFalse);
      expect(fake.seen, ['POST /v1/auth/login', 'GET /v1/auth/me']);

      final ticket = await session.createInvite();
      expect(ticket.inviteCode, 'A3K9M2');
      expect(ticket.space.memberCount, 1);
      expect(session.pairing.status, PairingStatus.pendingInvite);
      expect(session.pairing.memberCount, 1);
      expect((await fake.store.readInvite())?.code, 'A3K9M2');

      await session.acceptInvite('a3k9m2');
      expect(session.isPaired, isTrue);
      expect(session.pendingPairingDone, isTrue);
      expect(session.space?.memberCount, 2);
      expect(session.posts(), isEmpty);

      await expectLater(
        session.createPost(body: '  ', media: const []),
        throwsA(
          isA<ApiException>().having(
            (error) => error.code,
            'code',
            'EMPTY_POST',
          ),
        ),
      );
      expect(fake.seen.where((call) => call == 'POST /v1/posts'), isEmpty);

      await session.createPost(
        body: '今天天气真好',
        media: [
          MediaDraft(
            bytes: Uint8List.fromList([1, 2, 3]),
            contentType: 'image/jpeg',
          ),
        ],
      );
      expect(fake.seen, contains('POST /v1/media/uploads'));
      expect(fake.seen, contains('PUT http://uploads.test/obj'));
      expect(session.posts().single.body, '今天天气真好');
      expect(session.posts().single.media.single.url, 'http://cdn.test/a.jpg');

      await session.like(session.posts().single.id);
      expect(session.posts().single.likedByMe, isTrue);
      expect(session.posts().single.likeCount, 1);

      await session.createCheckin(name: '没有这家店xyz');
      expect(session.checkins().single.name, '没有这家店xyz');
      expect(session.checkins().single.placeId, isNull);

      final saved = await session.saveAnniversary(
        title: '在一起',
        eventDate: '2024-05-20',
        recurrence: 'yearly',
        remind: true,
      );
      expect(saved.remindLocalTime, '09:00');
      expect(saved.timezone, 'Asia/Shanghai');
      expect(session.anniversaries().single.title, '在一起');

      await session.registerDevice(platform: 'android', token: 'stub-android');
      expect(session.devices().single.token, 'stub-android');

      final postBody = jsonDecode(fake.postBody!) as Map<String, dynamic>;
      expect(postBody['body'], '今天天气真好');
      expect(postBody['media_ids'], ['media-1']);
      expect(fake.anniversaryBody?['remind_local_time'], '09:00');
      expect(fake.anniversaryBody?['timezone'], 'Asia/Shanghai');
      session.dispose();
    },
  );

  test('listPosts follows next_cursor', () async {
    final store = MemoryTokenStore()
      ..tokens = const StoredTokens(
        accessToken: 'access',
        refreshToken: 'refresh',
      );
    final api = HttpCoupleApi(
      baseUrl: 'http://example.test/v1',
      tokens: store,
      client: MockClient((request) async {
        expect(request.url.path, '/v1/posts');
        final cursor = request.url.queryParameters['cursor'];
        if (cursor == null) {
          return _json(200, {
            'items': [_postMap(id: 'p1', body: '较新')],
            'next_cursor': 'cursor-2',
          });
        }
        expect(cursor, 'cursor-2');
        return _json(200, {
          'items': [_postMap(id: 'p2', body: '较旧')],
          'next_cursor': null,
        });
      }),
      ownsClient: true,
    );
    final posts = await api.listPosts();
    expect(posts.map((post) => post.body), ['较新', '较旧']);
  });

  test('network failures become a Chinese NETWORK error', () async {
    final api = HttpCoupleApi(
      baseUrl: 'http://example.test/v1',
      tokens: MemoryTokenStore(),
      client: MockClient((request) async {
        throw http.ClientException('down');
      }),
      ownsClient: true,
    );
    await expectLater(
      api.login(email: 'a@example.com', password: 'password1'),
      throwsA(
        isA<ApiException>().having((error) => error.code, 'code', 'NETWORK'),
      ),
    );
  });
}

http.Response _json(int status, Object? body) {
  return http.Response(
    body == null ? '' : jsonEncode(body),
    status,
    headers: {'content-type': 'application/json'},
  );
}

Map<String, dynamic> _userMap() {
  return {
    'id': '11111111-1111-1111-1111-111111111111',
    'email': 'a@example.com',
    'display_name': '甲',
    'avatar_url': null,
    'created_at': '2026-10-31T00:00:00Z',
  };
}

Map<String, dynamic> _authMap({
  String access = 'access',
  String refresh = 'refresh',
}) {
  return {
    'access_token': access,
    'refresh_token': refresh,
    'token_type': 'Bearer',
    'expires_in': 3600,
    'user': _userMap(),
  };
}

Map<String, dynamic> _spaceMap({required int count}) {
  return {
    'id': '22222222-2222-2222-2222-222222222222',
    'member_count': count,
    'members': [
      {
        'user_id': _userMap()['id'],
        'display_name': '甲',
        'avatar_url': null,
        'joined_at': '2026-10-31T00:00:00Z',
      },
      if (count >= 2)
        {
          'user_id': '33333333-3333-3333-3333-333333333333',
          'display_name': '乙',
          'avatar_url': null,
          'joined_at': '2026-10-31T00:05:00Z',
        },
    ],
    'created_at': '2026-10-31T00:00:00Z',
  };
}

Map<String, dynamic> _postMap({String id = 'post-1', String body = '今天天气真好'}) {
  return {
    'id': id,
    'space_id': _spaceMap(count: 2)['id'],
    'author': _userMap(),
    'body': body,
    'media': [
      {
        'id': 'media-1',
        'url': 'http://cdn.test/a.jpg',
        'content_type': 'image/jpeg',
        'sort_order': 0,
      },
    ],
    'location': null,
    'like_count': 0,
    'liked_by_me': false,
    'comment_count': 0,
    'created_at': '2026-10-31T01:00:00Z',
    'updated_at': '2026-10-31T01:00:00Z',
  };
}

class _FlowApi {
  _FlowApi() {
    client = MockClient(_handle);
  }

  late final http.Client client;
  final store = MemoryTokenStore();
  final seen = <String>[];
  String? postBody;
  Map<String, dynamic>? anniversaryBody;
  var liked = false;
  var posted = false;

  Future<http.Response> _handle(http.Request request) async {
    final key = '${request.method} ${request.url}';
    final path = '${request.method} ${request.url.path}';
    seen.add(request.url.host == 'uploads.test' ? key : path);
    if (request.url.path == '/v1/auth/login') {
      final body = jsonDecode(request.body) as Map<String, dynamic>;
      expect(body['email'], 'a@example.com');
      expect(body['password'], 'password1');
      return _json(200, _authMap());
    }
    if (request.url.host != 'uploads.test') {
      expect(request.headers['authorization'], 'Bearer access');
    }
    if (request.url.path == '/v1/auth/me') {
      return _json(200, {
        'user': _userMap(),
        'pairing': posted || liked
            ? {
                'status': 'paired',
                'space_id': _spaceMap(count: 2)['id'],
                'member_count': 2,
              }
            : {'status': 'unpaired', 'space_id': null, 'member_count': null},
      });
    }
    if (request.url.path == '/v1/pairing/invites' && request.method == 'POST') {
      return _json(201, {
        'invite_code': 'A3K9M2',
        'expires_at': '2026-11-01T00:00:00Z',
        'space': _spaceMap(count: 1),
      });
    }
    if (request.url.path == '/v1/pairing/invites/accept') {
      expect(jsonDecode(request.body)['invite_code'], 'a3k9m2');
      return _json(200, _spaceMap(count: 2));
    }
    if (request.url.path == '/v1/media/uploads') {
      final body = jsonDecode(request.body) as Map<String, dynamic>;
      expect(body['content_type'], 'image/jpeg');
      expect(body['byte_size'], 3);
      return _json(201, {
        'media_id': 'media-1',
        'upload_url': 'http://uploads.test/obj',
        'upload_method': 'PUT',
        'upload_headers': {'Content-Type': 'image/jpeg'},
        'expires_at': '2026-10-31T01:15:00Z',
      });
    }
    if (request.url.host == 'uploads.test') {
      expect(request.bodyBytes, [1, 2, 3]);
      expect(request.headers['content-type'], 'image/jpeg');
      return http.Response('', 200);
    }
    if (request.url.path == '/v1/posts' && request.method == 'POST') {
      postBody = request.body;
      posted = true;
      return _json(201, _postMap());
    }
    if (request.url.path == '/v1/posts' && request.method == 'GET') {
      final item = _postMap();
      item['liked_by_me'] = liked;
      item['like_count'] = liked ? 1 : 0;
      return _json(200, {
        'items': posted ? [item] : <Object>[],
        'next_cursor': null,
      });
    }
    if (request.url.path.endsWith('/likes') && request.method == 'PUT') {
      liked = true;
      return _json(200, {
        'post_id': 'post-1',
        'liked_by_me': true,
        'like_count': 1,
      });
    }
    if (request.url.path == '/v1/checkins' && request.method == 'POST') {
      expect(jsonDecode(request.body)['name'], '没有这家店xyz');
      return _json(201, _checkinMap());
    }
    if (request.url.path == '/v1/checkins') {
      return _json(200, {
        'items':
            request.method == 'GET' &&
                seen.where((call) => call == 'POST /v1/checkins').isNotEmpty
            ? [_checkinMap()]
            : <Object>[],
        'next_cursor': null,
      });
    }
    if (request.url.path == '/v1/anniversaries' && request.method == 'POST') {
      anniversaryBody = jsonDecode(request.body) as Map<String, dynamic>;
      return _json(201, _anniversaryMap());
    }
    if (request.url.path == '/v1/anniversaries') {
      final created = seen.any((call) => call == 'POST /v1/anniversaries');
      return _json(200, {
        'items': created ? [_anniversaryMap()] : <Object>[],
        'next_cursor': null,
      });
    }
    if (request.url.path == '/v1/devices' && request.method == 'POST') {
      final body = jsonDecode(request.body) as Map<String, dynamic>;
      expect(body['platform'], 'android');
      return _json(200, {
        'id': 'device-1',
        'platform': 'android',
        'token': body['token'],
        'updated_at': '2026-10-31T02:00:00Z',
      });
    }
    fail('unexpected ${request.method} ${request.url}');
  }
}

Map<String, dynamic> _checkinMap() {
  return {
    'id': 'checkin-1',
    'space_id': _spaceMap(count: 2)['id'],
    'creator': _userMap(),
    'name': '没有这家店xyz',
    'address': null,
    'lat': null,
    'lng': null,
    'place_id': null,
    'checked_in_at': '2026-10-31T03:00:00Z',
    'created_at': '2026-10-31T03:00:00Z',
  };
}

Map<String, dynamic> _anniversaryMap() {
  return {
    'id': 'ann-1',
    'space_id': _spaceMap(count: 2)['id'],
    'creator': _userMap(),
    'title': '在一起',
    'event_date': '2024-05-20',
    'recurrence': 'yearly',
    'remind': true,
    'remind_days_before': 0,
    'remind_local_time': '09:00',
    'timezone': 'Asia/Shanghai',
    'created_at': '2026-10-31T04:00:00Z',
    'updated_at': '2026-10-31T04:00:00Z',
  };
}
