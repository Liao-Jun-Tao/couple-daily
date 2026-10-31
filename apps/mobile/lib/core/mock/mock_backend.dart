import 'dart:math';
import 'dart:typed_data';

import 'package:couple_daily/core/constants.dart';
import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/format.dart';
import 'package:couple_daily/core/ids.dart';
import 'package:couple_daily/core/models/models.dart';

/// In-memory stand-in for the couple-daily API.
///
/// Passwords stay in this process only. Nothing is sent to `backend/`.
class MockBackend {
  MockBackend({DateTime Function()? clock, Random? random})
    : _clock = clock ?? DateTime.now,
      _random = random ?? Random.secure();

  final DateTime Function() _clock;
  final Random _random;
  int _sequence = 0;

  final Map<String, _User> _usersById = {};
  final Map<String, _User> _usersByEmail = {};
  final Map<String, _Refresh> _refreshByToken = {};
  final Map<String, _Space> _spaces = {};
  final Map<String, String> _spaceIdByUser = {};
  final Map<String, _Invite> _invitesByCode = {};
  final Map<String, _Post> _posts = {};
  final Map<String, _Comment> _comments = {};
  final Map<String, _Checkin> _checkins = {};
  final Map<String, _Anniversary> _anniversaries = {};
  final Map<String, _Device> _devices = {};
  final Set<String> _likes = {};

  DateTime now() => _clock().toUtc();

  AuthResult register({
    required String email,
    required String password,
    required String displayName,
  }) {
    final normalized = _email(email);
    final secret = _password(password);
    final name = _displayName(displayName);
    if (_usersByEmail.containsKey(normalized)) {
      throw ApiException('EMAIL_TAKEN', '这个邮箱已经注册');
    }
    final user = _User(
      id: newId(_random),
      email: normalized,
      password: secret,
      displayName: name,
      createdAt: _stamp(),
    );
    _usersById[user.id] = user;
    _usersByEmail[user.email] = user;
    return _tokensFor(user);
  }

  AuthResult login({required String email, required String password}) {
    final user = _usersByEmail[_email(email)];
    if (user == null || user.password != password) {
      throw ApiException('INVALID_CREDENTIALS', '邮箱或密码不正确');
    }
    return _tokensFor(user);
  }

  AuthResult refresh({required String refreshToken}) {
    final row = _refreshByToken[refreshToken];
    if (row == null || !now().isBefore(row.expiresAt)) {
      throw ApiException('INVALID_CREDENTIALS', '登录已失效，请重新登录');
    }
    final user = _usersById[row.userId];
    if (user == null) {
      throw ApiException('INVALID_CREDENTIALS', '登录已失效，请重新登录');
    }
    _refreshByToken.remove(refreshToken);
    return _tokensFor(user);
  }

  UserPublic? userById(String id) {
    final user = _usersById[id];
    if (user == null) return null;
    return user.view();
  }

  PairingSnapshot pairingOf(String userId) {
    _requireUser(userId);
    final spaceId = _spaceIdByUser[userId];
    if (spaceId == null) return PairingSnapshot.unpaired;
    final space = _spaces[spaceId]!;
    if (space.memberCount >= 2) {
      return PairingSnapshot(
        status: PairingStatus.paired,
        spaceId: space.id,
        memberCount: space.memberCount,
      );
    }
    final invite = _latestInvite(space.id);
    return PairingSnapshot(
      status: PairingStatus.pendingInvite,
      spaceId: space.id,
      memberCount: space.memberCount,
      inviteCode: invite?.code,
      inviteExpiresAt: invite?.expiresAt,
    );
  }

  SpaceSnapshot? spaceOf(String userId) {
    final spaceId = _spaceIdByUser[userId];
    if (spaceId == null) return null;
    return _spaces[spaceId]!.view(this);
  }

  String inviteStatus(String code) {
    final invite = _invitesByCode[_normalizeCode(code) ?? ''];
    if (invite == null) {
      throw ApiException('INVITE_NOT_FOUND', '邀请码不存在');
    }
    return invite.status;
  }

