export const useApi = () => {
  const config = useRuntimeConfig()
  const baseURL = config.public.apiBase as string

  const apiFetch = <T>(path: string, opts: Record<string, any> = {}): Promise<T> => {
    return $fetch<T>(`${baseURL}${path}`, {
      ...opts,
      credentials: 'include' as const,
      headers: {
        'Content-Type': 'application/json',
        ...(opts.headers || {}),
      },
    })
  }

  return { apiFetch, baseURL }
}
