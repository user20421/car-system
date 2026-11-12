import { ApiError, asApiError, createRequestId } from './errors'
import type { Problem } from '../types/api'

type Query = Record<string, string | number | boolean | undefined>
type RequestOptions = { method?: string; body?: unknown; query?: Query; auth?: boolean; retry?: boolean }

export class HttpClient {
  private accessToken: string | null = null
  private refreshHandler?: () => Promise<string | null>
  private authFailureHandler?: () => void
  private refreshing: Promise<string | null> | null = null

  constructor(private readonly baseUrl = import.meta.env.VITE_API_BASE_URL || '/api/v1') {}

  setAccessToken(token: string | null) { this.accessToken = token }
  configureAuth(refresh: () => Promise<string | null>, onFailure: () => void) {
    this.refreshHandler = refresh
    this.authFailureHandler = onFailure
  }

  async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const { method = 'GET', body, query, auth = true, retry = true } = options
    const requestId = createRequestId()
    const url = new URL(`${this.baseUrl}${path}`, window.location.origin)
    Object.entries(query || {}).forEach(([key, value]) => {
      if (value !== undefined) url.searchParams.set(key, String(value))
    })
    const headers: HeadersInit = {
      Accept: 'application/json',
      'X-Request-ID': requestId,
      ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
      ...(auth && this.accessToken ? { Authorization: `Bearer ${this.accessToken}` } : {}),
    }
    try {
      const response = await fetch(url, {
        method, headers, credentials: 'include',
        body: body !== undefined ? JSON.stringify(body) : undefined,
      })
      if (response.status === 401 && auth && retry && this.refreshHandler) {
        const token = await this.refreshOnce()
        if (token) return this.request<T>(path, { ...options, retry: false })
        this.authFailureHandler?.()
      }
      if (!response.ok) throw new ApiError(await this.readProblem(response, requestId))
      if (response.status === 204) return undefined as T
      return await response.json() as T
    } catch (error) {
      if (error instanceof ApiError) throw error
      throw asApiError(error)
    }
  }

  private refreshOnce() {
    if (!this.refreshing && this.refreshHandler) {
      this.refreshing = this.refreshHandler().finally(() => { this.refreshing = null })
    }
    return this.refreshing || Promise.resolve(null)
  }

  private async readProblem(response: Response, fallbackRequestId: string): Promise<Problem> {
    const requestId = response.headers.get('X-Request-ID') || fallbackRequestId
    try {
      const data = await response.json() as Partial<Problem>
      return {
        type: data.type || 'about:blank', title: data.title || response.statusText || '请求失败',
        status: data.status || response.status, detail: data.detail, code: data.code || 'HTTP_ERROR',
        request_id: data.request_id || requestId, errors: data.errors, instance: data.instance,
      }
    } catch {
      return { type: 'about:blank', title: response.statusText || '请求失败', status: response.status, code: 'HTTP_ERROR', request_id: requestId }
    }
  }
}

export const http = new HttpClient()
