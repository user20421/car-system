import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/telemetry.dart';
import '../services/realtime_service.dart';
import '../state/auth_controller.dart';
import '../state/garage_controller.dart';
import '../state/vehicle_controller.dart';
import '../widgets/common.dart';
import '../widgets/vehicle_artwork.dart';
import 'alerts_screen.dart';
import 'charging_screen.dart';
import 'controls_screen.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key, required this.onNavigate});
  final ValueChanged<int> onNavigate;
  @override
  Widget build(BuildContext context) {
    final vehicleState = context.watch<VehicleController>();
    final garage = context.watch<GarageController>();
    final vehicle = garage.selected;
    if (vehicle == null) {
      return StateView<List>(state: garage.state, onRetry: garage.load,
        onLogin: () => context.read<AuthController>().logout(), builder: (_) => const SizedBox());
    }
    return RefreshIndicator(onRefresh: vehicleState.refreshSnapshot, child: CustomScrollView(slivers: [
      SliverPadding(padding: const EdgeInsets.fromLTRB(20, 8, 20, 32), sliver: SliverToBoxAdapter(child: Column(
        crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(_greeting(), style: Theme.of(context).textTheme.bodyMedium?.copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant)),
            const SizedBox(height: 3), Text(vehicle.displayName, style: Theme.of(context).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w700)),
          ])), TextButton.icon(onPressed: () => _showVehiclePicker(context), icon: const Icon(Icons.expand_more), label: const Text('切换'))]),
          const SizedBox(height: 14),
          StateView<Telemetry>(state: vehicleState.state, onRetry: vehicleState.refreshSnapshot,
            onLogin: () => context.read<AuthController>().logout(), builder: (telemetry) => Column(children: [
              if (vehicleState.state.status.name == 'offline') const OfflineBanner(),
              _Hero(telemetry: telemetry, name: vehicle.displayName, model: vehicle.model, connection: vehicleState.connection),
              const SizedBox(height: 20),
              LayoutBuilder(builder: (context, constraints) {
                final wide = constraints.maxWidth > 680;
                final metrics = [_Metric(icon: Icons.route_outlined, label: '预计续航', value: '${telemetry.rangeKm.round()}', unit: 'km'),
                  _Metric(icon: Icons.battery_charging_full_rounded, label: '当前电量', value: '${telemetry.batteryPercent.round()}', unit: '%'),
                  _Metric(icon: Icons.ac_unit_rounded, label: '车内温度', value: telemetry.cabinTempC?.toStringAsFixed(1) ?? '--', unit: '°C')];
                return GridView.count(crossAxisCount: wide ? 3 : 3, shrinkWrap: true, physics: const NeverScrollableScrollPhysics(),
                  childAspectRatio: wide ? 2.2 : 1.05, crossAxisSpacing: 10, mainAxisSpacing: 10,
                  children: metrics);
              }),
              const SizedBox(height: 24),
              SectionTitle('快捷控车', action: TextButton(onPressed: vehicle.canControl ? () => _openControls(context) : null,
                child: Text(vehicle.canControl ? '全部控制' : '仅查看权限'))),
              const SizedBox(height: 10),
              Row(children: [
                Expanded(child: _QuickAction(icon: telemetry.locked ? Icons.lock_rounded : Icons.lock_open_rounded,
                  label: telemetry.locked ? '车辆已锁' : '车辆已解锁', onTap: vehicle.canControl ? () => _openControls(context) : null)),
                const SizedBox(width: 12),
                Expanded(child: _QuickAction(icon: Icons.air_rounded, label: telemetry.hvac.mode == 'off' ? '空调已关闭' : '${telemetry.hvac.targetTempC.toStringAsFixed(0)}°C 空调运行中',
                  onTap: vehicle.canControl ? () => _openControls(context) : null)),
                const SizedBox(width: 12),
                Expanded(child: _QuickAction(icon: Icons.wb_cloudy_outlined, label: telemetry.defrost ? '除霜运行中' : '前后除霜',
                  onTap: vehicle.canControl ? () => _openControls(context) : null)),
              ]),
              const SizedBox(height: 24),
              const SectionTitle('出行与服务'),
              const SizedBox(height: 10),
              SurfaceCard(padding: EdgeInsets.zero, child: Column(children: [
                _LinkTile(icon: Icons.ev_station_outlined, title: '续航与充电', subtitle: '电量、续航与能耗趋势',
                  onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const ChargingScreen()))),
                const Divider(height: 1),
                _LinkTile(icon: Icons.route_outlined, title: '行程与趋势', subtitle: '回顾历史遥测与续航变化', onTap: () => onNavigate(2)),
                const Divider(height: 1),
                _LinkTile(icon: Icons.notifications_active_outlined, title: '车辆告警', subtitle: '查看并确认车辆提醒',
                  onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const AlertsScreen()))),
                const Divider(height: 1),
                _LinkTile(icon: Icons.handyman_outlined, title: '车辆服务', subtitle: '保养、道路救援与专属支持', onTap: () => onNavigate(3)),
              ])),
              const SizedBox(height: 14),
              Center(child: Text('遥测时间 ${_formatTime(telemetry.occurredAt)} · 非实时路况数据',
                style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant))),
            ])),
        ]))),
    ]));
  }
  static String _greeting() => switch (DateTime.now().hour) { < 11 => '早上好，准备出发', < 14 => '中午好，一路舒心', < 18 => '下午好，自在同行', _ => '晚上好，极光相伴' };
  static String _formatTime(DateTime value) => '${value.toLocal().hour.toString().padLeft(2, '0')}:${value.toLocal().minute.toString().padLeft(2, '0')}';
  static void _openControls(BuildContext context) => Navigator.push(context, MaterialPageRoute(builder: (_) => const ControlsScreen()));
  static void _showVehiclePicker(BuildContext context) {
    final garage = context.read<GarageController>();
    showModalBottomSheet<void>(context: context, showDragHandle: true, builder: (_) => SafeArea(child: Column(mainAxisSize: MainAxisSize.min,
      children: garage.vehicles.map((vehicle) => ListTile(title: Text(vehicle.displayName), subtitle: Text(vehicle.model),
        trailing: garage.selectedVin == vehicle.vin ? const Icon(Icons.check) : null,
        onTap: () { garage.select(vehicle.vin); Navigator.pop(context); })).toList())));
  }
}

