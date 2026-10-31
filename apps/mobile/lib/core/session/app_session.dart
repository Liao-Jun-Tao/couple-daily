import 'package:flutter/foundation.dart';

import 'package:couple_daily/core/mock/mock_backend.dart';
import 'package:couple_daily/core/models/models.dart';
import 'package:couple_daily/core/notifications/notification_gateway.dart';
import 'package:couple_daily/core/notifications/reminder_stub.dart';

/// Signed-in user plus the in-memory backend. Screens talk to this,
/// not to a network client.
class AppSession extends ChangeNotifier {
  AppSession({
    MockBackend? backend,
    NotificationGateway? notifications,
    ReminderStub? reminders,
  }) : backend = backend ?? MockBackend(),
       notifications = notifications ?? NotificationGateway(),
       reminders = reminders ?? ReminderStub();

  final MockBackend backend;
  final NotificationGateway notifications;
  final ReminderStub reminders;

  bool ready = false;
  bool pendingPairingDone = false;
  String? userId;
  String? accessToken;
  String? refreshToken;

  bool get isLoggedIn => userId != null;

  UserPublic? get user {
    final id = userId;
    if (id == null) return null;
    return backend.userById(id);
  }

  PairingSnapshot get pairing {
    final id = userId;
    if (id == null) return PairingSnapshot.unpaired;
    return backend.pairingOf(id);
  }

  bool get isPaired => pairing.status == PairingStatus.paired;

  SpaceSnapshot? get space {
    final id = userId;
    if (id == null) return null;
    return backend.spaceOf(id);
  }

  void markReady() {
    if (ready) return;
    ready = true;
    notifyListeners();
  }

  void clearPairingDoneFlag() {
    pendingPairingDone = false;
  }

  void register({
    required String email,
    required String password,
    required String displayName,
  }) {
    final result = backend.register(
      email: email,
      password: password,
      displayName: displayName,
    );
    _applyAuth(result);
  }

  void login({required String email, required String password}) {
    final result = backend.login(email: email, password: password);
    _applyAuth(result);
  }

  void logout() {
    userId = null;
    accessToken = null;
    refreshToken = null;
    pendingPairingDone = false;
    notifyListeners();
  }

  InviteTicket createInvite() {
    final ticket = backend.createInvite(_requireUser());
    notifyListeners();
    return ticket;
  }

  void acceptInvite(String code) {
    backend.acceptInvite(userId: _requireUser(), code: code);
    pendingPairingDone = true;
    notifyListeners();
  }

  List<Post> posts() => backend.listPosts(_requireUser());

  void createPost({
    required String body,
    required List<MediaDraft> media,
    LocationInput? location,
  }) {
    backend.createPost(
      userId: _requireUser(),
      body: body,
      media: media,
      location: location,
    );
    notifyListeners();
  }

  void deletePost(String postId) {
    backend.deletePost(userId: _requireUser(), postId: postId);
    notifyListeners();
  }

  void like(String postId) {
    backend.like(userId: _requireUser(), postId: postId);
    notifyListeners();
  }

  void unlike(String postId) {
    backend.unlike(userId: _requireUser(), postId: postId);
    notifyListeners();
  }

  List<Comment> comments(String postId) {
    return backend.listComments(userId: _requireUser(), postId: postId);
  }

  void addComment(String postId, String body) {
    backend.addComment(userId: _requireUser(), postId: postId, body: body);
    notifyListeners();
  }

  void deleteComment(String commentId) {
    backend.deleteComment(userId: _requireUser(), commentId: commentId);
    notifyListeners();
  }

  List<RestaurantCheckin> checkins() => backend.listCheckins(_requireUser());

  void createCheckin({
    required String name,
    String? address,
    double? lat,
    double? lng,
    String? placeId,
  }) {
    backend.createCheckin(
      userId: _requireUser(),
      name: name,
      address: address,
      lat: lat,
      lng: lng,
      placeId: placeId,
    );
    notifyListeners();
  }

  void deleteCheckin(String checkinId) {
    backend.deleteCheckin(userId: _requireUser(), checkinId: checkinId);
    notifyListeners();
  }

  List<Anniversary> anniversaries() => backend.listAnniversaries(_requireUser());

  Anniversary saveAnniversary({
    String? id,
    required String title,
    required String eventDate,
    required String recurrence,
    required bool remind,
  }) {
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

  void deleteAnniversary(String anniversaryId) {
    backend.deleteAnniversary(
      userId: _requireUser(),
      anniversaryId: anniversaryId,
    );
    reminders.cancel(anniversaryId);
    notifyListeners();
  }

  List<DeviceToken> devices() => backend.listDevices(_requireUser());

  void registerDevice({required String platform, required String token}) {
    backend.registerDevice(
      userId: _requireUser(),
      platform: platform,
      token: token,
    );
    notifyListeners();
  }

  void _applyAuth(AuthResult result) {
    userId = result.user.id;
    accessToken = result.accessToken;
    refreshToken = result.refreshToken;
    pendingPairingDone = false;
    notifyListeners();
  }

  String _requireUser() {
    final id = userId;
    if (id == null) {
      throw StateError('No signed-in user');
    }
    return id;
  }
}
