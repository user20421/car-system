import 'package:flutter/foundation.dart';
import '../models/vehicle.dart';
import '../services/repository.dart';
import 'load_state.dart';

class GarageController extends ChangeNotifier {
  GarageController(this.repository);
  final CarRepository repository;
  LoadState<List<Vehicle>> state = const LoadState.loading();
  String? selectedVin;
  List<Vehicle> get vehicles => state.data ?? const [];
  Vehicle? get selected {
    for (final vehicle in vehicles) {
      if (vehicle.vin == selectedVin) return vehicle;
    }
    return vehicles.isEmpty ? null : vehicles.first;
  }

  Future<void> load() async {
    state = const LoadState.loading();
    notifyListeners();
    try {
      final items = await repository.listVehicles();
      state = items.isEmpty ? const LoadState.empty(<Vehicle>[]) : LoadState.ready(items);
      if (items.isNotEmpty && !items.any((vehicle) => vehicle.vin == selectedVin)) selectedVin = items.first.vin;
    } on Object catch (error) {
      state = LoadState.failure(error);
    }
    notifyListeners();
  }

  void select(String vin) {
    if (!vehicles.any((vehicle) => vehicle.vin == vin)) return;
    selectedVin = vin;
    notifyListeners();
  }
}