  InviteTicket createInvite(String userId) {
    _requireUser(userId);
    final existingSpaceId = _spaceIdByUser[userId];
    if (existingSpaceId == null) {
      final space = _Space(
        id: newId(_random),
        createdAt: _stamp(),
        members: [
          _Membership(userId: userId, joinedAt: _stamp()),
        ],
      );
      _spaces[space.id] = space;
      _spaceIdByUser[userId] = space.id;
      return _issueInvite(spaceId: space.id, inviterId: userId);
    }
    final space = _spaces[existingSpaceId]!;
    if (space.memberCount >= 2) {
      throw ApiException('ALREADY_PAIRED', '你已经在一个空间里');
    }
    final current = _latestInvite(space.id);
    if (current != null &&
        current.status == 'pending' &&
        now().isBefore(current.expiresAt)) {
      return InviteTicket(
        inviteCode: current.code,
        expiresAt: current.expiresAt,
        space: space.view(this),
      );
    }
    if (current != null && current.status == 'pending') {
      current.status = 'expired';
    }
    return _issueInvite(spaceId: space.id, inviterId: userId);
  }

  SpaceSnapshot acceptInvite({required String userId, required String code}) {
    _requireUser(userId);
    final normalized = _normalizeCode(code);
    if (normalized == null) {
      throw ApiException('INVALID_INVITE_CODE', '邀请码须为 6 位字母或数字');
    }
    final invite = _invitesByCode[normalized];
    if (invite == null) {
      throw ApiException('INVITE_NOT_FOUND', '邀请码不存在');
    }
    if (!now().isBefore(invite.expiresAt) || invite.status == 'expired') {
      invite.status = 'expired';
      throw ApiException('INVITE_EXPIRED', '邀请码已过期');
    }
    if (invite.status == 'revoked') {
      throw ApiException('INVITE_REVOKED', '邀请码已撤销');
    }
    if (invite.status == 'accepted') {
      throw ApiException('INVITE_USED', '邀请码已使用');
    }
    final space = _spaces[invite.spaceId]!;
    if (space.memberCount >= 2) {
      throw ApiException('SPACE_FULL', '这个空间已经有两个人');
    }
    if (_spaceIdByUser.containsKey(userId)) {
      throw ApiException('ALREADY_PAIRED', '你已经在一个空间里');
    }
    space.members.add(_Membership(userId: userId, joinedAt: _stamp()));
    invite
      ..status = 'accepted'
      ..acceptedByUserId = userId
      ..acceptedAt = _stamp();
    _spaceIdByUser[userId] = space.id;
    return space.view(this);
  }

  List<Post> listPosts(String userId) {
    final spaceId = _requireSpace(userId);
    final rows = _posts.values
        .where((post) => post.spaceId == spaceId && post.deletedAt == null)
        .toList()
      ..sort((a, b) {
        final byTime = b.createdAt.compareTo(a.createdAt);
        if (byTime != 0) return byTime;
        return b.id.compareTo(a.id);
      });
    return rows.map((post) => post.view(this, userId)).toList(growable: false);
  }

