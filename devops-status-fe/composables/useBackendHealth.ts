const backendStatus = ref<'connected' | 'disconnected' | 'checking'>('checking')

let polling: ReturnType<typeof setInterval> | null = null

export const useBackendHealth = () => {
  const { apiFetch } = useApi()

  async function check() {
    try {
      await apiFetch<any>('/healthz')
      backendStatus.value = 'connected'
    } catch {
      backendStatus.value = 'disconnected'
    }
  }

  function startPolling(intervalMs = 15000) {
    if (polling) return
    check()
    polling = setInterval(check, intervalMs)
  }

  function stopPolling() {
    if (polling) {
      clearInterval(polling)
      polling = null
    }
  }

  return { backendStatus, check, startPolling, stopPolling }
}
