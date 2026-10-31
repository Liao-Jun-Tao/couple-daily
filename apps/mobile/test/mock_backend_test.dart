import 'dart:typed_data';

import 'package:couple_daily/core/constants.dart';
import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/mock/mock_backend.dart';
import 'package:couple_daily/core/models/models.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('register and login use email and password', () {
    final backend = MockBackend();
    final created = backend.register(
      email: 'Ada@Example.com',
      password: 'password1',
      displayName: 'Ada',
    );
    expect(created.user.email, 'ada@example.com');
    expect(created.accessToken, isNotEmpty);
    expect(created.refreshToken, isNotEmpty);

    final signedIn = backend.login(
      email: 'ADA@example.com',
      password: 'password1',
    );
    expect(signedIn.user.id, created.user.id);

    expect(
      () => backend.register(
        email: 'ada@example.com',
        password: 'password1',
        displayName: 'Other',
      ),
      throwsCode('EMAIL_TAKEN'),
    );
    expect(
      () => backend.register(
        email: 'new@example.com',
        password: 'short',
        displayName: 'New',
      ),
      throwsCode('WEAK_PASSWORD'),
    );
    expect(
      () => backend.login(email: 'ada@example.com', password: 'wrong-password'),
      throwsCode('INVALID_CREDENTIALS'),
    );

    final refreshed = backend.refresh(refreshToken: created.refreshToken);
    expect(refreshed.user.id, created.user.id);
    expect(
      () => backend.refresh(refreshToken: created.refreshToken),
      throwsCode('INVALID_CREDENTIALS'),
    );
  });

  test('invite is 6 chars, 24h, one-time, and opens a 1-member space', () {
    var current = DateTime.utc(2026, 10, 31, 8);
    final backend = MockBackend(clock: () => current);
    final inviter = backend.register(
      email: 'a@example.com',
      password: 'password1',
      displayName: '甲',
    );
    final partner = backend.register(
      email: 'b@example.com',
      password: 'password1',
      displayName: '乙',
    );
    final third = backend.register(
      email: 'c@example.com',
      password: 'password1',
      displayName: '丙',
    );

    final ticket = backend.createInvite(inviter.user.id);
    expect(ticket.inviteCode, matches(RegExp(r'^[A-Z0-9]{6}$')));
    expect(ticket.expiresAt.difference(current), inviteTtl);
    expect(ticket.space.memberCount, 1);
    expect(backend.pairingOf(inviter.user.id).status, PairingStatus.pendingInvite);
    expect(backend.pairingOf(inviter.user.id).memberCount, 1);
    expect(
      backend.createInvite(inviter.user.id).inviteCode,
      ticket.inviteCode,
    );

    expect(
      () => backend.acceptInvite(
        userId: inviter.user.id,
        code: ticket.inviteCode.toLowerCase(),
      ),
      throwsCode('ALREADY_PAIRED'),
    );

    current = ticket.expiresAt.subtract(const Duration(seconds: 1));
    final space = backend.acceptInvite(
      userId: partner.user.id,
      code: ticket.inviteCode.toLowerCase(),
    );
    expect(space.memberCount, 2);
    expect(backend.inviteStatus(ticket.inviteCode), 'accepted');
    expect(backend.pairingOf(inviter.user.id).status, PairingStatus.paired);
    expect(backend.pairingOf(partner.user.id).memberCount, 2);

    expect(
      () => backend.acceptInvite(
        userId: third.user.id,
        code: ticket.inviteCode,
      ),
      throwsCode('INVITE_USED'),
    );
    expect(backend.pairingOf(inviter.user.id).memberCount, 2);
    expect(
      () => backend.createInvite(inviter.user.id),
      throwsCode('ALREADY_PAIRED'),
    );
  });

  test('expired invite is rejected and can be reissued on the same space', () {
    var current = DateTime.utc(2026, 10, 31, 8);
    final backend = MockBackend(clock: () => current);
    final inviter = backend.register(
      email: 'a@example.com',
      password: 'password1',
      displayName: '甲',
    );
    final partner = backend.register(
      email: 'b@example.com',
      password: 'password1',
      displayName: '乙',
    );
    final ticket = backend.createInvite(inviter.user.id);
    current = ticket.expiresAt;
    expect(
      () => backend.acceptInvite(
        userId: partner.user.id,
        code: ticket.inviteCode,
      ),
      throwsCode('INVITE_EXPIRED'),
    );
    expect(backend.pairingOf(inviter.user.id).memberCount, 1);

    final reissued = backend.createInvite(inviter.user.id);
    expect(reissued.inviteCode, isNot(ticket.inviteCode));
    expect(reissued.space.id, ticket.space.id);
    expect(reissued.space.memberCount, 1);
    expect(backend.inviteStatus(ticket.inviteCode), 'expired');

    final space = backend.acceptInvite(
      userId: partner.user.id,
      code: reissued.inviteCode,
    );
    expect(space.memberCount, 2);
  });

  test('posts reject empty content and enforce image limits', () {
    final backend = MockBackend();
    final author = backend.register(
      email: 'a@example.com',
      password: 'password1',
      displayName: '甲',
    );
    final other = backend.register(
      email: 'b@example.com',
      password: 'password1',
      displayName: '乙',
    );
    final ticket = backend.createInvite(author.user.id);
    backend.acceptInvite(userId: other.user.id, code: ticket.inviteCode);

    expect(
      () => backend.createPost(
        userId: author.user.id,
        body: ' \n\t',
        media: const [],
      ),
      throwsCode('EMPTY_POST'),
    );

    final textOnly = backend.createPost(
      userId: author.user.id,
      body: '今天很好',
      media: const [],
    );
    expect(textOnly.media, isEmpty);

    final imageOnly = backend.createPost(
      userId: author.user.id,
      body: '   ',
      media: [
        MediaDraft(bytes: Uint8List.fromList([1]), contentType: 'image/png'),
      ],
    );
    expect(imageOnly.body, isEmpty);
    expect(imageOnly.media, hasLength(1));

    expect(
      () => backend.createPost(
        userId: author.user.id,
        body: '',
        media: [
          MediaDraft(bytes: Uint8List.fromList([1]), contentType: 'image/gif'),
        ],
      ),
      throwsCode('UNSUPPORTED_MEDIA'),
    );
    expect(
      () => backend.createPost(
        userId: author.user.id,
        body: '',
        media: List.generate(
          maxImagesPerPost + 1,
          (_) => MediaDraft(
            bytes: Uint8List.fromList([1]),
            contentType: 'image/jpeg',
          ),
        ),
      ),
      throwsCode('TOO_MANY_IMAGES'),
    );
    expect(
      () => backend.createPost(
        userId: author.user.id,
        body: '',
        media: [
          MediaDraft(
            bytes: Uint8List(maxImageBytes + 1),
            contentType: 'image/jpeg',
          ),
        ],
      ),
      throwsCode('IMAGE_TOO_LARGE'),
    );

    final atLimit = backend.createPost(
      userId: author.user.id,
      body: '九张',
      media: List.generate(
        maxImagesPerPost,
        (_) => MediaDraft(
          bytes: Uint8List.fromList([9]),
          contentType: 'image/webp',
        ),
      ),
      location: const LocationInput(lat: 31.24, lng: 121.49, name: '外滩'),
    );
    expect(atLimit.media, hasLength(maxImagesPerPost));
    expect(atLimit.location?.visibility, LocationVisibility.exact);

    expect(
      () => backend.deletePost(userId: other.user.id, postId: textOnly.id),
      throwsCode('NOT_AUTHOR'),
    );
    backend.deletePost(userId: author.user.id, postId: textOnly.id);
    expect(
      backend.listPosts(author.user.id).map((post) => post.id),
      isNot(contains(textOnly.id)),
    );
  });

  test('likes are per user and comments are author-deletable', () {
    final backend = MockBackend();
    final author = backend.register(
      email: 'a@example.com',
      password: 'password1',
      displayName: '甲',
    );
    final other = backend.register(
      email: 'b@example.com',
      password: 'password1',
      displayName: '乙',
    );
    final ticket = backend.createInvite(author.user.id);
    backend.acceptInvite(userId: other.user.id, code: ticket.inviteCode);
    final post = backend.createPost(
      userId: author.user.id,
      body: '看这里',
      media: const [],
    );

    backend.like(userId: author.user.id, postId: post.id);
    backend.like(userId: author.user.id, postId: post.id);
    backend.unlike(userId: other.user.id, postId: post.id);
    final liked = backend.listPosts(author.user.id).single;
    expect(liked.likeCount, 1);
    expect(liked.likedByMe, isTrue);
    expect(backend.listPosts(other.user.id).single.likedByMe, isFalse);

    backend.unlike(userId: author.user.id, postId: post.id);
    expect(backend.listPosts(author.user.id).single.likeCount, 0);

    final comment = backend.addComment(
      userId: other.user.id,
      postId: post.id,
      body: '收到',
    );
    expect(
      () => backend.deleteComment(
        userId: author.user.id,
        commentId: comment.id,
      ),
      throwsCode('NOT_AUTHOR'),
    );
    backend.deleteComment(userId: other.user.id, commentId: comment.id);
    expect(
      backend.listComments(userId: author.user.id, postId: post.id),
      isEmpty,
    );
  });

  test('manual check-in allows an empty place id', () {
    final backend = MockBackend();
    final user = backend.register(
      email: 'a@example.com',
      password: 'password1',
      displayName: '甲',
    );
    backend.createInvite(user.user.id);
    final checkin = backend.createCheckin(
      userId: user.user.id,
      name: ' 我家楼下 ',
      placeId: '  ',
    );
    expect(checkin.name, '我家楼下');
    expect(checkin.placeId, isNull);

    final other = backend.register(
      email: 'b@example.com',
      password: 'password1',
      displayName: '乙',
    );
    final ticket = backend.createInvite(user.user.id);
    backend.acceptInvite(userId: other.user.id, code: ticket.inviteCode);
    expect(
      () => backend.deleteCheckin(
        userId: other.user.id,
        checkinId: checkin.id,
      ),
      throwsCode('NOT_AUTHOR'),
    );
    backend.deleteCheckin(userId: user.user.id, checkinId: checkin.id);
    expect(backend.listCheckins(user.user.id), isEmpty);
  });

  test('anniversary reminder defaults to 09:00 Asia/Shanghai', () {
    final backend = MockBackend();
    final author = backend.register(
      email: 'a@example.com',
      password: 'password1',
      displayName: '甲',
    );
    final other = backend.register(
      email: 'b@example.com',
      password: 'password1',
      displayName: '乙',
    );
    final ticket = backend.createInvite(author.user.id);
    backend.acceptInvite(userId: other.user.id, code: ticket.inviteCode);
    final anniversary = backend.createAnniversary(
      userId: author.user.id,
      title: '在一起',
      eventDate: '2020-02-14',
    );
    expect(anniversary.remind, isTrue);
    expect(anniversary.remindDaysBefore, 0);
    expect(anniversary.remindLocalTime, defaultRemindLocalTime);
    expect(anniversary.timezone, defaultTimezone);
    expect(anniversary.recurrence, 'yearly');
    expect(
      () => backend.deleteAnniversary(
        userId: other.user.id,
        anniversaryId: anniversary.id,
      ),
      throwsCode('NOT_AUTHOR'),
    );
  });
}

Matcher throwsCode(String code) {
  return throwsA(
    isA<ApiException>().having((error) => error.code, 'code', code),
  );
}
