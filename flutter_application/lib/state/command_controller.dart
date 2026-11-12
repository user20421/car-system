import 'dart:async';
import 'package:flutter/foundation.dart';
import 'package:uuid/uuid.dart';
import '../models/command.dart';
import '../services/repository.dart';

class CommandController extends ChangeNotifier {
  CommandController(this.repository);
  final CarRepository repository;
  final _reducer = const CommandStateMachine();
  final _commands = <String, VehicleCommand>{};
  final _pollTimers = <String, Timer>{};
  bool _sending = false;
  void Function(VehicleCommand)? onTerminal;
  bool get isSending => _sending;
  bool get isBusy => _sending || (latest != null && !latest!.isTerminal);
  VehicleCommand? get latest => _commands.values.isEmpty ? null
      : _commands.values.reduce((a, b) => a.createdAt.isAfter(b.createdAt) ? a : b);

  Future<VehicleCommand> send(String vin, String type, {Map<String, String> params = const {}}) async {
    if (isBusy) throw const RepositoryException(RepositoryErrorKind.conflict, '上一条指令尚未完成，请等待车辆回执');
    _sending = true;
    notifyListeners();
    try {
      final command = await repository.createCommand(vin, type, params, const Uuid().v4(),
          DateTime.now().toUtc().add(const Duration(seconds: 30)));
      _accept(command);
      _poll(command);
      return command;
    } finally {
      _sending = false;
      notifyListeners();
    }
  }

  VehicleCommand? commandById(String id) => _commands[id];
  void acceptEvent(VehicleCommand command) => _accept(command);

  void _accept(VehicleCommand incoming) {
    final current = _commands[incoming.id];
    final wasTerminal = current?.isTerminal ?? false;
    final next = current == null ? incoming : _reducer.reduce(current, incoming);
    _commands[incoming.id] = next;
    if (next.isTerminal) {
      _pollTimers.remove(next.id)?.cancel();
      if (!wasTerminal) onTerminal?.call(next);
    }
    notifyListeners();
  }

  void _poll(VehicleCommand command) {
    _pollTimers[command.id]?.cancel();
    _pollTimers[command.id] = Timer.periodic(const Duration(milliseconds: 800), (timer) async {
      try {
        final next = await repository.getCommand(command.vin, command.id);
        _accept(next);
        if (next.isTerminal) timer.cancel();
      } on RepositoryException catch (error) {
        if (error.kind == RepositoryErrorKind.unauthorized || error.kind == RepositoryErrorKind.notFound) {
          timer.cancel();
        }
      } on Object {
        // Keep polling through transient failures; the visible state stays unchanged.
      }
    });
  }

  @override
  void dispose() {
    for (final timer in _pollTimers.values) { timer.cancel(); }
    super.dispose();
  }
}
