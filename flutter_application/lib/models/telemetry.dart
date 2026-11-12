class HvacState {
  const HvacState({this.mode = 'off', this.targetTempC = 22, this.fanLevel = 0, this.acOn = false, this.recirculation = false});
  final String mode;
  final double targetTempC;
  final int fanLevel;
  final bool acOn;
  final bool recirculation;
  factory HvacState.fromJson(Map<String, dynamic> json) => HvacState(mode: json['mode'] as String? ?? 'off',
        targetTempC: (json['target_temp_c'] as num? ?? 22).toDouble(), fanLevel: json['fan_level'] as int? ?? 0,
        acOn: json['ac_on'] as bool? ?? false, recirculation: json['recirculation'] as bool? ?? false);
}

class VehicleLocation {
  const VehicleLocation({required this.latitude, required this.longitude, this.heading});
  final double latitude;
  final double longitude;
  final double? heading;
  factory VehicleLocation.fromJson(Map<String, dynamic> json) => VehicleLocation(
        latitude: (json['latitude'] as num? ?? 0).toDouble(), longitude: (json['longitude'] as num? ?? 0).toDouble(),
        heading: (json['heading_deg'] as num?)?.toDouble());
}

class VehicleFault {
  const VehicleFault({required this.code, required this.severity, required this.active, this.message});
  final String code;
  final String severity;
  final bool active;
  final String? message;
  factory VehicleFault.fromJson(Map<String, dynamic> json) => VehicleFault(code: json['code'] as String? ?? '',
        severity: json['severity'] as String? ?? 'info', active: json['active'] as bool? ?? false, message: json['message'] as String?);
}

class Telemetry {
  const Telemetry({required this.vin, required this.deviceBootId, required this.sequence, required this.occurredAt,
    required this.speedKph, required this.rpm, required this.batteryPercent, required this.rangeKm, required this.locked,
    required this.defrost, required this.faults, this.receivedAt, this.location, this.hvac = const HvacState(), this.cabinTempC});
  final String vin;
  final String deviceBootId;
  final int sequence;
  final DateTime occurredAt;
  final DateTime? receivedAt;
  final double speedKph;
  final int rpm;
  final double batteryPercent;
  final double rangeKm;
  final bool locked;
  final HvacState hvac;
  final double? cabinTempC;
  final bool defrost;
  final VehicleLocation? location;
  final List<VehicleFault> faults;
  bool get isOnline => DateTime.now().difference(occurredAt).inMinutes < 5;
  Telemetry copyWith({DateTime? occurredAt, double? batteryPercent, double? rangeKm, bool? locked,
      HvacState? hvac, double? cabinTempC, bool? defrost, int? sequence}) => Telemetry(
        vin: vin, deviceBootId: deviceBootId, sequence: sequence ?? this.sequence + 1,
        occurredAt: occurredAt ?? DateTime.now().toUtc(), receivedAt: DateTime.now().toUtc(),
        speedKph: speedKph, rpm: rpm, batteryPercent: batteryPercent ?? this.batteryPercent,
        rangeKm: rangeKm ?? this.rangeKm, locked: locked ?? this.locked, defrost: defrost ?? this.defrost,
        hvac: hvac ?? this.hvac, cabinTempC: cabinTempC ?? this.cabinTempC, location: location, faults: faults);

  factory Telemetry.fromJson(Map<String, dynamic> json) => Telemetry(vin: json['vin'] as String? ?? '',
        deviceBootId: json['device_boot_id'] as String? ?? '', sequence: json['sequence'] as int? ?? 0,
        occurredAt: DateTime.fromMillisecondsSinceEpoch(json['occurred_at_ms'] as int? ?? 0, isUtc: true),
        receivedAt: json['received_at_ms'] == null ? null : DateTime.fromMillisecondsSinceEpoch(json['received_at_ms'] as int, isUtc: true),
        speedKph: (json['speed_kph'] as num? ?? 0).toDouble(), rpm: json['rpm'] as int? ?? 0,
        batteryPercent: (json['battery_percent'] as num? ?? 0).toDouble(), rangeKm: (json['range_km'] as num? ?? 0).toDouble(),
        locked: json['locked'] as bool? ?? true, defrost: json['defrost'] as bool? ?? false,
        hvac: HvacState.fromJson(json['hvac'] as Map<String, dynamic>? ?? const {}),
        cabinTempC: (json['cabin_temp_c'] as num?)?.toDouble(),
        location: json['location'] == null ? null : VehicleLocation.fromJson(json['location'] as Map<String, dynamic>),
        faults: (json['fault'] as List? ?? const []).map((e) => VehicleFault.fromJson(e as Map<String, dynamic>)).toList());
}
