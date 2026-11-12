import 'package:aurora_owner/theme/aurora_theme.dart';
import 'package:aurora_owner/widgets/telemetry_visuals.dart';
import 'package:aurora_owner/widgets/vehicle_artwork.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('品牌主题、自绘车辆和电池仪表在深浅色下渲染', (tester) async {
    await tester.pumpWidget(MaterialApp(theme: AuroraTheme.light(), home: const Scaffold(
      body: Column(children: [VehicleArtwork(height: 240), BatteryGauge(percent: 82)]))));
    expect(find.byType(VehicleArtwork), findsOneWidget);
    expect(find.byType(BatteryGauge), findsOneWidget);
    expect(find.text('82'), findsOneWidget);
    await tester.pumpWidget(MaterialApp(theme: AuroraTheme.dark(), home: const Scaffold(
      body: BatteryGauge(percent: 42))));
    expect(find.text('42'), findsOneWidget);
  });
}
