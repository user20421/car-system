import '../models/alert.dart';
import '../models/auth.dart';
import '../models/command.dart';
import '../models/telemetry.dart';
import '../models/vehicle.dart';
import 'api_client.dart';
import 'repository.dart';

class HttpCarRepository implements CarRepository {
  HttpCarRepository(this.api);
  final ApiClient api;
  @override
  bool get isDemo => false;
  @override
  Stream<Telemetry> get telemetryStream => const Stream.empty();
  @override
  Stream<VehicleCommand> get commandStream => const Stream.empty();
  @override
  Future<AuthSession> login(String username, String password) => api.login(username, password);
  @override
  Future<AuthSession?> restoreSession() => api.restoreSession();
  @override
  Future<void> logout() => api.logout();

  @override
  Future<List<Vehicle>> listVehicles() async {
    final data = await api.request<Map<String, dynamic>>('GET', '/api/v1/vehicles', query: {'page': 1, 'page_size': 100});
    return (data['items'] as List? ?? const []).map((item) => Vehicle.fromJson(item as Map<String, dynamic>)).toList();
  }

  @override
  Future<Telemetry> latestTelemetry(String vin) async {
    final data = await api.request<Map<String, dynamic>>('GET', '/api/v1/vehicles/$vin/telemetry/latest');
    return Telemetry.fromJson(data);
  }

  @override
  Future<List<Telemetry>> telemetryHistory(String vin, DateTime from, DateTime to) async {
    final data = await api.request<Map<String, dynamic>>('GET', '/api/v1/vehicles/$vin/telemetry/history', query: {
      'from_ms': from.toUtc().millisecondsSinceEpoch, 'to_ms': to.toUtc().millisecondsSinceEpoch, 'limit': 1000,
    });
    return (data['items'] as List? ?? const []).map((item) => Telemetry.fromJson(item as Map<String, dynamic>)).toList();
  }

  @override
  Future<VehicleCommand> createCommand(String vin, String type, Map<String, String> params, String idempotencyKey, DateTime expiresAt) async {
    final data = await api.request<Map<String, dynamic>>('POST', '/api/v1/vehicles/$vin/commands', data: {
      'idempotency_key': idempotencyKey, 'expires_at_ms': expiresAt.toUtc().millisecondsSinceEpoch,
      'type': type, 'params': params,
    });
    return VehicleCommand.fromJson(data);
  }

  @override
  Future<VehicleCommand> getCommand(String vin, String commandId) async {
    final data = await api.request<Map<String, dynamic>>('GET', '/api/v1/vehicles/$vin/commands/$commandId');
    return VehicleCommand.fromJson(data);
  }

  @override
  Future<List<VehicleAlert>> listAlerts({String? vin}) async {
    final data = await api.request<Map<String, dynamic>>('GET', '/api/v1/alerts', query: {
      'vin': ?vin, 'page': 1, 'page_size': 100,
    });
    return (data['items'] as List? ?? const []).map((item) => VehicleAlert.fromJson(item as Map<String, dynamic>)).toList();
  }

  @override
  Future<VehicleAlert> acknowledgeAlert(String id, String note) async {
    final data = await api.request<Map<String, dynamic>>('POST', '/api/v1/alerts/$id/ack', data: {'note': note});
    return VehicleAlert.fromJson(data);
  }

  @override
  Future<WsTicket> createWsTicket() async {
    final data = await api.request<Map<String, dynamic>>('POST', '/api/v1/ws-ticket');
    return WsTicket.fromJson(data);
  }
}
