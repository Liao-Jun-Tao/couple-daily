import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'package:couple_daily/core/api/token_store.dart';

const _accessKey = 'access_token';
const _refreshKey = 'refresh_token';
const _inviteUserKey = 'invite_user_id';
const _inviteCodeKey = 'invite_code';
const _inviteExpiresKey = 'invite_expires_at';

/// Keychain / encrypted preferences. Failures surface to the caller.
class SecureTokenStore implements TokenStore {
  SecureTokenStore({FlutterSecureStorage? storage})
    : _storage =
          storage ??
          const FlutterSecureStorage(
            aOptions: AndroidOptions(encryptedSharedPreferences: true),
          );

  final FlutterSecureStorage _storage;

  @override
  Future<void> clear() async {
    await _storage.delete(key: _accessKey);
    await _storage.delete(key: _refreshKey);
    await clearInvite();
  }

  @override
  Future<void> clearInvite() async {
    await _storage.delete(key: _inviteUserKey);
    await _storage.delete(key: _inviteCodeKey);
    await _storage.delete(key: _inviteExpiresKey);
  }

  @override
  Future<StoredInvite?> readInvite() async {
    final userId = await _storage.read(key: _inviteUserKey);
    final code = await _storage.read(key: _inviteCodeKey);
    final expires = await _storage.read(key: _inviteExpiresKey);
    if (userId == null ||
        userId.isEmpty ||
        code == null ||
        code.isEmpty ||
        expires == null) {
      return null;
    }
    final parsed = DateTime.tryParse(expires);
    if (parsed == null) return null;
    return StoredInvite(userId: userId, code: code, expiresAt: parsed.toUtc());
  }

  @override
  Future<StoredTokens?> readTokens() async {
    final access = await _storage.read(key: _accessKey);
    final refresh = await _storage.read(key: _refreshKey);
    if (access == null ||
        access.isEmpty ||
        refresh == null ||
        refresh.isEmpty) {
      return null;
    }
    return StoredTokens(accessToken: access, refreshToken: refresh);
  }

  @override
  Future<void> writeInvite(StoredInvite invite) async {
    await _storage.write(key: _inviteUserKey, value: invite.userId);
    await _storage.write(key: _inviteCodeKey, value: invite.code);
    await _storage.write(
      key: _inviteExpiresKey,
      value: invite.expiresAt.toUtc().toIso8601String(),
    );
  }

  @override
  Future<void> writeTokens(StoredTokens tokens) async {
    await _storage.write(key: _accessKey, value: tokens.accessToken);
    await _storage.write(key: _refreshKey, value: tokens.refreshToken);
  }
}
