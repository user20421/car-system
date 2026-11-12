import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { api } from '../api'
import { asApiError } from '../api/errors'
import type { Alert, Telemetry, Vehicle } from '../types/api'

export const useFleetStore = defineStore('fleet', () => {
  const vehicles = ref<Vehicle[]>([])
  const telemetry = ref<Record<string, Telemetry>>({})
  const alerts = ref<Alert[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const error = ref<ReturnType<typeof asApiError> | null>(null)
  const lastUpdated = ref<number | null>(null)
  const realtimeState = ref('idle')
  const selectedVin = ref('')
  const onlineCount = computed(() => vehicles.value.filter(v => isOnline(v.vin)).length)
  const movingCount = computed(() => vehicles.value.filter(v => (telemetry.value[v.vin]?.speed_kph || 0) > 1).length)
  const averageBattery = computed(() => {
    const values = vehicles.value.map(v => telemetry.value[v.vin]?.battery_percent).filter((v): v is number => v !== undefined)
    return values.length ? values.reduce((a, b) => a + b, 0) / values.length : 0
  })
  const openAlerts = computed(() => alerts.value.filter(a => a.status === 'open'))

  function isOnline(vin: string) {
    const item = telemetry.value[vin]
    return Boolean(item && Date.now() - item.occurred_at_ms < 120_000)
  }
  function updateTelemetry(value: Telemetry) {
    if (vehicles.value.some(v => v.vin === value.vin)) telemetry.value[value.vin] = value
    lastUpdated.value = Date.now()
  }
  async function load(force = false) {
    if (loading.value || (loaded.value && !force)) return
    loading.value = true
    error.value = null
    try {
      const [vehiclePage, alertPage] = await Promise.all([api.listVehicles(), api.listAlerts()])
      vehicles.value = vehiclePage.items
      alerts.value = alertPage.items
      if (!selectedVin.value) selectedVin.value = vehicles.value[0]?.vin || ''
      const snapshots = await Promise.allSettled(vehiclePage.items.map(v => api.latestTelemetry(v.vin)))
      snapshots.forEach(result => { if (result.status === 'fulfilled') updateTelemetry(result.value) })
      loaded.value = true
      lastUpdated.value = Date.now()
    } catch (cause) { error.value = asApiError(cause); throw error.value }
    finally { loading.value = false }
  }
  async function refreshSnapshots(vins = vehicles.value.map(v => v.vin)) {
    const snapshots = await Promise.allSettled(vins.map(vin => api.latestTelemetry(vin)))
    snapshots.forEach(result => { if (result.status === 'fulfilled') updateTelemetry(result.value) })
  }
  function upsertAlert(alert: Alert) {
    const index = alerts.value.findIndex(a => a.id === alert.id)
    if (index >= 0) alerts.value[index] = alert
    else alerts.value.unshift(alert)
  }
  return { vehicles, telemetry, alerts, loading, loaded, error, lastUpdated, realtimeState, selectedVin, onlineCount, movingCount, averageBattery, openAlerts, isOnline, updateTelemetry, load, refreshSnapshots, upsertAlert }
})
