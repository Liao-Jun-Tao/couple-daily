import 'dart:typed_data';

import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/models/models.dart';

Map<String, dynamic> asJsonMap(Object? value) {
  if (value is Map<String, dynamic>) return value;
  if (value is Map) return Map<String, dynamic>.from(value);
  throw ApiException('INTERNAL', '服务器返回了无法识别的数据');
}

DateTime parseApiTime(Object? value) {
  if (value is! String) {
    throw ApiException('INTERNAL', '服务器返回了无法识别的数据');
  }
  try {
    return DateTime.parse(value);
  } on FormatException {
    throw ApiException('INTERNAL', '服务器返回了无法识别的数据');
  }
}

int parseApiInt(Object? value) {
  if (value is int) return value;
  if (value is num) return value.toInt();
  throw ApiException('INTERNAL', '服务器返回了无法识别的数据');
}

String parseApiString(Object? value) {
  if (value is String) return value;
  throw ApiException('INTERNAL', '服务器返回了无法识别的数据');
}

UserPublic parseUser(Map<String, dynamic> json) {
  final avatar = json['avatar_url'];
  return UserPublic(
    id: parseApiString(json['id']),
    email: parseApiString(json['email']),
    displayName: parseApiString(json['display_name']),
    createdAt: parseApiTime(json['created_at']),
    avatarUrl: avatar is String && avatar.isNotEmpty ? avatar : null,
  );
}

AuthResult parseAuth(Map<String, dynamic> json) {
  return AuthResult(
    accessToken: parseApiString(json['access_token']),
    refreshToken: parseApiString(json['refresh_token']),
    user: parseUser(asJsonMap(json['user'])),
  );
}

PairingStatus parsePairingStatus(Object? value) {
  return switch (value) {
    'paired' => PairingStatus.paired,
    'pending_invite' => PairingStatus.pendingInvite,
    _ => PairingStatus.unpaired,
  };
}

PairingSnapshot parsePairing(Map<String, dynamic> json) {
  final count = json['member_count'];
  final spaceId = json['space_id'];
  return PairingSnapshot(
    status: parsePairingStatus(json['status']),
    spaceId: spaceId is String && spaceId.isNotEmpty ? spaceId : null,
    memberCount: count == null ? null : parseApiInt(count),
  );
}

class MeSnapshot {
  const MeSnapshot({required this.user, required this.pairing});

  final UserPublic user;
  final PairingSnapshot pairing;
}

MeSnapshot parseMe(Map<String, dynamic> json) {
  return MeSnapshot(
    user: parseUser(asJsonMap(json['user'])),
    pairing: parsePairing(asJsonMap(json['pairing'])),
  );
}

SpaceMember parseMember(Map<String, dynamic> json) {
  final avatar = json['avatar_url'];
  return SpaceMember(
    userId: parseApiString(json['user_id']),
    displayName: parseApiString(json['display_name']),
    joinedAt: parseApiTime(json['joined_at']),
    avatarUrl: avatar is String && avatar.isNotEmpty ? avatar : null,
  );
}

SpaceSnapshot parseSpace(Map<String, dynamic> json) {
  final rawMembers = json['members'];
  final members = <SpaceMember>[];
  if (rawMembers is List) {
    for (final item in rawMembers) {
      members.add(parseMember(asJsonMap(item)));
    }
  }
  return SpaceSnapshot(
    id: parseApiString(json['id']),
    memberCount: parseApiInt(json['member_count']),
    members: members,
    createdAt: parseApiTime(json['created_at']),
  );
}

InviteTicket parseInvite(Map<String, dynamic> json) {
  return InviteTicket(
    inviteCode: parseApiString(json['invite_code']),
    expiresAt: parseApiTime(json['expires_at']),
    space: parseSpace(asJsonMap(json['space'])),
  );
}

LocationVisibility parseVisibility(Object? value) {
  return switch (value) {
    'city' => LocationVisibility.city,
    'hidden' => LocationVisibility.hidden,
    _ => LocationVisibility.exact,
  };
}

LocationInput? parseLocation(Object? value) {
  if (value == null) return null;
  final json = asJsonMap(value);
  final visibility = parseVisibility(json['visibility']);
  final lat = json['lat'];
  final lng = json['lng'];
  final name = json['name'];
  final placeId = json['place_id'];
  if (lat is! num || lng is! num) {
    if (visibility == LocationVisibility.hidden) {
      return LocationInput(
        lat: 0,
        lng: 0,
        name: name is String ? name : null,
        placeId: placeId is String ? placeId : null,
        visibility: visibility,
      );
    }
    return null;
  }
  return LocationInput(
    lat: lat.toDouble(),
    lng: lng.toDouble(),
    name: name is String ? name : null,
    placeId: placeId is String ? placeId : null,
    visibility: visibility,
  );
}

