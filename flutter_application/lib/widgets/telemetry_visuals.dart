import 'dart:math' as math;
import 'package:flutter/material.dart';
import '../models/telemetry.dart';
import '../theme/aurora_theme.dart';

class BatteryGauge extends StatelessWidget {
  const BatteryGauge({super.key, required this.percent, this.size = 210});
  final double percent;
  final double size;
  @override
  Widget build(BuildContext context) => SizedBox(width: size, height: size, child: CustomPaint(
    painter: _BatteryPainter(percent, Theme.of(context).brightness == Brightness.dark),
    child: Center(child: Column(mainAxisSize: MainAxisSize.min, children: [
      Text('${percent.round()}', style: Theme.of(context).textTheme.displayMedium?.copyWith(fontWeight: FontWeight.w300, height: 1)),
      Text('% 电量', style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant)),
    ]))));
}
class _BatteryPainter extends CustomPainter {
  const _BatteryPainter(this.percent, this.dark);
  final double percent;
  final bool dark;
  @override
  void paint(Canvas canvas, Size size) {
    final center = size.center(Offset.zero);
    final rect = Rect.fromCircle(center: center, radius: size.width * .41);
    canvas.drawArc(rect, 0, math.pi * 2, false, Paint()..style = PaintingStyle.stroke..strokeWidth = 9
      ..color = dark ? Colors.white10 : const Color(0xFFE9ECF4));
    final paint = Paint()..style = PaintingStyle.stroke..strokeWidth = 10..strokeCap = StrokeCap.round
      ..shader = const SweepGradient(startAngle: 0, endAngle: math.pi * 1.5,
        colors: [AuroraColors.cyan, AuroraColors.indigo, AuroraColors.violet, AuroraColors.cyan]).createShader(rect);
    canvas.drawArc(rect, math.pi * .75, math.pi * 1.5 * percent / 100, false, paint);
  }
  @override
  bool shouldRepaint(covariant _BatteryPainter oldDelegate) => oldDelegate.percent != percent || oldDelegate.dark != dark;
}

class TrendChart extends StatelessWidget {
  const TrendChart({super.key, required this.items, this.range = false, this.height = 220});
  final List<Telemetry> items;
  final bool range;
  final double height;
  @override
  Widget build(BuildContext context) => SizedBox(height: height, width: double.infinity,
    child: CustomPaint(painter: _TrendPainter(items, range, Theme.of(context))));
}
class _TrendPainter extends CustomPainter {
  const _TrendPainter(this.items, this.range, this.theme);
  final List<Telemetry> items;
  final bool range;
  final ThemeData theme;
  @override
  void paint(Canvas canvas, Size size) {
    if (items.length < 2) return;
    const left = 38.0, bottom = 26.0, top = 14.0;
    final chart = Rect.fromLTWH(left, top, size.width - left - 8, size.height - top - bottom);
    final values = items.map((item) => range ? item.rangeKm : item.batteryPercent).toList();
    final minValue = values.reduce(math.min), maxValue = values.reduce(math.max);
    final spread = math.max(1.0, maxValue - minValue);
    final text = TextPainter(textDirection: TextDirection.ltr);
    for (var row = 0; row < 4; row++) {
      final y = chart.top + chart.height * row / 3;
      canvas.drawLine(Offset(chart.left, y), Offset(chart.right, y), Paint()..strokeWidth = 1
        ..color = theme.colorScheme.outlineVariant.withValues(alpha: .55));
      final value = maxValue - spread * row / 3;
      text.text = TextSpan(text: value.toStringAsFixed(0), style: TextStyle(fontSize: 10, color: theme.colorScheme.onSurfaceVariant));
      text.layout(); text.paint(canvas, Offset(0, y - 7));
    }
    Offset point(int index) => Offset(chart.left + chart.width * index / (items.length - 1),
        chart.bottom - (values[index] - minValue) / spread * chart.height);
    final path = Path()..moveTo(point(0).dx, point(0).dy);
    for (var i = 1; i < items.length; i++) {
      final previous = point(i - 1), current = point(i);
      final midX = (previous.dx + current.dx) / 2;
      path.cubicTo(midX, previous.dy, midX, current.dy, current.dx, current.dy);
    }
    final fill = Path.from(path)..lineTo(chart.right, chart.bottom)..lineTo(chart.left, chart.bottom)..close();
    canvas.drawPath(fill, Paint()..shader = LinearGradient(begin: Alignment.topCenter, end: Alignment.bottomCenter,
      colors: [AuroraColors.indigo.withValues(alpha: .2), AuroraColors.indigo.withValues(alpha: 0)]).createShader(chart));
    canvas.drawPath(path, Paint()..style = PaintingStyle.stroke..strokeWidth = 2.5..strokeCap = StrokeCap.round
      ..shader = const LinearGradient(colors: [AuroraColors.indigo, AuroraColors.cyan]).createShader(chart));
    canvas.drawCircle(point(items.length - 1), 4, Paint()..color = AuroraColors.cyan);
    text.text = TextSpan(text: '${items.length} 个采样点', style: TextStyle(fontSize: 10, color: theme.colorScheme.onSurfaceVariant));
    text.layout(); text.paint(canvas, Offset(chart.left, size.height - 16));
  }
  @override
  bool shouldRepaint(covariant _TrendPainter oldDelegate) => oldDelegate.items != items || oldDelegate.range != range;
}
