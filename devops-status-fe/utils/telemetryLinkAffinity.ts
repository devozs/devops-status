/** Telemetry rows as returned by GET /api/admin/telemetry (subset). */
export type TelemetryAffinityRow = {
  id: string
  adapter: string
  config_json?: Record<string, unknown>
}

export function telemetryClusterId(t: TelemetryAffinityRow): string {
  const c = t.config_json || {}
  if (t.adapter === 'kubernetes') {
    return String(c.cluster_id ?? '').trim()
  }
  if (t.adapter === 'liveness') {
    const src = String(c.source ?? '')
      .toLowerCase()
      .trim()
    if (src !== 'kubernetes') return ''
    const k = c.kubernetes as Record<string, unknown> | undefined
    return String(k?.cluster_id ?? '').trim()
  }
  return ''
}

/**
 * Whether telemetry may appear in the "add to service" picker when the service has this provider.
 * CLI has no service_provider_id in config — allowed when a provider is selected (workflow gate only).
 */
export function telemetryMatchesServiceProvider(t: TelemetryAffinityRow, serviceProviderId: string): boolean {
  const spId = serviceProviderId.trim()
  if (!spId) return false
  const c = t.config_json || {}
  if (t.adapter === 'http' || t.adapter === 'prometheus' || t.adapter === 'hlctl') {
    return String(c.service_provider_id ?? '').trim() === spId
  }
  if (t.adapter === 'liveness') {
    const src = String(c.source ?? '')
      .toLowerCase()
      .trim()
    if (src !== 'service_provider') return false
    return String(c.service_provider_id ?? '').trim() === spId
  }
  if (t.adapter === 'cli') {
    return true
  }
  return false
}
