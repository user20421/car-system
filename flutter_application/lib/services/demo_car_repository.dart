import 'dart:async';
import 'package:uuid/uuid.dart';
import '../models/alert.dart';
import '../models/auth.dart';
import '../models/command.dart';
import '../models/telemetry.dart';
import '../models/vehicle.dart';
import 'repository.dart';

/// Explicit, local-only repository for product demos and deterministic widget tests.
class DemoCarRepository implements CarRepository {
  DemoCarRepository() {
    _vehicles = [
      Vehicle(vin: 'LAURORA0000000001', displayName: '极光 · 日常座驾', model: 'AURORA S9', ownerUserId: _demoUser.id,
          role: VehicleRole.owner, modelYear: 2026, firmwareVersion: 'AOS 4.8.2', createdAt: DateTime(2025, 8, 18)),
      Vehicle(vin: 'LAURORA0000000002', displayName: '远山 · 家庭出行', model: 'AURORA M7', ownerUserId: _demoUser.id,
          role: VehicleRole.owner, modelYear: 2025, firmwareVersion: 'AOS 4.7.9', createdAt: DateTime(2025, 2, 6)),
      Vehicle(vin: 'LAURORA0000000003', displayName: '城市 · 共享车辆', model: 'AURORA S5', ownerUserId: 'another-owner',
          role: VehicleRole.viewer, modelYear: 2026, firmwareVersion: 'AOS 4.8.1', createdAt: DateTime(2026, 1, 12)),
    ];
    for (final vehicle in _vehicles) {
      _telemetry[vehicle.vin] = Telemetry(vin: vehicle.vin, deviceBootId: 'demo-boot', sequence: 42,
          occurredAt: now.toUtc(), receivedAt: now.toUtc(), speedKph: 0, rpm: 0,
          batteryPercent: vehicle.vin.endsWith('2') ? 42 : 82, rangeKm: vehicle.vin.endsWith('2') ? 238 : 516,
          locked: true, defrost: false, cabinTempC: 23.5,
          hvac: const HvacState(mode: 'auto', targetTempC: 22, fanLevel: 2, acOn: true),
          location: const VehicleLocation(latitude: 31.2304, longitude: 121.4737), faults: const []);
    }
    _alerts = [VehicleAlert(id: 'd2a93f38-c348-4a55-b0bd-e3621a07af9c', vin: _vehicles.first.vin,
        code: 'TIRE_PRESSURE_LOW', title: '左后轮胎压偏低，建议尽快检查', severity: 'warning', status: 'open',
        occurredAt: now.subtract(const Duration(hours: 2)))];
  }

  static final _demoUser = OwnerUser(id: '2e5c2273-e53b-49c0-8153-a5a2e54f95d0', username: 'demo',
      displayName: '林澈', roles: const ['user'], createdAt: DateTime(2024, 6, 1), email: 'demo@aurora.example');
  late final List<Vehicle> _vehicles;
  final _telemetry = <String, Telemetry>{};
  final _commands = <String, VehicleCommand>{};
  late List<VehicleAlert> _alerts;
  final _commandEvents = StreamController<VehicleCommand>.broadcast();
  AuthSession? _session;
  DateTime get now => DateTime.now();

  @override
  bool get isDemo => true;
  @override
  Stream<Telemetry> get telemetryStream => const Stream.empty();
  @override
  Stream<VehicleCommand> get commandStream => _commandEvents.stream;

  @override
  Future<AuthSession> login(String username, String password) async {
    await Future<void>.delayed(const Duration(milliseconds: 450));
    if (username != 'demo' || password != 'aurora-demo') {
      throw const RepositoryException(RepositoryErrorKind.unauthorized, '演示账号为 demo，密码为 aurora-demo');
    }
    return _session = AuthSession(accessToken: 'demo-access', refreshToken: 'demo-refresh',
        expiresAt: now.add(const Duration(hours: 8)), user: _demoUser);
  }

  @override
  Future<AuthSession?> restoreSession() async => null;
  @override
  Future<void> logout() async => _session = null;

  AuthSession _requireUser() {
    final session = _session;
    if (session == null) throw const RepositoryException(RepositoryErrorKind.unauthorized, '请先登录');
    return session;
  }

  void _checkAccess(String vin, {bool control = false}) {
    _requireUser();
    final matches = _vehicles.where((vehicle) => vehicle.vin == vin);
    if (matches.isEmpty) throw const RepositoryException(RepositoryErrorKind.notFound, '车辆不存在或不在你的名下');
    if (control && !matches.first.canControl) {
      throw const RepositoryException(RepositoryErrorKind.forbidden, '共享查看者不能控制车辆');
    }
  }

  @override
  Future<List<Vehicle>> listVehicles() async {
    _requireUser();
    await Future<void>.delayed(const Duration(milliseconds: 350));
    return List.unmodifiable(_vehicles);
  }

