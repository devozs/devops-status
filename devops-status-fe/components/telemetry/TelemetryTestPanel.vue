<template>
  <div class="test-panel">
    <h4 class="subheading">{{ requiresEnvironment ? 'Verify against environment' : 'Verify probe' }}</h4>
    <p v-if="!requiresEnvironment" class="help">
      This check runs from the API server using the configuration above (service provider, cluster, or command as applicable). Environments attach telemetry later — they are not required here.
    </p>
    <div v-if="requiresEnvironment" class="form-group">
      <label class="form-label">Environment</label>
      <AdminSelect
        v-model="environmentId"
        aria-label="Environment"
        placeholder="Select environment…"
        :options="environmentSelectOptions"
      />
    </div>
    <div class="actions">
      <button type="button" class="btn btn-sm" :disabled="!canVerify || busy" @click="onVerify">
        {{ busy ? 'Running…' : 'Verify' }}
      </button>
      <button
        v-if="adapter === 'prometheus'"
        type="button"
        class="btn btn-sm"
        :disabled="busy"
        @click="onConnectOnly"
      >
        Test connectivity
      </button>
    </div>
    <div v-if="result" class="test-out" :class="resultBoxClass">
      <strong>{{ verifyOk ? 'OK' : 'Failed' }}</strong>
      <span v-if="result.qos_level"> · QoS: {{ result.qos_level }}</span>
      <span v-if="result.latency_ms != null"> · {{ result.latency_ms }}ms</span>
      <span v-if="artifactoryDownloadLine" class="muted"> · {{ artifactoryDownloadLine }}</span>
      <p v-if="result.error" class="err">{{ result.error }}</p>
      <pre v-if="result.trace" class="trace trace--summary">{{ result.trace }}</pre>
      <template v-if="adapter === 'cli' && hasCliStructuredLogs">
        <div class="cli-logs-toolbar">
          <span v-if="cliRunnerLine" class="cli-meta muted">{{ cliRunnerLine }}</span>
          <button type="button" class="btn btn-sm btn-ghost" :disabled="!copyableCliText" @click="copyCliLogs">
            Copy CLI logs
          </button>
        </div>
        <div v-if="cliStdout" class="cli-log-block">
          <div class="cli-log-label">Stdout</div>
          <pre class="trace trace--cli">{{ cliStdout }}</pre>
        </div>
        <div v-if="cliStderr" class="cli-log-block">
          <div class="cli-log-label">Stderr</div>
          <pre class="trace trace--cli">{{ cliStderr }}</pre>
        </div>
        <div v-if="cliQosAttempts.length" class="cli-log-block">
          <div class="cli-log-label">QoS attempts</div>
          <pre class="trace trace--cli">{{ cliQosAttemptsFormatted }}</pre>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { AdapterType } from '~/types/telemetry'

const props = defineProps<{
  adapter: AdapterType
  environments: { id: string; name: string }[]
  clusters: { id: string; environment_id?: string | null; status: string; k8s_version?: string }[]
  k8sVersion?: string
  /** When true, only environments with a matching connected cluster are listed. */
  restrictToK8sEnv?: boolean
  /** When true, show the environment dropdown and require a selection before Verify. Default false — telemetry is verified on its own; environments bind telemetry elsewhere. */
  requiresEnvironment?: boolean
  busy?: boolean
}>()

const requiresEnvironment = computed(() => props.requiresEnvironment === true)

/** Two-way bind from parent (e.g. form save gating); falls back to local state when parent omits v-model. */
const environmentId = defineModel<string>('environmentId', { default: '' })

const result = ref<Record<string, unknown> | null>(null)

const verifyOk = computed(() => {
  const r = result.value
  if (!r) return false
  if (r.operational_ok !== undefined) return r.operational_ok === true
  return r.success === true
})

