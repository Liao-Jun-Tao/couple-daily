/// Locked product limits. The API enforces the same rules. The UI
/// repeats them so a request is not sent when it would be rejected.
const int maxImagesPerPost = 9;
const int maxImageBytes = 10485760;
const int inviteCodeLength = 6;
const Duration inviteTtl = Duration(hours: 24);
const int minPasswordLength = 8;
const int maxPasswordLength = 128;
const String defaultRemindLocalTime = '09:00';
const String defaultTimezone = 'Asia/Shanghai';
const int defaultRemindDaysBefore = 0;

const Set<String> allowedImageTypes = {
  'image/jpeg',
  'image/png',
  'image/heic',
  'image/webp',
};

const String inviteCodeAlphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
