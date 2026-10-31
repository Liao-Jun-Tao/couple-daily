import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

import 'package:couple_daily/core/api/api_parse.dart';
import 'package:couple_daily/core/api/http_api.dart';
import 'package:couple_daily/core/api/secure_token_store.dart';
import 'package:couple_daily/core/api/token_store.dart';
import 'package:couple_daily/core/config/api_config.dart';
import 'package:couple_daily/core/constants.dart';
import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/mock/mock_backend.dart';
import 'package:couple_daily/core/models/models.dart';
import 'package:couple_daily/core/notifications/notification_gateway.dart';
import 'package:couple_daily/core/notifications/reminder_stub.dart';

/// Signed-in user plus either the in-memory mock or the real `/v1` API.
///
/// Screens talk to this. Pass [backend] or `useMock: true` to stay offline.
/// The default follows [ApiConfig.fromEnvironment] (`USE_MOCK` defaults to
/// false).
class AppSession extends ChangeNotifier {
  AppSession({
    MockBackend? backend,
    bool? useMock,
    ApiConfig? config,
    TokenStore? tokenStore,
    http.Client? httpClient,
    NotificationGateway? notifications,
    ReminderStub? reminders,
  }) : notifications = notifications ?? NotificationGateway(),
       reminders = reminders ?? ReminderStub(),
       _config = config ?? ApiConfig.fromEnvironment {
    final mock = backend != null || (useMock ?? _config.useMock);
    if (mock) {
      _backend = backend ?? MockBackend();
      _api = null;
      _tokens = null;
      _postsReady = true;
    } else {
      _backend = null;
      _tokens = tokenStore ?? SecureTokenStore();
      _api = HttpCoupleApi(
        baseUrl: _config.normalizedBaseUrl,
        tokens: _tokens!,
        client: httpClient ?? http.Client(),
        ownsClient: httpClient == null,
        onSessionExpired: _onSessionExpired,
      );
    }
  }

  final NotificationGateway notifications;
  final ReminderStub reminders;
  final ApiConfig _config;

  late final MockBackend? _backend;
  late final HttpCoupleApi? _api;
  late final TokenStore? _tokens;

  bool ready = false;
  bool pendingPairingDone = false;
  bool _postsReady = false;
  String? userId;
  String? accessToken;
  String? refreshToken;
  String? startupError;
  String? loadError;

  UserPublic? _user;
  PairingSnapshot _pairing = PairingSnapshot.unpaired;
  SpaceSnapshot? _space;
  List<Post> _posts = const [];
  final Map<String, List<Comment>> _comments = {};
  List<RestaurantCheckin> _checkins = const [];
  List<Anniversary> _anniversaries = const [];
  List<DeviceToken> _devices = const [];

  bool get useMock => _backend != null;

  bool get postsReady => _postsReady;

  /// In-memory backend. Only available when [useMock] is true.
  MockBackend get backend {
    final backend = _backend;
    if (backend == null) {
      throw StateError('Mock backend is only available in mock mode');
    }
    return backend;
  }

  bool get isLoggedIn => userId != null;

  UserPublic? get user {
    final backend = _backend;
    if (backend != null) {
      final id = userId;
      if (id == null) return null;
      return backend.userById(id);
    }
    return _user;
  }

  PairingSnapshot get pairing {
    final backend = _backend;
    if (backend != null) {
      final id = userId;
      if (id == null) return PairingSnapshot.unpaired;
      return backend.pairingOf(id);
    }
    return _pairing;
  }

  bool get isPaired => pairing.status == PairingStatus.paired;

  SpaceSnapshot? get space {
    final backend = _backend;
    if (backend != null) {
      final id = userId;
      if (id == null) return null;
      return backend.spaceOf(id);
    }
    return _space;
  }

  /// Splash calls this after the hold. Mock mode is local; real mode
  /// restores tokens and sends an expired refresh back to auth.
  Future<void> bootstrap() async {
    if (ready) return;
    try {
      if (_api != null) await _restore();
    } on Object catch (error) {
      startupError = messageOf(error);
      _resetMemory();
    }
    ready = true;
    notifyListeners();
  }

