import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
const mocks = vi.hoisted(() => ({ listVehicles: vi.fn(), listAlerts: vi.fn(), latestTelemetry: vi.fn() }))
vi.mock('../api', () => ({ api: mocks, useMockApi: true }))
import { useFleetStore } from '../stores/fleet'

const vehicle = { vin: 'LAURORA0000000001', display_name: 'A9', model: 'A9', owner_user_id: 'u1', access_role: 'owner' as const, created_at: '' }
const telemetry = { vin: vehicle.vin, device_boot_id: 'boot', sequence: 1, occurred_at_ms: Date.now(), speed_kph: 42, rpm: 900, battery_percent: 80, range_km: 400, locked: false, defrost: false, fault: [] }

describe('fleet store', () => {
  beforeEach(() => setActivePinia(createPinia()))
  it('retains partial snapshots when one vehicle telemetry request fails', async () => {
    mocks.listVehicles.mockResolvedValue({ items: [vehicle], page: { number: 1, size: 100, total_items: 1, total_pages: 1 } })
    mocks.listAlerts.mockResolvedValue({ items: [], page: { number: 1, size: 100, total_items: 0, total_pages: 0 } })
    mocks.latestTelemetry.mockResolvedValue(telemetry)
    const store = useFleetStore()
    await store.load()
    expect(store.vehicles).toHaveLength(1)
    expect(store.telemetry[vehicle.vin].speed_kph).toBe(42)
    expect(store.onlineCount).toBe(1)
    expect(store.loaded).toBe(true)
  })
  it('marks stale telemetry offline', async () => {
    const store = useFleetStore()
    store.vehicles = [vehicle]
    store.updateTelemetry({ ...telemetry, occurred_at_ms: Date.now() - 300_000 })
    expect(store.isOnline(vehicle.vin)).toBe(false)
  })
})