  Post createPost({
    required String userId,
    required String body,
    required List<MediaDraft> media,
    LocationInput? location,
  }) {
    final spaceId = _requireSpace(userId);
    final trimmed = body.trim();
    if (trimmed.isEmpty && media.isEmpty) {
      throw ApiException('EMPTY_POST', '写点文字，或至少加一张图片');
    }
    if (trimmed.length > 5000) {
      throw ApiException('BODY_TOO_LONG', '正文不能超过 5000 字');
    }
    if (media.length > maxImagesPerPost) {
      throw ApiException('TOO_MANY_IMAGES', '最多 9 张图片');
    }
    final storedMedia = <_Media>[];
    for (var index = 0; index < media.length; index++) {
      final item = media[index];
      if (!allowedImageTypes.contains(item.contentType)) {
        throw ApiException('UNSUPPORTED_MEDIA', '仅支持 jpeg、png、heic、webp');
      }
      if (item.byteSize > maxImageBytes) {
        throw ApiException('IMAGE_TOO_LARGE', '单张图片不能超过 10MB');
      }
      storedMedia.add(
        _Media(
          id: newId(_random),
          bytes: Uint8List.fromList(item.bytes),
          contentType: item.contentType,
          sortOrder: index,
        ),
      );
    }
    final storedLocation = _location(location);
    final nowStamp = _stamp();
    final post = _Post(
      id: newId(_random),
      spaceId: spaceId,
      authorUserId: userId,
      body: trimmed,
      media: storedMedia,
      location: storedLocation,
      createdAt: nowStamp,
      updatedAt: nowStamp,
    );
    _posts[post.id] = post;
    return post.view(this, userId);
  }

  void deletePost({required String userId, required String postId}) {
    final spaceId = _requireSpace(userId);
    final post = _posts[postId];
    if (post == null || post.spaceId != spaceId || post.deletedAt != null) {
      throw ApiException('NOT_FOUND', '动态不存在');
    }
    if (post.authorUserId != userId) {
      throw ApiException('NOT_AUTHOR', '只有作者可以删除');
    }
    post.deletedAt = _stamp();
  }

  void like({required String userId, required String postId}) {
    final post = _visiblePost(userId, postId);
    _likes.add('${post.id}|$userId');
  }

  void unlike({required String userId, required String postId}) {
    final post = _visiblePost(userId, postId);
    _likes.remove('${post.id}|$userId');
  }

  List<Comment> listComments({required String userId, required String postId}) {
    _visiblePost(userId, postId);
    final rows = _comments.values
        .where(
          (comment) => comment.postId == postId && comment.deletedAt == null,
        )
        .toList()
      ..sort((a, b) => a.createdAt.compareTo(b.createdAt));
    return rows.map((comment) => comment.view(this)).toList(growable: false);
  }

  Comment addComment({
    required String userId,
    required String postId,
    required String body,
  }) {
    final post = _visiblePost(userId, postId);
    final trimmed = body.trim();
    if (trimmed.isEmpty) {
      throw ApiException('EMPTY_COMMENT', '评论不能为空');
    }
    if (trimmed.length > 2000) {
      throw ApiException('COMMENT_TOO_LONG', '评论不能超过 2000 字');
    }
    final comment = _Comment(
      id: newId(_random),
      spaceId: post.spaceId,
      postId: post.id,
      authorUserId: userId,
      body: trimmed,
      createdAt: _stamp(),
    );
    _comments[comment.id] = comment;
    return comment.view(this);
  }

  void deleteComment({required String userId, required String commentId}) {
    final spaceId = _requireSpace(userId);
    final comment = _comments[commentId];
    if (comment == null ||
        comment.spaceId != spaceId ||
        comment.deletedAt != null) {
      throw ApiException('NOT_FOUND', '评论不存在');
    }
    if (comment.authorUserId != userId) {
      throw ApiException('NOT_AUTHOR', '只有作者可以删除');
    }
    comment.deletedAt = _stamp();
  }

  List<RestaurantCheckin> listCheckins(String userId) {
    final spaceId = _requireSpace(userId);
    final rows = _checkins.values
        .where((row) => row.spaceId == spaceId && row.deletedAt == null)
        .toList()
      ..sort((a, b) => b.checkedInAt.compareTo(a.checkedInAt));
    return rows.map((row) => row.view(this)).toList(growable: false);
  }

