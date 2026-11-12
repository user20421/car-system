import '../../models/auth.dart';
import '../session_store.dart';

/// Browser sessions are memory-only; refresh tokens are never persisted to Web Storage.
/// A deployment can replace this class with an HttpOnly Secure SameSite cookie session.
class PlatformSessionStore implements SessionStore {
  static AuthSession? _memorySession;
  @override
  bool get usesCookieSession => true;
  @override
  Future<AuthSession?> read() async => _memorySession;
  @override
  Future<void> write(AuthSession session) async => _memorySession = session;
  @override
  Future<void> clear() async => _memorySession = null;
}
