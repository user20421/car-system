enum CommandStatus { pending, sent, received, succeeded, failed, expired }
class CommandFailure {
  const CommandFailure({required this.code, required this.message, required this.retryable, this.details = const {}});
  final String code;
  final String message;
  final bool retryable;
  final Map<String, String> details;
  factory CommandFailure.fromJson(Map<String, dynamic> json) => CommandFailure(code: json['code'] as String? ?? 'UNKNOWN',
        message: json['message'] as String? ?? '车辆未能执行指令', retryable: json['retryable'] as bool? ?? false,
        details: json['details'] is Map
            ? (json['details'] as Map).map((key, value) => MapEntry('$key', '$value'))
            : const <String, String>{});
}

class VehicleCommand {
  const VehicleCommand({required this.id, required this.vin, required this.requestedBy, required this.idempotencyKey,
    required this.expiresAt, required this.type, required this.params, required this.status, required this.createdAt,
    required this.updatedAt, this.error});
  final String id;
  final String vin;
  final String requestedBy;
  final String idempotencyKey;
  final DateTime expiresAt;
  final String type;
  final Map<String, String> params;
  final CommandStatus status;
  final DateTime createdAt;
  final DateTime updatedAt;
  final CommandFailure? error;
  bool get isTerminal => status == CommandStatus.succeeded || status == CommandStatus.failed || status == CommandStatus.expired;
  VehicleCommand copyWith({CommandStatus? status, DateTime? updatedAt, CommandFailure? error}) => VehicleCommand(
        id: id, vin: vin, requestedBy: requestedBy, idempotencyKey: idempotencyKey, expiresAt: expiresAt, type: type,
        params: params, status: status ?? this.status, createdAt: createdAt, updatedAt: updatedAt ?? this.updatedAt,
        error: error ?? this.error);
  factory VehicleCommand.fromJson(Map<String, dynamic> json) => VehicleCommand(
        id: json['command_id'] as String? ?? '', vin: json['vin'] as String? ?? '', requestedBy: json['requested_by'] as String? ?? '',
        idempotencyKey: json['idempotency_key'] as String? ?? '',
        expiresAt: DateTime.fromMillisecondsSinceEpoch(json['expires_at_ms'] as int? ?? 0, isUtc: true),
        type: json['type'] as String? ?? '',
        params: json['params'] is Map
            ? (json['params'] as Map).map((key, value) => MapEntry('$key', '$value'))
            : const <String, String>{},
        status: CommandStatus.values.firstWhere((status) => status.name == json['status'], orElse: () => CommandStatus.failed),
        createdAt: DateTime.fromMillisecondsSinceEpoch(json['created_at_ms'] as int? ?? 0, isUtc: true),
        updatedAt: DateTime.fromMillisecondsSinceEpoch(json['updated_at_ms'] as int? ?? 0, isUtc: true),
        error: json['error'] is Map
            ? CommandFailure.fromJson(Map<String, dynamic>.from(json['error'] as Map))
            : null);
}

class CommandStateMachine {
  const CommandStateMachine();
  VehicleCommand reduce(VehicleCommand current, VehicleCommand incoming) {
    if (current.id != incoming.id || current.vin != incoming.vin || current.isTerminal) return current;
    if (incoming.status == CommandStatus.expired || incoming.status == CommandStatus.succeeded || incoming.status == CommandStatus.failed) return incoming;
    const order = [CommandStatus.pending, CommandStatus.sent, CommandStatus.received];
    return order.indexOf(incoming.status) >= order.indexOf(current.status) ? incoming : current;
  }
}
