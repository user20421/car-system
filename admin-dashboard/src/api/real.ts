import { ApiError } from './errors'
import { http } from './http'
import type { ApiAdapter, Alert, AlertList, AuditEvent, CreateCommandInput, DeviceInfo, Telemetry, TelemetryHistory, TokenResponse, User, Vehicle, VehicleCommand, VehicleList, WSTicket } from '../types/api'

function unsupported<T>(feature: string): Promise<T> {
  return Promise.reject(new ApiError({
    type: 'about:blank', title: '当前后端暂未提供此功能', status: 501,
    detail: `${feature}不在当前 OpenAPI 契约中；可显式启用演示模式体验界面。`,
    code: 'ENDPOINT_NOT_IN_CONTRACT', request_id: 'unavailable',
  }))
}

export class RealApiAdapter implements ApiAdapter {
  login(username: string, password: string) {
    return http.request<TokenResponse>('/auth/login', { method: 'POST', body: { username, password }, auth: false })
  }
  refresh(refreshToken: string) {
    return http.request<TokenResponse>('/auth/refresh', {
      method: 'POST', auth: false, body: { refresh_token: refreshToken },
    })
  }
  async logout(refreshToken: string) {
    await http.request<void>('/auth/logout', {
      method: 'POST', retry: false,
      ...(refreshToken ? { body: { refresh_token: refreshToken } } : {}),
    })
  }
  me() { return http.request<User>('/auth/me') }
  listVehicles(page = 1, pageSize = 100) { return http.request<VehicleList>('/vehicles', { query: { page, page_size: pageSize } }) }
  getVehicle(vin: string) { return http.request<Vehicle>(`/vehicles/${vin}`) }
  latestTelemetry(vin: string) { return http.request<Telemetry>(`/vehicles/${vin}/telemetry/latest`) }
  telemetryHistory(vin: string, fromMs: number, toMs: number) {
    return http.request<TelemetryHistory>(`/vehicles/${vin}/telemetry/history`, { query: { from_ms: fromMs, to_ms: toMs, limit: 500 } })
  }
  createCommand(vin: string, input: CreateCommandInput) {
    return http.request<VehicleCommand>(`/vehicles/${vin}/commands`, { method: 'POST', body: input })
  }
  getCommand(vin: string, commandId: string) { return http.request<VehicleCommand>(`/vehicles/${vin}/commands/${commandId}`) }
  listAlerts(params: { vin?: string; status?: string; page?: number } = {}) {
    return http.request<AlertList>('/alerts', { query: { ...params, page_size: 100 } })
  }
  acknowledgeAlert(id: string, note: string) { return http.request<Alert>(`/alerts/${id}/ack`, { method: 'POST', body: { note } }) }
  createWSTicket() { return http.request<WSTicket>('/ws-ticket', { method: 'POST' }) }
  listUsers() { return unsupported<User[]>('用户目录') }
  listDevices() { return unsupported<DeviceInfo[]>('设备目录') }
  listAuditEvents() { return unsupported<AuditEvent[]>('审计事件') }
  createUser(_input: { username: string; display_name: string; email: string; password: string; roles: ('user' | 'admin')[] }) {
    return unsupported<User>('创建用户')
  }
  updateUserStatus(_id: string, _status: NonNullable<User['status']>) {
    return unsupported<User>('用户状态管理')
  }
  provisionDevice(_input: { vin: string; serial_number: string }) {
    return unsupported<DeviceInfo>('设备预配置')
  }
  updateVehicle(_vin: string, _displayName: string) {
    return unsupported<Vehicle>('车辆资料编辑')
  }
}
