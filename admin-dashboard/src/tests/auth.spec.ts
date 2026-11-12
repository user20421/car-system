import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const mocks = vi.hoisted(() => ({
  login: vi.fn(), refresh: vi.fn(), logout: vi.fn(),
}))
vi.mock('../api', () => ({ api: mocks, useMockApi: true }))
import { useAuthStore } from '../stores/auth'
import { http } from '../api/http'

const session = { access_token: 'access-one', refresh_token: 'refresh-one', token_type: 'Bearer' as const, expires_in: 900, user: { id: 'u1', username: 'operator', display_name: 'Operator', roles: ['admin' as const], created_at: '' } }

describe('auth store', () => {
  beforeEach(() => { setActivePinia(createPinia()); http.setAccessToken(null) })
  it('keeps access and refresh credentials in memory only', async () => {
    mocks.login.mockResolvedValue(session)
    const store = useAuthStore()
    await store.login('operator', 'password123')
    expect(store.isAuthenticated).toBe(true)
    expect(store.accessToken).toBe('access-one')
    expect(localStorage.getItem('access_token')).toBeNull()
    expect(localStorage.getItem('refresh_token')).toBeNull()
  })
  it('rotates both tokens and clears the session on refresh failure', async () => {
    mocks.login.mockResolvedValue(session)
    mocks.refresh.mockResolvedValue({ ...session, access_token: 'access-two', refresh_token: 'refresh-two' })
    const store = useAuthStore()
    await store.login('operator', 'password123')
    await store.refresh()
    expect(store.accessToken).toBe('access-two')
    expect(store.refreshToken).toBe('refresh-two')
    mocks.refresh.mockRejectedValue(new Error('expired'))
    await store.refresh()
    expect(store.isAuthenticated).toBe(false)
    expect(store.accessToken).toBeNull()
  })
  it('clears memory even when server logout fails', async () => {
    mocks.login.mockResolvedValue(session)
    mocks.logout.mockRejectedValue(new Error('offline'))
    const store = useAuthStore()
    await store.login('operator', 'password123')
    await expect(store.logout()).rejects.toThrow('offline')
    expect(store.isAuthenticated).toBe(false)
  })
})
