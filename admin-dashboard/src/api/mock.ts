import type { ApiAdapter, Alert, AlertList, AuditEvent, CreateCommandInput, DeviceInfo, Telemetry, TelemetryHistory, TokenResponse, User, Vehicle, VehicleCommand, VehicleList, WSTicket } from '../types/api'

const wait = (ms = 180) => new Promise(resolve => setTimeout(resolve, ms))
const now = () => Date.now()

const users: User[] = [
  { id: 'u-001', username: 'aurora.admin', display_name: '林屿', email: 'lin.yu@aurora.auto', roles: ['admin'], created_at: '2025-01-08T08:00:00Z', status: 'active', last_active_at: '2026-11-12T10:46:00Z' },
  { id: 'u-002', username: 'ops.chen', display_name: '陈序', email: 'chen.xu@aurora.auto', roles: ['admin'], created_at: '2025-04-16T08:00:00Z', status: 'active', last_active_at: '2026-11-12T09:18:00Z' },
  { id: 'u-003', username: 'driver.song', display_name: '宋遥', email: 'song.yao@aurora.auto', roles: ['user'], created_at: '2025-08-21T08:00:00Z', status: 'active', last_active_at: '2026-11-11T15:30:00Z' },
  { id: 'u-004', username: 'viewer.zhou', display_name: '周岚', email: 'zhou.lan@aurora.auto', roles: ['user'], created_at: '2026-02-10T08:00:00Z', status: 'invited' },
  { id: 'u-005', username: 'service.gu', display_name: '顾川', email: 'gu.chuan@aurora.auto', roles: ['user'], created_at: '2026-06-18T08:00:00Z', status: 'suspended' },
]

const vehicles: Vehicle[] = [
  { vin: 'LAURORA0000000001', display_name: 'AURORA 01', model: 'A9 曜境版', model_year: 2026, owner_user_id: 'u-003', access_role: 'owner', firmware_version: '4.8.2', created_at: '2025-11-20T08:00:00Z', license_plate: '沪A·D80216', color: '#7971f8' },
  { vin: 'LAURORA0000000002', display_name: '星轨 07', model: 'A7 长续航版', model_year: 2026, owner_user_id: 'u-003', access_role: 'driver', firmware_version: '4.8.2', created_at: '2025-12-12T08:00:00Z', license_plate: '沪A·D67702', color: '#3cb9c7' },
  { vin: 'LAURORA0000000003', display_name: '城市漫游者', model: 'A7 四驱版', model_year: 2025, owner_user_id: 'u-004', access_role: 'owner', firmware_version: '4.7.9', created_at: '2025-09-08T08:00:00Z', license_plate: '苏E·D10983', color: '#e7a865' },
  { vin: 'LAURORA0000000004', display_name: '远山 12', model: 'A9 曜境版', model_year: 2026, owner_user_id: 'u-005', access_role: 'viewer', firmware_version: '4.8.1', created_at: '2026-01-10T08:00:00Z', license_plate: '浙A·D56219', color: '#8b93aa' },
  { vin: 'LAURORA0000000005', display_name: '青屿 03', model: 'A5 都市版', model_year: 2025, owner_user_id: 'u-003', access_role: 'owner', firmware_version: '4.7.9', created_at: '2025-07-18T08:00:00Z', license_plate: '沪C·D32381', color: '#5b91df' },
  { vin: 'LAURORA0000000006', display_name: '极光 21', model: 'A9 曜境版', model_year: 2026, owner_user_id: 'u-004', access_role: 'driver', firmware_version: '4.8.2', created_at: '2026-03-09T08:00:00Z', license_plate: '沪A·D95270', color: '#bc78cf' },
]

const locations = [
  [31.2304, 121.4737, 68], [31.222, 121.458, 42], [31.241, 121.501, 0],
  [31.205, 121.435, 0], [31.252, 121.48, 26], [31.218, 121.521, 105],
]

function telemetry(vin: string, at = now()): Telemetry {
  const index = Math.max(0, vehicles.findIndex(v => v.vin === vin))
  const moving = index !== 2 && index !== 3
  const phase = at / 45000 + index
  return {
    vin, device_boot_id: `boot-${index + 1}-a91`, sequence: Math.floor(at / 5000), occurred_at_ms: index === 3 ? at - 28 * 60_000 : at - index * 1400,
    received_at_ms: at, speed_kph: moving ? Math.max(0, locations[index][2] + Math.sin(phase) * 12) : 0,
    rpm: moving ? 950 + Math.round(Math.sin(phase) * 220) : 0,
    battery_percent: Math.round((82 - index * 7 + Math.sin(phase) * .3) * 10) / 10,
    range_km: Math.round(486 - index * 39), locked: !moving,
    location: { latitude: locations[index][0] + Math.sin(phase) * .0004, longitude: locations[index][1] + Math.cos(phase) * .0005, heading_deg: locations[index][2], horizontal_accuracy_m: 4.2 },
    hvac: { mode: index % 2 ? 'cool' : 'auto', target_temp_c: 22, fan_level: 2, ac_on: true, recirculation: false },
    cabin_temp_c: 22.4 + index * .3, defrost: false,
    fault: index === 2 ? [{ code: 'TPMS_RL_LOW', severity: 'warning', message: '左后轮胎压偏低', active: true }] : [],
  }
}

