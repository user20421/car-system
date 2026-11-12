import { ref, watch } from 'vue'
import { defineStore } from 'pinia'

export type Settings = { refreshInterval: number; telemetryRetention: string; alertSound: boolean; compactTables: boolean; mapAutoFit: boolean; sessionTimeout: number }
const defaults: Settings = { refreshInterval: 5, telemetryRetention: '90d', alertSound: true, compactTables: false, mapAutoFit: true, sessionTimeout: 30 }

export const useSettingsStore = defineStore('settings', () => {
  const saved = localStorage.getItem('aurora-settings')
  let initial = defaults
  try { if (saved) initial = { ...defaults, ...JSON.parse(saved) as Partial<Settings> } } catch { /* use safe defaults */ }
  const settings = ref<Settings>(initial)
  const savedAt = ref<number | null>(null)
  watch(settings, value => { localStorage.setItem('aurora-settings', JSON.stringify(value)); savedAt.value = Date.now() }, { deep: true })
  function reset() { settings.value = { ...defaults } }
  return { settings, savedAt, reset }
})
