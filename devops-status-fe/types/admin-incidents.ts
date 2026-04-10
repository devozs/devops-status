/** GET /api/admin/incidents enriched row */
export interface AdminIncidentInfraServiceProvider {
  id: string
  name: string
  provider_type: string
}

export interface AdminIncidentInfraK8sCluster {
  id: string
  name: string
}

export interface AdminIncidentInfra {
  service_provider?: AdminIncidentInfraServiceProvider
  k8s_cluster?: AdminIncidentInfraK8sCluster
}

export interface AdminLinkedTelemetry {
  id: string
  name: string
  display_name: string
  adapter: string
}

export type IncidentDegradationQoS = {
  kind: 'qos'
  adapter: string
  pass_rate: number
  qos_level: string
}

export type IncidentDegradationOperational = {
  kind: 'operational'
  adapter: string
}

export type IncidentDegradation = IncidentDegradationQoS | IncidentDegradationOperational | Record<string, unknown>

export interface AdminIncidentListItem {
  id: string
  target_type: string
  target_id: string
  title: string
  status: string
  severity: string
  started_at: string
  resolved_at?: string | null
  source_telemetry_id?: string | null
  degradation?: IncidentDegradation | null
  resolved_by?: 'system' | 'admin' | null
  created_at: string
  updated_at: string
  target_name: string
  target_slug: string
  infra?: AdminIncidentInfra | null
  linked_telemetry: AdminLinkedTelemetry[]
  resolution: 'open' | 'automatic' | 'manual' | 'unknown'
  qos_pass_rate_percent?: number | null
  /** Enriched for history / list: first update or degradation-derived */
  issue_message?: string | null
  /** How it ended + latest update when distinct from issue */
  resolution_message?: string | null
}

export interface IncidentUpdate {
  id: string
  incident_id: string
  status: string
  message: string
  author: string
  created_at: string
}

export interface AdminIncidentDetailResponse {
  incident: AdminIncidentListItem
  updates: IncidentUpdate[]
}
