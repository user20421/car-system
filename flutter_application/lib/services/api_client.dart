import 'dart:async';
import 'package:dio/dio.dart';
import '../models/auth.dart';
import 'repository.dart';
import 'session_store.dart';
import 'platform/adapter_io.dart' if (dart.library.html) 'platform/adapter_web.dart' as platform;

class ApiClient {
  ApiClient({String? baseUrl, SessionStore? store, this.onSessionExpired})
      : store = store ?? createSessionStore(),
        dio = Dio(BaseOptions(
          baseUrl: baseUrl ?? const String.fromEnvironment('AURORA_API_BASE_URL', defaultValue: 'https://localhost:8443'),
          connectTimeout: const Duration(seconds: 12), receiveTimeout: const Duration(seconds: 20),
          headers: {'accept': 'application/json'},
        )) {
    platform.configureAdapter(dio);
    dio.interceptors.add(InterceptorsWrapper(onRequest: _onRequest, onError: _onError));
  }

  final Dio dio;
  final SessionStore store;
  void Function()? onSessionExpired;
  AuthSession? session;
  Future<AuthSession>? _refreshing;

  Future<void> _onRequest(RequestOptions options, RequestInterceptorHandler handler) async {
    if (options.extra['skipAuth'] != true && session != null && session!.accessToken.isNotEmpty) {
      options.headers['authorization'] = 'Bearer ${session!.accessToken}';
    }
    handler.next(options);
  }

  Future<void> _onError(DioException error, ErrorInterceptorHandler handler) async {
    final options = error.requestOptions;
    if (error.response?.statusCode != 401 || options.extra['skipRefresh'] == true || session == null) {
      return handler.next(error);
    }
    try {
      await refresh();
      final response = await dio.fetch(options..extra['skipRefresh'] = true);
      return handler.resolve(response);
    } catch (_) {
      await clearSession();
      onSessionExpired?.call();
      return handler.next(error);
    }
  }

  Future<AuthSession> login(String username, String password) async {
    final response = await request<Map<String, dynamic>>('POST', '/api/v1/auth/login',
        data: {'username': username, 'password': password}, skipAuth: true);
    final next = AuthSession.fromJson(response);
    session = next;
    await store.write(next);
    return next;
  }

  Future<AuthSession?> restoreSession() async {
    session = await store.read();
    if (session == null && store.usesCookieSession) {
      // Cookie deployments can restore an HttpOnly session without exposing tokens to JavaScript.
      final userJson = await request<Map<String, dynamic>>('GET', '/api/v1/auth/me', skipAuth: true, skipRefresh: true);
      session = AuthSession(accessToken: '', refreshToken: '', expiresAt: DateTime.now().add(const Duration(minutes: 15)),
          user: OwnerUser.fromJson(userJson));
      return session;
    }
    if (session == null) return null;
    if (store.usesCookieSession && session!.accessToken.isEmpty) return session;
    return refresh();
  }

  Future<AuthSession> refresh() async {
    if (_refreshing != null) return _refreshing!;
    final current = session;
    if (current == null) throw const RepositoryException(RepositoryErrorKind.unauthorized, '登录已过期');
    _refreshing = _refresh(current);
    try {
      return await _refreshing!;
    } finally {
      _refreshing = null;
    }
  }

  Future<AuthSession> _refresh(AuthSession current) async {
    final response = await request<Map<String, dynamic>>('POST', '/api/v1/auth/refresh',
        data: {'refresh_token': current.refreshToken}, skipAuth: true, skipRefresh: true);
    final next = AuthSession.fromJson(response);
    session = next;
    await store.write(next);
    return next;
  }

  Future<void> logout() async {
    try {
      if (session != null) {
        await request<void>('POST', '/api/v1/auth/logout', data: {'refresh_token': session!.refreshToken});
      }
    } finally {
      await clearSession();
    }
  }

  Future<void> clearSession() async {
    session = null;
    await store.clear();
  }

  Future<T> request<T>(String method, String path, {Object? data, Map<String, dynamic>? query,
      bool skipAuth = false, bool skipRefresh = false}) async {
    try {
      final response = await dio.request<dynamic>(path, data: data, queryParameters: query,
          options: Options(method: method, extra: {'skipAuth': skipAuth, 'skipRefresh': skipRefresh}));
      return response.data as T;
    } on DioException catch (error) {
      throw mapDioError(error);
    } on TypeError {
      throw const RepositoryException(RepositoryErrorKind.server, '服务返回了无法识别的数据');
    }
  }

  RepositoryException mapDioError(DioException error) {
    final status = error.response?.statusCode;
    final body = error.response?.data;
    final detail = body is Map ? (body['detail'] ?? body['title'])?.toString() : null;
    final code = body is Map ? body['code']?.toString() : null;
    final kind = switch (status) {
      401 => RepositoryErrorKind.unauthorized,
      403 => RepositoryErrorKind.forbidden,
      404 => RepositoryErrorKind.notFound,
      409 => RepositoryErrorKind.conflict,
      _ when error.type == DioExceptionType.connectionError || error.type == DioExceptionType.connectionTimeout ||
          error.type == DioExceptionType.receiveTimeout => RepositoryErrorKind.offline,
      _ => RepositoryErrorKind.server,
    };
    return RepositoryException(kind, detail ?? switch (kind) {
      RepositoryErrorKind.offline => '网络连接不可用，请检查网络后重试',
      RepositoryErrorKind.unauthorized => '登录已过期，请重新登录',
      RepositoryErrorKind.forbidden => '你没有执行此操作的权限',
      RepositoryErrorKind.notFound => '未找到车辆或数据',
      RepositoryErrorKind.conflict => '操作状态已变化，请刷新后重试',
      _ => '服务暂时不可用，请稍后重试',
    }, code);
  }
}