  @override
  Future<Telemetry> latestTelemetry(String vin) async {
    _checkAccess(vin);
    await Future<void>.delayed(const Duration(milliseconds: 260));
    final current = _telemetry[vin]!;
    final next = current.copyWith(batteryPercent: (current.batteryPercent - .01).clamp(0, 100));
    return _telemetry[vin] = next;
  }

  @override
  Future<List<Telemetry>> telemetryHistory(String vin, DateTime from, DateTime to) async {
    _checkAccess(vin);
    await Future<void>.delayed(const Duration(milliseconds: 400));
    final latest = _telemetry[vin]!;
    final count = to.difference(from).inHours > 30 ? 28 : 24;
    return List.generate(count, (index) {
      final progress = index / (count - 1);
      return latest.copyWith(occurredAt: from.add(to.difference(from) * progress),
          sequence: index, batteryPercent: (latest.batteryPercent - 8 + 8 * progress + (index % 4)).clamp(0, 100),
          rangeKm: (latest.rangeKm - 50 + 50 * progress + (index % 3) * 5).clamp(0, 900));
    });
  }

  @override
  Future<VehicleCommand> createCommand(String vin, String type, Map<String, String> params,
      String idempotencyKey, DateTime expiresAt) async {
    _checkAccess(vin, control: true);
    await Future<void>.delayed(const Duration(milliseconds: 350));
    final existing = _commands.values.where((command) => command.idempotencyKey == idempotencyKey);
    if (existing.isNotEmpty) return existing.first;
    final timestamp = now.toUtc();
    final command = VehicleCommand(id: const Uuid().v4(), vin: vin, requestedBy: _demoUser.id,
        idempotencyKey: idempotencyKey, expiresAt: expiresAt, type: type, params: params,
        status: CommandStatus.pending, createdAt: timestamp, updatedAt: timestamp);
    _commands[command.id] = command;
    return command;
  }

  @override
  Future<VehicleCommand> getCommand(String vin, String commandId) async {
    _checkAccess(vin, control: true);
    await Future<void>.delayed(const Duration(milliseconds: 180));
    final current = _commands[commandId];
    if (current == null || current.vin != vin) {
      throw const RepositoryException(RepositoryErrorKind.notFound, '指令不存在');
    }
    // Advance once per poll instead of reading the wall clock. This keeps demo
    // receipts deterministic in both real async code and Flutter's fake async tests.
    final status = current.isTerminal ? current.status : switch (current.status) {
      CommandStatus.pending => CommandStatus.sent,
      CommandStatus.sent => CommandStatus.received,
      _ => CommandStatus.succeeded,
    };
    final next = current.copyWith(status: status, updatedAt: now.toUtc());
    _commands[commandId] = next;
    if (next.isTerminal && !current.isTerminal) _apply(next);
    _commandEvents.add(next);
    return next;
  }

  void _apply(VehicleCommand command) {
    final telemetry = _telemetry[command.vin]!;
    _telemetry[command.vin] = switch (command.type) {
      'lock' => telemetry.copyWith(locked: true),
      'unlock' => telemetry.copyWith(locked: false),
      'defrost_on' => telemetry.copyWith(defrost: true),
      'defrost_off' => telemetry.copyWith(defrost: false),
      'hvac_on' => telemetry.copyWith(hvac: const HvacState(mode: 'auto', targetTempC: 22, fanLevel: 2, acOn: true)),
      'hvac_off' => telemetry.copyWith(hvac: const HvacState()),
      'set_temperature' => telemetry.copyWith(hvac: HvacState(mode: 'auto', fanLevel: 2, acOn: true,
          targetTempC: double.tryParse(command.params['target_temp_c'] ?? '') ?? 22)),
      _ => telemetry,
    };
  }

  @override
  Future<List<VehicleAlert>> listAlerts({String? vin}) async {
    _requireUser();
    if (vin != null) _checkAccess(vin);
    await Future<void>.delayed(const Duration(milliseconds: 250));
    return _alerts.where((alert) => vin == null || alert.vin == vin).toList();
  }

  @override
  Future<VehicleAlert> acknowledgeAlert(String id, String note) async {
    _requireUser();
    final index = _alerts.indexWhere((alert) => alert.id == id);
    if (index < 0) throw const RepositoryException(RepositoryErrorKind.notFound, '告警不存在');
    final alert = _alerts[index];
    _checkAccess(alert.vin);
    final next = VehicleAlert(id: alert.id, vin: alert.vin, code: alert.code, title: alert.title,
        severity: alert.severity, status: 'acknowledged', occurredAt: alert.occurredAt,
        acknowledgedAt: now.toUtc(), note: note);
    return _alerts[index] = next;
  }

  @override
  Future<WsTicket> createWsTicket() async => WsTicket(ticket: 'demo-ticket-not-a-real-credential-0000',
      expiresAt: now.add(const Duration(minutes: 1)), websocketUrl: 'wss://demo.invalid/ws');

  void dispose() => _commandEvents.close();
}
