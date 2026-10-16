import 'dart:async';
import 'package:flutter/material.dart';
import 'package:fl_chart/fl_chart.dart';
import '../services/api.dart';
import '../services/ws_service.dart';
import 'login_page.dart';

class HomePage extends StatefulWidget {
  const HomePage({super.key});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  final _ws = WsService();
  StreamSubscription? _sub;

  String? _vin;
  bool _locked = true;
  bool _acOn = false;
  bool _wsOnline = false;
  Map<String, dynamic> _latest = {
    'speed': 0.0, 'rpm': 0, 'battery': 0.0, 'lat': 0.0, 'lng': 0.0, 'fault_code': ''
  };
  List<dynamic> _commands = [];
  String? _feedback;
  Timer? _feedbackTimer;

  // 历史曲线数据（speed / battery）
  final List<FlSpot> _speedSpots = [];
  final List<FlSpot> _batterySpots = [];
  double _xCounter = 0;

  static const _cmdLabels = {
    'lock': '🔒 锁车',
    'unlock': '🔓 解锁',
    'ac_on': '❄️ 开空调',
    'ac_off': '💨 关空调',
    'honk': '📢 鸣笛',
  };
  static const _statusLabels = {
    'pending': '待下发',
    'sent': '已下发',
    'executed': '已执行',
    'failed': '失败',
  };

  @override
  void initState() {
    super.initState();
    _load();
    _ws.connect();
    _sub = _ws.messages.listen(_onMessage);
    _ws.connectionState.listen((on) {
      if (mounted) setState(() => _wsOnline = on);
    });
  }

  Future<void> _load() async {
    try {
      final vehicles = await Api.myVehicles();
      if (vehicles.isNotEmpty && mounted) {
        final v = vehicles.first;
        setState(() {
          _vin = v['vin'];
          _locked = v['locked'] ?? true;
          _acOn = v['ac_on'] ?? false;
        });
        final latest = await Api.latestTelemetry(_vin!);
        if (latest != null && mounted) setState(() => _latest = latest);
        await _loadHistory();
        await _loadCommands();
      }
    } catch (_) {/* 后端未就绪时静默 */}
  }

  /// 拉取历史曲线
  Future<void> _loadHistory() async {
    if (_vin == null) return;
    final history = await Api.telemetryHistory(_vin!, limit: 60);
    if (!mounted) return;
    setState(() {
      _speedSpots.clear();
      _batterySpots.clear();
      _xCounter = 0;
      for (final t in history) {
        _speedSpots.add(FlSpot(_xCounter, (t['speed'] as num).toDouble()));
        _batterySpots.add(FlSpot(_xCounter, (t['battery'] as num).toDouble()));
        _xCounter++;
      }
    });
  }

  Future<void> _loadCommands() async {
    if (_vin == null) return;
    final cmds = await Api.commands(_vin!);
    if (mounted) setState(() => _commands = cmds);
  }

  void _onMessage(Map<String, dynamic> msg) {
    if (msg['vin'] != _vin) return;
    if (msg['type'] == 'ack') {
      _showFeedback('车机已执行「${_cmdLabels[msg['command']]}」✅');
      if (msg['command'] == 'lock') setState(() => _locked = true);
      if (msg['command'] == 'unlock') setState(() => _locked = false);
      if (msg['command'] == 'ac_on') setState(() => _acOn = true);
      if (msg['command'] == 'ac_off') setState(() => _acOn = false);
      _loadCommands();
      return;
    }
    if (msg['type'] == 'command') return;
    if (msg.containsKey('speed')) {
      setState(() {
        _latest = msg;
        // 追加曲线点，保持最近 120 个点
        _speedSpots.add(FlSpot(_xCounter, (msg['speed'] as num).toDouble()));
        _batterySpots.add(FlSpot(_xCounter, (msg['battery'] as num).toDouble()));
        _xCounter++;
        while (_speedSpots.length > 120) {
          _speedSpots.removeAt(0);
          _batterySpots.removeAt(0);
        }
      });
    }
  }

