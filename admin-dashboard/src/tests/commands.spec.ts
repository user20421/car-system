import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { CommandTracker } from '../services/commands'
import type { VehicleCommand } from '../types/api'

const base: VehicleCommand = { command_id: 'cmd-1', vin: 'LAURORA0000000001', requested_by: 'u1', idempotency_key: 'key-000001', expires_at_ms: Date.now() + 60_000, type: 'lock', params: {}, status: 'pending', created_at_ms: Date.now(), updated_at_ms: Date.now() }

describe('CommandTracker', () => {
  beforeEach(() => { vi.useFakeTimers() })
  afterEach(() => { vi.useRealTimers() })
  it('deduplicates in-flight identical operations and follows authoritative states', async () => {
    const createCommand = vi.fn().mockResolvedValue({ ...base })
    const getCommand = vi.fn().mockResolvedValueOnce({ ...base, status: 'sent' }).mockResolvedValueOnce({ ...base, status: 'received' }).mockResolvedValueOnce({ ...base, status: 'succeeded' })
    const tracker = new CommandTracker({ createCommand, getCommand })
    const first = tracker.send(base.vin, 'lock')
    const duplicate = tracker.send(base.vin, 'lock')
    expect(first).toBe(duplicate)
    await vi.advanceTimersByTimeAsync(0)
    await vi.advanceTimersByTimeAsync(700)
    await vi.advanceTimersByTimeAsync(700)
    await vi.advanceTimersByTimeAsync(700)
    const result = await first
    expect(result.status).toBe('succeeded')
    expect(createCommand).toHaveBeenCalledTimes(1)
    expect(getCommand).toHaveBeenCalledTimes(3)
    expect(tracker.pendingCount).toBe(0)
  })
  it('never converts failed or expired server states into success', async () => {
    const createCommand = vi.fn().mockResolvedValue({ ...base, status: 'failed', error: { code: 'VEHICLE_BUSY', message: 'Busy', retryable: true } })
    const tracker = new CommandTracker({ createCommand, getCommand: vi.fn() })
    const result = await tracker.send(base.vin, 'unlock')
    expect(result.status).toBe('failed')
    expect(result.error?.code).toBe('VEHICLE_BUSY')
  })
})
