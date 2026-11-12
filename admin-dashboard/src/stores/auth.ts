import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { api } from '../api'
import { http } from '../api/http'
import type { TokenResponse, User } from '../types/api'
import { hasPermission, type Permission } from '../utils/permissions'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const accessToken = ref<string | null>(null)
  const refreshToken = ref<string | null>(null)
  const initialized = ref(false)
  const loading = ref(false)
  const error = ref('')
  const isAuthenticated = computed(() => Boolean(accessToken.value && user.value))
  const isAdmin = computed(() => Boolean(user.value?.roles.includes('admin')))
  const can = computed(() => (permission: Permission) => hasPermission(user.value, permission))

  function applySession(session: TokenResponse) {
    accessToken.value = session.access_token
    refreshToken.value = session.refresh_token || refreshToken.value
    user.value = session.user
    http.setAccessToken(session.access_token)
    error.value = ''
  }
  function clearSession() {
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    http.setAccessToken(null)
  }
  async function login(username: string, password: string) {
    loading.value = true
    error.value = ''
    try { applySession(await api.login(username, password)) }
    catch (cause) { error.value = cause instanceof Error ? cause.message : '登录失败'; throw cause }
    finally { loading.value = false; initialized.value = true }
  }
  async function refresh(): Promise<string | null> {
    try {
      // Native-style refresh tokens stay in memory. An empty body also lets the
      // browser restore an HttpOnly Secure cookie session after a full reload.
      applySession(await api.refresh(refreshToken.value || ''))
      return accessToken.value
    } catch { clearSession(); return null }
  }
  async function initialize() {
    if (initialized.value) return
    await refresh()
    initialized.value = true
  }
  async function logout() {
    const token = refreshToken.value || ''
    try { if (accessToken.value) await api.logout(token) } finally { clearSession() }
  }
  return { user, accessToken, refreshToken, initialized, loading, error, isAuthenticated, isAdmin, can, login, logout, refresh, initialize, clearSession }
})

export function configureHttpAuth() {
  const auth = useAuthStore()
  http.configureAuth(() => auth.refresh(), () => auth.clearSession())
}
