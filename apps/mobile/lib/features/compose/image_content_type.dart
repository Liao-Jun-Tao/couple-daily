import 'package:couple_daily/core/constants.dart';
import 'package:image_picker/image_picker.dart';

String? contentTypeForName(String name) {
  final lower = name.toLowerCase();
  if (lower.endsWith('.jpg') || lower.endsWith('.jpeg')) return 'image/jpeg';
  if (lower.endsWith('.png')) return 'image/png';
  if (lower.endsWith('.heic') || lower.endsWith('.heif')) return 'image/heic';
  if (lower.endsWith('.webp')) return 'image/webp';
  return null;
}

String? resolveContentType(XFile file) {
  final mime = file.mimeType?.toLowerCase();
  if (mime == 'image/jpg') return 'image/jpeg';
  if (mime == 'image/heif') return 'image/heic';
  if (mime != null && allowedImageTypes.contains(mime)) return mime;
  return contentTypeForName(file.name);
}
