export type Role = 'user' | 'admin'
export type AccessRole = 'owner' | 'driver' | 'viewer'
export type Severity = 'info' | 'warning' | 'critical'
export type CommandStatus = 'pending' | 'sent' | 'received' | 'succeeded' | 'failed' | 'expired'
export type CommandType = 'lock' | 'unlock' | 'hvac_on' | 'hvac_off' | 'set_temperature' | 'defrost_on' | 'defrost_off' | 'honk_flash'

export interface User {
  id: string
  username: string
  display_name: string
  email?: string
  roles: Role[]
  created_at: string
  status?: 'active' | 'invited' | 'suspended'
  last_active_at?: string
}

export interface Vehicle {
  vin: string
  display_name: string
  model: string
  model_year?: number
  owner_user_id: string
  access_role: AccessRole
  firmware_version?: string
  created_at: string
  color?: string
  license_plate?: string
}

export interface Location {
  latitude: number
  longitude: number
  altitude_m?: number
  heading_deg?: number
  horizontal_accuracy_m?: number
}

export interface Fault {
  code: string
  severity: Severity
  message?: string
  active: boolean
}

export interface Telemetry {
  vin: string
  device_boot_id: string
  sequence: number
  occurred_at_ms: number
  received_at_ms?: number
  speed_kph: number
  rpm: number
  battery_percent: number
  range_km: number
  location?: Location
  locked: boolean
  hvac?: { mode: 'off' | 'auto' | 'cool' | 'heat'; target_temp_c?: number; fan_level?: number; ac_on?: boolean; recirculation?: boolean }
  cabin_temp_c?: number
  defrost: boolean
  fault: Fault[]
}

export interface PageMeta { number: number; size: number; total_items: number; total_pages: number }
export interface VehicleList { items: Vehicle[]; page: PageMeta }
export interface TelemetryHistory { items: Telemetry[]; next_cursor: string | null }

export interface TokenResponse {
  access_token: string
  refresh_token: string
  token_type: 'Bearer'
  expires_in: number
  user: User
}

export interface CommandError { code: string; message: string; retryable: boolean; details?: Record<string, string> }
export interface VehicleCommand {
  command_id: string
  vin: string
  requested_by: string
  idempotency_key: string
  expires_at_ms: number
  type: CommandType
  params: Record<string, string>
  status: CommandStatus
  error?: CommandError | null
  created_at_ms: number
  updated_at_ms: number
}
export interface CreateCommandInput {
  type: CommandType
  params: Record<string, string>
  idempotency_key: string
  expires_at_ms: number
}

export interface Alert {
  id: string
  vin: string
  code: string
  title: string
  severity: Severity
  status: 'open' | 'acknowledged'
  occurred_at_ms: number
  acknowledged_by?: string | null
  acknowledged_at_ms?: number | null
  acknowledgement_note?: string | null
}
export interface AlertList { items: Alert[]; page: PageMeta }

export interface WSTicket { ticket: string; expires_at_ms: number; websocket_url: string }
export interface Problem {
  type: string
  title: string
  status: number
  detail?: string
  instance?: string
  code: string
  request_id: string
  errors?: { field: string; message: string }[]
}

export interface AuditEvent {
  id: string
  actor: string
  action: string
  target: string
  result: 'success' | 'denied' | 'failed'
  request_id: string
  occurred_at_ms: number
  ip?: string
  detail?: string
}

export interface DeviceInfo {
  id: string
  vin: string
  serial_number: string
  hardware: string
  firmware_version: string
  status: 'online' | 'offline' | 'provisioning'
  last_seen_at_ms: number
  certificate_expires_at: string
}

export interface ApiAdapter {
  login(username: string, password: string): Promise<TokenResponse>
  refresh(refreshToken: string): Promise<TokenResponse>
  logout(refreshToken: string): Promise<void>
  me(): Promise<User>
  listVehicles(page?: number, pageSize?: number): Promise<VehicleList>
  getVehicle(vin: string): Promise<Vehicle>
  latestTelemetry(vin: string): Promise<Telemetry>
  telemetryHistory(vin: string, fromMs: number, toMs: number): Promise<TelemetryHistory>
  createCommand(vin: string, input: CreateCommandInput): Promise<VehicleCommand>
  getCommand(vin: string, commandId: string): Promise<VehicleCommand>
  listAlerts(params?: { vin?: string; status?: string; page?: number }): Promise<AlertList>
  acknowledgeAlert(id: string, note: string): Promise<Alert>
  createWSTicket(): Promise<WSTicket>
  listUsers(): Promise<User[]>
  listDevices(): Promise<DeviceInfo[]>
  listAuditEvents(): Promise<AuditEvent[]>
  createUser(input: { username: string; display_name: string; email: string; password: string; roles: Role[] }): Promise<User>
  updateUserStatus(id: string, status: NonNullable<User['status']>): Promise<User>
  provisionDevice(input: { vin: string; serial_number: string }): Promise<DeviceInfo>
  updateVehicle(vin: string, displayName: string): Promise<Vehicle>
}
