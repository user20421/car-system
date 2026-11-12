import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/telemetry.dart';
import '../state/auth_controller.dart';
import '../state/garage_controller.dart';
import '../state/history_controller.dart';
import '../widgets/common.dart';
import '../widgets/telemetry_visuals.dart';

class JourneysScreen extends StatefulWidget {
  const JourneysScreen({super.key});
  @override
  State<JourneysScreen> createState() => _JourneysScreenState();
}
class _JourneysScreenState extends State<JourneysScreen> {
  String? _loadedVin;
  bool _showRange = false;
  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final vin = context.watch<GarageController>().selected?.vin;
    if (vin != null && vin != _loadedVin) {
      _loadedVin = vin;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) context.read<HistoryController>().load(vin);
      });
    }
  }
  @override
  Widget build(BuildContext context) {
    final history = context.watch<HistoryController>();
    final vin = context.read<GarageController>().selected?.vin;
    if (vin == null) return const Center(child: Text('暂无车辆'));
    return RefreshIndicator(onRefresh: () => history.load(vin), child: ListView(padding: const EdgeInsets.fromLTRB(20, 12, 20, 32), children: [
      Text('每一程，都有迹可循', style: Theme.of(context).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w700)),
      const SizedBox(height: 6), Text('从历史遥测了解车辆使用与续航变化', style: Theme.of(context).textTheme.bodyMedium?.copyWith(
        color: Theme.of(context).colorScheme.onSurfaceVariant)),
      const SizedBox(height: 22),
      Row(children: [Expanded(child: SegmentedButton<int>(segments: const [ButtonSegment(value: 24, label: Text('近 24 小时')),
        ButtonSegment(value: 168, label: Text('近 7 天'))], selected: {history.hours},
        onSelectionChanged: (selection) => history.load(vin, nextHours: selection.first))),
        const SizedBox(width: 10), IconButton.filledTonal(tooltip: '切换指标', onPressed: () => setState(() => _showRange = !_showRange),
          icon: Icon(_showRange ? Icons.battery_full : Icons.route_outlined))]),
      const SizedBox(height: 18),
      SurfaceCard(child: StateView<List<Telemetry>>(state: history.state, onRetry: () => history.load(vin),
        onLogin: () => context.read<AuthController>().expireSession(), emptyTitle: '所选时段没有历史记录', builder: (items) => Column(
          crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(_showRange ? '预计续航 · km' : '动力电池 · %', style: const TextStyle(fontWeight: FontWeight.w700)),
            const SizedBox(height: 16), TrendChart(items: items, range: _showRange, height: 260), const SizedBox(height: 12),
            Text('${items.first.occurredAt.toLocal().month}/${items.first.occurredAt.toLocal().day} — '
              '${items.last.occurredAt.toLocal().month}/${items.last.occurredAt.toLocal().day}',
              style: Theme.of(context).textTheme.bodySmall),
          ]))),
      const SizedBox(height: 16),
      StateView<List<Telemetry>>(state: history.state, onRetry: () => history.load(vin), builder: (items) => Row(children: [
        Expanded(child: _Stat(label: '记录点数', value: '${items.length}', icon: Icons.sensors_rounded)),
        const SizedBox(width: 12), Expanded(child: _Stat(label: '续航变化', value: '${(items.last.rangeKm - items.first.rangeKm).abs().round()} km', icon: Icons.route)),
      ])),
      const SizedBox(height: 16),
      Container(padding: const EdgeInsets.all(16), decoration: BoxDecoration(color: Theme.of(context).colorScheme.primaryContainer.withValues(alpha: .45),
        borderRadius: BorderRadius.circular(18)), child: Text('说明：服务端当前提供历史遥测而非独立行程事件，因此本页不虚构起终点或里程。采样间隔较大时，趋势仅用于观察变化。',
        style: Theme.of(context).textTheme.bodySmall)),
    ]));
  }
}
class _Stat extends StatelessWidget {
  const _Stat({required this.label, required this.value, required this.icon});
  final String label, value;
  final IconData icon;
  @override
  Widget build(BuildContext context) => SurfaceCard(child: Row(children: [Icon(icon, color: Theme.of(context).colorScheme.primary),
    const SizedBox(width: 12), Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Text(value, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700)), Text(label, style: Theme.of(context).textTheme.bodySmall)]))]));
}
