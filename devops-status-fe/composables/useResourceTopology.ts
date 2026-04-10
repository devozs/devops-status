import type { ResourceTopology } from '~/types/resource-topology'

export function useResourceTopology() {
  const { apiFetch } = useApi()

  function fetchAdmin(kind: 'service' | 'environment', id: string) {
    const path =
      kind === 'service'
        ? `/api/admin/services/${encodeURIComponent(id)}/topology`
        : `/api/admin/environments/${encodeURIComponent(id)}/topology`
    return apiFetch<ResourceTopology>(path)
  }

  function fetchPublic(kind: 'service' | 'environment', slug: string) {
    const path =
      kind === 'service'
        ? `/api/public/services/${encodeURIComponent(slug)}/topology`
        : `/api/public/environments/${encodeURIComponent(slug)}/topology`
    return apiFetch<ResourceTopology>(path)
  }

  return { fetchAdmin, fetchPublic }
}
