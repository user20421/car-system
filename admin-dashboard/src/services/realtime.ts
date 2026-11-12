import { api, useMockApi } from '../api'
import type { Alert, CommandStatus, Telemetry } from '../types/api'

export type RealtimeState = 'idle' | 'connecting' | 'live' | 'reconnecting' | 'offline' | 'closed'
type Events = {
  telemetry: (value: Telemetry) => void
  command: (value: { command_id: string; vin: string; status: CommandStatus; updated_at_ms: number }) => void
  alert: (value: Alert) => void
  state: (value: RealtimeState) => void
  snapshot: (vins: string[]) => void
}

export class RealtimeClient {
  private socket: WebSocket | null = null
  private listeners: { [K in keyof Events]: Set<Events[K]> } = { telemetry: new Set(), command: new Set(), alert: new Set(), state: new Set(), snapshot: new Set() }
  private vins = new Set<string>()
  private attempts = 0
  private stopped = true
  private reconnectTimer?: ReturnType<typeof setTimeout>
  private mockTimer?: ReturnType<typeof setInterval>
  state: RealtimeState = 'idle'

  on<K extends keyof Events>(event: K, listener: Events[K]) {
    this.listeners[event].add(listener)
    return () => { this.listeners[event].delete(listener) }
  }
  private emit<K extends keyof Events>(event: K, value: Parameters<Events[K]>[0]) {
    this.listeners[event].forEach(listener => { (listener as (arg: typeof value) => void)(value) })
  }
  private setState(state: RealtimeState) { this.state = state; this.emit('state', state) }

  subscribe(vins: string[]) {
    // The server pushes every VIN authorized for the ticket owner; no client
    // subscription frames are part of the current OpenAPI/WSS contract.
    vins.forEach(vin => this.vins.add(vin))
  }
  unsubscribe(vin: string) { this.vins.delete(vin) }

  async start() {
    this.stopped = false
    if (useMockApi) { this.startMock(); return }
    await this.connect()
  }
  private async connect() {
    if (this.stopped || !navigator.onLine) { this.setState('offline'); this.emit('snapshot', [...this.vins]); return }
    this.setState(this.attempts ? 'reconnecting' : 'connecting')
    try {
      const ticket = await api.createWSTicket()
      if (this.stopped) return
      const url = new URL(ticket.websocket_url)
      if (import.meta.env.DEV) {
        // Vite proxies /api (including Upgrade) so the backend Origin check
        // sees the same host during local development.
        url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        url.host = window.location.host
      }
      url.searchParams.set('ticket', ticket.ticket)
      const socket = new WebSocket(url)
      this.socket = socket
      socket.onopen = () => { this.attempts = 0; this.setState('live') }
      socket.onmessage = event => this.handleMessage(event.data)
      socket.onclose = () => { this.socket = null; this.emit('snapshot', [...this.vins]); this.scheduleReconnect() }
      socket.onerror = () => socket.close()
    } catch { this.scheduleReconnect() }
  }
  private scheduleReconnect() {
    if (this.stopped) return
    this.setState(navigator.onLine ? 'reconnecting' : 'offline')
    const delay = Math.min(30_000, 1000 * 2 ** this.attempts) + Math.random() * 400
    this.attempts += 1
    clearTimeout(this.reconnectTimer)
    this.reconnectTimer = setTimeout(() => void this.connect(), delay)
  }
  private handleMessage(raw: string) {
    try {
      const message = JSON.parse(raw) as { type: string; data?: unknown; payload?: unknown }
      const data = message.data ?? message.payload
      if (message.type === 'telemetry') this.emit('telemetry', data as Telemetry)
      if (message.type === 'command' || message.type === 'command.updated') this.emit('command', data as Parameters<Events['command']>[0])
      if (message.type === 'alert' || message.type === 'alert.created') this.emit('alert', data as Alert)
    } catch { /* Ignore malformed realtime frames; HTTPS remains authoritative. */ }
  }
  private startMock() {
    this.setState('live')
    const tick = async () => {
      for (const vin of this.vins) {
        try { this.emit('telemetry', await api.latestTelemetry(vin)) } catch { /* retain last known snapshot */ }
      }
    }
    void tick()
    this.mockTimer = setInterval(() => void tick(), 5000)
  }
  stop() {
    this.stopped = true
    clearTimeout(this.reconnectTimer)
    clearInterval(this.mockTimer)
    this.socket?.close()
    this.socket = null
    this.setState('closed')
  }
}
