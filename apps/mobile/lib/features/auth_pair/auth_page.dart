import 'package:flutter/material.dart';

import 'package:couple_daily/core/errors/api_exception.dart';
import 'package:couple_daily/core/session/session_scope.dart';

enum _AuthMode { login, register }

/// Email + password only. Phone / SMS OTP is not part of this client.
class AuthPage extends StatefulWidget {
  const AuthPage({super.key});

  @override
  State<AuthPage> createState() => _AuthPageState();
}

class _AuthPageState extends State<AuthPage> {
  final _email = TextEditingController();
  final _password = TextEditingController();
  final _name = TextEditingController();
  _AuthMode _mode = _AuthMode.login;
  String? _error;

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    _name.dispose();
    super.dispose();
  }

  void _submit() {
    final session = SessionScope.read(context);
    try {
      if (_mode == _AuthMode.register) {
        session.register(
          email: _email.text,
          password: _password.text,
          displayName: _name.text,
        );
      } else {
        session.login(email: _email.text, password: _password.text);
      }
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = messageOf(error));
    }
  }

  @override
  Widget build(BuildContext context) {
    final registering = _mode == _AuthMode.register;
    return Scaffold(
      appBar: AppBar(title: const Text('情侣日常')),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          SegmentedButton<_AuthMode>(
            segments: const [
              ButtonSegment(value: _AuthMode.login, label: Text('登录')),
              ButtonSegment(value: _AuthMode.register, label: Text('注册')),
            ],
            selected: {_mode},
            onSelectionChanged: (value) {
              setState(() {
                _mode = value.first;
                _error = null;
              });
            },
          ),
          const SizedBox(height: 8),
          Text(
            '使用邮箱和密码。首版没有手机号，也没有短信验证码。',
            style: Theme.of(context).textTheme.bodyMedium,
          ),
          const SizedBox(height: 20),
          if (registering) ...[
            TextField(
              key: const Key('auth-name'),
              controller: _name,
              textInputAction: TextInputAction.next,
              decoration: const InputDecoration(labelText: '昵称'),
            ),
            const SizedBox(height: 12),
          ],
          TextField(
            key: const Key('auth-email'),
            controller: _email,
            keyboardType: TextInputType.emailAddress,
            autocorrect: false,
            textInputAction: TextInputAction.next,
            decoration: const InputDecoration(labelText: '邮箱'),
          ),
          const SizedBox(height: 12),
          TextField(
            key: const Key('auth-password'),
            controller: _password,
            obscureText: true,
            decoration: const InputDecoration(
              labelText: '密码',
              helperText: '至少 8 位',
            ),
            onSubmitted: (_) => _submit(),
          ),
          if (_error != null) ...[
            const SizedBox(height: 12),
            Text(
              _error!,
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
          ],
          const SizedBox(height: 20),
          FilledButton(
            key: const Key('auth-submit'),
            onPressed: _submit,
            child: Text(registering ? '注册' : '登录'),
          ),
        ],
      ),
    );
  }
}
