import 'dart:async';
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';
import '../models/command.dart';
import '../models/telemetry.dart';
import 'repository.dart';

enum RealtimeStatus { disconnected, connecting, connected, reconnecting }

class RealtimeService {
  RealtimeService(this.repository);
  final CarRepository repository;
  final _telemetryController = StreamController<Telemetry>.broadcast();
  final _commandController = StreamController<VehicleCommand>.broadcast();
  final _statusController = StreamController<RealtimeStatus>.broadcast();
  WebSocketChannel? _channel;
  StreamSubscription? _socketSubscription;
  StreamSubscription? _demoSubscription;
  Timer? _retryTimer;
  String? _vin;
  int _attempt = 0;
  bool _closed = false;
  void Function()? onSnapshotNeeded;

  Stream<Telemetry> get telemetry => _telemetryController.stream;
  Stream<VehicleCommand> get commands => _commandController.stream;
  Stream<RealtimeStatus> get status => _statusController.stream;

  Future<void> connect(String vin) async {
    if (_vin == vin && _channel != null) return;
    await close();
    _closed = false;
    _vin = vin;
    _attempt = 0;
    _demoSubscription = repository.commandStream.listen(_commandController.add);
    if (repository.isDemo) {
      _statusController.add(RealtimeStatus.connected);
      onSnapshotNeeded?.call();
      return;
    }
    await _open();
  }

  Future<void> _open() async {
    if (_closed || _vin == null) return;
    _statusController.add(_attempt == 0 ? RealtimeStatus.connecting : RealtimeStatus.reconnecting);
    try {
      // A fresh short-lived ticket is requested for every attempt; tickets are never reused.
      final ticket = await repository.createWsTicket();
      if (_closed) return;
      final base = Uri.parse(ticket.websocketUrl);
      final uri = base.replace(queryParameters: {...base.queryParameters, 'ticket': ticket.ticket});
      final channel = WebSocketChannel.connect(uri);
      await channel.ready;
      if (_closed) {
        await channel.sink.close();
        return;
      }
      _channel = channel;
      _attempt = 0;
      _statusController.add(RealtimeStatus.connected);
      // WSS is a hint channel. An HTTPS snapshot reconciles anything missed while disconnected.
      onSnapshotNeeded?.call();
      _socketSubscription = channel.stream.listen(_onMessage,
          onError: (_) => _scheduleReconnect(), onDone: _scheduleReconnect, cancelOnError: true);
    } catch (_) {
      _scheduleReconnect();
    }
  }

  void _onMessage(dynamic raw) {
    try {
      final event = jsonDecode(raw as String) as Map<String, dynamic>;
      final type = (event['type'] ?? event['event']) as String? ?? '';
      final data = event['data'] as Map<String, dynamic>? ?? event;
      if (type.contains('telemetry') && data['vin'] == _vin) {
        _telemetryController.add(Telemetry.fromJson(data));
      } else if (type.contains('command') && data['vin'] == _vin) {
        _commandController.add(VehicleCommand.fromJson(data));
      }
    } on Object {
      // Ignore malformed push frames; the next HTTPS snapshot remains authoritative.
    }
  }

  void _scheduleReconnect() {
    if (_closed || _retryTimer != null) return;
    _channel = null;
    _attempt++;
    _statusController.add(RealtimeStatus.reconnecting);
    final delay = Duration(seconds: (1 << _attempt.clamp(0, 5)).clamp(1, 30));
    _retryTimer = Timer(delay, () {
      _retryTimer = null;
      _open();
    });
  }

  Future<void> close() async {
    _closed = true;
    _retryTimer?.cancel();
    _retryTimer = null;
    await _socketSubscription?.cancel();
    _socketSubscription = null;
    await _demoSubscription?.cancel();
    _demoSubscription = null;
    final channel = _channel;
    _channel = null;
    if (channel != null) await channel.sink.close();
  }

  Future<void> dispose() async {
    await close();
    await _telemetryController.close();
    await _commandController.close();
    await _statusController.close();
  }
}
