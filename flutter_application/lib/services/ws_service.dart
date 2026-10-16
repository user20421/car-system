import 'dart:async';
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';

/// WebSocket 实时通道：自动重连，广播 JSON 消息流
class WsService {
  static const String wsUrl = 'ws://localhost:8080/ws';

  final _controller = StreamController<Map<String, dynamic>>.broadcast();
  WebSocketChannel? _channel;
  Timer? _reconnectTimer;
  bool _disposed = false;

  Stream<Map<String, dynamic>> get messages => _controller.stream;
  final _connected = StreamController<bool>.broadcast();
  Stream<bool> get connectionState => _connected.stream;

  void connect() {
    if (_disposed) return;
    try {
      _channel = WebSocketChannel.connect(Uri.parse(wsUrl));
      _connected.add(true);
      _channel!.stream.listen(
        (data) {
          try {
            _controller.add(jsonDecode(data as String) as Map<String, dynamic>);
          } catch (_) {/* 忽略非 JSON */}
        },
        onDone: _scheduleReconnect,
        onError: (_) => _scheduleReconnect(),
      );
    } catch (_) {
      _scheduleReconnect();
    }
  }

  void _scheduleReconnect() {
    _connected.add(false);
    if (_disposed) return;
    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(const Duration(seconds: 3), connect);
  }

  void dispose() {
    _disposed = true;
    _reconnectTimer?.cancel();
    _channel?.sink.close();
    _controller.close();
    _connected.close();
  }
}
