import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_application/main.dart';

void main() {
  testWidgets('App 启动显示登录页', (WidgetTester tester) async {
    await tester.pumpWidget(const CarApp());
    expect(find.text('Car-System 车主端'), findsOneWidget);
    expect(find.byIcon(Icons.directions_car), findsOneWidget);
  });
}
