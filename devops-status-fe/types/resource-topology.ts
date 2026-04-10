export interface ResourceTopologyInfra {
  type: 'service_provider' | 'k8s_cluster'
  id: string
  name: string
  provider_type?: string
}

export interface ResourceTopologyResource {
  id: string
  name: string
  slug: string
}

export interface ResourceTopologyTelemetryRef {
  id: string
  name: string
  display_name?: string
  adapter: string
}

export interface ResourceTopologyLinkTuning {
  sample_interval_sec: number
  window_size: number
  consecutive_failures_to_down: number
  consecutive_success_to_recover: number
}

export interface ResourceTopologyTelemetryRow {
  telemetry: ResourceTopologyTelemetryRef
  link: ResourceTopologyLinkTuning
}

export interface ResourceTopology {
  kind: 'service' | 'environment'
  resource: ResourceTopologyResource
  infra?: ResourceTopologyInfra | null
  telemetries: ResourceTopologyTelemetryRow[]
}
