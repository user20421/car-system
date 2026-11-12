import 'package:flutter/material.dart';

abstract final class AuroraColors {
  static const indigo = Color(0xFF5856D6);
  static const violet = Color(0xFF8B5CF6);
  static const cyan = Color(0xFF16C8C0);
  static const ink = Color(0xFF11152A);
  static const success = Color(0xFF16856B);
  static const warning = Color(0xFFB56A13);
}

class AuroraTheme {
  static ThemeData light() => _theme(Brightness.light);
  static ThemeData dark() => _theme(Brightness.dark);
  static ThemeData _theme(Brightness brightness) {
    final dark = brightness == Brightness.dark;
    final scheme = ColorScheme.fromSeed(seedColor: AuroraColors.indigo, brightness: brightness,
        surface: dark ? const Color(0xFF151824) : const Color(0xFFF7F8FC),
        primary: dark ? const Color(0xFFAAA7FF) : AuroraColors.indigo,
        secondary: dark ? const Color(0xFF65E5DD) : AuroraColors.cyan);
    return ThemeData(useMaterial3: true, brightness: brightness, colorScheme: scheme,
      scaffoldBackgroundColor: dark ? const Color(0xFF0D101A) : const Color(0xFFF4F6FB),
      fontFamily: 'sans-serif', visualDensity: VisualDensity.standard,
      appBarTheme: AppBarTheme(backgroundColor: Colors.transparent, elevation: 0, centerTitle: false,
          foregroundColor: dark ? Colors.white : AuroraColors.ink),
      inputDecorationTheme: InputDecorationTheme(filled: true,
          fillColor: dark ? const Color(0xFF1B2030) : Colors.white,
          border: OutlineInputBorder(borderRadius: BorderRadius.circular(18), borderSide: BorderSide.none),
          enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(18),
              borderSide: BorderSide(color: dark ? Colors.white12 : const Color(0xFFE2E5F0))),
          focusedBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(18),
              borderSide: const BorderSide(color: AuroraColors.indigo, width: 1.5)),
          contentPadding: const EdgeInsets.symmetric(horizontal: 18, vertical: 17)),
      filledButtonTheme: FilledButtonThemeData(style: FilledButton.styleFrom(minimumSize: const Size(64, 52),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(17)),
          textStyle: const TextStyle(fontWeight: FontWeight.w700, letterSpacing: .2))),
      outlinedButtonTheme: OutlinedButtonThemeData(style: OutlinedButton.styleFrom(minimumSize: const Size(64, 50),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(17)))),
      navigationBarTheme: const NavigationBarThemeData(height: 68, backgroundColor: Colors.transparent,
          elevation: 0, labelBehavior: NavigationDestinationLabelBehavior.alwaysShow),
      dividerTheme: DividerThemeData(color: dark ? Colors.white10 : const Color(0xFFE6E8F0)),
    );
  }
}
