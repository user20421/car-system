import 'package:aurora_owner/services/demo_car_repository.dart';
import 'package:aurora_owner/services/repository.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  late DemoCarRepository repository;
  setUp(() async {
    repository = DemoCarRepository();
    await repository.login('demo', 'aurora-demo');
  });
  tearDown(() => repository.dispose());

  test('车辆列表包含服务端授权的车主和共享查看角色', () async {
    final vehicles = await repository.listVehicles();
    expect(vehicles, hasLength(3));
    expect(vehicles.where((vehicle) => vehicle.canControl), hasLength(2));
    expect(vehicles.last.canControl, isFalse);
  });

  test('仅查看车辆拒绝控车，不泄露任意 VIN 数据', () async {
    await expectLater(repository.createCommand('LAURORA0000000003', 'unlock', {}, 'viewer-key-1',
      DateTime.now().add(const Duration(seconds: 30))),
      throwsA(isA<RepositoryException>().having((error) => error.kind, 'kind', RepositoryErrorKind.forbidden)));
    await expectLater(repository.latestTelemetry('LNOTAUTHORIZED001'),
      throwsA(isA<RepositoryException>().having((error) => error.kind, 'kind', RepositoryErrorKind.notFound)));
  });

  test('未登录无法枚举车辆', () async {
    await repository.logout();
    await expectLater(repository.listVehicles(),
      throwsA(isA<RepositoryException>().having((error) => error.kind, 'kind', RepositoryErrorKind.unauthorized)));
  });
}
