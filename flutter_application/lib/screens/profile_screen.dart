import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/repository.dart';
import '../state/auth_controller.dart';
import '../state/garage_controller.dart';
import '../state/load_state.dart';
import '../state/settings_controller.dart';
import '../widgets/common.dart';

class ProfileScreen extends StatelessWidget {
  const ProfileScreen({super.key});
  Future<void> _logout(BuildContext context) async {
    final confirmed = await showDialog<bool>(context: context, builder: (dialogContext) => AlertDialog(
      title: const Text('退出当前账号？'), content: const Text('退出后将撤销当前会话，原生设备上保存的刷新令牌也会被删除。'),
      actions: [TextButton(onPressed: () => Navigator.pop(dialogContext, false), child: const Text('取消')),
        FilledButton(onPressed: () => Navigator.pop(dialogContext, true), child: const Text('退出登录'))]));
    if (confirmed == true && context.mounted) await context.read<AuthController>().logout();
  }
  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthController>();
    final settings = context.watch<SettingsController>();
    final user = auth.session?.user;
    if (user == null) {
      return StateView<String>(state: const LoadState.unauthorized(), onRetry: () {},
        onLogin: () => context.read<AuthController>().expireSession(), builder: (_) => const SizedBox());
    }
    final vehicles = context.watch<GarageController>().vehicles;
    return ListView(padding: const EdgeInsets.fromLTRB(20, 12, 20, 32), children: [
      SurfaceCard(child: Row(children: [Container(width: 64, height: 64, decoration: const BoxDecoration(shape: BoxShape.circle,
        gradient: LinearGradient(colors: [Color(0xFF5856D6), Color(0xFF16C8C0)])),
        child: Center(child: Text(user.displayName.characters.first, style: const TextStyle(fontSize: 25, color: Colors.white, fontWeight: FontWeight.w700)))),
        const SizedBox(width: 18), Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(user.displayName, style: Theme.of(context).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700)),
          const SizedBox(height: 4), Text(user.email ?? '@${user.username}', style: Theme.of(context).textTheme.bodySmall),
          const SizedBox(height: 8), Text('${vehicles.length} 辆可访问车辆 · 权限由服务端校验', style: TextStyle(fontSize: 11,
            color: Theme.of(context).colorScheme.primary)),
        ]))])),
      const SizedBox(height: 24), const SectionTitle('外观与体验'), const SizedBox(height: 10),
      SurfaceCard(padding: EdgeInsets.zero, child: Column(children: [
        ListTile(leading: const Icon(Icons.brightness_6_outlined), title: const Text('主题外观'),
          subtitle: Text(switch (settings.themeMode) { ThemeMode.system => '跟随系统', ThemeMode.light => '浅色', ThemeMode.dark => '深色' }),
          trailing: DropdownButton<ThemeMode>(value: settings.themeMode, underline: const SizedBox(), items: const [
            DropdownMenuItem(value: ThemeMode.system, child: Text('自动')), DropdownMenuItem(value: ThemeMode.light, child: Text('浅色')),
            DropdownMenuItem(value: ThemeMode.dark, child: Text('深色'))], onChanged: (value) => settings.setThemeMode(value!))),
        const Divider(height: 1), SwitchListTile(secondary: const Icon(Icons.vibration_rounded), title: const Text('触觉反馈'),
          subtitle: const Text('关键操作提供轻微反馈'), value: settings.hapticFeedback, onChanged: settings.setHaptics),
      ])),
      const SizedBox(height: 24), const SectionTitle('账号与安全'), const SizedBox(height: 10),
      SurfaceCard(padding: EdgeInsets.zero, child: Column(children: [
        ListTile(leading: const Icon(Icons.shield_outlined), title: const Text('会话保护'),
          subtitle: Text(kIsWeb ? 'Web：内存会话 + 浏览器 Cookie 策略接口' : '刷新令牌已保存在系统安全存储中')),
        const Divider(height: 1), const ListTile(leading: Icon(Icons.verified_user_outlined), title: Text('车辆权限'),
          subtitle: Text('每次车辆与控车请求均由服务端重新鉴权')),
        const Divider(height: 1), ListTile(leading: const Icon(Icons.info_outline), title: const Text('关于 AURORA'),
          subtitle: const Text('AURORA 车主 1.0.0 · API v1')),
      ])),
      if (context.read<CarRepository>().isDemo) const Padding(padding: EdgeInsets.all(16), child: DemoBadge()),
      const SizedBox(height: 24), OutlinedButton.icon(onPressed: () => _logout(context), icon: const Icon(Icons.logout),
        label: const Text('退出登录'), style: OutlinedButton.styleFrom(foregroundColor: Theme.of(context).colorScheme.error)),
    ]);
  }
}