const alerts: Alert[] = [
  { id: 'a-001', vin: vehicles[2].vin, code: 'TPMS_RL_LOW', title: '左后轮胎压低于安全阈值', severity: 'warning', status: 'open', occurred_at_ms: now() - 8 * 60_000 },
  { id: 'a-002', vin: vehicles[3].vin, code: 'VEHICLE_OFFLINE', title: '车辆离线超过 20 分钟', severity: 'critical', status: 'open', occurred_at_ms: now() - 26 * 60_000 },
  { id: 'a-003', vin: vehicles[0].vin, code: 'GEOFENCE_EXIT', title: '车辆驶出「徐汇运营区」', severity: 'info', status: 'open', occurred_at_ms: now() - 42 * 60_000 },
  { id: 'a-004', vin: vehicles[4].vin, code: 'BATTERY_LOW', title: '动力电池电量低于 30%', severity: 'warning', status: 'acknowledged', occurred_at_ms: now() - 2 * 3600_000, acknowledged_by: 'u-001', acknowledged_at_ms: now() - 90 * 60_000, acknowledgement_note: '已通知车主前往附近充电站。' },
  { id: 'a-005', vin: vehicles[1].vin, code: 'SOS_TEST', title: '紧急呼叫自检完成', severity: 'info', status: 'acknowledged', occurred_at_ms: now() - 5 * 3600_000, acknowledged_by: 'u-002', acknowledged_at_ms: now() - 4 * 3600_000 },
]

const commands = new Map<string, VehicleCommand>()
const provisionedDevices: DeviceInfo[] = []
const audit: AuditEvent[] = [
  { id: 'evt-1042', actor: '林屿', action: 'alert.acknowledge', target: 'ALERT a-004', result: 'success', request_id: '01JCE8N7A1M4K8', occurred_at_ms: now() - 12 * 60_000, ip: '10.24.8.16', detail: '确认低电量告警' },
  { id: 'evt-1041', actor: '陈序', action: 'command.create', target: vehicles[0].vin, result: 'success', request_id: '01JCE8K2F7P2X9', occurred_at_ms: now() - 31 * 60_000, ip: '10.24.8.21', detail: '创建 lock 指令' },
  { id: 'evt-1040', actor: '宋遥', action: 'vehicle.control', target: vehicles[3].vin, result: 'denied', request_id: '01JCE8G9C3D1Q7', occurred_at_ms: now() - 64 * 60_000, ip: '10.24.12.8', detail: 'viewer 角色无控制权限' },
  { id: 'evt-1039', actor: '林屿', action: 'auth.login', target: 'aurora.admin', result: 'success', request_id: '01JCE8B4R2M8H4', occurred_at_ms: now() - 2 * 3600_000, ip: '10.24.8.16' },
  { id: 'evt-1038', actor: '系统', action: 'device.provision', target: vehicles[5].vin, result: 'success', request_id: '01JCE87A6N4V2P', occurred_at_ms: now() - 4 * 3600_000, detail: '设备证书签发完成' },
]

