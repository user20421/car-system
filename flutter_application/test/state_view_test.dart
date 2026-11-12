import 'package:aurora_owner/state/load_state.dart';
import 'package:aurora_owner/widgets/common.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Future<void> pumpState(WidgetTester tester, LoadState<String> state) => tester.pumpWidget(
    MaterialApp(home: Scaffold(body: StateView<String>(state: state, onRetry: () {}, onLogin: () {},
      builder: (value) => Text(value)))));

  testWidgets('关键子页面状态有明确反馈', (tester) async {
    await pumpState(tester, const LoadState.loading());
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    await pumpState(tester, const LoadState.empty());
    expect(find.text('这里还很安静'), findsOneWidget);
    await pumpState(tester, const LoadState.error('服务错误'));
    expect(find.text('服务错误'), findsOneWidget);
    await pumpState(tester, const LoadState.offline());
    expect(find.text('当前处于离线状态'), findsOneWidget);
    await pumpState(tester, const LoadState.unauthorized());
    expect(find.text('登录状态已失效'), findsOneWidget);
    expect(find.text('重新登录'), findsOneWidget);
    await pumpState(tester, const LoadState.ready('车辆数据'));
    expect(find.text('车辆数据'), findsOneWidget);
  });
}
