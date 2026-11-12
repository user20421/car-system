import '../models/alert.dart';
import '../models/auth.dart';
import '../models/command.dart';
import '../models/telemetry.dart';
import '../models/vehicle.dart';

enum RepositoryErrorKind { offline, unauthorized, forbidden, notFound, conflict, server, unknown }

class RepositoryException implements Exception {
  const RepositoryException(this.kind, this.message, [this.code]);
  final RepositoryErrorKind kind;
  final String message;
  final String? code;
  @override
  String toString() => message;
}

abstract interface class CarRepository {
  bool get isDemo;
  Stream<Telemetry> get telemetryStream;
  Stream<VehicleCommand> get commandStream;
  Future<AuthSession> login(String username, String password);
  Future<AuthSession?> restoreSession();
  Future<void> logout();
  Future<List<Vehicle>> listVehicles();
  Future<Telemetry> latestTelemetry(String vin);
  Future<List<Telemetry>> telemetryHistory(String vin, DateTime from, DateTime to);
  Future<VehicleCommand> createCommand(String vin, String type, Map<String, String> params, String idempotencyKey, DateTime expiresAt);
  Future<VehicleCommand> getCommand(String vin, String commandId);
  Future<List<VehicleAlert>> listAlerts({String? vin});
  Future<VehicleAlert> acknowledgeAlert(String id, String note);
  Future<WsTicket> createWsTicket();
}