/** Artifactory liveness download sample summary from verify response. */
const artifactoryDownloadLine = computed(() => {
  if (props.adapter !== 'liveness') return ''
  const r = result.value
  if (!r) return ''
  const meta = r.metadata
  if (!meta || typeof meta !== 'object') return ''
  const m = meta as Record<string, unknown>
  if (String(m.liveness_check || '').toLowerCase() !== 'artifactory') return ''
  const bytesRaw = m.download_bytes
  const bytes = typeof bytesRaw === 'number' ? bytesRaw : Number(bytesRaw)
  const bpsRaw = m.avg_bytes_per_sec ?? r.raw_value
  const bps = typeof bpsRaw === 'number' ? bpsRaw : Number(bpsRaw)
  if (!Number.isFinite(bps) || bps <= 0) return ''
  const mibPerSec = bps / (1024 * 1024)
  const parts = [`~${mibPerSec.toFixed(2)} MiB/s avg`]
  if (Number.isFinite(bytes) && bytes > 0) {
    if (bytes >= 1024 * 1024) {
      parts.push(`${(bytes / (1024 * 1024)).toFixed(2)} MiB sampled`)
    } else if (bytes >= 1024) {
      parts.push(`${(bytes / 1024).toFixed(1)} KiB sampled`)
    } else {
      parts.push(`${Math.round(bytes)} B sampled`)
    }
  }
  return parts.join(', ')
})

const cliLogsObj = computed(() => {
  const r = result.value
  if (!r) return null
  const direct = r.cli_logs
  if (direct && typeof direct === 'object') return direct as Record<string, unknown>
  const meta = r.metadata
  if (meta && typeof meta === 'object') return meta as Record<string, unknown>
  return null
})

function strVal(v: unknown): string {
  return typeof v === 'string' ? v : ''
}

const cliStdout = computed(() => strVal(cliLogsObj.value?.cli_stdout))
const cliStderr = computed(() => strVal(cliLogsObj.value?.cli_stderr))

const cliQosAttempts = computed(() => {
  const raw = cliLogsObj.value?.cli_qos_attempts
  return Array.isArray(raw) ? raw : []
})

const cliQosAttemptsFormatted = computed(() => {
  try {
    return JSON.stringify(cliQosAttempts.value, null, 2)
  } catch {
    return String(cliQosAttempts.value)
  }
})

const cliRunnerLine = computed(() => {
  const o = cliLogsObj.value
  if (!o) return ''
  const runner = strVal(o.cli_runner)
  const image = strVal(o.cli_image)
  const job = strVal(o.job)
  const parts: string[] = []
  if (runner) parts.push(`runner=${runner}`)
  if (image) parts.push(`image=${image}`)
  if (job) parts.push(`job=${job}`)
  return parts.join(' · ')
})

const hasCliStructuredLogs = computed(() => {
  if (props.adapter !== 'cli' && props.adapter !== 'kubernetes') return false
  return Boolean(
    cliStdout.value ||
      cliStderr.value ||
      cliQosAttempts.value.length ||
      cliRunnerLine.value,
  )
})

const copyableCliText = computed(() => {
  const parts: string[] = []
  if (cliRunnerLine.value) parts.push(cliRunnerLine.value)
  if (cliStdout.value) parts.push('--- stdout ---\n' + cliStdout.value)
  if (cliStderr.value) parts.push('--- stderr ---\n' + cliStderr.value)
  if (cliQosAttempts.value.length) parts.push('--- qos attempts ---\n' + cliQosAttemptsFormatted.value)
  return parts.join('\n\n')
})

async function copyCliLogs() {
  const t = copyableCliText.value
  if (!t || !navigator.clipboard?.writeText) return
  try {
    await navigator.clipboard.writeText(t)
  } catch {
    /* ignore */
  }
}

/** When operational check passes, tint the box by QoS tier (matches threshold band colors). */
const resultBoxClass = computed(() => {
  const r = result.value
  if (!r) return ''
  if (!verifyOk.value) return 'test-out--fail'
  const raw = r.qos_level
  const tier = typeof raw === 'string' ? raw.toLowerCase().trim() : ''
  if (tier === 'green') return 'test-out--qos-green'
  if (tier === 'yellow') return 'test-out--qos-yellow'
  if (tier === 'red') return 'test-out--qos-red'
  return 'test-out--ok'
})

