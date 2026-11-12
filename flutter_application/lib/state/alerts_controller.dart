import 'package:flutter/foundation.dart';
import '../models/alert.dart';
import '../services/repository.dart';
import 'load_state.dart';

class AlertsController extends ChangeNotifier {
  AlertsController(this.repository);
  final CarRepository repository;
  LoadState<List<VehicleAlert>> state = const LoadState.loading();
  final _acknowledging = <String>{};
  bool isAcknowledging(String id) => _acknowledging.contains(id);

  Future<void> load({String? vin}) async {
    state = const LoadState.loading();
    notifyListeners();
    try {
      final alerts = await repository.listAlerts(vin: vin);
      alerts.sort((a, b) => b.occurredAt.compareTo(a.occurredAt));
      state = alerts.isEmpty ? const LoadState.empty(<VehicleAlert>[]) : LoadState.ready(alerts);
    } on Object catch (error) {
      state = LoadState.failure(error);
    }
    notifyListeners();
  }

  Future<bool> acknowledge(VehicleAlert alert) async {
    if (_acknowledging.contains(alert.id)) return false;
    _acknowledging.add(alert.id);
    notifyListeners();
    try {
      final updated = await repository.acknowledgeAlert(alert.id, '车主已在 AURORA App 中确认');
      final items = List<VehicleAlert>.from(state.data ?? const []);
      final index = items.indexWhere((item) => item.id == alert.id);
      if (index >= 0) items[index] = updated;
      state = LoadState.ready(items);
      return true;
    } finally {
      _acknowledging.remove(alert.id);
      notifyListeners();
    }
  }
}
