import type { CommandType, Role, User, Vehicle } from '../types/api'

export type Permission = 'fleet:view' | 'alerts:ack' | 'users:manage' | 'audit:view' | 'settings:manage' | 'vehicle:control'

const rolePermissions: Record<Role, Permission[]> = {
  user: ['fleet:view', 'alerts:ack', 'vehicle:control'],
  admin: ['fleet:view', 'alerts:ack', 'users:manage', 'audit:view', 'settings:manage', 'vehicle:control'],
}

export function hasPermission(user: User | null, permission: Permission) {
  return Boolean(user?.roles.some(role => rolePermissions[role]?.includes(permission)))
}

export function canControlVehicle(user: User | null, vehicle: Vehicle | null, _command?: CommandType) {
  if (!user || !vehicle || !hasPermission(user, 'vehicle:control')) return false
  if (user.roles.includes('admin')) return true
  return vehicle.access_role === 'owner' || vehicle.access_role === 'driver'
}

export function canAccessRoute(user: User | null, permission?: Permission) {
  if (!permission) return true
  return hasPermission(user, permission)
}