const emit = defineEmits<{
  verify: [payload: Record<string, unknown>]
}>()

const filteredEnvs = computed(() => {
  if (!props.restrictToK8sEnv) {
    return props.environments
  }
  const want = (props.k8sVersion || '').trim()
  const clusterEnv = new Set<string>()
  for (const c of props.clusters) {
    if (c.status !== 'connected' || !c.environment_id) continue
    const ver = (c.k8s_version || '').replace(/^v/, '')
    const minor = ver.split('.').slice(0, 2).join('.')
    if (want && minor !== want) continue
    clusterEnv.add(c.environment_id)
  }
  return props.environments.filter((e) => clusterEnv.has(e.id))
})

const environmentSelectOptions = computed(() =>
  filteredEnvs.value.map((e) => ({ value: e.id, label: e.name })),
)

const canVerify = computed(() => (requiresEnvironment.value ? !!environmentId.value : true))

watch(
  () => [props.restrictToK8sEnv, props.k8sVersion, requiresEnvironment.value] as const,
  () => {
    if (!requiresEnvironment.value) {
      environmentId.value = ''
      return
    }
    if (
      props.restrictToK8sEnv &&
      environmentId.value &&
      !filteredEnvs.value.some((e) => e.id === environmentId.value)
    ) {
      environmentId.value = ''
    }
  },
)

function onVerify() {
  result.value = null
  const payload: Record<string, unknown> = { test_connect_only: false }
  if (requiresEnvironment.value && environmentId.value) payload.environment_id = environmentId.value
  emit('verify', payload)
}

function onConnectOnly() {
  result.value = null
  const payload: Record<string, unknown> = { test_connect_only: true }
  if (requiresEnvironment.value && environmentId.value) payload.environment_id = environmentId.value
  emit('verify', payload)
}

defineExpose({
  setResult(r: Record<string, unknown> | null) {
    result.value = r
  },
})
</script>

<style scoped>
.test-panel { margin-top: 16px; padding-top: 12px; border-top: 1px solid var(--color-border); }
.subheading { font-size: 0.95rem; margin: 0 0 8px; font-weight: 600; }
.form-group { margin-bottom: 8px; max-width: 400px; }
.form-label { font-size: 0.85rem; font-weight: 500; color: var(--color-form-label); }
.actions { display: flex; gap: 8px; margin-top: 8px; flex-wrap: wrap; }
.test-out { margin-top: 10px; padding: 10px; border-radius: 6px; font-size: 0.85rem; border: 1px solid transparent; }
.test-out--ok { background: #dcfce7; color: #166534; border-color: rgba(34, 197, 94, 0.35); }
.test-out--fail { background: #fef2f2; color: #991b1b; border-color: rgba(239, 68, 68, 0.35); }
.test-out--qos-green { background: rgba(34, 197, 94, 0.18); color: #14532d; border-color: rgba(34, 197, 94, 0.45); }
.test-out--qos-yellow { background: rgba(234, 179, 8, 0.22); color: #713f12; border-color: rgba(234, 179, 8, 0.5); }
.test-out--qos-red { background: rgba(239, 68, 68, 0.18); color: #7f1d1d; border-color: rgba(239, 68, 68, 0.45); }
.err { margin: 6px 0 0; }
.trace { margin-top: 8px; font-size: 0.75rem; white-space: pre-wrap; word-break: break-word; }
.trace--summary { max-height: 140px; overflow: auto; }
.trace--cli { max-height: 280px; overflow: auto; margin-top: 4px; }
.cli-logs-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 8px; flex-wrap: wrap; margin-top: 10px; }
.cli-meta { font-size: 0.72rem; }
.cli-log-block { margin-top: 10px; }
.cli-log-label { font-size: 0.72rem; font-weight: 600; color: var(--color-text-secondary); text-transform: uppercase; letter-spacing: 0.02em; }
.btn-ghost { background: transparent; border: 1px solid var(--color-border-strong); }
.muted { color: var(--color-text-secondary); }
.help { font-size: 0.8rem; color: var(--color-text-secondary); margin: 0 0 8px; line-height: 1.4; }
</style>
