import 'dart:io';

/// iOS stub tokens stand in for APNs. Android stub tokens stand in for FCM.
String currentPushPlatform() {
  if (Platform.isIOS) return 'ios';
  return 'android';
}

String pushChannelName(String platform) {
  return platform == 'ios' ? 'APNs' : 'FCM';
}
