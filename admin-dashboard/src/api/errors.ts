import type { Problem } from '../types/api'

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly requestId: string
  readonly problem?: Problem

  constructor(problem: Problem) {
    super(problem.detail || problem.title)
    this.name = 'ApiError'
    this.status = problem.status
    this.code = problem.code
    this.requestId = problem.request_id
    this.problem = problem
  }

  get unauthorized() { return this.status === 401 }
  get forbidden() { return this.status === 403 }
}

export function asApiError(error: unknown) {
  if (error instanceof ApiError) return error
  return new ApiError({
    type: 'about:blank', title: '网络连接失败', status: 0,
    detail: error instanceof Error ? error.message : '无法连接运营服务，请检查网络后重试。',
    code: 'NETWORK_ERROR', request_id: 'unavailable',
  })
}

export function createRequestId() {
  return globalThis.crypto?.randomUUID?.() || `req-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
