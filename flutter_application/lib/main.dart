import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'screens/login_screen.dart';
import 'screens/owner_shell.dart';
import 'services/api_client.dart';
import 'services/demo_car_repository.dart';
import 'services/http_car_repository.dart';
import 'services/repository.dart';
import 'state/alerts_controller.dart';
import 'state/auth_controller.dart';
import 'state/command_controller.dart';
import 'state/connectivity_controller.dart';
import 'state/garage_controller.dart';
import 'state/history_controller.dart';
import 'state/settings_controller.dart';
import 'state/vehicle_controller.dart';
import 'theme/aurora_theme.dart';
import 'widgets/common.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  const demo = bool.fromEnvironment('AURORA_DEMO');
  runApp(const AuroraRoot(demo: demo));
}

class AuroraRoot extends StatefulWidget {
  const AuroraRoot({super.key, required this.demo});
  final bool demo;
  @override
  State<AuroraRoot> createState() => _AuroraRootState();
}
class _AuroraRootState extends State<AuroraRoot> {
  late final CarRepository repository;
  late final AuthController auth;
  late final GarageController garage;
  late final HistoryController history;
  late final VehicleController vehicle;
  late final CommandController commands;
  late final AlertsController alerts;
  late final SettingsController settings;
  late final ConnectivityController connectivity;
  late final DemoCarRepository? demoRepository;

  @override
  void initState() {
    super.initState();
    if (widget.demo) {
      demoRepository = DemoCarRepository();
      repository = demoRepository!;
    } else {
      demoRepository = null;
      final api = ApiClient();
      repository = HttpCarRepository(api);
      api.onSessionExpired = () => auth.expireSession;
    }
    auth = AuthController(repository);
    garage = GarageController(repository);
    history = HistoryController(repository);
    vehicle = VehicleController(repository);
    commands = CommandController(repository);
    alerts = AlertsController(repository);
    settings = SettingsController();
    connectivity = ConnectivityController();
    vehicle.onCommand = commands.acceptEvent;
    commands.onTerminal = (_) => vehicle.refreshSnapshot();
    auth.restore();
  }

  @override
  void dispose() {
    auth.dispose(); garage.dispose(); history.dispose(); vehicle.dispose(); commands.dispose(); alerts.dispose();
    settings.dispose(); connectivity.dispose(); demoRepository?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => MultiProvider(providers: [
    Provider<CarRepository>.value(value: repository),
    ChangeNotifierProvider.value(value: auth), ChangeNotifierProvider.value(value: garage),
    ChangeNotifierProvider.value(value: history), ChangeNotifierProvider.value(value: vehicle), ChangeNotifierProvider.value(value: commands),
    ChangeNotifierProvider.value(value: alerts), ChangeNotifierProvider.value(value: settings),
    ChangeNotifierProvider.value(value: connectivity),
  ], child: const AuroraApp());
}

class AuroraApp extends StatelessWidget {
  const AuroraApp({super.key});
  @override
  Widget build(BuildContext context) {
    final settings = context.watch<SettingsController>();
    final auth = context.watch<AuthController>();
    return MaterialApp(title: 'AURORA 车主', debugShowCheckedModeBanner: false,
      theme: AuroraTheme.light(), darkTheme: AuroraTheme.dark(), themeMode: settings.themeMode,
      home: AnimatedSwitcher(duration: const Duration(milliseconds: 500), switchInCurve: Curves.easeOut,
        child: switch (auth.status) {
          AuthStatus.restoring => const SplashScreen(key: ValueKey('splash')),
          AuthStatus.unauthenticated => const LoginScreen(key: ValueKey('login')),
          AuthStatus.authenticated => OwnerShell(key: ValueKey(auth.session!.user.id)),
        }));
  }
}

class SplashScreen extends StatefulWidget {
  const SplashScreen({super.key});
  @override
  State<SplashScreen> createState() => _SplashScreenState();
}
class _SplashScreenState extends State<SplashScreen> with SingleTickerProviderStateMixin {
  late final AnimationController controller = AnimationController(vsync: this, duration: const Duration(milliseconds: 1200))..forward();
  @override
  void dispose() { controller.dispose(); super.dispose(); }
  @override
  Widget build(BuildContext context) => Scaffold(body: Container(decoration: const BoxDecoration(
    gradient: LinearGradient(colors: [Color(0xFF11152A), Color(0xFF24204A)], begin: Alignment.topLeft, end: Alignment.bottomRight)),
    child: Center(child: FadeTransition(opacity: CurvedAnimation(parent: controller, curve: Curves.easeOut),
      child: ScaleTransition(scale: Tween(begin: .94, end: 1.0).animate(controller),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Theme(data: AuroraTheme.dark(), child: const AuroraLogo()), const SizedBox(height: 18),
          const Text('让每一次出发，心有极光', style: TextStyle(color: Colors.white54, letterSpacing: 2, fontSize: 12)),
        ]))))));
}
