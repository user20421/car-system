class OwnerUser {
  const OwnerUser({required this.id, required this.username, required this.displayName, required this.roles, required this.createdAt, this.email});
  final String id;
  final String username;
  final String displayName;
  final List<String> roles;
  final DateTime createdAt;
  final String? email;

  factory OwnerUser.fromJson(Map<String, dynamic> json) => OwnerUser(
        id: json['id'] as String? ?? '', username: json['username'] as String? ?? '',
        displayName: json['display_name'] as String? ?? '',
        roles: (json['roles'] as List? ?? const []).map((e) => '$e').toList(),
        createdAt: DateTime.tryParse(json['created_at'] as String? ?? '') ?? DateTime.fromMillisecondsSinceEpoch(0),
        email: json['email'] as String?);
  Map<String, dynamic> toJson() => {'id': id, 'username': username, 'display_name': displayName,
        'roles': roles, 'created_at': createdAt.toIso8601String(), if (email != null) 'email': email};
}

class AuthSession {
  const AuthSession({required this.accessToken, required this.refreshToken, required this.expiresAt, required this.user});
  final String accessToken;
  final String refreshToken;
  final DateTime expiresAt;
  final OwnerUser user;
  bool get isExpired => DateTime.now().isAfter(expiresAt);

  factory AuthSession.fromJson(Map<String, dynamic> json) => AuthSession(
        accessToken: json['access_token'] as String? ?? '', refreshToken: json['refresh_token'] as String? ?? '',
        expiresAt: DateTime.now().add(Duration(seconds: json['expires_in'] as int? ?? 900)),
        user: OwnerUser.fromJson(json['user'] as Map<String, dynamic>? ?? const {}));
  Map<String, dynamic> toJson() => {'access_token': accessToken, 'refresh_token': refreshToken,
        'expires_at': expiresAt.toIso8601String(), 'user': user.toJson()};
  factory AuthSession.fromStoredJson(Map<String, dynamic> json) => AuthSession(
        accessToken: json['access_token'] as String? ?? '', refreshToken: json['refresh_token'] as String? ?? '',
        expiresAt: DateTime.tryParse(json['expires_at'] as String? ?? '') ?? DateTime.fromMillisecondsSinceEpoch(0),
        user: OwnerUser.fromJson(json['user'] as Map<String, dynamic>? ?? const {}));
}