  RestaurantCheckin createCheckin({
    required String userId,
    required String name,
    String? address,
    double? lat,
    double? lng,
    String? placeId,
    DateTime? checkedInAt,
  }) {
    final spaceId = _requireSpace(userId);
    final trimmed = name.trim();
    if (trimmed.isEmpty || trimmed.length > 128) {
      throw ApiException('CHECKIN_NAME_REQUIRED', '店名不能为空，且不超过 128 字');
    }
    if ((lat == null) != (lng == null)) {
      throw ApiException('INVALID_LOCATION', '经纬度需要同时填写');
    }
    final row = _Checkin(
      id: newId(_random),
      spaceId: spaceId,
      creatorUserId: userId,
      name: trimmed,
      address: _blankToNull(address),
      lat: lat,
      lng: lng,
      placeId: _blankToNull(placeId),
      checkedInAt: checkedInAt?.toUtc() ?? _stamp(),
      createdAt: _stamp(),
    );
    _checkins[row.id] = row;
    return row.view(this);
  }

  void deleteCheckin({required String userId, required String checkinId}) {
    final spaceId = _requireSpace(userId);
    final row = _checkins[checkinId];
    if (row == null || row.spaceId != spaceId || row.deletedAt != null) {
      throw ApiException('NOT_FOUND', '打卡不存在');
    }
    if (row.creatorUserId != userId) {
      throw ApiException('NOT_AUTHOR', '只有作者可以删除');
    }
    row.deletedAt = _stamp();
  }

  List<Anniversary> listAnniversaries(String userId) {
    final spaceId = _requireSpace(userId);
    final rows = _anniversaries.values
        .where((row) => row.spaceId == spaceId && row.deletedAt == null)
        .toList()
      ..sort((a, b) => a.eventDate.compareTo(b.eventDate));
    return rows.map((row) => row.view(this)).toList(growable: false);
  }

  Anniversary createAnniversary({
    required String userId,
    required String title,
    required String eventDate,
    String recurrence = 'yearly',
    bool remind = true,
  }) {
    final spaceId = _requireSpace(userId);
    final storedTitle = _title(title);
    final storedDate = _eventDate(eventDate);
    final storedRecurrence = _recurrence(recurrence);
    final stamp = _stamp();
    final row = _Anniversary(
      id: newId(_random),
      spaceId: spaceId,
      creatorUserId: userId,
      title: storedTitle,
      eventDate: storedDate,
      recurrence: storedRecurrence,
      remind: remind,
      remindDaysBefore: defaultRemindDaysBefore,
      remindLocalTime: defaultRemindLocalTime,
      timezone: defaultTimezone,
      createdAt: stamp,
      updatedAt: stamp,
    );
    _anniversaries[row.id] = row;
    return row.view(this);
  }

  Anniversary updateAnniversary({
    required String userId,
    required String anniversaryId,
    required String title,
    required String eventDate,
    required String recurrence,
    required bool remind,
  }) {
    final spaceId = _requireSpace(userId);
    final row = _anniversaries[anniversaryId];
    if (row == null || row.spaceId != spaceId || row.deletedAt != null) {
      throw ApiException('NOT_FOUND', '纪念日不存在');
    }
    if (row.creatorUserId != userId) {
      throw ApiException('NOT_AUTHOR', '只有作者可以修改');
    }
    row
      ..title = _title(title)
      ..eventDate = _eventDate(eventDate)
      ..recurrence = _recurrence(recurrence)
      ..remind = remind
      ..remindLocalTime = defaultRemindLocalTime
      ..timezone = defaultTimezone
      ..remindDaysBefore = defaultRemindDaysBefore
      ..updatedAt = _stamp();
    return row.view(this);
  }

  void deleteAnniversary({
    required String userId,
    required String anniversaryId,
  }) {
    final spaceId = _requireSpace(userId);
    final row = _anniversaries[anniversaryId];
    if (row == null || row.spaceId != spaceId || row.deletedAt != null) {
      throw ApiException('NOT_FOUND', '纪念日不存在');
    }
    if (row.creatorUserId != userId) {
      throw ApiException('NOT_AUTHOR', '只有作者可以删除');
    }
    row.deletedAt = _stamp();
  }

  List<DeviceToken> listDevices(String userId) {
    _requireUser(userId);
    return _devices.values
        .where((row) => row.userId == userId)
        .map((row) => row.view())
        .toList(growable: false);
  }

