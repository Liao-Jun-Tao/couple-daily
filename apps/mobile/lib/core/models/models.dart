import 'dart:typed_data';

enum PairingStatus { unpaired, pendingInvite, paired }

enum LocationVisibility { exact, city, hidden }

extension LocationVisibilityLabel on LocationVisibility {
  /// 完整 is the default (exact coordinates + place name).
  String get label => switch (this) {
    LocationVisibility.exact => '完整',
    LocationVisibility.city => '城市',
    LocationVisibility.hidden => '隐藏',
  };
}

class UserPublic {
  const UserPublic({
    required this.id,
    required this.email,
    required this.displayName,
    required this.createdAt,
    this.avatarUrl,
  });

  final String id;
  final String email;
  final String displayName;
  final DateTime createdAt;
  final String? avatarUrl;
}

class AuthResult {
  const AuthResult({
    required this.accessToken,
    required this.refreshToken,
    required this.user,
  });

  final String accessToken;
  final String refreshToken;
  final UserPublic user;
}

class PairingSnapshot {
  const PairingSnapshot({
    required this.status,
    this.spaceId,
    this.memberCount,
    this.inviteCode,
    this.inviteExpiresAt,
  });

  final PairingStatus status;
  final String? spaceId;
  final int? memberCount;
  final String? inviteCode;
  final DateTime? inviteExpiresAt;

  static const unpaired = PairingSnapshot(status: PairingStatus.unpaired);
}

class SpaceMember {
  const SpaceMember({
    required this.userId,
    required this.displayName,
    required this.joinedAt,
    this.avatarUrl,
  });

  final String userId;
  final String displayName;
  final DateTime joinedAt;
  final String? avatarUrl;
}

class SpaceSnapshot {
  const SpaceSnapshot({
    required this.id,
    required this.memberCount,
    required this.members,
    required this.createdAt,
  });

  final String id;
  final int memberCount;
  final List<SpaceMember> members;
  final DateTime createdAt;
}

class InviteTicket {
  const InviteTicket({
    required this.inviteCode,
    required this.expiresAt,
    required this.space,
  });

  final String inviteCode;
  final DateTime expiresAt;
  final SpaceSnapshot space;
}

class LocationInput {
  const LocationInput({
    required this.lat,
    required this.lng,
    this.name,
    this.placeId,
    this.visibility = LocationVisibility.exact,
  });

  final double lat;
  final double lng;
  final String? name;
  final String? placeId;
  final LocationVisibility visibility;
}

class MediaDraft {
  const MediaDraft({required this.bytes, required this.contentType});

  final Uint8List bytes;
  final String contentType;
  int get byteSize => bytes.length;
}

class MediaRef {
  const MediaRef({
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

class Post {
  const Post({
    required this.id,
    required this.spaceId,
    required this.author,
    required this.body,
    required this.media,
    required this.likeCount,
    required this.likedByMe,
    required this.commentCount,
    required this.createdAt,
    required this.updatedAt,
    this.location,
  });

  final String id;
  final String spaceId;
  final UserPublic author;
  final String body;
  final List<MediaRef> media;
  final LocationInput? location;
  final int likeCount;
  final bool likedByMe;
  final int commentCount;
  final DateTime createdAt;
  final DateTime updatedAt;
}

class Comment {
  const Comment({
    required this.id,
    required this.postId,
    required this.author,
    required this.body,
    required this.createdAt,
  });

  final String id;
  final String postId;
  final UserPublic author;
  final String body;
  final DateTime createdAt;
}

class RestaurantCheckin {
  const RestaurantCheckin({
    required this.id,
    required this.spaceId,
    required this.creator,
    required this.name,
    required this.checkedInAt,
    required this.createdAt,
    this.address,
    this.lat,
    this.lng,
    this.placeId,
  });

  final String id;
  final String spaceId;
  final UserPublic creator;
  final String name;
  final String? address;
  final double? lat;
  final double? lng;
  final String? placeId;
  final DateTime checkedInAt;
  final DateTime createdAt;
}

class Anniversary {
  const Anniversary({
    required this.id,
    required this.spaceId,
    required this.creator,
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
  final UserPublic creator;
  final String title;
  final String eventDate;
  final String recurrence;
  final bool remind;
  final int remindDaysBefore;
  final String remindLocalTime;
  final String timezone;
  final DateTime createdAt;
  final DateTime updatedAt;
}

class DeviceToken {
  const DeviceToken({
    required this.id,
    required this.platform,
    required this.token,
    required this.updatedAt,
  });

  final String id;
  final String platform;
  final String token;
  final DateTime updatedAt;
}

class ReminderIntent {
  const ReminderIntent({
    required this.anniversaryId,
    required this.title,
    required this.eventDate,
    required this.remindLocalTime,
    required this.timezone,
  });

  final String anniversaryId;
  final String title;
  final String eventDate;

  /// Local wall-clock time. New rows are stored as 09:00.
  final String remindLocalTime;
  final String timezone;
}