MediaRef parseMedia(Map<String, dynamic> json) {
  final url = json['url'];
  return MediaRef(
    id: parseApiString(json['id']),
    bytes: Uint8List(0),
    contentType: parseApiString(json['content_type']),
    sortOrder: parseApiInt(json['sort_order']),
    url: url is String && url.isNotEmpty ? url : null,
  );
}

Post parsePost(Map<String, dynamic> json) {
  final rawMedia = json['media'];
  final media = <MediaRef>[];
  if (rawMedia is List) {
    for (final item in rawMedia) {
      media.add(parseMedia(asJsonMap(item)));
    }
  }
  return Post(
    id: parseApiString(json['id']),
    spaceId: parseApiString(json['space_id']),
    author: parseUser(asJsonMap(json['author'])),
    body: parseApiString(json['body']),
    media: media,
    location: parseLocation(json['location']),
    likeCount: parseApiInt(json['like_count']),
    likedByMe: json['liked_by_me'] == true,
    commentCount: parseApiInt(json['comment_count']),
    createdAt: parseApiTime(json['created_at']),
    updatedAt: parseApiTime(json['updated_at']),
  );
}

class LikeStatus {
  const LikeStatus({
    required this.postId,
    required this.likedByMe,
    required this.likeCount,
  });

  final String postId;
  final bool likedByMe;
  final int likeCount;
}

LikeStatus parseLike(Map<String, dynamic> json) {
  return LikeStatus(
    postId: parseApiString(json['post_id']),
    likedByMe: json['liked_by_me'] == true,
    likeCount: parseApiInt(json['like_count']),
  );
}

Comment parseComment(Map<String, dynamic> json) {
  return Comment(
    id: parseApiString(json['id']),
    postId: parseApiString(json['post_id']),
    author: parseUser(asJsonMap(json['author'])),
    body: parseApiString(json['body']),
    createdAt: parseApiTime(json['created_at']),
  );
}

RestaurantCheckin parseCheckin(Map<String, dynamic> json) {
  final address = json['address'];
  final lat = json['lat'];
  final lng = json['lng'];
  final placeId = json['place_id'];
  return RestaurantCheckin(
    id: parseApiString(json['id']),
    spaceId: parseApiString(json['space_id']),
    creator: parseUser(asJsonMap(json['creator'])),
    name: parseApiString(json['name']),
    address: address is String && address.isNotEmpty ? address : null,
    lat: lat is num ? lat.toDouble() : null,
    lng: lng is num ? lng.toDouble() : null,
    placeId: placeId is String && placeId.isNotEmpty ? placeId : null,
    checkedInAt: parseApiTime(json['checked_in_at']),
    createdAt: parseApiTime(json['created_at']),
  );
}

Anniversary parseAnniversary(Map<String, dynamic> json) {
  return Anniversary(
    id: parseApiString(json['id']),
    spaceId: parseApiString(json['space_id']),
    creator: parseUser(asJsonMap(json['creator'])),
    title: parseApiString(json['title']),
    eventDate: parseApiString(json['event_date']),
    recurrence: parseApiString(json['recurrence']),
    remind: json['remind'] == true,
    remindDaysBefore: parseApiInt(json['remind_days_before']),
    remindLocalTime: parseApiString(json['remind_local_time']),
    timezone: parseApiString(json['timezone']),
    createdAt: parseApiTime(json['created_at']),
    updatedAt: parseApiTime(json['updated_at']),
  );
}

DeviceToken parseDevice(Map<String, dynamic> json) {
  return DeviceToken(
    id: parseApiString(json['id']),
    platform: parseApiString(json['platform']),
    token: parseApiString(json['token']),
    updatedAt: parseApiTime(json['updated_at']),
  );
}

String? parseNextCursor(Map<String, dynamic> json) {
  final next = json['next_cursor'];
  if (next is String && next.isNotEmpty) return next;
  return null;
}

List<Map<String, dynamic>> parseItems(Map<String, dynamic> json) {
  final items = json['items'];
  if (items is! List) {
    throw ApiException('INTERNAL', '服务器返回了无法识别的数据');
  }
  return [for (final item in items) asJsonMap(item)];
}
