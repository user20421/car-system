import 'package:flutter/foundation.dart';
import '../models/auth.dart';
import '../services/repository.dart';

enum AuthStatus { restoring, unauthenticated, authenticated }
class AuthController extends ChangeNotifier {
  AuthController(this.repository);
  final CarRepository repository;
  AuthStatus status = AuthStatus.restoring;
  AuthSession? session;
  bool isBusy = false;
  String? loginError;

  Future<void> restore() async {
    try {
      session = await repository.restoreSession();
      status = session == null ? AuthStatus.unauthenticated : AuthStatus.authenticated;
    } catch (_) {
      session = null;
      status = AuthStatus.unauthenticated;
    }
    notifyListeners();
  }

  Future<bool> login(String username, String password) async {
    if (isBusy) return false;
    isBusy = true;
    loginError = null;
    notifyListeners();
    try {
      session = await repository.login(username.trim(), password);
      status = AuthStatus.authenticated;
      return true;
    } on Object catch (error) {
      loginError = error is RepositoryException ? error.message : '登录失败，请稍后重试';
      return false;
    } finally {
      isBusy = false;
      notifyListeners();
    }
  }

  Future<void> logout() async {
    try {
      await repository.logout();
    } on Object {
      // Local logout still completes offline; the short-lived access token expires server-side.
    } finally {
      session = null;
      status = AuthStatus.unauthenticated;
      notifyListeners();
    }
  }

  void expireSession() {
    session = null;
    status = AuthStatus.unauthenticated;
    loginError = '会话已过期，请重新登录';
    notifyListeners();
  }
}