export class MockApiAdapter implements ApiAdapter {
  async login(username: string, password: string): Promise<TokenResponse> {
    await wait(500)
    if (!username || password.length < 8) throw new Error('请输入有效账号与至少 8 位密码')
    const user = username.includes('viewer') ? { ...users[3], roles: ['user'] as ['user'] } : users[0]
    return { access_token: `demo-access-${now()}`, refresh_token: `demo-refresh-${now()}`, token_type: 'Bearer', expires_in: 900, user }
  }
  async refresh(_refreshToken: string): Promise<TokenResponse> {
    await wait()
    return { access_token: `demo-access-${now()}`, refresh_token: `demo-refresh-${now()}`, token_type: 'Bearer', expires_in: 900, user: users[0] }
  }
  async logout() { await wait(80) }
  async me() { await wait(); return users[0] }
  async listVehicles(page = 1, pageSize = 100): Promise<VehicleList> {
    await wait(420)
    return { items: vehicles.slice((page - 1) * pageSize, page * pageSize), page: { number: page, size: pageSize, total_items: vehicles.length, total_pages: Math.ceil(vehicles.length / pageSize) } }
  }
  async getVehicle(vin: string) { await wait(); const vehicle = vehicles.find(v => v.vin === vin); if (!vehicle) throw new Error('车辆不存在或无权访问'); return vehicle }
  async latestTelemetry(vin: string) { await wait(90); return telemetry(vin) }
  async telemetryHistory(vin: string, fromMs: number, toMs: number): Promise<TelemetryHistory> {
    await wait(350)
    const count = 48
    return { items: Array.from({ length: count }, (_, i) => telemetry(vin, fromMs + (toMs - fromMs) * i / (count - 1))), next_cursor: null }
  }
  async createCommand(vin: string, input: CreateCommandInput): Promise<VehicleCommand> {
    await wait(350)
    const command: VehicleCommand = { command_id: crypto.randomUUID(), vin, requested_by: 'u-001', ...input, status: 'pending', created_at_ms: now(), updated_at_ms: now() }
    commands.set(command.command_id, command)
    return structuredClone(command)
  }
  async getCommand(_vin: string, commandId: string) {
    await wait(100)
    const command = commands.get(commandId)
    if (!command) throw new Error('指令不存在')
    const elapsed = now() - command.created_at_ms
    command.status = elapsed < 700 ? 'pending' : elapsed < 1500 ? 'sent' : elapsed < 2600 ? 'received' : command.expires_at_ms < now() ? 'expired' : 'succeeded'
    command.updated_at_ms = now()
    return structuredClone(command)
  }
  async listAlerts(params: { vin?: string; status?: string; page?: number } = {}): Promise<AlertList> {
    await wait(300)
    const items = alerts.filter(a => (!params.vin || a.vin === params.vin) && (!params.status || a.status === params.status))
    return { items, page: { number: 1, size: 100, total_items: items.length, total_pages: 1 } }
  }
  async acknowledgeAlert(id: string, note: string) { await wait(350); const alert = alerts.find(a => a.id === id); if (!alert) throw new Error('告警不存在'); Object.assign(alert, { status: 'acknowledged', acknowledged_by: 'u-001', acknowledged_at_ms: now(), acknowledgement_note: note }); return { ...alert } }
  async createWSTicket(): Promise<WSTicket> { return { ticket: 'demo-only-no-network-ticket-000000', expires_at_ms: now() + 30_000, websocket_url: 'wss://localhost:8443/api/v1/ws' } }
  async listUsers() { await wait(320); return structuredClone(users) }
  async listDevices(): Promise<DeviceInfo[]> { await wait(320); return [...provisionedDevices, ...vehicles.map((v, i) => ({ id: `dev-${i + 1}`, vin: v.vin, serial_number: `AUR-TCU-${String(2801 + i)}`, hardware: i % 2 ? 'TCU-Gen3 / Rev.B' : 'TCU-Gen3 / Rev.A', firmware_version: v.firmware_version || '4.8.0', status: (i === 3 ? 'offline' : 'online') as 'offline' | 'online', last_seen_at_ms: now() - i * 1400, certificate_expires_at: `202${7 + i % 2}-11-12T00:00:00Z` }))] }
  async listAuditEvents() { await wait(350); return structuredClone(audit) }
  async createUser(input: { username: string; display_name: string; email: string; password: string; roles: ('user' | 'admin')[] }) {
    await wait(400)
    const user: User = { id: `u-${String(users.length + 1).padStart(3, '0')}`, ...input, status: 'invited', created_at: new Date().toISOString() }
    users.push(user)
    return { ...user }
  }
  async updateUserStatus(id: string, status: NonNullable<User['status']>) { await wait(); const user = users.find(u => u.id === id); if (!user) throw new Error('用户不存在'); user.status = status; return { ...user } }
  async provisionDevice(input: { vin: string; serial_number: string }) {
    await wait(400)
    const device: DeviceInfo = { id: `dev-${Date.now()}`, ...input, hardware: 'TCU-Gen3 / Rev.B', firmware_version: '4.8.2', status: 'provisioning', last_seen_at_ms: now(), certificate_expires_at: '2028-11-12T00:00:00Z' }
    provisionedDevices.push(device)
    return { ...device }
  }
  async updateVehicle(vin: string, displayName: string) { await wait(); const vehicle = vehicles.find(v => v.vin === vin); if (!vehicle) throw new Error('车辆不存在'); vehicle.display_name = displayName; return { ...vehicle } }
}
