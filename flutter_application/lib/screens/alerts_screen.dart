import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/alert.dart';
import '../services/repository.dart';
import '../state/alerts_controller.dart';
import '../state/auth_controller.dart';
import '../widgets/common.dart';

class AlertsScreen extends StatelessWidget {
  const AlertsScreen({super.key});
  Future<void> _acknowledge(BuildContext context, VehicleAlert alert) async {
    final confirmed = await showDialog<bool>(context: context, builder: (dialogContext) => AlertDialog(
      title: const Text('确认已知悉此提醒？'), content: Text(alert.title),
      actions: [TextButton(onPressed: () => Navigator.pop(dialogContext, false), child: const Text('取消')),
        FilledButton(onPressed: () => Navigator.pop(dialogContext, true), child: const Text('确认'))]));
    if (confirmed != true || !context.mounted) return;
    try {
      await context.read<AlertsController>().acknowledge(alert);
    } on RepositoryException catch (error) {
      if (context.mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(error.message)));
    }
  }
  @override
  Widget build(BuildContext context) {
    final controller = context.watch<AlertsController>();
    return Scaffold(
      appBar: AppBar(title: const Text('车辆告警')),
      body: SafeArea(
        child: RefreshIndicator(
          onRefresh: controller.load,
          child: ListView(
            padding: const EdgeInsets.all(20),
            children: [
              StateView<List<VehicleAlert>>(
                state: controller.state,
                onRetry: controller.load,
                onLogin: () => context.read<AuthController>().expireSession(),
                emptyTitle: '没有车辆告警',
                builder: (alerts) => Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('车辆提醒会保留确认记录。确认不代表故障已消除，请按提示检查车辆。',
                      style: Theme.of(context).textTheme.bodySmall),
                    const SizedBox(height: 18),
                    ...alerts.map((alert) => Padding(
                      padding: const EdgeInsets.only(bottom: 12),
                      child: _AlertCard(alert: alert, onAcknowledge: () => _acknowledge(context, alert)),
                    )),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
  static String _format(DateTime time) {
    final local = time.toLocal();
    return '${local.year}-${local.month.toString().padLeft(2, '0')}-${local.day.toString().padLeft(2, '0')} '
      '${local.hour.toString().padLeft(2, '0')}:${local.minute.toString().padLeft(2, '0')}';
  }
}

class _AlertCard extends StatelessWidget {
  const _AlertCard({required this.alert, required this.onAcknowledge});
  final VehicleAlert alert;
  final VoidCallback onAcknowledge;
  @override
  Widget build(BuildContext context) {
    final acknowledging = context.watch<AlertsController>().isAcknowledging(alert.id);
    return SurfaceCard(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        Icon(alert.severity == 'critical' ? Icons.error_outline : Icons.warning_amber_rounded,
          color: alert.severity == 'info' ? Theme.of(context).colorScheme.primary : Colors.orange),
        const SizedBox(width: 12),
        Expanded(child: Text(alert.title, style: const TextStyle(fontWeight: FontWeight.w700))),
      ]),
      const SizedBox(height: 12),
      Text('${alert.code} · ${AlertsScreen._format(alert.occurredAt)}', style: Theme.of(context).textTheme.bodySmall),
      const SizedBox(height: 14),
      Row(children: [
        Text(alert.isOpen ? '待确认' : '已确认',
          style: TextStyle(fontSize: 12, color: alert.isOpen ? Colors.orange : Theme.of(context).colorScheme.primary)),
        const Spacer(),
        if (alert.isOpen) FilledButton.tonal(onPressed: acknowledging ? null : onAcknowledge,
          child: acknowledging ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2))
              : const Text('我已知悉')),
      ]),
    ]));
  }
}
