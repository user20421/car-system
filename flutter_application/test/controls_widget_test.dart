import 'package:aurora_owner/models/vehicle.dart';
import 'package:aurora_owner/screens/controls_screen.dart';
import 'package:aurora_owner/services/demo_car_repository.dart';
import 'package:aurora_owner/services/repository.dart';
import 'package:aurora_owner/state/auth_controller.dart';
import 'package:aurora_owner/state/command_controller.dart';
import 'package:aurora_owner/state/connectivity_controller.dart';
import 'package:aurora_owner/state/garage_controller.dart';
import 'package:aurora_owner/state/settings_controller.dart';
import 'package:aurora_owner/state/vehicle_controller.dart';
import 'package:aurora_owner/theme/aurora_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';

void main() {
  late _ClockDemoRepository repository;
  late GarageController garage;
  late VehicleController vehicle;
  late CommandController commands;
  late ConnectivityController connectivity;
  late SettingsController settings;
  late AuthController auth;
  setUp(() async {
    repository = _ClockDemoRepository();
    await repository.login('demo', 'aurora-demo');
    garage = GarageController(repository);
    vehicle = VehicleController(repository);
    commands = CommandController(repository);
    connectivity = ConnectivityController(listenToPlatform: false);
    settings = SettingsController();
    auth = AuthController(repository);
    await garage.load();
    await vehicle.select(garage.selected!.vin);
  });
  tearDown(() {
    garage.dispose(); vehicle.dispose(); commands.dispose(); connectivity.dispose(); settings.dispose(); auth.dispose(); repository.dispose();
  });
  Future<void> pumpControls(WidgetTester tester) => tester.pumpWidget(MultiProvider(providers: [
    Provider<CarRepository>.value(value: repository), ChangeNotifierProvider.value(value: garage),
    ChangeNotifierProvider.value(value: vehicle), ChangeNotifierProvider.value(value: commands),
    ChangeNotifierProvider.value(value: connectivity), ChangeNotifierProvider.value(value: settings),
    ChangeNotifierProvider.value(value: auth),
  ], child: MaterialApp(theme: AuroraTheme.light(), home: const ControlsScreen())));

  testWidgets('控车必须二次确认，提交后先展示 pending 再等待回执', (tester) async {
    await pumpControls(tester);
    await tester.pump();
    await tester.tap(find.byKey(const Key('control-lock')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(find.text('确认解锁车辆？'), findsOneWidget);
    await tester.tap(find.text('确认发送'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 450));
    await tester.pump(const Duration(milliseconds: 300));
    expect(find.text('指令已安全创建，等待下发'), findsOneWidget);
    expect(find.text('车辆已确认执行成功'), findsNothing);
    for (var i = 0; i < 8 && find.text('车辆已确认执行成功').evaluate().isEmpty; i++) {
      repository.current = repository.current.add(const Duration(milliseconds: 700));
      await tester.pump(const Duration(milliseconds: 800));
      await tester.pump(const Duration(milliseconds: 300));
    }
    expect(find.text('车辆已确认执行成功'), findsOneWidget);
    vehicle.stop();
  });

  testWidgets('viewer 无法发送控车且界面明确解释', (tester) async {
    garage.select('LAURORA0000000003');
    final selecting = vehicle.select('LAURORA0000000003');
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    await selecting;
    expect(garage.selected!.role, VehicleRole.viewer);
    await pumpControls(tester);
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text('你拥有仅查看权限，不能发送控车指令'), findsOneWidget);
    expect(tester.widget<FilledButton>(find.byKey(const Key('control-lock'))).onPressed, isNull);
    vehicle.stop();
  });
}

class _ClockDemoRepository extends DemoCarRepository {
  DateTime current = DateTime(2026, 11, 12, 12);
  @override
  DateTime get now => current;
}
