export type AdapterType = 'http' | 'prometheus' | 'kubernetes' | 'cli'
export type ExecutionTarget = 'backend' | 'k8s_cluster'

export const K8S_VERSIONS = ['1.26', '1.27', '1.28', '1.29', '1.30', '1.31', '1.32'] as const

export const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'HEAD'] as const

export const PROM_OPERATORS = ['gt', 'gte', 'lt', 'lte', 'eq'] as const

export const CLI_TEXT_OPERATORS = ['eq', 'neq'] as const

export type CliMetricValueKind = 'number' | 'text'

export const K8S_CHECK_TYPES = ['api_health', 'deployment_ready', 'pod_status', 'node_status'] as const

export const CLI_RUNNER_PRESETS = ['alpine', 'ubuntu_24'] as const
export type CliRunnerPreset = (typeof CLI_RUNNER_PRESETS)[number]

export interface HttpConfig {
  service_provider_id: string
  path: string
  method: string
  headers?: Record<string, string>
  body?: string
  expected_status?: number
}

export interface PrometheusConfig {
  service_provider_id: string
  query: string
  threshold?: number
  operator?: string
}

export interface KubernetesConfig {
  cluster_id: string
  k8s_version: string
  check_type: string
  namespace?: string
  resource_name?: string
  label_selector?: string
}

export interface CliConfig {
  command?: string
  args?: string[]
  /** Arbitrary shell body for metric QoS mode (after container_prep); allowlist not applied. */
  cli_shell?: string
  /** Shell snippet run before the probe command (same sh -c as the command). */
  container_prep?: string
  /** Base image preset; server maps to CLI_RUNNER_IMAGE_ALPINE / CLI_RUNNER_IMAGE_UBUNTU_24. */
  runner?: CliRunnerPreset
  /** Optional exact image ref; must match a server-configured allowlist. */
  runner_image?: string
  env?: Record<string, string>
  parse_json: boolean
  execution_target: ExecutionTarget
  cluster_id?: string
  k8s_version?: string
}

export interface HttpK8sQosThresholds {
  green_max_ms: number
  yellow_max_ms: number
  /** Upper budget (ms); must be > yellow_max_ms. Latency QoS levels still use green/yellow cutoffs; red is above yellow. */
  red_max_ms: number
}

export interface PrometheusQosThresholds {
  green_operator: string
  green_threshold: number
  yellow_operator: string
  yellow_threshold: number
}

export interface CliQosThresholds {
  green_command: string
  green_args: string[]
  yellow_command: string
  yellow_args: string[]
  red_command: string
  red_args: string[]
}

/** Prometheus-style bands for one shell probe; red is implicit. */
export interface CliMetricNumberQosThresholds {
  value_kind: 'number'
  green_operator: string
  green_threshold: number
  yellow_operator: string
  yellow_threshold: number
}

export interface CliMetricTextQosThresholds {
  value_kind: 'text'
  green_operator: string
  green_text: string
  yellow_operator: string
  yellow_text: string
}

export interface TelemetryServiceProviderRow {
  id: string
  name: string
  provider_type: string
  has_prometheus_credentials?: boolean
}

export interface TelemetryRow {
  id: string
  name: string
  /** Short label for UIs where the probe is shown under an environment or service; **name** stays unique. */
  display_name?: string
  adapter: AdapterType
  config_json: Record<string, unknown>
  qos_thresholds?: Record<string, unknown>
  execution_target?: ExecutionTarget
  timeout_ms: number
  retries?: number
}

/** Modal seed: omit `id` for create-as-duplicate (POST); include `id` for edit (PUT). */
export type TelemetryFormInitial = Omit<TelemetryRow, 'id'> & { id?: string }

export interface TelemetryK8sClusterRow {
  id: string
  name?: string
  environment_id?: string | null
  status: string
  k8s_version?: string
  has_api_credential?: boolean
}
