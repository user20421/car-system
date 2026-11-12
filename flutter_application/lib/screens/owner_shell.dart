import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/repository.dart';
import '../state/alerts_controller.dart';
import '../state/connectivity_controller.dart';
import '../state/garage_controller.dart';
import '../state/vehicle_controller.dart';
import '../widgets/common.dart';
import 'alerts_screen.dart';
import 'garage_screen.dart';
import 'home_screen.dart';
import 'journeys_screen.dart';
import 'profile_screen.dart';
import 'services_screen.dart';

class OwnerShell extends StatefulWidget {
  const OwnerShell({super.key});
  @override
  State<OwnerShell> createState() => _OwnerShellState();
}
class _OwnerShellState extends State<OwnerShell> {
  int _index = 0;
  String? _vin;
  @override
  void initState() {
    super.initState();
    final garage = context.read<GarageController>()..addListener(_syncVehicle);
    garage.load();
    context.read<AlertsController>().load();
  }
  void _syncVehicle() {
    final next = context.read<GarageController>().selected?.vin;
    if (next != null && next != _vin) {
      _vin = next;
      context.read<VehicleController>().select(next);
    }
  }
  @override
  void dispose() { context.read<GarageController>().removeListener(_syncVehicle); super.dispose(); }
  void _openAlerts() => Navigator.push(context, MaterialPageRoute(builder: (_) => const AlertsScreen()));
  @override
  Widget build(BuildContext context) {
    final online = context.watch<ConnectivityController>().online;
    final demo = context.read<CarRepository>().isDemo;
    const titles = ['我的座驾', '我的车辆', '行程与趋势', '车辆服务', '个人中心'];
    final screens = [HomeScreen(onNavigate: (index) => setState(() => _index = index)),
      const GarageScreen(), const JourneysScreen(), const ServicesScreen(), const ProfileScreen()];
    final content = SafeArea(bottom: false, child: Column(children: [
      Padding(padding: const EdgeInsets.fromLTRB(20, 8, 12, 4), child: Row(children: [
        const AuroraLogo(compact: true), const SizedBox(width: 12),
        Expanded(child: Text(titles[_index], style: Theme.of(context).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700))),
        if (demo) const DemoBadge(), const SizedBox(width: 4),
        IconButton(tooltip: '车辆告警', onPressed: _openAlerts, icon: const Icon(Icons.notifications_none_rounded)),
      ])),
      if (!online) const OfflineBanner(),
      Expanded(child: IndexedStack(index: _index, children: screens)),
    ]));
    return LayoutBuilder(builder: (context, constraints) {
      if (constraints.maxWidth >= 920) {
        return Scaffold(body: Row(children: [NavigationRail(selectedIndex: _index, onDestinationSelected: (i) => setState(() => _index = i),
          labelType: NavigationRailLabelType.all, leading: const SizedBox(height: 20), destinations: const [
            NavigationRailDestination(icon: Icon(Icons.space_dashboard_outlined), selectedIcon: Icon(Icons.space_dashboard), label: Text('座驾')),
            NavigationRailDestination(icon: Icon(Icons.directions_car_outlined), label: Text('车辆')),
            NavigationRailDestination(icon: Icon(Icons.route_outlined), label: Text('行程')),
            NavigationRailDestination(icon: Icon(Icons.handyman_outlined), label: Text('服务')),
            NavigationRailDestination(icon: Icon(Icons.person_outline), label: Text('我的')),
          ]), const VerticalDivider(width: 1), Expanded(child: content)]));
      }
      return Scaffold(body: content, bottomNavigationBar: SafeArea(child: NavigationBar(selectedIndex: _index,
        onDestinationSelected: (i) => setState(() => _index = i), destinations: const [
          NavigationDestination(icon: Icon(Icons.space_dashboard_outlined), selectedIcon: Icon(Icons.space_dashboard), label: '座驾'),
          NavigationDestination(icon: Icon(Icons.directions_car_outlined), label: '车辆'),
          NavigationDestination(icon: Icon(Icons.route_outlined), label: '行程'),
          NavigationDestination(icon: Icon(Icons.handyman_outlined), label: '服务'),
          NavigationDestination(icon: Icon(Icons.person_outline), label: '我的'),
        ])));
    });
  }
}