  DeviceToken registerDevice({
    required String userId,
    required String platform,
    required String token,
  }) {
    _requireUser(userId);
    if (platform != 'ios' && platform != 'android') {
      throw ApiException('INVALID_PLATFORM', '推送平台只能是 ios 或 android');
    }
    final trimmed = token.trim();
    if (trimmed.isEmpty || trimmed.length > 512) {
      throw ApiException('INVALID_TOKEN', '设备 token 无效');
    }
    final existing = _devices.values.where(
      (row) => row.userId == userId && row.token == trimmed,
    );
    final stamp = _stamp();
    if (existing.isNotEmpty) {
      final row = existing.first
        ..platform = platform
        ..updatedAt = stamp;
      return row.view();
    }
    final row = _Device(
      id: newId(_random),
      userId: userId,
      platform: platform,
      token: trimmed,
      updatedAt: stamp,
    );
    _devices[row.id] = row;
    return row.view();
  }

  InviteTicket _issueInvite({
    required String spaceId,
    required String inviterId,
  }) {
    final created = now();
    final invite = _Invite(
      code: _newCode(),
      inviterUserId: inviterId,
      spaceId: spaceId,
      expiresAt: created.add(inviteTtl),
    );
    _invitesByCode[invite.code] = invite;
    return InviteTicket(
      inviteCode: invite.code,
      expiresAt: invite.expiresAt,
      space: _spaces[spaceId]!.view(this),
    );
  }

  String _newCode() {
    for (var attempt = 0; attempt < 8; attempt++) {
      final buffer = StringBuffer();
      for (var index = 0; index < inviteCodeLength; index++) {
        buffer.write(
          inviteCodeAlphabet[_random.nextInt(inviteCodeAlphabet.length)],
        );
      }
      final code = buffer.toString();
      if (!_invitesByCode.containsKey(code)) return code;
    }
    throw ApiException('INVITE_FAILED', '暂时无法生成邀请码');
  }

  _Invite? _latestInvite(String spaceId) {
    final rows = _invitesByCode.values
        .where((invite) => invite.spaceId == spaceId)
        .toList()
      ..sort((a, b) => b.expiresAt.compareTo(a.expiresAt));
    if (rows.isEmpty) return null;
    return rows.first;
  }

  AuthResult _tokensFor(_User user) {
    final refreshToken = newId(_random);
    _refreshByToken[refreshToken] = _Refresh(
      userId: user.id,
      expiresAt: now().add(const Duration(days: 30)),
    );
    return AuthResult(
      accessToken: 'mock-access-${user.id}',
      refreshToken: refreshToken,
      user: user.view(),
    );
  }

  _Post _visiblePost(String userId, String postId) {
    final spaceId = _requireSpace(userId);
    final post = _posts[postId];
    if (post == null || post.spaceId != spaceId || post.deletedAt != null) {
      throw ApiException('NOT_FOUND', '动态不存在');
    }
    return post;
  }

  String _requireSpace(String userId) {
    _requireUser(userId);
    final spaceId = _spaceIdByUser[userId];
    if (spaceId == null) {
      throw ApiException('FORBIDDEN', '需要先完成配对');
    }
    return spaceId;
  }

  void _requireUser(String userId) {
    if (!_usersById.containsKey(userId)) {
      throw ApiException('UNAUTHORIZED', '请先登录');
    }
  }

  DateTime _stamp() {
    _sequence += 1;
    return now().add(Duration(microseconds: _sequence));
  }

  LocationInput? _location(LocationInput? location) {
    if (location == null) return null;
    if (location.lat < -90 ||
        location.lat > 90 ||
        location.lng < -180 ||
        location.lng > 180) {
      throw ApiException('INVALID_LOCATION', '经纬度超出范围');
    }
    return LocationInput(
      lat: location.lat,
      lng: location.lng,
      name: _blankToNull(location.name),
      placeId: _blankToNull(location.placeId),
      visibility: location.visibility,
    );
  }

