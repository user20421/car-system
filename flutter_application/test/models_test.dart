import 'package:aurora_owner/models/alert.dart';
import 'package:aurora_owner/models/auth.dart';
import 'package:aurora_owner/models/command.dart';
import 'package:aurora_owner/models/telemetry.dart';
import 'package:aurora_owner/models/vehicle.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('登录响应与嵌套用户按 OpenAPI 字段解析', () {
    final session = AuthSession.fromJson({'access_token': 'access', 'refresh_token': 'refresh',
      'token_type': 'Bearer', 'expires_in': 900, 'user': {'id': 'u1', 'username': 'owner',
        'display_name': '车主', 'roles': ['user'], 'created_at': '2026-01-01T00:00:00Z'}});
    expect(session.accessToken, 'access');
    expect(session.refreshToken, 'refresh');
    expect(session.user.displayName, '车主');
    expect(session.isExpired, isFalse);
    expect(session.expiresAt.difference(DateTime.now()).inMinutes, lessThanOrEqualTo(15));
  });

  test('多车角色、遥测嵌套结构和 UTC 毫秒正确解析', () {
    final vehicle = Vehicle.fromJson({'vin': 'LAURORA0000000001', 'display_name': '极光', 'model': 'S9',
      'owner_user_id': 'u1', 'access_role': 'viewer', 'created_at': '2026-01-01T00:00:00Z'});
    expect(vehicle.role, VehicleRole.viewer);
    expect(vehicle.canControl, isFalse);
    final telemetry = Telemetry.fromJson({'vin': vehicle.vin, 'device_boot_id': 'b1', 'sequence': 9,
      'occurred_at_ms': 1767225600000, 'speed_kph': 0, 'rpm': 0, 'battery_percent': 82.5,
      'range_km': 510, 'locked': true, 'defrost': false, 'cabin_temp_c': 21.5,
      'hvac': {'mode': 'auto', 'target_temp_c': 22, 'fan_level': 2, 'ac_on': true},
      'location': {'latitude': 31.2, 'longitude': 121.4}, 'fault': [
        {'code': 'TPMS', 'severity': 'warning', 'active': true, 'message': '胎压'}]});
    expect(telemetry.occurredAt.isUtc, isTrue);
    expect(telemetry.occurredAt.millisecondsSinceEpoch, 1767225600000);
    expect(telemetry.hvac.targetTempC, 22);
    expect(telemetry.location!.latitude, 31.2);
    expect(telemetry.faults.single.code, 'TPMS');
  });

  test('指令与告警终态及结构化错误解析', () {
    final command = VehicleCommand.fromJson({'command_id': 'c1', 'vin': 'v1', 'requested_by': 'u1',
      'idempotency_key': '12345678', 'expires_at_ms': 10, 'type': 'lock', 'params': {},
      'status': 'failed', 'created_at_ms': 1, 'updated_at_ms': 2,
      'error': <dynamic, dynamic>{'code': 'VEHICLE_BUSY', 'message': '车辆忙碌', 'retryable': true,
        'details': <dynamic, dynamic>{'reason': 'charging'}}});
    expect(command.status, CommandStatus.failed);
    expect(command.isTerminal, isTrue);
    expect(command.error!.retryable, isTrue);
    expect(command.error!.details, {'reason': 'charging'});
    final alert = VehicleAlert.fromJson({'id': 'a1', 'vin': 'v1', 'code': 'TIRE', 'title': '胎压',
      'severity': 'warning', 'status': 'open', 'occurred_at_ms': 100});
    expect(alert.isOpen, isTrue);
    expect(alert.occurredAt.millisecondsSinceEpoch, 100);
  });
}
