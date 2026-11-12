import '../models/auth.dart';
import 'platform/session_store_io.dart' if (dart.library.html) 'platform/session_store_web.dart' as platform;

abstract interface class SessionStore {
  Future<AuthSession?> read();
  Future<void> write(AuthSession session);
  Future<void> clear();
  bool get usesCookieSession;
}

SessionStore createSessionStore() => platform.PlatformSessionStore();
