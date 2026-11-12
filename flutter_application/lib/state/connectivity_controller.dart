import 'dart:async';
import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:flutter/foundation.dart';

class ConnectivityController extends ChangeNotifier {
  ConnectivityController({bool listenToPlatform = true}) {
    if (listenToPlatform) {
      _subscription = Connectivity().onConnectivityChanged.listen(_update);
      Connectivity().checkConnectivity().then(_update);
    }
  }
  StreamSubscription<List<ConnectivityResult>>? _subscription;
  bool online = true;
  void _update(List<ConnectivityResult> results) {
    final next = results.any((result) => result != ConnectivityResult.none);
    if (online != next) {
      online = next;
      notifyListeners();
    }
  }
  @override
  void dispose() {
    _subscription?.cancel();
    super.dispose();
  }
}
