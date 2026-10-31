/// Machine-readable failure from the in-memory mock.
class ApiException implements Exception {
  ApiException(this.code, this.message);

  final String code;
  final String message;

  @override
  String toString() => '$code: $message';
}

String messageOf(Object error) {
  if (error is ApiException) return error.message;
  return '出现了一点问题，请重试';
}
