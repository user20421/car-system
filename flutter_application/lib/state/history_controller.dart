import 'package:flutter/foundation.dart';
import '../models/telemetry.dart';
import '../services/repository.dart';
import 'load_state.dart';

class HistoryController extends ChangeNotifier {
  HistoryController(this.repository);
  final CarRepository repository;
  LoadState<List<Telemetry>> state = const LoadState.loading();
  String? vin;
  int hours = 24;

  Future<void> load(String nextVin, {int? nextHours}) async {
    vin = nextVin;
    hours = nextHours ?? hours;
    state = const LoadState.loading();
    notifyListeners();
    try {
      final to = DateTime.now().toUtc();
      final items = await repository.telemetryHistory(nextVin, to.subtract(Duration(hours: hours)), to);
      if (vin != nextVin) return;
      state = items.isEmpty ? const LoadState.empty(<Telemetry>[]) : LoadState.ready(items);
    } on Object catch (error) {
      if (vin == nextVin) state = LoadState.failure(error);
    }
    notifyListeners();
  }
}
