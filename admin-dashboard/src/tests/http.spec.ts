import { describe, expect, it, vi } from 'vitest'
import { HttpClient } from '../api/http'
import { ApiError } from '../api/errors'

const json = (value: unknown, status = 200, requestId = 'req-test') => new Response(JSON.stringify(value), { status, headers: { 'Content-Type': 'application/json', 'X-Request-ID': requestId } })

describe('HTTP client', () => {
  it('shares one refresh across concurrent 401 responses and retries with new access token', async () => {
    const client = new HttpClient('/api/v1')
    client.setAccessToken('expired')
    const refresh = vi.fn().mockImplementation(async () => { client.setAccessToken('fresh'); return 'fresh' })
    client.configureAuth(refresh, vi.fn())
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (_input, init) => {
      const headers = new Headers(init?.headers)
      if (headers.get('Authorization') === 'Bearer expired') return json({ title: 'Expired', status: 401, code: 'UNAUTHORIZED', type: 'about:blank', request_id: 'r1' }, 401)
      return json({ ok: true })
    })
    const [a, b] = await Promise.all([client.request('/vehicles'), client.request('/alerts')])
    expect(a).toEqual({ ok: true }); expect(b).toEqual({ ok: true })
    expect(refresh).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledTimes(4)
  })
  it('preserves Problem Details and request id', async () => {
    const client = new HttpClient('/api/v1')
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(json({ type: 'about:blank', title: 'Forbidden', detail: 'Vehicle role denied', status: 403, code: 'FORBIDDEN', request_id: 'req-403' }, 403))
    const error: ApiError = await client.request<ApiError>('/vehicles/vin').catch(cause => cause as ApiError)
    expect(error).toBeInstanceOf(ApiError)
    expect(error.status).toBe(403); expect(error.requestId).toBe('req-403'); expect(error.message).toBe('Vehicle role denied')
  })
})
