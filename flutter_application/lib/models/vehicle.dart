enum VehicleRole { owner, driver, viewer }

class Vehicle {
  const Vehicle({required this.vin, required this.displayName, required this.model, required this.ownerUserId,
    required this.role, required this.createdAt, this.modelYear, this.firmwareVersion});
  final String vin;
  final String displayName;
  final String model;
  final String ownerUserId;
  final VehicleRole role;
  final DateTime createdAt;
  final int? modelYear;
  final String? firmwareVersion;
  bool get canControl => role != VehicleRole.viewer;

  factory Vehicle.fromJson(Map<String, dynamic> json) => Vehicle(
        vin: json['vin'] as String? ?? '', displayName: json['display_name'] as String? ?? '',
        model: json['model'] as String? ?? '', ownerUserId: json['owner_user_id'] as String? ?? '',
        role: VehicleRole.values.firstWhere((role) => role.name == json['access_role'], orElse: () => VehicleRole.viewer),
        createdAt: DateTime.tryParse(json['created_at'] as String? ?? '') ?? DateTime.fromMillisecondsSinceEpoch(0),
        modelYear: json['model_year'] as int?, firmwareVersion: json['firmware_version'] as String?);
  Map<String, dynamic> toJson() => {'vin': vin, 'display_name': displayName, 'model': model,
        'owner_user_id': ownerUserId, 'access_role': role.name, 'created_at': createdAt.toIso8601String(),
        if (modelYear != null) 'model_year': modelYear, if (firmwareVersion != null) 'firmware_version': firmwareVersion};
}

class PageMeta {
  const PageMeta({required this.number, required this.size, required this.totalItems, required this.totalPages});
  final int number;
  final int size;
  final int totalItems;
  final int totalPages;
  factory PageMeta.fromJson(Map<String, dynamic> json) => PageMeta(number: json['number'] as int? ?? 1,
        size: json['size'] as int? ?? 20, totalItems: json['total_items'] as int? ?? 0, totalPages: json['total_pages'] as int? ?? 0);
}
