import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/repository.dart';
import '../state/auth_controller.dart';
import '../widgets/common.dart';
import '../widgets/vehicle_artwork.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});
  @override
  State<LoginScreen> createState() => _LoginScreenState();
}
class _LoginScreenState extends State<LoginScreen> {
  final _formKey = GlobalKey<FormState>();
  final _username = TextEditingController();
  final _password = TextEditingController();
  bool _obscure = true;
  @override
  void initState() {
    super.initState();
    if (context.read<CarRepository>().isDemo) {
      _username.text = 'demo';
      _password.text = 'aurora-demo';
    }
  }
  @override
  void dispose() { _username.dispose(); _password.dispose(); super.dispose(); }
  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    FocusScope.of(context).unfocus();
    await context.read<AuthController>().login(_username.text, _password.text);
  }
  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthController>();
    final demo = context.read<CarRepository>().isDemo;
    return Scaffold(body: Container(decoration: BoxDecoration(gradient: LinearGradient(
      colors: Theme.of(context).brightness == Brightness.dark ? const [Color(0xFF0D101A), Color(0xFF191735)]
          : const [Color(0xFFF4F6FB), Color(0xFFE9EAF9)], begin: Alignment.topLeft, end: Alignment.bottomRight)),
      child: SafeArea(child: LayoutBuilder(builder: (context, constraints) {
        final wide = constraints.maxWidth > 900;
        final form = Center(child: SingleChildScrollView(padding: EdgeInsets.symmetric(horizontal: wide ? 64 : 24, vertical: 28),
          child: ConstrainedBox(constraints: const BoxConstraints(maxWidth: 440), child: AutofillGroup(child: Form(key: _formKey,
            child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
              const AuroraLogo(), const SizedBox(height: 48),
              Row(children: [Expanded(child: Text('欢迎回来', style: Theme.of(context).textTheme.headlineLarge?.copyWith(
                fontWeight: FontWeight.w700, letterSpacing: -1))), if (demo) const DemoBadge()]),
              const SizedBox(height: 10),
              Text('登录以连接你的 AURORA，掌握每一程。', style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                color: Theme.of(context).colorScheme.onSurfaceVariant)),
              const SizedBox(height: 34),
              TextFormField(key: const Key('login-username'), controller: _username, autofillHints: const [AutofillHints.username],
                textInputAction: TextInputAction.next, decoration: const InputDecoration(labelText: '账号', prefixIcon: Icon(Icons.person_outline)),
                validator: (value) => value == null || value.trim().isEmpty ? '请输入账号' : null),
              const SizedBox(height: 16),
              TextFormField(key: const Key('login-password'), controller: _password, obscureText: _obscure,
                autofillHints: const [AutofillHints.password], onFieldSubmitted: (_) => _submit(),
                decoration: InputDecoration(labelText: '密码', prefixIcon: const Icon(Icons.lock_outline),
                  suffixIcon: IconButton(onPressed: () => setState(() => _obscure = !_obscure),
                    icon: Icon(_obscure ? Icons.visibility_outlined : Icons.visibility_off_outlined))),
                validator: (value) => value == null || value.length < 8 ? '密码至少 8 位' : null),
              if (auth.loginError != null) Padding(padding: const EdgeInsets.only(top: 16),
                child: Text(auth.loginError!, style: TextStyle(color: Theme.of(context).colorScheme.error))),
              const SizedBox(height: 24),
              FilledButton(key: const Key('login-submit'), onPressed: auth.isBusy ? null : _submit,
                child: auth.isBusy ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2))
                    : Text(demo ? '进入演示车库' : '安全登录')),
              const SizedBox(height: 20),
              Row(mainAxisAlignment: MainAxisAlignment.center, children: [Icon(Icons.shield_outlined, size: 16,
                color: Theme.of(context).colorScheme.onSurfaceVariant), const SizedBox(width: 6),
                Flexible(child: Text('加密连接 · 仅展示你名下的车辆', style: TextStyle(fontSize: 12,
                  color: Theme.of(context).colorScheme.onSurfaceVariant)))]),
              if (demo) Padding(padding: const EdgeInsets.only(top: 18), child: Text('演示数据仅在本机生成，不会发起控车请求。',
                textAlign: TextAlign.center, style: Theme.of(context).textTheme.bodySmall)),
            ]))))));
        if (!wide) return form;
        return Row(children: [Expanded(child: Padding(padding: const EdgeInsets.all(40), child: Column(
          mainAxisAlignment: MainAxisAlignment.center, children: [const Spacer(),
            Text('BEYOND\nTHE HORIZON', style: Theme.of(context).textTheme.displayMedium?.copyWith(
              fontWeight: FontWeight.w200, letterSpacing: 5, height: 1.15)),
            const SizedBox(height: 20), const VehicleArtwork(height: 300), const Spacer(),
            Align(alignment: Alignment.bottomLeft, child: Text('AURORA INTELLIGENT MOBILITY',
              style: TextStyle(fontSize: 10, letterSpacing: 3, color: Theme.of(context).colorScheme.onSurfaceVariant))),
          ]))), SizedBox(width: 510, child: form)]);
      }))));
  }
}
