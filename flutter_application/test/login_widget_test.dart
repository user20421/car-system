import 'package:aurora_owner/screens/login_screen.dart';
import 'package:aurora_owner/services/demo_car_repository.dart';
import 'package:aurora_owner/services/repository.dart';
import 'package:aurora_owner/state/auth_controller.dart';
import 'package:aurora_owner/theme/aurora_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';

void main() {
  testWidgets('显式演示登录展示品牌、演示标识并建立会话', (tester) async {
    final repository = DemoCarRepository();
    final auth = AuthController(repository);
    addTearDown(() { auth.dispose(); repository.dispose(); });
    await tester.pumpWidget(MultiProvider(providers: [Provider<CarRepository>.value(value: repository),
      ChangeNotifierProvider.value(value: auth)], child: MaterialApp(theme: AuroraTheme.light(), home: const LoginScreen())));
    await tester.pump();
    expect(find.text('AURORA'), findsOneWidget);
    expect(find.text('演示模式'), findsOneWidget);
    expect(find.byKey(const Key('login-username')), findsOneWidget);
    await tester.tap(find.byKey(const Key('login-submit')));
    await tester.pump();
    expect(auth.isBusy, isTrue);
    await tester.pump(const Duration(milliseconds: 500));
    expect(auth.status, AuthStatus.authenticated);
    expect(auth.session!.user.displayName, '林澈');
  });

  testWidgets('真实 API 默认不预填演示账号', (tester) async {
    final repository = _RealMarkedDemoRepository();
    final auth = AuthController(repository);
    addTearDown(() { auth.dispose(); repository.dispose(); });
    await tester.pumpWidget(MultiProvider(providers: [Provider<CarRepository>.value(value: repository),
      ChangeNotifierProvider.value(value: auth)], child: const MaterialApp(home: LoginScreen())));
    await tester.pump();
    expect(tester.widget<TextFormField>(find.byKey(const Key('login-password'))).controller!.text, isEmpty);
  });
}

class _RealMarkedDemoRepository extends DemoCarRepository {
  @override
  bool get isDemo => false;
}
