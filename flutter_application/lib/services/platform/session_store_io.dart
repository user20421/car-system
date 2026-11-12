import 'dart:convert';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import '../../models/auth.dart';
import '../session_store.dart';

class PlatformSessionStore implements SessionStore {
  PlatformSessionStore() : _storage = const FlutterSecureStorage(
    aOptions: AndroidOptions(),
    iOptions: IOSOptions(accessibility: KeychainAccessibility.first_unlock_this_device),
  );
  final FlutterSecureStorage _storage;
  static const _key = 'aurora.native.session.v1';
  @override
  bool get usesCookieSession => false;
  @override
  Future<AuthSession?> read() async {
    final value = await _storage.read(key: _key);
    if (value == null) return null;
    try {
      return AuthSession.fromStoredJson(jsonDecode(value) as Map<String, dynamic>);
    } on FormatException {
      await clear();
      return null;
    }
  }
  @override
  Future<void> write(AuthSession session) => _storage.write(key: _key, value: jsonEncode(session.toJson()));
  @override
  Future<void> clear() => _storage.delete(key: _key);
}
