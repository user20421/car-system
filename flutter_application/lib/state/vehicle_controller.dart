import 'dart:async';
import 'package:flutter/foundation.dart';
import '../models/command.dart';
import '../models/telemetry.dart';
import '../services/realtime_service.dart';
import '../services/repository.dart';
import 'load_state.dart';

class VehicleController extends ChangeNotifier {
  VehicleController(this.repository) : realtime = RealtimeService(repository) {
    realtime.onSnapshotNeeded = refreshSnapshot;
    _telemetrySub = realtime.telemetry.listen((value) {
      if (value.vin == vin) {
        state = LoadState.ready(value);
        notifyListeners();
      }
    });
    _commandSub = realtime.commands.listen((command) => onCommand?.call(command));
    _statusSub = realtime.status.listen((value) {
      connection = value;
      notifyListeners();
    });
  }
  final CarRepository repository;
  final RealtimeService realtime;
  LoadState<Telemetry> state = const LoadState.loading();
  RealtimeStatus connection = RealtimeStatus.disconnected;
  String? vin;
  Timer? _snapshotTimer;
  StreamSubscription? _telemetrySub;
  StreamSubscription? _commandSub;
  StreamSubscription? _statusSub;
  void Function(VehicleCommand)? onCommand;

  Future<void> select(String nextVin) async {
    if (vin == nextVin && state.status != LoadStatus.loading) return;
    vin = nextVin;
    state = const LoadState.loading();
    notifyListeners();
    if (repository.isDemo) {
      connection = RealtimeStatus.connected;
    } else {
      await realtime.connect(nextVin);
      _snapshotTimer?.cancel();
      _snapshotTimer = Timer.periodic(const Duration(seconds: 30), (_) => refreshSnapshot());
    }
    await refreshSnapshot();
  }

  Future<void> refreshSnapshot() async {
    final currentVin = vin;
    if (currentVin == null) return;
    try {
      final value = await repository.latestTelemetry(currentVin);
      if (vin == currentVin) {
        state = LoadState.ready(value);
        notifyListeners();
      }
    } on Object catch (error) {
      if (vin == currentVin) {
        final failure = LoadState<Telemetry>.failure(error);
        state = failure.status == LoadStatus.offline && state.data != null ? LoadState.offline(state.data) : failure;
        notifyListeners();
      }
    }
  }

  void stop() {
    _snapshotTimer?.cancel();
    _snapshotTimer = null;
  }

  @override
  void dispose() {
    stop();
    _telemetrySub?.cancel();
    _commandSub?.cancel();
    _statusSub?.cancel();
    realtime.dispose();
    super.dispose();
  }
}
