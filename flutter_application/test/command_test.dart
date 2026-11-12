import 'dart:async';
import 'package:aurora_owner/models/command.dart';
import 'package:aurora_owner/services/demo_car_repository.dart';
import 'package:aurora_owner/services/repository.dart';
import 'package:aurora_owner/state/command_controller.dart';
import 'package:flutter_test/flutter_test.dart';

VehicleCommand command(CommandStatus status, {DateTime? createdAt}) => VehicleCommand(id: 'c1', vin: 'v1',
  requestedBy: 'u1', idempotencyKey: '12345678', expiresAt: DateTime.fromMillisecondsSinceEpoch(10000),
  type: 'lock', params: const {}, status: status, createdAt: createdAt ?? DateTime.fromMillisecondsSinceEpoch(1),
  updatedAt: DateTime.fromMillisecondsSinceEpoch(2));

void main() {
  test('指令只允许向前推进且终态不可逆', () {
    const machine = CommandStateMachine();
    final sent = machine.reduce(command(CommandStatus.pending), command(CommandStatus.sent));
    expect(sent.status, CommandStatus.sent);
    expect(machine.reduce(sent, command(CommandStatus.pending)).status, CommandStatus.sent);
    final succeeded = machine.reduce(sent, command(CommandStatus.succeeded));
    expect(machine.reduce(succeeded, command(CommandStatus.failed)).status, CommandStatus.succeeded);
    expect(machine.reduce(succeeded, command(CommandStatus.expired)).status, CommandStatus.succeeded);
  });

  test('提交期间防重，接受指令后不伪造成功', () async {
    final repository = _SlowDemoRepository();
    await repository.login('demo', 'aurora-demo');
    final controller = CommandController(repository);
    final first = controller.send('LAURORA0000000001', 'lock');
    await Future<void>.delayed(Duration.zero);
    expect(controller.isSending, isTrue);
    await expectLater(controller.send('LAURORA0000000001', 'lock'),
      throwsA(isA<RepositoryException>().having((error) => error.kind, 'kind', RepositoryErrorKind.conflict)));
    repository.gate.complete();
    final accepted = await first;
    expect(accepted.status, CommandStatus.pending);
    expect(controller.latest!.isTerminal, isFalse);
    await expectLater(controller.send('LAURORA0000000001', 'lock'),
      throwsA(isA<RepositoryException>().having((error) => error.kind, 'kind', RepositoryErrorKind.conflict)));
    controller.dispose();
    repository.dispose();
  });

  test('演示车辆只通过后续车端回执到达成功', () async {
    final repository = DemoCarRepository();
    await repository.login('demo', 'aurora-demo');
    final command = await repository.createCommand('LAURORA0000000001', 'lock', {}, 'unique-key-123',
      DateTime.now().add(const Duration(seconds: 30)));
    expect(command.status, CommandStatus.pending);
    await Future<void>.delayed(const Duration(milliseconds: 650));
    expect((await repository.getCommand(command.vin, command.id)).status, CommandStatus.sent);
    await Future<void>.delayed(const Duration(milliseconds: 600));
    expect((await repository.getCommand(command.vin, command.id)).status, CommandStatus.received);
    await Future<void>.delayed(const Duration(milliseconds: 900));
    expect((await repository.getCommand(command.vin, command.id)).status, CommandStatus.succeeded);
    repository.dispose();
  });
}

class _SlowDemoRepository extends DemoCarRepository {
  final gate = Completer<void>();
  @override
  Future<VehicleCommand> createCommand(String vin, String type, Map<String, String> params,
      String idempotencyKey, DateTime expiresAt) async {
    await gate.future;
    return super.createCommand(vin, type, params, idempotencyKey, expiresAt);
  }
}
