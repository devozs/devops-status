/** Public incident stub from GET /api/public/status/summary (per service/environment). */
export interface PublicIncidentStub {
  id: string
  title: string
  severity: string
  status: string
  started_at: string
  resolved_at?: string | null
  source_telemetry_id?: string | null
}

export interface StatusSummaryDay {
  date: string
  availability_pct: number
  qos_level: string
  total_samples?: number
  failed_samples?: number
}

export interface StatusSummaryItem {
  name: string
  slug: string
  description?: string
  status?: string
  uptime_pct?: number
  days?: StatusSummaryDay[]
  recent_incidents?: PublicIncidentStub[]
}
