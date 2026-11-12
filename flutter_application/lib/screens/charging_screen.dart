import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/telemetry.dart';
import '../state/auth_controller.dart';
import '../state/garage_controller.dart';
import '../state/history_controller.dart';
import '../state/vehicle_controller.dart';
import '../widgets/common.dart';
import '../widgets/telemetry_visuals.dart';

class ChargingScreen extends StatefulWidget {
  const ChargingScreen({super.key});
  @override
  State<ChargingScreen> createState() => _ChargingScreenState();
}
class _ChargingScreenState extends State<ChargingScreen> {
  String? _loadedVin;
  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final vin = context.read<GarageController>().selected?.vin;
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
    final vehicleState = context.watch<VehicleController>();
    final vehicle = context.watch<GarageController>().selected;
    return Scaffold(appBar: AppBar(title: const Text('续航与充电')), body: SafeArea(child: RefreshIndicator(
      onRefresh: () => Future.wait([vehicleState.refreshSnapshot(), history.load(vehicle!.vin)]),
      child: ListView(padding: const EdgeInsets.fromLTRB(20, 8, 20, 32), children: [
        StateView<Telemetry>(state: vehicleState.state, onRetry: vehicleState.refreshSnapshot,
          onLogin: () => context.read<AuthController>().expireSession(), builder: (telemetry) => Column(children: [
            SurfaceCard(child: Column(children: [BatteryGauge(percent: telemetry.batteryPercent), const SizedBox(height: 8),
              Text('预计可行驶 ${telemetry.rangeKm.round()} km', style: Theme.of(context).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700)),
              const SizedBox(height: 6), Text('基于当前能耗与车况估算，实际续航随环境变化',
                textAlign: TextAlign.center, style: Theme.of(context).textTheme.bodySmall),
              const SizedBox(height: 22),
              Container(padding: const EdgeInsets.all(14), decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.surfaceContainerHighest.withValues(alpha: .55), borderRadius: BorderRadius.circular(16)),
                child: Row(children: [Icon(Icons.info_outline, size: 19, color: Theme.of(context).colorScheme.primary),
                  const SizedBox(width: 10), const Expanded(child: Text('当前协议未提供充电枪与充电功率数据，暂不推断充电状态。',
                    style: TextStyle(fontSize: 12)))])),
            ])),
            const SizedBox(height: 24),
          ])),
        Row(children: [Expanded(child: Text('电量与续航趋势', style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700))),
          SegmentedButton<int>(segments: const [ButtonSegment(value: 24, label: Text('24h')), ButtonSegment(value: 168, label: Text('7d'))],
            selected: {history.hours}, onSelectionChanged: (selection) => history.load(vehicle!.vin, nextHours: selection.first),
            style: const ButtonStyle(visualDensity: VisualDensity.compact))]),
        const SizedBox(height: 14),
        SurfaceCard(child: StateView<List<Telemetry>>(state: history.state,
          onRetry: () => history.load(vehicle!.vin), onLogin: () => context.read<AuthController>().expireSession(),
          emptyTitle: '所选时段没有遥测记录', builder: (items) => Column(children: [
            TrendChart(items: items), const SizedBox(height: 14),
            Row(mainAxisAlignment: MainAxisAlignment.spaceAround, children: [
              _Summary(label: '起始电量', value: '${items.first.batteryPercent.toStringAsFixed(0)}%'),
              _Summary(label: '最新电量', value: '${items.last.batteryPercent.toStringAsFixed(0)}%'),
              _Summary(label: '续航变化', value: '${(items.last.rangeKm - items.first.rangeKm).toStringAsFixed(0)} km'),
            ]),
          ]))),
      ]))));
  }
}
class _Summary extends StatelessWidget {
  const _Summary({required this.label, required this.value});
  final String label, value;
  @override
  Widget build(BuildContext context) => Column(children: [Text(value, style: const TextStyle(fontWeight: FontWeight.w700, fontSize: 16)),
    const SizedBox(height: 4), Text(label, style: Theme.of(context).textTheme.bodySmall)]);
}