  void markReady() {
    if (ready) return;
    ready = true;
    notifyListeners();
  }

  void clearPairingDoneFlag() {
    pendingPairingDone = false;
  }

  Future<void> register({
    required String email,
    required String password,
    required String displayName,
  }) async {
    final backend = _backend;
    if (backend != null) {
      _applyAuth(
        backend.register(
          email: email,
          password: password,
          displayName: displayName,
        ),
      );
      return;
    }
    final result = await _api!.register(
      email: email,
      password: password,
      displayName: displayName,
    );
    await _adopt(result);
  }

  Future<void> login({required String email, required String password}) async {
    final backend = _backend;
    if (backend != null) {
      _applyAuth(backend.login(email: email, password: password));
      return;
    }
    final result = await _api!.login(email: email, password: password);
    await _adopt(result);
  }

  Future<void> logout() async {
    if (_api != null) {
      for (final device in List<DeviceToken>.of(_devices)) {
        try {
          await _api.deleteDevice(device.token);
        } on Object {
          // Logging out still finishes when device unregister fails.
        }
      }
      try {
        await _tokens?.clear();
      } on Object {
        // A keystore failure must not trap the user on the signed-in shell.
      }
    }
    _resetMemory();
    startupError = null;
    notifyListeners();
  }

  Future<InviteTicket> createInvite() async {
    final backend = _backend;
    if (backend != null) {
      final ticket = backend.createInvite(_requireUser());
      notifyListeners();
      return ticket;
    }
    final ticket = await _api!.createInvite();
    _space = ticket.space;
    _pairing = PairingSnapshot(
      status: ticket.space.memberCount >= 2
          ? PairingStatus.paired
          : PairingStatus.pendingInvite,
      spaceId: ticket.space.id,
      memberCount: ticket.space.memberCount,
      inviteCode: ticket.inviteCode,
      inviteExpiresAt: ticket.expiresAt,
    );
    final id = userId;
    if (id != null) {
      try {
        await _tokens!.writeInvite(
          StoredInvite(
            userId: id,
            code: ticket.inviteCode,
            expiresAt: ticket.expiresAt,
          ),
        );
      } on Object {
        // The code is still on screen for this session.
      }
    }
    notifyListeners();
    return ticket;
  }

  Future<void> acceptInvite(String code) async {
    final backend = _backend;
    if (backend != null) {
      backend.acceptInvite(userId: _requireUser(), code: code);
      pendingPairingDone = true;
      notifyListeners();
      return;
    }
    final space = await _api!.acceptInvite(code);
    _applySpace(space, pairingDone: space.memberCount >= 2);
    try {
      await _tokens?.clearInvite();
    } on Object {
      // Pairing already succeeded.
    }
    if (space.memberCount >= 2) await _loadPairedCaches();
    notifyListeners();
  }

  List<Post> posts() {
    final backend = _backend;
    if (backend != null) return backend.listPosts(_requireUser());
    return _posts;
  }

  Future<void> refreshPosts() async {
    final backend = _backend;
    if (backend != null) return;
    try {
      final loaded = await _api!.space();
      if (loaded != null) _space = loaded;
      _posts = await _api.listPosts();
      loadError = null;
    } on ApiException catch (error) {
      loadError = error.message;
    }
    _postsReady = true;
    notifyListeners();
  }

  Future<void> createPost({
    required String body,
    required List<MediaDraft> media,
    LocationInput? location,
  }) async {
    final backend = _backend;
    if (backend != null) {
      backend.createPost(
        userId: _requireUser(),
        body: body,
        media: media,
        location: location,
      );
      notifyListeners();
      return;
    }
    if (body.trim().isEmpty && media.isEmpty) {
      throw ApiException('EMPTY_POST', chineseApiMessage('EMPTY_POST', ''));
    }
    if (media.length > maxImagesPerPost) {
      throw ApiException('VALIDATION_ERROR', '最多 9 张图片');
    }
    final ids = <String>[];
    for (final item in media) {
      if (item.byteSize > maxImageBytes ||
          !allowedImageTypes.contains(item.contentType)) {
        throw ApiException(
          'MEDIA_INVALID',
          chineseApiMessage('MEDIA_INVALID', ''),
        );
      }
      ids.add(await _api!.uploadMedia(item));
    }
    final created = await _api!.createPost(
      body: body,
      mediaIds: ids,
      location: location,
    );
    try {
      _posts = await _api.listPosts();
      loadError = null;
    } on ApiException {
      _posts = [created, ..._posts.where((post) => post.id != created.id)];
    }
    _postsReady = true;
    notifyListeners();
  }

