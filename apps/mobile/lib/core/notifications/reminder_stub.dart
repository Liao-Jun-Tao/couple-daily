import 'package:couple_daily/core/models/models.dart';

/// Records anniversary reminders locally. This does not talk to APNs or FCM.
class ReminderStub {
  final List<ReminderIntent> intents = [];

  void sync(Anniversary anniversary) {
    intents.removeWhere((item) => item.anniversaryId == anniversary.id);
    if (!anniversary.remind) return;
    intents.add(
      ReminderIntent(
        anniversaryId: anniversary.id,
        title: anniversary.title,
        eventDate: anniversary.eventDate,
        remindLocalTime: anniversary.remindLocalTime,
        timezone: anniversary.timezone,
      ),
    );
  }

  void cancel(String anniversaryId) {
    intents.removeWhere((item) => item.anniversaryId == anniversaryId);
  }
}
