import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/vehicle.dart';
import '../state/auth_controller.dart';
import '../state/garage_controller.dart';
import '../widgets/common.dart';

class ServicesScreen extends StatelessWidget {
  const ServicesScreen({super.key});
  void _contact(BuildContext context, String service) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$service：请通过 AURORA 官方客服热线联系服务顾问')));
  }
  @override
  Widget build(BuildContext context) {
    final garage = context.watch<GarageController>();
    return StateView<List<Vehicle>>(state: garage.state, onRetry: garage.load,
      onLogin: () => context.read<AuthController>().expireSession(), emptyTitle: '暂无可用车辆服务', builder: (vehicles) {
        final vehicle = garage.selected!;
        return ListView(padding: const EdgeInsets.fromLTRB(20, 12, 20, 32), children: [
          Container(padding: const EdgeInsets.all(24), decoration: BoxDecoration(borderRadius: BorderRadius.circular(28),
            gradient: const LinearGradient(colors: [Color(0xFF5856D6), Color(0xFF343166)])),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [const Icon(Icons.handyman_outlined, color: Colors.white70),
              const SizedBox(height: 28), const Text('AURORA CARE', style: TextStyle(color: Colors.white70, fontSize: 10, letterSpacing: 3)),
              const SizedBox(height: 8), const Text('全心守护，始终在场', style: TextStyle(color: Colors.white, fontSize: 23, fontWeight: FontWeight.w700)),
              const SizedBox(height: 8), Text('${vehicle.displayName} · ${vehicle.model}', style: const TextStyle(color: Colors.white60))])),
          const SizedBox(height: 22),
          GridView.count(crossAxisCount: MediaQuery.sizeOf(context).width > 700 ? 4 : 2, shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(), mainAxisSpacing: 12, crossAxisSpacing: 12, childAspectRatio: 1.35,
            children: [
              _Service(icon: Icons.build_circle_outlined, title: '保养服务', subtitle: '联系服务顾问', onTap: () => _contact(context, '保养预约')),
              _Service(icon: Icons.emergency_outlined, title: '道路救援', subtitle: '7 × 24 小时守护', onTap: () => _contact(context, '道路救援')),
              _Service(icon: Icons.support_agent_rounded, title: '专属顾问', subtitle: '官方服务支持', onTap: () => _contact(context, '专属顾问')),
              _Service(icon: Icons.description_outlined, title: '用车指南', subtitle: '了解车辆功能', onTap: () => _contact(context, '用车指南')),
            ]),
          const SizedBox(height: 22), const SectionTitle('车辆信息'), const SizedBox(height: 10),
          SurfaceCard(padding: EdgeInsets.zero, child: Column(children: [
            ListTile(title: const Text('车辆识别代码'), subtitle: Text(vehicle.vin)),
            const Divider(height: 1), ListTile(title: const Text('当前系统'), subtitle: Text(vehicle.firmwareVersion ?? '暂未提供')),
            const Divider(height: 1), const ListTile(title: Text('OTA 更新'), subtitle: Text('当前用户协议未开放升级状态查询，请勿轻信非官方升级信息')),
          ])),
          const SizedBox(height: 16), Text('服务权益与可用项目以车辆所在地区的 AURORA 官方政策为准。',
            style: Theme.of(context).textTheme.bodySmall),
        ]);
      });
  }
}
class _Service extends StatelessWidget {
  const _Service({required this.icon, required this.title, required this.subtitle, required this.onTap});
  final IconData icon;
  final String title, subtitle;
  final VoidCallback onTap;
  @override
  Widget build(BuildContext context) => SurfaceCard(onTap: onTap, padding: const EdgeInsets.all(16), child: Column(
    crossAxisAlignment: CrossAxisAlignment.start, mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
      Icon(icon, color: Theme.of(context).colorScheme.primary, size: 28),
      Text(title, style: const TextStyle(fontWeight: FontWeight.w700)),
      Text(subtitle, maxLines: 1, overflow: TextOverflow.ellipsis, style: Theme.of(context).textTheme.bodySmall),
    ]));
}