class _Hero extends StatelessWidget {
  const _Hero({required this.telemetry, required this.name, required this.model, required this.connection});
  final Telemetry telemetry;
  final String name;
  final String model;
  final RealtimeStatus connection;
  @override
  Widget build(BuildContext context) => Container(height: 292, decoration: BoxDecoration(borderRadius: BorderRadius.circular(30),
    gradient: const LinearGradient(colors: [Color(0xFF202544), Color(0xFF343166)], begin: Alignment.topLeft, end: Alignment.bottomRight),
    boxShadow: [BoxShadow(color: const Color(0xFF5856D6).withValues(alpha: .16), blurRadius: 32, offset: const Offset(0, 16))]),
    child: Stack(children: [
      Positioned.fill(child: VehicleArtwork(height: 270)),
      Positioned(left: 22, top: 20, child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(model, style: const TextStyle(color: Colors.white70, fontSize: 11, letterSpacing: 2)), const SizedBox(height: 6),
        Row(children: [Container(width: 7, height: 7, decoration: BoxDecoration(shape: BoxShape.circle,
          color: connection == RealtimeStatus.connected ? const Color(0xFF65F6EE) : Colors.orange)), const SizedBox(width: 7),
          Text(connection == RealtimeStatus.connected ? '实时连接' : '正在同步', style: const TextStyle(color: Colors.white70, fontSize: 11))]),
      ])),
      Positioned(right: 20, top: 20, child: Icon(telemetry.locked ? Icons.lock_rounded : Icons.lock_open_rounded, color: Colors.white70, size: 20)),
      const Positioned(left: 22, bottom: 18, child: Text('AURORA INTELLIGENT COCKPIT', style: TextStyle(color: Colors.white38, fontSize: 8, letterSpacing: 2))),
    ]));
}
class _Metric extends StatelessWidget {
  const _Metric({required this.icon, required this.label, required this.value, required this.unit});
  final IconData icon;
  final String label, value, unit;
  @override
  Widget build(BuildContext context) => SurfaceCard(padding: const EdgeInsets.all(13), child: Column(crossAxisAlignment: CrossAxisAlignment.start,
    mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [Icon(icon, size: 19, color: Theme.of(context).colorScheme.primary),
      Flexible(child: FittedBox(fit: BoxFit.scaleDown, alignment: Alignment.centerLeft,
        child: Text.rich(TextSpan(children: [TextSpan(text: value), TextSpan(text: ' $unit', style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w400))]),
          style: Theme.of(context).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700)))),
      Text(label, maxLines: 1, overflow: TextOverflow.ellipsis, style: Theme.of(context).textTheme.bodySmall),
    ]));
}
class _QuickAction extends StatelessWidget {
  const _QuickAction({required this.icon, required this.label, this.onTap});
  final IconData icon;
  final String label;
  final VoidCallback? onTap;
  @override
  Widget build(BuildContext context) => SurfaceCard(padding: const EdgeInsets.symmetric(vertical: 18, horizontal: 8), onTap: onTap,
    child: Column(children: [Icon(icon, color: onTap == null ? Theme.of(context).disabledColor : Theme.of(context).colorScheme.primary),
      const SizedBox(height: 9), Text(label, maxLines: 2, textAlign: TextAlign.center, style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w600))]));
}
class _LinkTile extends StatelessWidget {
  const _LinkTile({required this.icon, required this.title, required this.subtitle, required this.onTap});
  final IconData icon;
  final String title, subtitle;
  final VoidCallback onTap;
  @override
  Widget build(BuildContext context) => ListTile(contentPadding: const EdgeInsets.symmetric(horizontal: 18, vertical: 5), onTap: onTap,
    leading: Icon(icon), title: Text(title, style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14)),
    subtitle: Text(subtitle, style: const TextStyle(fontSize: 11)), trailing: const Icon(Icons.chevron_right));
}