  Future<void> deletePost(String postId) async {
    final backend = _backend;
    if (backend != null) {
      backend.deletePost(userId: _requireUser(), postId: postId);
      notifyListeners();
      return;
    }
    await _api!.deletePost(postId);
    _posts = [
      for (final post in _posts)
        if (post.id != postId) post,
    ];
    _comments.remove(postId);
    notifyListeners();
  }

  Future<void> like(String postId) async {
    final backend = _backend;
    if (backend != null) {
      backend.like(userId: _requireUser(), postId: postId);
      notifyListeners();
      return;
    }
    _applyLike(await _api!.like(postId));
    notifyListeners();
  }

  Future<void> unlike(String postId) async {
    final backend = _backend;
    if (backend != null) {
      backend.unlike(userId: _requireUser(), postId: postId);
      notifyListeners();
      return;
    }
    _applyLike(await _api!.unlike(postId));
    notifyListeners();
  }

  List<Comment> comments(String postId) {
    final backend = _backend;
    if (backend != null) {
      return backend.listComments(userId: _requireUser(), postId: postId);
    }
    return _comments[postId] ?? const [];
  }

  Future<void> refreshComments(String postId) async {
    if (_backend != null) return;
    _comments[postId] = await _api!.listComments(postId);
    notifyListeners();
  }

  Future<void> addComment(String postId, String body) async {
    final backend = _backend;
    if (backend != null) {
      backend.addComment(userId: _requireUser(), postId: postId, body: body);
      notifyListeners();
      return;
    }
    await _api!.addComment(postId, body);
    _comments[postId] = await _api.listComments(postId);
    _bumpCommentCount(postId, _comments[postId]!.length);
    notifyListeners();
  }

  Future<void> deleteComment(String commentId) async {
    final backend = _backend;
    if (backend != null) {
      backend.deleteComment(userId: _requireUser(), commentId: commentId);
      notifyListeners();
      return;
    }
    await _api!.deleteComment(commentId);
    String? postId;
    for (final entry in _comments.entries) {
      if (entry.value.any((comment) => comment.id == commentId)) {
        postId = entry.key;
      }
    }
    if (postId != null) {
      _comments[postId] = await _api.listComments(postId);
      _bumpCommentCount(postId, _comments[postId]!.length);
    }
    notifyListeners();
  }

  List<RestaurantCheckin> checkins() {
    final backend = _backend;
    if (backend != null) return backend.listCheckins(_requireUser());
    return _checkins;
  }

  Future<void> refreshCheckins() async {
    if (_backend != null) return;
    _checkins = await _api!.listCheckins();
    notifyListeners();
  }

  Future<void> createCheckin({
    required String name,
    String? address,
    double? lat,
    double? lng,
    String? placeId,
  }) async {
    final backend = _backend;
    if (backend != null) {
      backend.createCheckin(
        userId: _requireUser(),
        name: name,
        address: address,
        lat: lat,
        lng: lng,
        placeId: placeId,
      );
      notifyListeners();
      return;
    }
    await _api!.createCheckin(
      name: name,
      address: address,
      lat: lat,
      lng: lng,
      placeId: placeId,
    );
    _checkins = await _api.listCheckins();
    notifyListeners();
  }

  Future<void> deleteCheckin(String checkinId) async {
    final backend = _backend;
    if (backend != null) {
      backend.deleteCheckin(userId: _requireUser(), checkinId: checkinId);
      notifyListeners();
      return;
    }
    await _api!.deleteCheckin(checkinId);
    _checkins = [
      for (final item in _checkins)
        if (item.id != checkinId) item,
    ];
    notifyListeners();
  }

