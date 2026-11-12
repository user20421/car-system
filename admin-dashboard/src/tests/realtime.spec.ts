import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({ createWSTicket: vi.fn() }))
vi.mock('../api', () => ({ api: mocks, useMockApi: false }))
import { RealtimeClient } from '../services/realtime'

class FakeWebSocket {
  static instances: FakeWebSocket[] = []
  readonly url: string
  readonly sent: string[] = []
  readyState = 0
  onopen: ((event: Event) => void) | null = null
  onmessage: ((event: { data: string }) => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null
  constructor(url: string | URL) { this.url = String(url); FakeWebSocket.instances.push(this) }
  send(data: string) { this.sent.push(data) }
  close() { this.readyState = 3; this.onclose?.() }
}

describe('realtime WSS contract', () => {
  beforeEach(() => {
    FakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', FakeWebSocket)
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: true })
    mocks.createWSTicket.mockResolvedValue({
      ticket: 'single-use-ticket', expires_at_ms: Date.now() + 60_000,
      websocket_url: 'wss://localhost:8443/api/v1/ws',
    })
  })
  afterEach(() => vi.unstubAllGlobals())

  it('uses the ticket endpoint without inventing subscription frames', async () => {
    const client = new RealtimeClient()
    const telemetry = vi.fn()
    client.on('telemetry', telemetry)
    client.subscribe(['LTEST000000000001'])
    await client.start()
    const socket = FakeWebSocket.instances[0]
    const url = new URL(socket.url)
    expect(url.pathname).toBe('/api/v1/ws')
    expect(url.searchParams.get('ticket')).toBe('single-use-ticket')
    expect(url.host).toBe(window.location.host)

    socket.readyState = 1
    socket.onopen?.(new Event('open'))
    expect(socket.sent).toEqual([])
    expect(client.state).toBe('live')

    socket.onmessage?.({ data: JSON.stringify({ type: 'telemetry', data: { vin: 'LTEST000000000001' } }) })
    expect(telemetry).toHaveBeenCalledWith({ vin: 'LTEST000000000001' })
    client.stop()
  })
})
