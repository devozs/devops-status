/**
 * Admin routes use a session cookie issued by the API origin (e.g. :8080) while Nuxt runs on :3000.
 * On a full browser refresh, SSR runs in Node and cannot send that cookie to the API, so /me would
 * always fail and log users out. Only validate the session in the browser where credentials work.
 */
export default defineNuxtRouteMiddleware(async (to) => {
  if (!to.path.startsWith('/admin') || to.path === '/admin/login') return

  if (import.meta.server) {
    return
  }

  const { apiFetch } = useApi()

  try {
    await apiFetch<any>('/api/admin/auth/me')
  } catch {
    return navigateTo('/admin/login')
  }
})