  List<Anniversary> anniversaries() {
    final backend = _backend;
    if (backend != null) return backend.listAnniversaries(_requireUser());
    return _anniversaries;
  }

  Future<void> refreshAnniversaries() async {
    if (_backend != null) return;
    _anniversaries = await _api!.listAnniversaries();
    notifyListeners();
  }

  Future<Anniversary> saveAnniversary({
    String? id,
    required String title,
    required String eventDate,
    required String recurrence,
    required bool remind,
  }) async {
    final backend = _backend;
    if (backend != null) {
      final user = _requireUser();
      final Anniversary saved;
      if (id == null) {
        saved = backend.createAnniversary(
          userId: user,
          title: title,
          eventDate: eventDate,
          recurrence: recurrence,
          remind: remind,
        );
      } else {
        saved = backend.updateAnniversary(
          userId: user,
          anniversaryId: id,
          title: title,
          eventDate: eventDate,
          recurrence: recurrence,
          remind: remind,
        );
      }
      reminders.sync(saved);
      notifyListeners();
      return saved;
    }
    final Anniversary saved;
    if (id == null) {
      saved = await _api!.createAnniversary(
        title: title,
        eventDate: eventDate,
        recurrence: recurrence,
        remind: remind,
      );
    } else {
      saved = await _api!.updateAnniversary(
        anniversaryId: id,
        title: title,
        eventDate: eventDate,
        recurrence: recurrence,
        remind: remind,
      );
    }
    reminders.sync(saved);
    _anniversaries = await _api.listAnniversaries();
    notifyListeners();
    return saved;
  }

  Future<void> deleteAnniversary(String anniversaryId) async {
    final backend = _backend;
    if (backend != null) {
      backend.deleteAnniversary(
        userId: _requireUser(),
        anniversaryId: anniversaryId,
      );
      reminders.cancel(anniversaryId);
      notifyListeners();
      return;
    }
    await _api!.deleteAnniversary(anniversaryId);
    reminders.cancel(anniversaryId);
    _anniversaries = [
      for (final item in _anniversaries)
        if (item.id != anniversaryId) item,
    ];
    notifyListeners();
  }

  List<DeviceToken> devices() {
    final backend = _backend;
    if (backend != null) return backend.listDevices(_requireUser());
    return _devices;
  }

  Future<void> registerDevice({
    required String platform,
    required String token,
  }) async {
    final backend = _backend;
    if (backend != null) {
      backend.registerDevice(
        userId: _requireUser(),
        platform: platform,
        token: token,
      );
      notifyListeners();
      return;
    }
    final device = await _api!.registerDevice(platform: platform, token: token);
    _devices = [
      for (final item in _devices)
        if (item.token != device.token) item,
      device,
    ];
    notifyListeners();
  }

  @override
  void dispose() {
    _api?.close();
    super.dispose();
  }

  Future<void> _restore() async {
    final StoredTokens? stored;
    try {
      stored = await _tokens!.readTokens();
    } on Object {
      startupError = '无法读取本机登录状态，请重新登录';
      return;
    }
    if (stored == null) return;
    accessToken = stored.accessToken;
    refreshToken = stored.refreshToken;
    try {
      await _hydrate();
    } on ApiException catch (error) {
      if (_isAuthFailure(error)) {
        try {
          await _tokens.clear();
        } on Object {
          // Still drop the in-memory session.
        }
        _resetMemory();
        return;
      }
      if (userId == null) {
        startupError = error.message;
        accessToken = null;
        refreshToken = null;
        return;
      }
      loadError = error.message;
    }
  }

  Future<void> _adopt(AuthResult result) async {
    startupError = null;
    userId = result.user.id;
    _user = result.user;
    accessToken = result.accessToken;
    refreshToken = result.refreshToken;
    pendingPairingDone = false;
    try {
      await _hydrate();
    } on Object {
      userId = null;
      _user = null;
      accessToken = null;
      refreshToken = null;
      rethrow;
    }
    notifyListeners();
  }

