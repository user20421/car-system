import 'dart:math' as math;
import 'package:flutter/material.dart';

class VehicleArtwork extends StatefulWidget {
  const VehicleArtwork({super.key, this.height = 250, this.glow = true});
  final double height;
  final bool glow;
  @override
  State<VehicleArtwork> createState() => _VehicleArtworkState();
}
class _VehicleArtworkState extends State<VehicleArtwork> with SingleTickerProviderStateMixin {
  late final AnimationController _controller = AnimationController(vsync: this, duration: const Duration(seconds: 5))..repeat(reverse: true);
  @override
  void dispose() { _controller.dispose(); super.dispose(); }
  @override
  Widget build(BuildContext context) => SizedBox(height: widget.height, width: double.infinity,
    child: AnimatedBuilder(animation: _controller, builder: (context, _) => CustomPaint(
      painter: AuroraCarPainter(progress: _controller.value, glow: widget.glow,
        dark: Theme.of(context).brightness == Brightness.dark))));
}

class AuroraCarPainter extends CustomPainter {
  const AuroraCarPainter({required this.progress, required this.glow, required this.dark});
  final double progress;
  final bool glow;
  final bool dark;
  @override
  void paint(Canvas canvas, Size size) {
    final scale = math.min(size.width / 430, size.height / 245);
    canvas.save();
    canvas.translate((size.width - 430 * scale) / 2, (size.height - 245 * scale) / 2);
    canvas.scale(scale);
    final ground = Paint()..shader = RadialGradient(colors: [
      const Color(0xFF5856D6).withValues(alpha: glow ? .14 + progress * .09 : .08), Colors.transparent])
      .createShader(const Rect.fromLTWH(38, 180, 358, 42))
      ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 20);
    canvas.drawOval(const Rect.fromLTWH(38, 180, 358, 42), ground);
    for (var i = 0; i < 3; i++) {
      final line = Paint()..color = const Color(0xFF16C8C0).withValues(alpha: (.1 - i * .02) * progress)
        ..strokeWidth = 1;
      canvas.drawArc(Rect.fromLTWH(65 - i * 16.0, 22 - i * 12.0, 300 + i * 32.0, 190 + i * 18.0), .1, 2.9, false, line);
    }
    final body = Path()
      ..moveTo(45, 155)..quadraticBezierTo(47, 121, 84, 111)
      ..lineTo(130, 98)..quadraticBezierTo(164, 48, 210, 43)
      ..lineTo(279, 43)..quadraticBezierTo(311, 48, 345, 98)
      ..quadraticBezierTo(386, 105, 395, 137)..lineTo(393, 163)
      ..quadraticBezierTo(384, 178, 355, 179)..lineTo(82, 179)
      ..quadraticBezierTo(48, 176, 45, 155);
    final paint = Paint()..shader = const LinearGradient(begin: Alignment.topLeft, end: Alignment.bottomRight,
      colors: [Color(0xFF9BA8D8), Color(0xFF3B4778), Color(0xFF151C38), Color(0xFF59658F)],
      stops: [0, .35, .72, 1]).createShader(const Rect.fromLTWH(40, 40, 360, 145));
    canvas.drawShadow(body, Colors.black.withValues(alpha: .35), 12, true);
    canvas.drawPath(body, paint);
    final glass = Path()..moveTo(141, 96)..quadraticBezierTo(169, 57, 207, 53)
      ..lineTo(271, 53)..quadraticBezierTo(297, 58, 326, 96)..close();
    canvas.drawPath(glass, Paint()..shader = const LinearGradient(begin: Alignment.topCenter, end: Alignment.bottomCenter,
      colors: [Color(0xFF9BDCE7), Color(0xFF263454), Color(0xFF151D36)]).createShader(const Rect.fromLTWH(140, 50, 190, 50)));
    canvas.drawLine(const Offset(226, 53), const Offset(229, 96), Paint()..color = const Color(0xFF657198)..strokeWidth = 3);
    final shoulder = Paint()..shader = const LinearGradient(colors: [Color(0xFF9FF9F0), Color(0xFF7F77EE)])
      .createShader(const Rect.fromLTWH(66, 106, 304, 30))
      ..strokeWidth = 2..style = PaintingStyle.stroke;
    canvas.drawPath(Path()..moveTo(66, 124)..quadraticBezierTo(203, 106, 370, 119), shoulder);
    canvas.drawLine(const Offset(230, 105), const Offset(230, 164), Paint()..color = Colors.white10..strokeWidth = 1);
    canvas.drawRRect(RRect.fromRectAndRadius(const Rect.fromLTWH(246, 111, 24, 4), const Radius.circular(2)), Paint()..color = Colors.white54);
    final lamp = Paint()..shader = const LinearGradient(colors: [Colors.white, Color(0xFF65F6EE)])
      .createShader(const Rect.fromLTWH(342, 108, 40, 20))
      ..maskFilter = const MaskFilter.blur(BlurStyle.solid, 2);
    canvas.drawPath(Path()..moveTo(342, 108)..quadraticBezierTo(370, 110, 382, 124)..lineTo(352, 119)..close(), lamp);
    canvas.drawRRect(RRect.fromRectAndRadius(const Rect.fromLTWH(49, 130, 15, 19), const Radius.circular(4)), Paint()..color = const Color(0xFFFF6E8A));
    for (final x in [112.0, 325.0]) {
      canvas.drawCircle(Offset(x, 174), 34, Paint()..color = const Color(0xFF0A0D17));
      canvas.drawCircle(Offset(x, 174), 24, Paint()..color = const Color(0xFF7782A1));
      canvas.drawCircle(Offset(x, 174), 19, Paint()..color = const Color(0xFF171D31));
      final spokes = Paint()..color = const Color(0xFFB9C4DA)..strokeWidth = 2;
      for (var i = 0; i < 6; i++) {
        final angle = i * math.pi / 3 + progress * .08;
        canvas.drawLine(Offset(x, 174), Offset(x + math.cos(angle) * 16, 174 + math.sin(angle) * 16), spokes);
      }
      canvas.drawCircle(Offset(x, 174), 4, Paint()..color = const Color(0xFF16C8C0));
    }
    canvas.restore();
  }
  @override
  bool shouldRepaint(covariant AuroraCarPainter oldDelegate) => oldDelegate.progress != progress || oldDelegate.dark != dark;
}
