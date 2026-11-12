import { ref, watch } from 'vue'
import { defineStore } from 'pinia'

export type Toast = { id: number; title: string; message?: string; tone: 'success' | 'error' | 'info' }

export const useUiStore = defineStore('ui', () => {
  const storedTheme = localStorage.getItem('aurora-theme')
  const theme = ref<'light' | 'dark'>(storedTheme === 'dark' ? 'dark' : 'light')
  const sidebarCollapsed = ref(false)
  const toasts = ref<Toast[]>([])
  watch(theme, value => {
    document.documentElement.dataset.theme = value
    localStorage.setItem('aurora-theme', value)
  }, { immediate: true })
  function toggleTheme() { theme.value = theme.value === 'light' ? 'dark' : 'light' }
  function toast(title: string, tone: Toast['tone'] = 'info', message?: string) {
    const id = Date.now() + Math.random()
    toasts.value.push({ id, title, message, tone })
    setTimeout(() => { toasts.value = toasts.value.filter(item => item.id !== id) }, 4200)
  }
  return { theme, sidebarCollapsed, toasts, toggleTheme, toast }
})
