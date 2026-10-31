import 'package:couple_daily/core/notifications/notification_gateway.dart';

/// Lead sentence for the in-app tip. Anniversary reminders are the reason
/// the app asks; the tip must not block the rest of the product.
const notificationPermissionTipLead = '纪念日提醒需要通知权限。';

bool notificationPermissionIsDenied(NotificationPermissionResult result) {
  return result == NotificationPermissionResult.denied ||
      result == NotificationPermissionResult.permanentlyDenied;
}

String notificationPermissionTipText(NotificationPermissionResult result) {
  switch (result) {
    case NotificationPermissionResult.permanentlyDenied:
      return '$notificationPermissionTipLead'
          '系统已拒绝且不再询问。请到系统设置中允许通知，'
          '否则纪念日当天的提醒无法送达。可以关闭这条提示，继续使用应用。';
    case NotificationPermissionResult.denied:
      return '$notificationPermissionTipLead'
          '当前系统通知未开启，纪念日当天的提醒无法送达。'
          '可以关闭这条提示，继续使用应用。';
    case NotificationPermissionResult.granted:
    case NotificationPermissionResult.unavailable:
      return notificationPermissionTipLead;
  }
}
