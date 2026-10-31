/// Machine-readable API failure. [message] is Chinese for the UI.
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

/// Maps a contract error code to Chinese. The server decides the code
/// (for example SPACE_FULL before INVITE_USED). This function does not
/// reorder or replace that code.
String chineseApiMessage(String code, String serverMessage) {
  switch (code) {
    case 'EMAIL_TAKEN':
      return '这个邮箱已经注册';
    case 'INVALID_CREDENTIALS':
      return '邮箱或密码不正确';
    case 'UNAUTHORIZED':
    case 'TOKEN_INVALID':
      return '登录已失效，请重新登录';
    case 'ALREADY_PAIRED':
      return '你已经在一个空间里';
    case 'INVITE_USED':
      return '邀请码已使用';
    case 'INVITE_EXPIRED':
      return '邀请码已过期';
    case 'INVITE_INVALID':
    case 'INVALID_INVITE_CODE':
      return '邀请码须为 6 位字母或数字';
    case 'INVITE_NOT_FOUND':
      return '邀请码不存在';
    case 'SPACE_FULL':
      return '这个空间已经有两个人';
    case 'NOT_PAIRED':
      return '请先完成配对';
    case 'FORBIDDEN':
      return '只有作者可以操作';
    case 'NOT_FOUND':
      return '内容不存在或已被删除';
    case 'EMPTY_POST':
      return '写一点文字，或者加一张图片';
    case 'MEDIA_INVALID':
      return '图片无效或上传失败，请重试';
    case 'RATE_LIMITED':
      return '尝试太多次，请稍后再试';
    case 'NETWORK':
      return '网络不可用，请检查 API 地址后重试';
    case 'VALIDATION_ERROR':
      return validationMessage(serverMessage);
    default:
      return '出现了一点问题（$code），请重试';
  }
}

String validationMessage(String serverMessage) {
  final text = serverMessage.toLowerCase();
  if (text.contains('password')) return '密码至少 8 位，最多 128 位';
  if (text.contains('display_name')) return '请填写 1 到 64 个字符的昵称';
  if (text.contains('email')) return '请填写有效邮箱';
  if (text.contains('invite')) return '邀请码须为 6 位字母或数字';
  if (text.contains('body')) return '正文不能超过 5000 字';
  if (text.contains('media') ||
      text.contains('content_type') ||
      text.contains('byte_size')) {
    return '图片需为 jpeg、png、heic 或 webp，且单张不超过 10MB';
  }
  if (text.contains('location')) return '位置信息不正确';
  if (text.contains('title')) return '请填写纪念日名称';
  if (text.contains('event_date')) return '日期格式应为 YYYY-MM-DD';
  if (text.contains('name')) return '请填写名称';
  if (text.contains('json')) return '请求格式不正确';
  return '填写内容不符合要求';
}
