class VehicleAlert {
  const VehicleAlert({required this.id, required this.vin, required this.code, required this.title, required this.severity,
    required this.status, required this.occurredAt, this.acknowledgedAt, this.note});
  final String id;
  final String vin;
  final String code;
  final String title;
  final String severity;
  final String status;
  final DateTime occurredAt;
  final DateTime? acknowledgedAt;
  final String? note;
  bool get isOpen => status == 'open';
  factory VehicleAlert.fromJson(Map<String, dynamic> json) => VehicleAlert(id: json['id'] as String? ?? '',
        vin: json['vin'] as String? ?? '', code: json['code'] as String? ?? '', title: json['title'] as String? ?? '',
        severity: json['severity'] as String? ?? 'info', status: json['status'] as String? ?? 'open',
        occurredAt: DateTime.fromMillisecondsSinceEpoch(json['occurred_at_ms'] as int? ?? 0, isUtc: true),
        acknowledgedAt: json['acknowledged_at_ms'] == null ? null : DateTime.fromMillisecondsSinceEpoch(json['acknowledged_at_ms'] as int, isUtc: true),
        note: json['acknowledgement_note'] as String?);
}

class WsTicket {
  const WsTicket({required this.ticket, required this.expiresAt, required this.websocketUrl});
  final String ticket;
  final DateTime expiresAt;
  final String websocketUrl;
  factory WsTicket.fromJson(Map<String, dynamic> json) => WsTicket(ticket: json['ticket'] as String? ?? '',
        expiresAt: DateTime.fromMillisecondsSinceEpoch(json['expires_at_ms'] as int? ?? 0, isUtc: true),
        websocketUrl: json['websocket_url'] as String? ?? '');
}
