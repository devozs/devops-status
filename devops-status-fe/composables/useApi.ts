export const useApi = () => {
  const config = useRuntimeConfig()

  function resolveApiBase(): string {
    const raw = config.public.apiBase as string | undefined
    if (typeof raw === 'string' && raw.trim() !== '') {
      return raw.replace(/\/$/, '')
    }

    if (import.meta.dev) {
      return 'http://127.0.0.1:8080'
    }

    // Production without NUXT_PUBLIC_API_BASE: same-origin (browser) or request URL (SSR).
    if (import.meta.server) {
      try {
        return useRequestURL().origin
      } catch {
        return ''
      }
    }
    return ''
  }

  const apiFetch = <T>(path: string, opts: Record<string, any> = {}): Promise<T> => {
    return $fetch<T>(`${resolveApiBase()}${path}`, {
      ...opts,
      credentials: 'include' as const,
      headers: {
        'Content-Type': 'application/json',
        ...(opts.headers || {}),
      },
    })
  }

  return {
    apiFetch,
    get baseURL() {
      return resolveApiBase()
    },
  }
}
