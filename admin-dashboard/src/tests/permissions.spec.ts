import { describe, expect, it } from 'vitest'
import { canControlVehicle, hasPermission } from '../utils/permissions'
import type { User, Vehicle } from '../types/api'

const admin: User = { id: '1', username: 'admin', display_name: 'Admin', roles: ['admin'], created_at: '' }
const member: User = { ...admin, id: '2', roles: ['user'] }
const vehicle: Vehicle = { vin: 'LAURORA0000000001', display_name: 'A9', model: 'A9', owner_user_id: '1', access_role: 'owner', created_at: '' }

describe('permission policy', () => {
  it('separates admin operations from fleet viewing', () => {
    expect(hasPermission(member, 'fleet:view')).toBe(true)
    expect(hasPermission(member, 'users:manage')).toBe(false)
    expect(hasPermission(admin, 'users:manage')).toBe(true)
  })
  it('requires both global control permission and per-vehicle role', () => {
    expect(canControlVehicle(member, vehicle)).toBe(true)
    expect(canControlVehicle(member, { ...vehicle, access_role: 'viewer' })).toBe(false)
    expect(canControlVehicle(admin, vehicle)).toBe(true)
    expect(canControlVehicle(admin, { ...vehicle, access_role: 'viewer' })).toBe(true)
    expect(canControlVehicle(null, vehicle)).toBe(false)
  })
})