  /// 按钮状态感知：与当前状态相同的操作不允许重复执行
  bool _cmdEnabled(String cmd) {
    switch (cmd) {
      case 'lock': return !_locked;
      case 'unlock': return _locked;
      case 'ac_on': return !_acOn;
      case 'ac_off': return _acOn;
      default: return true; // honk 永远可用
    }
  }

  Future<void> _sendCommand(String cmd) async {
    if (_vin == null || !_cmdEnabled(cmd)) return;
    try {
      await Api.sendCommand(_vin!, cmd);
      _showFeedback('指令「${_cmdLabels[cmd]}」已下发，等待车机执行...');
      await _loadCommands();
    } catch (_) {
      _showFeedback('下发失败，请检查网络');
    }
  }

  void _showFeedback(String text) {
    setState(() => _feedback = text);
    _feedbackTimer?.cancel();
    _feedbackTimer = Timer(const Duration(seconds: 4), () {
      if (mounted) setState(() => _feedback = null);
    });
  }

  Future<void> _logout() async {
    await Api.logout();
    if (!mounted) return;
    Navigator.of(context).pushReplacement(
      MaterialPageRoute(builder: (_) => const LoginPage()),
    );
  }

  @override
  void dispose() {
    _sub?.cancel();
    _ws.dispose();
    _feedbackTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final battery = (_latest['battery'] as num?)?.toDouble() ?? 0;
    final faultCode = (_latest['fault_code'] as String?) ?? '';
    return Scaffold(
      appBar: AppBar(
        title: const Text('🚗 我的爱车'),
        actions: [
          Center(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: Text(
                _wsOnline ? '● 实时连接' : '● 已断开',
                style: TextStyle(color: _wsOnline ? Colors.greenAccent : Colors.redAccent),
              ),
            ),
          ),
          IconButton(icon: const Icon(Icons.logout), onPressed: _logout),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            if (_vin == null)
              const Center(
                child: Padding(
                  padding: EdgeInsets.all(40),
                  child: Text('名下暂无车辆', style: TextStyle(color: Colors.grey)),
                ),
              )
            else ...[
              // 故障告警横幅
              if (faultCode.isNotEmpty)
                Container(
                  width: double.infinity,
                  margin: const EdgeInsets.only(bottom: 12),
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.red.withOpacity(0.2),
                    border: Border.all(color: Colors.redAccent),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Text('⚠️ 车辆故障告警：$faultCode，请尽快检修',
                      style: const TextStyle(color: Colors.redAccent, fontWeight: FontWeight.bold)),
                ),

              // 车况卡片
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(18),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(_vin!,
                          style: const TextStyle(fontFamily: 'monospace', fontSize: 13, color: Colors.grey)),
                      const SizedBox(height: 14),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceAround,
                        children: [
                          _stat('车速', '${(_latest['speed'] as num?)?.toStringAsFixed(1)} km/h'),
                          _stat('转速', '${_latest['rpm']} rpm'),
                          _stat('位置',
                              '${(_latest['lat'] as num?)?.toStringAsFixed(4)}, ${(_latest['lng'] as num?)?.toStringAsFixed(4)}'),
                        ],
                      ),
                      const SizedBox(height: 16),
                      Row(
                        children: [
                          const Icon(Icons.battery_std, size: 20),
                          const SizedBox(width: 8),
                          Expanded(
                            child: LinearProgressIndicator(
                              value: battery / 100,
                              minHeight: 10,
                              borderRadius: BorderRadius.circular(5),
                              color: battery > 50
                                  ? Colors.green
                                  : battery > 20
                                      ? Colors.orange
                                      : Colors.red,
                            ),
                          ),
                          const SizedBox(width: 10),
                          Text('${battery.toStringAsFixed(1)}%'),
                        ],
                      ),
                      const SizedBox(height: 12),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceAround,
                        children: [
                          _stateChip(_locked ? '🔒 已上锁' : '🔓 已解锁',
                              _locked ? Colors.green : Colors.orange),
                          _stateChip(_acOn ? '❄️ 空调开' : '💨 空调关',
                              _acOn ? Colors.lightBlue : Colors.grey),
                        ],
                      ),
                    ],
                  ),
                ),
              ),

              // 车况曲线
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(18),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('📈 车况趋势（实时）',
                          style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
                      const SizedBox(height: 12),
                      SizedBox(height: 160, child: _buildChart()),
                      const SizedBox(height: 6),
                      const Row(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          _LegendDot(color: Color(0xFF4FC3F7), label: '车速 km/h'),
                          SizedBox(width: 20),
                          _LegendDot(color: Color(0xFF3FB950), label: '电量 %'),
                        ],
                      ),
                    ],
                  ),
                ),
              ),

              // 远程控制（状态感知按钮）
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(18),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('🎮 远程控制',
                          style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
                      const SizedBox(height: 12),
                      Wrap(
                        spacing: 10,
                        runSpacing: 10,
                        children: [
                          for (final e in _cmdLabels.entries)
                            OutlinedButton(
                              onPressed: _cmdEnabled(e.key) ? () => _sendCommand(e.key) : null,
                              child: Text(e.value),
                            ),
                        ],
                      ),
                      if (_feedback != null) ...[
                        const SizedBox(height: 12),
                        Container(
                          width: double.infinity,
                          padding: const EdgeInsets.all(10),
                          decoration: BoxDecoration(
                            color: Colors.blue.withOpacity(0.15),
                            borderRadius: BorderRadius.circular(8),
                          ),
                          child: Text(_feedback!, style: const TextStyle(color: Colors.lightBlue)),
                        ),
                      ],
                    ],
                  ),
                ),
              ),

              // 指令历史
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(18),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('📜 指令历史',
                          style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
                      const SizedBox(height: 10),
                      if (_commands.isEmpty)
                        const Text('暂无指令记录', style: TextStyle(color: Colors.grey)),
                      for (final c in _commands)
                        ListTile(
                          dense: true,
                          contentPadding: EdgeInsets.zero,
                          title: Text(_cmdLabels[c['command']] ?? c['command']),
                          subtitle: Text(c['created_at'].toString().substring(0, 19)),
                          trailing: _stateChip(
                            _statusLabels[c['status']] ?? c['status'],
                            c['status'] == 'executed'
                                ? Colors.green
                                : c['status'] == 'failed'
                                    ? Colors.red
                                    : Colors.grey,
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildChart() {
    if (_speedSpots.length < 2) {
      return const Center(child: Text('数据收集中...', style: TextStyle(color: Colors.grey)));
    }
    final minX = _speedSpots.first.x;
    final maxX = _speedSpots.last.x;
    return LineChart(
      LineChartData(
        minX: minX,
        maxX: maxX,
        minY: 0,
        gridData: FlGridData(
          show: true,
          drawVerticalLine: false,
          getDrawingHorizontalLine: (_) => const FlLine(color: Color(0xFF21262D), strokeWidth: 1),
        ),
        borderData: FlBorderData(show: false),
        titlesData: const FlTitlesData(show: false),
        lineTouchData: const LineTouchData(enabled: false),
        lineBarsData: [
          _lineBar(_speedSpots, const Color(0xFF4FC3F7)),
          _lineBar(_batterySpots, const Color(0xFF3FB950)),
        ],
      ),
      duration: Duration.zero,
    );
  }

  LineChartBarData _lineBar(List<FlSpot> spots, Color color) {
    return LineChartBarData(
      spots: spots,
      isCurved: true,
      color: color,
      barWidth: 2,
      dotData: const FlDotData(show: false),
      belowBarData: BarAreaData(show: true, color: color.withOpacity(0.08)),
    );
  }

  Widget _stat(String label, String value) {
    return Column(
      children: [
        Text(value, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
        const SizedBox(height: 4),
        Text(label, style: const TextStyle(fontSize: 12, color: Colors.grey)),
      ],
    );
  }

  Widget _stateChip(String text, Color color) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.15),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(text, style: TextStyle(color: color, fontSize: 12)),
    );
  }
}

class _LegendDot extends StatelessWidget {
  final Color color;
  final String label;
  const _LegendDot({required this.color, required this.label});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(width: 10, height: 10, decoration: BoxDecoration(color: color, shape: BoxShape.circle)),
        const SizedBox(width: 5),
        Text(label, style: const TextStyle(fontSize: 12, color: Colors.grey)),
      ],
    );
  }
}
