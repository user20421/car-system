import 'package:flutter/material.dart';
import '../state/load_state.dart';
import '../theme/aurora_theme.dart';

class AuroraLogo extends StatelessWidget {
  const AuroraLogo({super.key, this.compact = false});
  final bool compact;
  @override
  Widget build(BuildContext context) => Row(mainAxisSize: MainAxisSize.min, children: [
    Container(width: compact ? 30 : 38, height: compact ? 30 : 38,
      decoration: BoxDecoration(borderRadius: BorderRadius.circular(compact ? 10 : 13),
        gradient: const LinearGradient(colors: [AuroraColors.indigo, AuroraColors.violet, AuroraColors.cyan],
          begin: Alignment.topLeft, end: Alignment.bottomRight)),
      child: Icon(Icons.change_history_rounded, color: Colors.white, size: compact ? 20 : 25)),
    if (!compact) ...[const SizedBox(width: 11), Text('AURORA', style: Theme.of(context).textTheme.titleLarge?.copyWith(
      fontWeight: FontWeight.w800, letterSpacing: 3.2))],
  ]);
}

class DemoBadge extends StatelessWidget {
  const DemoBadge({super.key});
  @override
  Widget build(BuildContext context) => Container(padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 5),
    decoration: BoxDecoration(color: AuroraColors.cyan.withValues(alpha: .13), borderRadius: BorderRadius.circular(20)),
    child: Text('演示模式', style: TextStyle(color: Theme.of(context).colorScheme.secondary, fontSize: 11, fontWeight: FontWeight.w700)));
}

class SurfaceCard extends StatelessWidget {
  const SurfaceCard({super.key, required this.child, this.padding = const EdgeInsets.all(20), this.onTap});
  final Widget child;
  final EdgeInsetsGeometry padding;
  final VoidCallback? onTap;
  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Material(color: dark ? const Color(0xFF171B29) : Colors.white,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24),
        side: BorderSide(color: dark ? Colors.white.withValues(alpha: .06) : Colors.white)),
      clipBehavior: Clip.antiAlias, elevation: 0,
      child: InkWell(onTap: onTap, child: Padding(padding: padding, child: child)));
  }
}

class SectionTitle extends StatelessWidget {
  const SectionTitle(this.title, {super.key, this.action});
  final String title;
  final Widget? action;
  @override
  Widget build(BuildContext context) => Row(children: [Expanded(child: Text(title,
      style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700))), ?action]);
}

class StateView<T> extends StatelessWidget {
  const StateView({super.key, required this.state, required this.builder, required this.onRetry, this.onLogin, this.emptyTitle = '这里还很安静'});
  final LoadState<T> state;
  final Widget Function(T data) builder;
  final VoidCallback onRetry;
  final VoidCallback? onLogin;
  final String emptyTitle;
  @override
  Widget build(BuildContext context) => switch (state.status) {
    LoadStatus.loading => const Center(child: Padding(padding: EdgeInsets.all(48), child: CircularProgressIndicator())),
    LoadStatus.ready => builder(state.data as T),
    LoadStatus.empty => _Message(icon: Icons.inbox_outlined, title: emptyTitle,
        message: '暂时没有可显示的内容', action: '刷新', onAction: onRetry),
    LoadStatus.error => _Message(icon: Icons.cloud_off_outlined, title: '暂时未能加载',
        message: state.message ?? '请稍后重试', action: '重试', onAction: onRetry),
    LoadStatus.offline => state.data != null ? builder(state.data as T) : _Message(icon: Icons.wifi_off_rounded,
        title: '当前处于离线状态', message: '连接网络后可同步最新车辆数据', action: '重试', onAction: onRetry),
    LoadStatus.unauthorized => _Message(icon: Icons.lock_outline_rounded, title: '登录状态已失效',
        message: '请重新登录后查看你的车辆', action: '重新登录', onAction: onLogin),
  };
}

class _Message extends StatelessWidget {
  const _Message({required this.icon, required this.title, required this.message, required this.action, this.onAction});
  final IconData icon;
  final String title;
  final String message;
  final String action;
  final VoidCallback? onAction;
  @override
  Widget build(BuildContext context) => Center(child: Padding(padding: const EdgeInsets.all(32), child: Column(
    mainAxisSize: MainAxisSize.min, children: [Icon(icon, size: 42, color: Theme.of(context).colorScheme.primary),
      const SizedBox(height: 16), Text(title, style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700)),
      const SizedBox(height: 7), Text(message, textAlign: TextAlign.center, style: Theme.of(context).textTheme.bodyMedium?.copyWith(
        color: Theme.of(context).colorScheme.onSurfaceVariant)),
      if (onAction != null) ...[const SizedBox(height: 18), OutlinedButton(onPressed: onAction, child: Text(action))],
    ])));
}

class OfflineBanner extends StatelessWidget {
  const OfflineBanner({super.key});
  @override
  Widget build(BuildContext context) => Material(color: const Color(0xFF38333F), child: SafeArea(bottom: false,
    child: Padding(padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
      child: Row(children: const [Icon(Icons.wifi_off, size: 16, color: Colors.white), SizedBox(width: 8),
        Expanded(child: Text('网络已断开 · 显示最近一次数据', style: TextStyle(color: Colors.white, fontSize: 12)))]))));
}
