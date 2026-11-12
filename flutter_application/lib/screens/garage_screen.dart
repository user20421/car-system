import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/vehicle.dart';
import '../state/auth_controller.dart';
import '../state/garage_controller.dart';
import '../widgets/common.dart';
import '../widgets/vehicle_artwork.dart';

class GarageScreen extends StatelessWidget {
  const GarageScreen({super.key});
  @override
  Widget build(BuildContext context) {
    final garage = context.watch<GarageController>();
    return RefreshIndicator(onRefresh: garage.load, child: CustomScrollView(slivers: [
      SliverPadding(padding: const EdgeInsets.fromLTRB(20, 14, 20, 32), sliver: SliverToBoxAdapter(
        child: StateView<List<Vehicle>>(state: garage.state, onRetry: garage.load,
          onLogin: () => context.read<AuthController>().logout(), emptyTitle: '你的车库还没有车辆',
          builder: (vehicles) => LayoutBuilder(builder: (context, constraints) {
            final columns = constraints.maxWidth > 1050 ? 3 : constraints.maxWidth > 620 ? 2 : 1;
            if (columns == 1) {
              return Column(children: vehicles.map((vehicle) => Padding(
                padding: const EdgeInsets.only(bottom: 16), child: _VehicleCard(vehicle: vehicle))).toList());
            }
            return GridView.builder(shrinkWrap: true, physics: const NeverScrollableScrollPhysics(),
              gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(crossAxisCount: columns,
                childAspectRatio: 1.25, crossAxisSpacing: 16, mainAxisSpacing: 16),
              itemCount: vehicles.length, itemBuilder: (_, index) => _VehicleCard(vehicle: vehicles[index]));
          })))),
    ]));
  }
}
class _VehicleCard extends StatelessWidget {
  const _VehicleCard({required this.vehicle});
  final Vehicle vehicle;
  @override
  Widget build(BuildContext context) {
    final selected = context.watch<GarageController>().selectedVin == vehicle.vin;
    final role = switch (vehicle.role) { VehicleRole.owner => '车主', VehicleRole.driver => '授权驾驶人', VehicleRole.viewer => '仅查看' };
    return SurfaceCard(padding: EdgeInsets.zero, onTap: () {
      context.read<GarageController>().select(vehicle.vin);
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('已切换至 ${vehicle.displayName}'), duration: const Duration(seconds: 1)));
    }, child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      SizedBox(height: 160, child: Stack(children: [const Positioned.fill(child: VehicleArtwork(height: 170, glow: false)),
        Positioned(top: 16, right: 16, child: Icon(selected ? Icons.check_circle : Icons.circle_outlined,
          color: selected ? Theme.of(context).colorScheme.primary : Theme.of(context).colorScheme.outline)),
      ])),
      Padding(padding: const EdgeInsets.fromLTRB(20, 0, 20, 20), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(vehicle.displayName, maxLines: 1, overflow: TextOverflow.ellipsis,
          style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700)),
        const SizedBox(height: 5), Text('${vehicle.model} · ${vehicle.modelYear ?? ''}', style: Theme.of(context).textTheme.bodySmall),
        const SizedBox(height: 14), Row(children: [Container(padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
          decoration: BoxDecoration(color: Theme.of(context).colorScheme.primaryContainer, borderRadius: BorderRadius.circular(8)),
          child: Text(role, style: TextStyle(fontSize: 10, color: Theme.of(context).colorScheme.onPrimaryContainer))),
          const Spacer(), Text(vehicle.vin, style: const TextStyle(fontSize: 9, letterSpacing: .5))]),
      ])),
    ]));
  }
}
