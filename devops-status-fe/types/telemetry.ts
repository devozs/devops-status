export type AdapterType = 'http' | 'prometheus' | 'kubernetes' | 'liveness' | 'cli' | 'hlctl'

/** Service provider types eligible for liveness (service_provider source). */
export const LIVENESS_SERVICE_PROVIDER_TYPES = [
  'prometheus',
  'grafana',
  'elasticsearch',
  'jenkins',
  'artifactory',
  'rancher',
] as const
export type LivenessServiceProviderType = (typeof LIVENESS_SERVICE_PROVIDER_TYPES)[number]

export type LivenessConfigSource = 'kubernetes' | 'service_provider'
export type ExecutionTarget = 'backend' | 'k8s_cluster'

export const K8S_VERSIONS = ['1.26', '1.27', '1.28', '1.29', '1.30', '1.31', '1.32'] as const

export const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'HEAD'] as const

export const PROM_OPERATORS = ['gt', 'gte', 'lt', 'lte', 'eq'] as const

export const CLI_TEXT_OPERATORS = ['eq', 'neq'] as const

export type CliMetricValueKind = 'number' | 'text'

export const K8S_CHECK_TYPES = ['api_health', 'deployment_ready', 'pod_status', 'node_status'] as const

export const CLI_RUNNER_PRESETS = ['toolkit', 'alpine', 'ubuntu_24'] as const
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
  /** Legacy HTTP API checks; ignored when cli_shell is set. */
  check_type?: string
  namespace?: string
  resource_name?: string
  label_selector?: string
  /** When set, probe runs as a Job on the cluster (same as CLI metric shell). */
  cli_shell?: string
  container_prep?: string
  runner?: CliRunnerPreset
  runner_image?: string
  /** Passed into the Job container (e.g. base64 kubeconfig); admin-visible in API. */
  env?: Record<string, string>
  /** Pod nodeSelector for shell Jobs on the cluster. */
  node_selector?: Record<string, string>
  parse_json?: boolean
  success_exit?: number
  /** Optional link to a telemetry shell hint (must match hint body on save). */
  job_env_hint_id?: string
  job_node_selector_hint_id?: string
  container_prep_hint_id?: string
  probe_shell_hint_id?: string
}

/** Optional Artifactory bandwidth sample (liveness + service_provider + Artifactory SP only). */
export interface LivenessArtifactoryConfig {
  download_repository_path?: string
  download_max_bytes?: number
}

/** Default / max sample size for Artifactory download probe (must match server). */
export const ARTIFACTORY_DOWNLOAD_DEFAULT_MAX_BYTES = 5 * 1024 * 1024
export const ARTIFACTORY_DOWNLOAD_ABS_MAX_BYTES = 50 * 1024 * 1024

/** Stored in telemetry.config_json for adapter liveness. */
export interface LivenessConfig {
  source: LivenessConfigSource
  kubernetes?: KubernetesConfig
  service_provider_id?: string
  /** Bandwidth test; only used when the linked service provider is Artifactory. */
  artifactory?: LivenessArtifactoryConfig
  timeout_ms?: number
}

/** Stored in telemetry.qos_thresholds for adapter liveness. */
export interface LivenessQosLatency {
  green_max_ms: number
  yellow_max_ms: number
  red_max_ms: number
}

export interface LivenessQosValue {
  green_operator: string
  green_threshold: number
  yellow_operator: string
  yellow_threshold: number
}

export interface LivenessQosThresholds {
  latency: LivenessQosLatency
  value?: LivenessQosValue
}

export interface CliConfig {
  command?: string
  args?: string[]
  /** Arbitrary shell body for metric QoS mode (after container_prep); allowlist not applied. */
  cli_shell?: string
  /** Shell snippet run before the probe command (same sh -c as the command). */
  container_prep?: string
  /** Base image preset; server maps to CLI_RUNNER_IMAGE_TOOLKIT / _ALPINE / _UBUNTU_24. */
  runner?: CliRunnerPreset
  /** Optional exact image ref; must match a server-configured allowlist. */
  runner_image?: string
  env?: Record<string, string>
  /** Pod nodeSelector when the probe runs as a Kubernetes Job (k8s_cluster or backend with CLI_BACKEND_EXECUTOR=k8s). */
  node_selector?: Record<string, string>
  parse_json: boolean
  execution_target: ExecutionTarget
  cluster_id?: string
  k8s_version?: string
  job_env_hint_id?: string
  job_node_selector_hint_id?: string
  container_prep_hint_id?: string
  probe_shell_hint_id?: string
}

/** HLCTL: runs in management after hlctl kubeconfig; credentials from HLCTL service provider. */
export interface HlctlConfig {
  service_provider_id: string
  command?: string
  args?: string[]
  cli_shell?: string
  container_prep?: string
  env?: Record<string, string>
  parse_json: boolean
  job_env_hint_id?: string
  container_prep_hint_id?: string
  probe_shell_hint_id?: string
}

export type ShellHintKind = 'job_env' | 'job_node_selector' | 'container_prep' | 'probe_shell'

export const SHELL_HINT_KINDS: ShellHintKind[] = [
  'job_env',
  'job_node_selector',
  'container_prep',
  'probe_shell',
]

export interface TelemetryShellHintRow {
  id: string
  kind: ShellHintKind
  title: string
  body: string
  description: string
  sort_order: number
  created_at: string
  updated_at: string
}

/** PUT /api/admin/telemetry-shell-hints/:id may include these after a body change. */
export interface TelemetryShellHintSaveResponse extends TelemetryShellHintRow {
  telemetry_rows_updated?: number
  telemetry_sync_errors?: string[]
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
