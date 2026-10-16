import 'package:dio/dio.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// 后端 API 客户端：统一鉴权、401 处理
class Api {
  static const String baseUrl = 'http://localhost:8080';
  static final Dio _dio = Dio(BaseOptions(
    baseUrl: baseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 10),
  ));

  static String? token;

  /// 启动时恢复本地 Token
  static Future<void> init() async {
    final prefs = await SharedPreferences.getInstance();
    token = prefs.getString('token');
    _dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) {
        if (token != null) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        handler.next(options);
      },
    ));
  }

  static Future<String?> login(String username, String password) async {
    try {
      final res = await _dio.post('/api/v1/auth/login',
          data: {'username': username, 'password': password});
      token = res.data['data']['token'];
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString('token', token!);
      await prefs.setString('username', username);
      return null; // null 表示成功
    } on DioException catch (e) {
      if (e.response?.statusCode == 401) return '用户名或密码错误';
      if (e.response != null) return '服务器错误 (${e.response?.statusCode})';
      return '无法连接后端 (localhost:8080)，请确认后端已启动';
    } catch (e) {
      return '未知错误: $e';
    }
  }

  static Future<void> logout() async {
    token = null;
    final prefs = await SharedPreferences.getInstance();
    await prefs.clear();
  }

  /// 我的车辆列表
  static Future<List<dynamic>> myVehicles() async {
    final res = await _dio.get('/api/v1/my/vehicles');
    return res.data['data'] ?? [];
  }

  /// 车辆最新车况（Redis 优先）
  static Future<Map<String, dynamic>?> latestTelemetry(String vin) async {
    try {
      final res = await _dio.get('/api/v1/vehicle/$vin/telemetry/latest');
      return Map<String, dynamic>.from(res.data['data']);
    } catch (_) {
      return null;
    }
  }

  /// 下发控车指令
  static Future<int> sendCommand(String vin, String command) async {
    final res = await _dio.post('/api/v1/vehicle/$vin/command',
        data: {'command': command});
    return res.data['data']['id'];
  }

  /// 车辆详情（含锁/空调状态）
  static Future<Map<String, dynamic>?> getVehicle(String vin) async {
    try {
      final res = await _dio.get('/api/v1/vehicle/$vin');
      return Map<String, dynamic>.from(res.data['data']);
    } catch (_) {
      return null;
    }
  }

  /// 车况历史（时间正序）
  static Future<List<dynamic>> telemetryHistory(String vin, {int limit = 60}) async {
    try {
      final res = await _dio.get('/api/v1/vehicle/$vin/telemetry/history?limit=$limit');
      return res.data['data'] ?? [];
    } catch (_) {
      return [];
    }
  }

  /// 指令历史
  static Future<List<dynamic>> commands(String vin, {int limit = 10}) async {
    final res = await _dio.get('/api/v1/vehicle/$vin/commands?limit=$limit');
    return res.data['data'] ?? [];
  }
}
