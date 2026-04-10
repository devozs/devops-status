export const ADMIN_DASHBOARD_PREVIEW_LIMIT = 5

export type AdminDashboardPreviewLine = {
  main: string
  meta?: string
}

function truncateText(s: string, max: number): string {
  if (!s) return ''
  if (s.length <= max) return s
  return `${s.slice(0, max - 1)}…`
}

function k8sStatusLabel(c: Record<string, unknown>): string {
  if (
    c.status === 'connected'
    && c.has_api_credential
    && c.api_verify_ok === false
  ) {
    return 'connected · API not verified'
  }
  return String(c.status ?? '—')
}

function extractError(reason: unknown): string {
  const r = reason as Record<string, unknown> | undefined
  const data = r?.data as Record<string, unknown> | undefined
  const err = data?.error ?? r?.message
  if (typeof err === 'string' && err) return err
  return 'Failed to load'
}

export function useAdminDashboardData() {
  const { apiFetch } = useApi()
  const lim = ADMIN_DASHBOARD_PREVIEW_LIMIT

  const servicesLoading = ref(true)
  const services = ref<AdminDashboardPreviewLine[]>([])
  const servicesTotal = ref(0)
  const servicesError = ref('')

  const environmentsLoading = ref(true)
  const environments = ref<AdminDashboardPreviewLine[]>([])
  const environmentsTotal = ref(0)
  const environmentsError = ref('')

  const telemetryLoading = ref(true)
  const telemetry = ref<AdminDashboardPreviewLine[]>([])
  const telemetryTotal = ref(0)
  const telemetryError = ref('')

  const linksLoading = ref(true)
  const links = ref<AdminDashboardPreviewLine[]>([])
  const linksTotal = ref(0)
  const linksError = ref('')
  const linksSummary = ref('')

  const k8sClustersLoading = ref(true)
  const k8sClusters = ref<AdminDashboardPreviewLine[]>([])
  const k8sClustersTotal = ref(0)
  const k8sClustersError = ref('')

  const incidentsLoading = ref(true)
  const incidents = ref<AdminDashboardPreviewLine[]>([])
  const incidentsTotal = ref(0)
  const incidentsError = ref('')

  async function refresh() {
    servicesLoading.value = true
    servicesError.value = ''
    environmentsLoading.value = true
    environmentsError.value = ''
    telemetryLoading.value = true
    telemetryError.value = ''
    linksLoading.value = true
    linksError.value = ''
    linksSummary.value = ''
    k8sClustersLoading.value = true
    k8sClustersError.value = ''
    incidentsLoading.value = true
    incidentsError.value = ''

    const pSvc = apiFetch<unknown[]>('/api/admin/services')
    const pEnv = apiFetch<unknown[]>('/api/admin/environments')
    const pTel = apiFetch<unknown[]>('/api/admin/telemetry')
    const pLinks = apiFetch<{ service_links?: unknown[]; environment_links?: unknown[] }>('/api/admin/telemetry-links')
    const pK8s = apiFetch<unknown[]>('/api/admin/k8s-clusters')
    const pInc = apiFetch<unknown[]>('/api/admin/incidents')

    pSvc
      .then((list) => {
        const arr = list || []
        servicesTotal.value = arr.length
        services.value = arr.slice(0, lim).map((row: any) => ({
          main: row.name || row.id || '—',
          meta: row.slug ? String(row.slug) : undefined,
        }))
      })
      .catch((e) => {
        servicesError.value = extractError(e)
        services.value = []
        servicesTotal.value = 0
      })
      .finally(() => {
        servicesLoading.value = false
      })

    pEnv
      .then((list) => {
        const arr = list || []
        environmentsTotal.value = arr.length
        environments.value = arr.slice(0, lim).map((row: any) => ({
          main: row.name || row.id || '—',
        }))
      })
      .catch((e) => {
        environmentsError.value = extractError(e)
        environments.value = []
        environmentsTotal.value = 0
      })
      .finally(() => {
        environmentsLoading.value = false
      })

    pTel
      .then((list) => {
        const arr = list || []
        telemetryTotal.value = arr.length
        telemetry.value = arr.slice(0, lim).map((row: any) => ({
          main: row.name || row.id || '—',
          meta: row.adapter ? String(row.adapter) : undefined,
        }))
      })
      .catch((e) => {
        telemetryError.value = extractError(e)
        telemetry.value = []
        telemetryTotal.value = 0
      })
      .finally(() => {
        telemetryLoading.value = false
      })

    pK8s
      .then((list) => {
        const arr = list || []
        k8sClustersTotal.value = arr.length
        k8sClusters.value = arr.slice(0, lim).map((row: any) => ({
          main: row.name || row.id || '—',
          meta: k8sStatusLabel(row),
        }))
      })
      .catch((e) => {
        k8sClustersError.value = extractError(e)
        k8sClusters.value = []
        k8sClustersTotal.value = 0
      })
      .finally(() => {
        k8sClustersLoading.value = false
      })

    pInc
      .then((list) => {
        const arr = list || []
        incidentsTotal.value = arr.length
        incidents.value = arr.slice(0, lim).map((row: any) => ({
          main: truncateText(String(row.title || '—'), 48),
          meta: [row.target_name, row.status].filter(Boolean).join(' · ') || undefined,
        }))
      })
      .catch((e) => {
        incidentsError.value = extractError(e)
        incidents.value = []
        incidentsTotal.value = 0
      })
      .finally(() => {
        incidentsLoading.value = false
      })

    Promise.allSettled([pSvc, pEnv, pTel, pLinks])
      .then((results) => {
        const svcList
          = results[0].status === 'fulfilled' ? (results[0].value || []) : []
        const envList
          = results[1].status === 'fulfilled' ? (results[1].value || []) : []
        const telList
          = results[2].status === 'fulfilled' ? (results[2].value || []) : []
        const linkPack
          = results[3].status === 'fulfilled' ? results[3].value : null

        if (results[3].status === 'rejected') {
          linksError.value = extractError(results[3].reason)
          links.value = []
          linksTotal.value = 0
          return
        }

        const sb = linkPack?.service_links || []
        const eb = linkPack?.environment_links || []

        const svcById = new Map<string, string>(
          svcList.map((r: any) => [String(r.id), String(r.name || r.id)]),
        )
        const envById = new Map<string, string>(
          envList.map((r: any) => [String(r.id), String(r.name || r.id)]),
        )
        const telById = new Map<string, string>(
          telList.map((r: any) => [String(r.id), String(r.name || r.id)]),
        )

        const svcLines: AdminDashboardPreviewLine[] = (sb as any[]).map((b: any) => {
          const target = svcById.get(String(b.service_id)) || String(b.service_id)
          const tel = telById.get(String(b.telemetry_id)) || String(b.telemetry_id)
          return {
            main: target,
            meta: `Service → ${tel}`,
          }
        })
        const envLines: AdminDashboardPreviewLine[] = (eb as any[]).map((b: any) => {
          const target = envById.get(String(b.environment_id)) || String(b.environment_id)
          const tel = telById.get(String(b.telemetry_id)) || String(b.telemetry_id)
          return {
            main: target,
            meta: `Environment → ${tel}`,
          }
        })
        const combined = [...svcLines, ...envLines]
        linksTotal.value = combined.length
        linksSummary.value = `${sb.length} service · ${eb.length} environment`
        links.value = combined.slice(0, lim)
      })
      .catch((e) => {
        linksError.value = extractError(e)
        links.value = []
        linksTotal.value = 0
      })
      .finally(() => {
        linksLoading.value = false
      })
  }

  onMounted(refresh)

  return {
    refresh,
    services,
    servicesTotal,
    servicesError,
    servicesLoading,
    environments,
    environmentsTotal,
    environmentsError,
    environmentsLoading,
    telemetry,
    telemetryTotal,
    telemetryError,
    telemetryLoading,
    bindings: links,
    bindingsTotal: linksTotal,
    bindingsError: linksError,
    bindingsSummary: linksSummary,
    bindingsLoading: linksLoading,
    links,
    linksTotal,
    linksError,
    linksSummary,
    linksLoading,
    k8sClusters,
    k8sClustersTotal,
    k8sClustersError,
    k8sClustersLoading,
    incidents,
    incidentsTotal,
    incidentsError,
    incidentsLoading,
  }
}
