import type { ApiAdapter } from '../types/api'
import { MockApiAdapter } from './mock'
import { RealApiAdapter } from './real'

export const useMockApi = import.meta.env.VITE_USE_MOCK === 'true'
export const api: ApiAdapter = useMockApi ? new MockApiAdapter() : new RealApiAdapter()