  String _email(String value) {
    final normalized = value.trim().toLowerCase();
    final valid = RegExp(r'^[^@\s]+@[^@\s]+\.[^@\s]+$').hasMatch(normalized);
    if (!valid || normalized.length > 320) {
      throw ApiException('INVALID_EMAIL', '请输入有效邮箱');
    }
    return normalized;
  }

  String _password(String value) {
    if (value.length < minPasswordLength || value.length > maxPasswordLength) {
      throw ApiException('WEAK_PASSWORD', '密码至少 8 位');
    }
    return value;
  }

  String _displayName(String value) {
    final trimmed = value.trim();
    if (trimmed.isEmpty || trimmed.length > 64) {
      throw ApiException('INVALID_DISPLAY_NAME', '请填写 1–64 字的昵称');
    }
    return trimmed;
  }

  String _title(String value) {
    final trimmed = value.trim();
    if (trimmed.isEmpty || trimmed.length > 128) {
      throw ApiException('INVALID_TITLE', '请填写 1–128 字的名称');
    }
    return trimmed;
  }

  String _eventDate(String value) {
    if (parseDateOnly(value) == null) {
      throw ApiException('INVALID_DATE', '日期格式应为 YYYY-MM-DD');
    }
    return value;
  }

  String _recurrence(String value) {
    if (value != 'yearly' && value != 'once') {
      throw ApiException('INVALID_RECURRENCE', '重复方式只能是每年或一次');
    }
    return value;
  }

  String? _blankToNull(String? value) {
    if (value == null) return null;
    final trimmed = value.trim();
    if (trimmed.isEmpty) return null;
    return trimmed;
  }

  String? _normalizeCode(String raw) {
    final code = raw.trim().toUpperCase();
    if (code.length != inviteCodeLength) return null;
    for (final unit in code.codeUnits) {
      final char = String.fromCharCode(unit);
      if (!inviteCodeAlphabet.contains(char)) return null;
    }
    return code;
  }
}

class _User {
  _User({
    required this.id,
    required this.email,
    required this.password,
    required this.displayName,
    required this.createdAt,
  });

  final String id;
  final String email;
  final String password;
  final String displayName;
  final DateTime createdAt;

  UserPublic view() {
    return UserPublic(
      id: id,
      email: email,
      displayName: displayName,
      createdAt: createdAt,
    );
  }
}

class _Refresh {
  _Refresh({required this.userId, required this.expiresAt});

  final String userId;
  final DateTime expiresAt;
}

class _Membership {
  _Membership({required this.userId, required this.joinedAt});

  final String userId;
  final DateTime joinedAt;
}

class _Space {
  _Space({
    required this.id,
    required this.createdAt,
    required this.members,
  });

  final String id;
  final DateTime createdAt;
  final List<_Membership> members;

  int get memberCount => members.length;

  SpaceSnapshot view(MockBackend backend) {
    return SpaceSnapshot(
      id: id,
      memberCount: memberCount,
      createdAt: createdAt,
      members: [
        for (final member in members)
          SpaceMember(
            userId: member.userId,
            displayName: backend._usersById[member.userId]!.displayName,
            joinedAt: member.joinedAt,
          ),
      ],
    );
  }
}

class _Invite {
  _Invite({
    required this.code,
    required this.inviterUserId,
    required this.spaceId,
    required this.expiresAt,
  });

  final String code;
  final String inviterUserId;
  final String spaceId;
  final DateTime expiresAt;
  String status = 'pending';
  String? acceptedByUserId;
  DateTime? acceptedAt;
}

class _Media {
  _Media({
    required this.id,
    required this.bytes,
    required this.contentType,
    required this.sortOrder,
  });

  final String id;
  final Uint8List bytes;
  final String contentType;
  final int sortOrder;
}

class _Post {
  _Post({
    required this.id,
    required this.spaceId,
    required this.authorUserId,
    required this.body,
    required this.media,
    required this.location,
    required this.createdAt,
    required this.updatedAt,
  });

