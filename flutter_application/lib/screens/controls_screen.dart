import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import '../models/command.dart';
import '../models/telemetry.dart';
import '../services/repository.dart';
import '../state/auth_controller.dart';
import '../state/command_controller.dart';
import '../state/connectivity_controller.dart';
import '../state/garage_controller.dart';
import '../state/load_state.dart';
import '../state/settings_controller.dart';
import '../state/vehicle_controller.dart';
import '../theme/aurora_theme.dart';
import '../widgets/command_progress.dart';
import '../widgets/common.dart';

class ControlsScreen extends StatefulWidget {
  const ControlsScreen({super.key});
  @override
  State<ControlsScreen> createState() => _ControlsScreenState();
}
class _ControlsScreenState extends State<ControlsScreen> {
  double _temperature = 22;
  bool _initialized = false;
  Future<void> _send(String type, {Map<String, String> params = const {}}) async {
    final vehicle = context.read<GarageController>().selected!;
    final confirmed = await showDialog<bool>(context: context, builder: (dialogContext) => AlertDialog(
      icon: Icon(type == 'unlock' ? Icons.lock_open_rounded : Icons.shield_outlined),
      title: Text('确认${commandNames[type] ?? '执行指令'}？'),
      content: Text(type == 'unlock' ? '请确认车辆处于安全环境。解锁指令将发送至 ${vehicle.displayName}，执行结果以车辆回执为准。'
          : '指令将通过加密连接发送至 ${vehicle.displayName}。请勿在车辆行驶中操作，执行结果以车辆回执为准。'),
      actions: [TextButton(onPressed: () => Navigator.pop(dialogContext, false), child: const Text('取消')),
        FilledButton(onPressed: () => Navigator.pop(dialogContext, true), child: const Text('确认发送'))],
    ));
    if (confirmed != true || !mounted) return;
    if (context.read<SettingsController>().hapticFeedback) HapticFeedback.lightImpact();
    try {
      final command = await context.read<CommandController>().send(vehicle.vin, type, params: params);
      if (mounted) showCommandProgress(context, command.id);
    } on RepositoryException catch (error) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(error.message)));
      if (error.kind == RepositoryErrorKind.unauthorized && mounted) context.read<AuthController>().expireSession();
    }
  }
  @override
  Widget build(BuildContext context) {
    final vehicleState = context.watch<VehicleController>();
    final commandController = context.watch<CommandController>();
    final vehicle = context.watch<GarageController>().selected;
    final online = context.watch<ConnectivityController>().online;
    if (vehicle == null) return const Scaffold(body: Center(child: Text('暂无车辆')));
    final offline = vehicleState.state.status == LoadStatus.offline;
    final canControl = vehicle.canControl && online && !offline;
    return Scaffold(appBar: AppBar(title: const Text('车辆控制')),
      body: SafeArea(child: StateView<Telemetry>(state: vehicleState.state, onRetry: vehicleState.refreshSnapshot,
        onLogin: () => context.read<AuthController>().expireSession(), builder: (telemetry) {
          if (!_initialized) { _temperature = telemetry.hvac.targetTempC; _initialized = true; }
          final disabledReason = !vehicle.canControl ? '你拥有仅查看权限，不能发送控车指令' : !online || offline ? '离线时不能发送控车指令' : null;
          return ListView(padding: const EdgeInsets.fromLTRB(20, 8, 20, 30), children: [
            if (disabledReason != null) Container(margin: const EdgeInsets.only(bottom: 16), padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(color: Colors.orange.withValues(alpha: .12), borderRadius: BorderRadius.circular(16)),
              child: Text(disabledReason)),
            SurfaceCard(child: Column(children: [
              Row(children: [Expanded(child: Text('门锁与车辆安全', style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700))),
                Icon(telemetry.locked ? Icons.lock_rounded : Icons.lock_open_rounded, color: Theme.of(context).colorScheme.primary)]),
              const SizedBox(height: 12), Text(telemetry.locked ? '车辆当前已锁定' : '车辆当前处于解锁状态',
                style: TextStyle(color: Theme.of(context).colorScheme.onSurfaceVariant)),
              const SizedBox(height: 20),
              Row(children: [Expanded(child: FilledButton.tonalIcon(key: const Key('control-lock'),
                onPressed: canControl && !commandController.isBusy ? () => _send(telemetry.locked ? 'unlock' : 'lock') : null,
                icon: const Icon(Icons.lock_outline), label: Text(telemetry.locked ? '解锁车辆' : '立即锁车'))),
                const SizedBox(width: 12), Expanded(child: OutlinedButton.icon(onPressed: canControl && !commandController.isBusy
                  ? () => _send('honk_flash') : null, icon: const Icon(Icons.campaign_outlined), label: const Text('鸣笛闪灯')))]),
            ])),
            const SizedBox(height: 16),
            SurfaceCard(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [Expanded(child: Text('智能空调', style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700))),
                Switch(value: telemetry.hvac.mode != 'off', onChanged: canControl && !commandController.isBusy
                  ? (_) => _send(telemetry.hvac.mode == 'off' ? 'hvac_on' : 'hvac_off') : null)]),
              const SizedBox(height: 14),
              Center(child: Text('${_temperature.toStringAsFixed(1)}°', style: Theme.of(context).textTheme.displaySmall?.copyWith(
                fontWeight: FontWeight.w300, color: AuroraColors.indigo))),
              Center(child: Text('目标温度 · 车内 ${telemetry.cabinTempC?.toStringAsFixed(1) ?? '--'}°C',
                style: Theme.of(context).textTheme.bodySmall)),
              Slider(min: 16, max: 30, divisions: 28, value: _temperature, label: '${_temperature.toStringAsFixed(1)}°C',
                onChanged: canControl ? (value) => setState(() => _temperature = value) : null),
              SizedBox(width: double.infinity, child: FilledButton.tonal(onPressed: canControl && !commandController.isBusy
                ? () => _send('set_temperature', params: {'target_temp_c': _temperature.toStringAsFixed(1)}) : null,
                child: const Text('发送温度设定'))),
            ])),
            const SizedBox(height: 16),
            SurfaceCard(child: Row(children: [Icon(Icons.wb_cloudy_outlined, color: Theme.of(context).colorScheme.primary),
              const SizedBox(width: 16), Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                const Text('前后风挡除霜', style: TextStyle(fontWeight: FontWeight.w700)),
                Text(telemetry.defrost ? '除霜正在运行' : '快速改善视野', style: Theme.of(context).textTheme.bodySmall),
              ])), Switch(value: telemetry.defrost, onChanged: canControl && !commandController.isBusy
                ? (_) => _send(telemetry.defrost ? 'defrost_off' : 'defrost_on') : null)])),
            if (commandController.latest != null) ...[
              const SizedBox(height: 24), const SectionTitle('最近指令'), const SizedBox(height: 10),
              SurfaceCard(onTap: () => showCommandProgress(context, commandController.latest!.id), child: Row(children: [
                Icon(commandController.latest!.isTerminal ? Icons.check_circle_outline : Icons.sync_rounded,
                  color: commandController.latest!.status == CommandStatus.failed ? Colors.orange : AuroraColors.cyan),
                const SizedBox(width: 14), Expanded(child: Text(commandNames[commandController.latest!.type] ?? '车辆指令')),
                Text(_shortStatus(commandController.latest!.status)), const Icon(Icons.chevron_right),
              ])),
            ],
            const SizedBox(height: 16), Text('提示：发送成功不等于执行成功。只有在收到车辆终态回执后，页面才会显示成功。',
              style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant)),
          ]);
        })));
  }
  String _shortStatus(CommandStatus status) => switch (status) { CommandStatus.pending => '等待下发', CommandStatus.sent => '已发送',
    CommandStatus.received => '车端已接收', CommandStatus.succeeded => '成功', CommandStatus.failed => '失败', CommandStatus.expired => '已过期' };
}
