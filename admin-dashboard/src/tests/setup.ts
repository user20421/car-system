import { afterEach, vi } from 'vitest'
import { config } from '@vue/test-utils'

class ResizeObserverMock { observe() {} unobserve() {} disconnect() {} }
globalThis.ResizeObserver = ResizeObserverMock
Object.defineProperty(window, 'matchMedia', { writable: true, value: vi.fn().mockReturnValue({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() }) })
config.global.stubs = { transition: false, 'transition-group': false }
afterEach(() => { vi.clearAllMocks(); localStorage.clear() })
