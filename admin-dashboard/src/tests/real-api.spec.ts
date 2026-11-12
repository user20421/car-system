import { beforeEach, describe, expect, it, vi } from 'vitest'
import { RealApiAdapter } from '../api/real'
import { http } from '../api/http'

const json = (value: unknown, status = 200) => new Response(JSON.stringify(value), {
  status, headers: { 'Content-Type': 'application/json' },
})
const session = {
  access_token: 'access-two', refresh_token: 'refresh-two', token_type: 'Bearer', expires_in: 900,
  user: { id: 'u1', username: 'owner', display_name: 'Owner', roles: ['user'], created_at: '' },
}

describe('real API adapter contract', () => {
  beforeEach(() => http.setAccessToken('access-one'))

  it('sends refresh_token in the OpenAPI JSON request body', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(json(session))
    await new RealApiAdapter().refresh('refresh-one')
    const [input, init] = fetchMock.mock.calls[0]
    expect(new URL(String(input)).pathname).toBe('/api/v1/auth/refresh')
    expect(init?.method).toBe('POST')
    expect(JSON.parse(String(init?.body))).toEqual({ refresh_token: 'refresh-one' })
  })

  it('uses contract pagination names for vehicle lists', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(json({
      items: [], page: { number: 2, size: 25, total_items: 0, total_pages: 0 },
    }))
    await new RealApiAdapter().listVehicles(2, 25)
    const url = new URL(String(fetchMock.mock.calls[0][0]))
    expect(url.pathname).toBe('/api/v1/vehicles')
    expect(Object.fromEntries(url.searchParams)).toEqual({ page: '2', page_size: '25' })
  })

  it('requests a real single-use WSS ticket', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(json({
      ticket: 'a-ticket-that-is-long-enough', expires_at_ms: 1, websocket_url: 'wss://localhost:8443/api/v1/ws',
    }, 201))
    await new RealApiAdapter().createWSTicket()
    const [input, init] = fetchMock.mock.calls[0]
    expect(new URL(String(input)).pathname).toBe('/api/v1/ws-ticket')
    expect(init?.method).toBe('POST')
  })

  it('fails explicitly instead of calling non-contract admin endpoints', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
    await expect(new RealApiAdapter().listUsers()).rejects.toMatchObject({
      status: 501, code: 'ENDPOINT_NOT_IN_CONTRACT',
    })
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
