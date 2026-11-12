import 'package:flutter/material.dart';

class SettingsController extends ChangeNotifier {
  ThemeMode themeMode = ThemeMode.system;
  bool hapticFeedback = true;

  void setThemeMode(ThemeMode value) {
    themeMode = value;
    notifyListeners();
  }
  void setHaptics(bool value) {
    hapticFeedback = value;
    notifyListeners();
  }
}