  final String id;
  final String spaceId;
  final String authorUserId;
  final String body;
  final List<_Media> media;
  final LocationInput? location;
  final DateTime createdAt;
  DateTime updatedAt;
  DateTime? deletedAt;

  Post view(MockBackend backend, String viewerId) {
    final likeCount = backend._likes
        .where((key) => key.startsWith('$id|'))
        .length;
    return Post(
      id: id,
      spaceId: spaceId,
      author: backend._usersById[authorUserId]!.view(),
      body: body,
      media: [
        for (final item in media)
          MediaRef(
            id: item.id,
            bytes: item.bytes,
            contentType: item.contentType,
            sortOrder: item.sortOrder,
          ),
      ],
      location: location,
      likeCount: likeCount,
      likedByMe: backend._likes.contains('$id|$viewerId'),
      commentCount: backend._comments.values
          .where((comment) => comment.postId == id && comment.deletedAt == null)
          .length,
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
  }
}

class _Comment {
  _Comment({
    required this.id,
    required this.spaceId,
    required this.postId,
    required this.authorUserId,
    required this.body,
    required this.createdAt,
  });

  final String id;
  final String spaceId;
  final String postId;
  final String authorUserId;
  final String body;
  final DateTime createdAt;
  DateTime? deletedAt;

  Comment view(MockBackend backend) {
    return Comment(
      id: id,
      postId: postId,
      author: backend._usersById[authorUserId]!.view(),
      body: body,
      createdAt: createdAt,
    );
  }
}

class _Checkin {
  _Checkin({
    required this.id,
    required this.spaceId,
    required this.creatorUserId,
    required this.name,
    required this.address,
    required this.lat,
    required this.lng,
    required this.placeId,
    required this.checkedInAt,
    required this.createdAt,
  });

  final String id;
  final String spaceId;
  final String creatorUserId;
  final String name;
  final String? address;
  final double? lat;
  final double? lng;
  final String? placeId;
  final DateTime checkedInAt;
  final DateTime createdAt;
  DateTime? deletedAt;

  RestaurantCheckin view(MockBackend backend) {
    return RestaurantCheckin(
      id: id,
      spaceId: spaceId,
      creator: backend._usersById[creatorUserId]!.view(),
      name: name,
      address: address,
      lat: lat,
      lng: lng,
      placeId: placeId,
      checkedInAt: checkedInAt,
      createdAt: createdAt,
    );
  }
}

class _Anniversary {
  _Anniversary({
    required this.id,
    required this.spaceId,
    required this.creatorUserId,
    required this.title,
    required this.eventDate,
    required this.recurrence,
    required this.remind,
    required this.remindDaysBefore,
    required this.remindLocalTime,
    required this.timezone,
    required this.createdAt,
    required this.updatedAt,
  });

  final String id;
  final String spaceId;
  final String creatorUserId;
  String title;
  String eventDate;
  String recurrence;
  bool remind;
  int remindDaysBefore;
  String remindLocalTime;
  String timezone;
  final DateTime createdAt;
  DateTime updatedAt;
  DateTime? deletedAt;

  Anniversary view(MockBackend backend) {
    return Anniversary(
      id: id,
      spaceId: spaceId,
      creator: backend._usersById[creatorUserId]!.view(),
      title: title,
      eventDate: eventDate,
      recurrence: recurrence,
      remind: remind,
      remindDaysBefore: remindDaysBefore,
      remindLocalTime: remindLocalTime,
      timezone: timezone,
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
  }
}

class _Device {
  _Device({
    required this.id,
    required this.userId,
    required this.platform,
    required this.token,
    required this.updatedAt,
  });

  final String id;
  final String userId;
  String platform;
  final String token;
  DateTime updatedAt;

  DeviceToken view() {
    return DeviceToken(
      id: id,
      platform: platform,
      token: token,
      updatedAt: updatedAt,
    );
  }
}
