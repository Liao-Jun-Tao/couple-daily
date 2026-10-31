/// Runtime switch for the couple-daily client.
///
/// `--dart-define=USE_MOCK=true` keeps the in-memory backend.
/// The default is the real API. Override the host with
/// `--dart-define=API_BASE_URL=http://10.0.2.2:8080/v1`.
class ApiConfig {
  const ApiConfig({required this.useMock, required this.baseUrl});

  final bool useMock;
  final String baseUrl;

  static const fromEnvironment = ApiConfig(
    useMock: bool.fromEnvironment('USE_MOCK', defaultValue: false),
    baseUrl: String.fromEnvironment(
      'API_BASE_URL',
      defaultValue: 'http://10.0.2.2:8080/v1',
    ),
  );

  static const defaultBaseUrl = 'http://10.0.2.2:8080/v1';

  String get normalizedBaseUrl {
    final trimmed = baseUrl.trim();
    final raw = trimmed.isEmpty ? defaultBaseUrl : trimmed;
    var url = raw;
    while (url.endsWith('/')) {
      url = url.substring(0, url.length - 1);
    }
    return url;
  }
}
