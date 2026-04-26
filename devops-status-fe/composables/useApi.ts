/** Matches devops-status-be/internal/middleware/csrf.go (cookie readable by JS for double-submit). */
const CSRF_COOKIE = 'csrf_token'
const CSRF_HEADER = 'X-CSRF-Token'

function readCsrfCookie(): string | undefined {
  if (import.meta.server || typeof document === 'undefined') return undefined
  const m = document.cookie.match(new RegExp(`(?:^|;\\s*)${CSRF_COOKIE}=([^;]*)`))
  if (!m?.[1]) return undefined
  try {
    return decodeURIComponent(m[1])
  } catch {
    return m[1]
  }
}

export const useApi = () => {
  const config = useRuntimeConfig()

  function resolveApiBase(): string {
    const rawPublic = config.public.apiBase as string | undefined
    const rawInternal = (config as { apiBaseInternal?: string }).apiBaseInternal as string | undefined

    // SSR must reach the API inside the cluster; the public HTTPS URL often fails from the pod (hairpin, egress, TLS).
    // Browsers still use NUXT_PUBLIC_API_BASE (same host as the SPA) or same-origin ''.
    if (import.meta.server && typeof rawInternal === 'string' && rawInternal.trim() !== '') {
      return rawInternal.replace(/\/$/, '')
    }

    if (typeof rawPublic === 'string' && rawPublic.trim() !== '') {
      return rawPublic.replace(/\/$/, '')
    }

    if (import.meta.dev) {
      return 'http://127.0.0.1:8080'
    }

    // Production without NUXT_PUBLIC_API_BASE: same-origin (browser) or request URL (SSR fallback).
    if (import.meta.server) {
      try {
        return useRequestURL().origin
      } catch {
        return ''
      }
    }
    return ''
  }

  const apiFetch = async <T>(path: string, opts: Record<string, any> = {}): Promise<T> => {
    const method = String(opts.method || 'GET').toUpperCase()
    const needsCsrf = ['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)
    const base = resolveApiBase()

    // Production CSRF: cookie is set on GETs to the API. If the SPA never hit a GET before this POST
    // (or the cookie was cleared), fetch /meta first so ensureCSRFCookie runs and document.cookie updates.
    // Login is intentionally unauthenticated and does not use CSRF on the server — skip preflight.
    const pathOnly = path.split('?')[0] || ''
    const isLoginPost = method === 'POST' && pathOnly === '/api/admin/auth/login'
    if (needsCsrf && import.meta.client && !readCsrfCookie() && !isLoginPost) {
      await $fetch(`${base}/api/admin/meta`, {
        method: 'GET',
        credentials: 'include',
      })
    }

    const csrf = needsCsrf ? readCsrfCookie() : undefined

    return $fetch<T>(`${base}${path}`, {
      ...opts,
      credentials: 'include' as const,
      headers: {
        'Content-Type': 'application/json',
        ...(csrf ? { [CSRF_HEADER]: csrf } : {}),
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
