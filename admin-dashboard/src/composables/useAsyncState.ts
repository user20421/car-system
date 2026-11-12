import { computed, ref } from 'vue'
import { asApiError } from '../api/errors'

export function useAsyncState() {
  const loading = ref(false)
  const error = ref<ReturnType<typeof asApiError> | null>(null)
  const loaded = ref(false)
  const isOffline = ref(!navigator.onLine)
  const empty = computed(() => loaded.value && !loading.value && !error.value)
  async function run<T>(task: () => Promise<T>) {
    loading.value = true
    error.value = null
    try {
      const result = await task()
      loaded.value = true
      return result
    } catch (cause) {
      error.value = asApiError(cause)
      throw error.value
    } finally {
      loading.value = false
    }
  }
  return { loading, error, loaded, empty, isOffline, run }
}
