class StoredTokens {
  const StoredTokens({required this.accessToken, required this.refreshToken});

  final String accessToken;
  final String refreshToken;
}

class StoredInvite {
  const StoredInvite({
    required this.userId,
    required this.code,
    required this.expiresAt,
  });

  final String userId;
  final String code;
  final DateTime expiresAt;
}

/// Access and refresh tokens, plus the invite code the API returns only once.
abstract class TokenStore {
  Future<StoredTokens?> readTokens();

  Future<void> writeTokens(StoredTokens tokens);

  Future<StoredInvite?> readInvite();

  Future<void> writeInvite(StoredInvite invite);

  Future<void> clearInvite();

  Future<void> clear();
}

class MemoryTokenStore implements TokenStore {
  StoredTokens? tokens;
  StoredInvite? invite;

  @override
  Future<void> clear() async {
    tokens = null;
    invite = null;
  }

  @override
  Future<void> clearInvite() async {
    invite = null;
  }

  @override
  Future<StoredInvite?> readInvite() async => invite;

  @override
  Future<StoredTokens?> readTokens() async => tokens;

  @override
  Future<void> writeInvite(StoredInvite value) async {
    invite = value;
  }

  @override
  Future<void> writeTokens(StoredTokens value) async {
    tokens = value;
  }
}