  Future<void> _hydrate() async {
    final me = await _api!.me();
    _user = me.user;
    userId = me.user.id;
    var pairing = me.pairing;
    if (pairing.status == PairingStatus.pendingInvite) {
      StoredInvite? invite;
      try {
        invite = await _tokens!.readInvite();
      } on Object {
        invite = null;
      }
      if (invite != null && invite.userId == me.user.id) {
        pairing = PairingSnapshot(
          status: pairing.status,
          spaceId: pairing.spaceId ?? _space?.id,
          memberCount: pairing.memberCount,
          inviteCode: invite.code,
          inviteExpiresAt: invite.expiresAt,
        );
      }
    } else {
      try {
        await _tokens!.clearInvite();
      } on Object {
        // The in-memory pairing no longer shows a code.
      }
    }
    _pairing = pairing;
    if (pairing.status == PairingStatus.unpaired) {
      _space = null;
      _posts = const [];
      _checkins = const [];
      _anniversaries = const [];
      _postsReady = true;
      return;
    }
    try {
      _space = await _api.space();
    } on ApiException catch (error) {
      loadError = error.message;
    }
    if (pairing.status == PairingStatus.paired) {
      await _loadPairedCaches();
    } else {
      _postsReady = true;
    }
  }

  Future<void> _loadPairedCaches() async {
    try {
      _posts = await _api!.listPosts();
      loadError = null;
    } on ApiException catch (error) {
      loadError = error.message;
    }
    _postsReady = true;
    try {
      _checkins = await _api!.listCheckins();
    } on ApiException catch (error) {
      loadError ??= error.message;
    }
    try {
      _anniversaries = await _api!.listAnniversaries();
    } on ApiException catch (error) {
      loadError ??= error.message;
    }
  }

  void _applyAuth(AuthResult result) {
    userId = result.user.id;
    accessToken = result.accessToken;
    refreshToken = result.refreshToken;
    pendingPairingDone = false;
    startupError = null;
    notifyListeners();
  }

  void _applySpace(SpaceSnapshot space, {required bool pairingDone}) {
    _space = space;
    _pairing = PairingSnapshot(
      status: space.memberCount >= 2
          ? PairingStatus.paired
          : PairingStatus.pendingInvite,
      spaceId: space.id,
      memberCount: space.memberCount,
    );
    pendingPairingDone = pairingDone;
  }

  void _applyLike(LikeStatus status) {
    _posts = [
      for (final post in _posts)
        if (post.id == status.postId)
          Post(
            id: post.id,
            spaceId: post.spaceId,
            author: post.author,
            body: post.body,
            media: post.media,
            location: post.location,
            likeCount: status.likeCount,
            likedByMe: status.likedByMe,
            commentCount: post.commentCount,
            createdAt: post.createdAt,
            updatedAt: post.updatedAt,
          )
        else
          post,
    ];
  }

  void _bumpCommentCount(String postId, int count) {
    _posts = [
      for (final post in _posts)
        if (post.id == postId)
          Post(
            id: post.id,
            spaceId: post.spaceId,
            author: post.author,
            body: post.body,
            media: post.media,
            location: post.location,
            likeCount: post.likeCount,
            likedByMe: post.likedByMe,
            commentCount: count,
            createdAt: post.createdAt,
            updatedAt: post.updatedAt,
          )
        else
          post,
    ];
  }

  void _onSessionExpired() {
    _resetMemory();
    final tokens = _tokens;
    if (tokens != null) unawaited(tokens.clear());
    if (ready) notifyListeners();
  }

  void _resetMemory() {
    userId = null;
    _user = null;
    accessToken = null;
    refreshToken = null;
    pendingPairingDone = false;
    _pairing = PairingSnapshot.unpaired;
    _space = null;
    _posts = const [];
    _comments.clear();
    _checkins = const [];
    _anniversaries = const [];
    _devices = const [];
    loadError = null;
    _postsReady = _backend != null;
  }

  bool _isAuthFailure(ApiException error) {
    return error.code == 'UNAUTHORIZED' || error.code == 'TOKEN_INVALID';
  }

  String _requireUser() {
    final id = userId;
    if (id == null) throw StateError('No signed-in user');
    return id;
  }
}
