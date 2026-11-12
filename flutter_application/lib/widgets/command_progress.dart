import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/command.dart';
import '../state/command_controller.dart';
import '../theme/aurora_theme.dart';

const commandNames = {
  'lock': '锁车', 'unlock': '解锁车辆', 'hvac_on': '开启空调', 'hvac_off': '关闭空调',
  'set_temperature': '设定温度', 'defrost_on': '开启除霜', 'defrost_off': '关闭除霜', 'honk_flash': '鸣笛闪灯',
};

Future<void> showCommandProgress(BuildContext context, String commandId) => showModalBottomSheet<void>(
  context: context, isDismissible: true, backgroundColor: Colors.transparent,
  builder: (sheetContext) => Container(decoration: BoxDecoration(color: Theme.of(context).scaffoldBackgroundColor,
      borderRadius: const BorderRadius.vertical(top: Radius.circular(30))),
    padding: const EdgeInsets.fromLTRB(24, 16, 24, 30),
    child: SafeArea(top: false, child: Consumer<CommandController>(builder: (context, controller, _) {
      final command = controller.commandById(commandId);
      if (command == null) return const SizedBox(height: 180, child: Center(child: CircularProgressIndicator()));
      return CommandProgressView(command: command);
    }))));

class CommandProgressView extends StatelessWidget {
  const CommandProgressView({super.key, required this.command});
  final VehicleCommand command;
  @override
  Widget build(BuildContext context) {
    final status = command.status;
    final activeIndex = switch (status) { CommandStatus.pending => 0, CommandStatus.sent => 1,
      CommandStatus.received => 2, _ => 3 };
    return Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
      Center(child: Container(width: 38, height: 4, decoration: BoxDecoration(color: Theme.of(context).dividerColor,
        borderRadius: BorderRadius.circular(4)))),
      const SizedBox(height: 24),
      Row(children: [Expanded(child: Text(commandNames[command.type] ?? '车辆指令',
        style: Theme.of(context).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w700))),
        if (!command.isTerminal) const SizedBox(width: 22, height: 22, child: CircularProgressIndicator(strokeWidth: 2)),
        if (status == CommandStatus.succeeded) const Icon(Icons.check_circle_rounded, color: AuroraColors.success, size: 30),
        if (status == CommandStatus.failed || status == CommandStatus.expired)
          const Icon(Icons.error_rounded, color: Colors.orange, size: 30),
      ]),
      const SizedBox(height: 8),
      Text(_statusText(status), style: Theme.of(context).textTheme.bodyMedium?.copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant)),
      const SizedBox(height: 24),
      Row(children: List.generate(4, (index) => Expanded(child: Row(children: [
        _StepDot(index: index, activeIndex: activeIndex, terminal: command.isTerminal, succeeded: status == CommandStatus.succeeded),
        if (index < 3) Expanded(child: Container(height: 2, color: index < activeIndex ? AuroraColors.cyan : Theme.of(context).dividerColor)),
      ])))),
      const SizedBox(height: 8),
      const Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
        Text('已创建', style: TextStyle(fontSize: 10)), Text('已下发', style: TextStyle(fontSize: 10)),
        Text('车端接收', style: TextStyle(fontSize: 10)), Text('最终结果', style: TextStyle(fontSize: 10)),
      ]),
      if (command.error != null) Padding(padding: const EdgeInsets.only(top: 18), child: Text(
        '${command.error!.code} · ${command.error!.message}', style: const TextStyle(color: Colors.orange))),
      const SizedBox(height: 22),
      if (command.isTerminal) SizedBox(width: double.infinity, child: FilledButton(
        onPressed: () => Navigator.pop(context), child: const Text('完成'))),
    ]);
  }
  static String _statusText(CommandStatus status) => switch (status) {
    CommandStatus.pending => '指令已安全创建，等待下发', CommandStatus.sent => '指令已发送，等待车辆确认',
    CommandStatus.received => '车辆已接收，正在执行', CommandStatus.succeeded => '车辆已确认执行成功',
    CommandStatus.failed => '车辆未能完成指令，请查看原因', CommandStatus.expired => '指令已过期，车辆未执行',
  };
}
class _StepDot extends StatelessWidget {
  const _StepDot({required this.index, required this.activeIndex, required this.terminal, required this.succeeded});
  final int index;
  final int activeIndex;
  final bool terminal;
  final bool succeeded;
  @override
  Widget build(BuildContext context) {
    final done = index < activeIndex || (terminal && succeeded);
    return Container(width: 12, height: 12, decoration: BoxDecoration(shape: BoxShape.circle,
      color: done ? AuroraColors.cyan : index == activeIndex ? AuroraColors.indigo : Theme.of(context).dividerColor));
  }
}
