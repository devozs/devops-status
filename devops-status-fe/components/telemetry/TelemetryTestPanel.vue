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
        v-if="busy"
        type="button"
        class="btn btn-sm btn-stop-verify"
        @click="emit('abort-verify')"
      >
        Stop
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
    <div v-if="showTerminal" class="terminal-wrap">
      <div class="terminal-head">
        <span class="terminal-title">Execution log</span>
        <div class="terminal-head-actions">
          <button
            type="button"
            class="btn btn-sm btn-ghost terminal-copy"
            :disabled="!copyableExecutionLog"
            @click="copyExecutionLog"
          >
            Copy log
          </button>
          <Sun v-if="streamIdle" class="terminal-sun" :size="15" :stroke-width="2" aria-hidden="true" />
        </div>
      </div>
      <div v-if="streamHostLine" class="terminal-host">{{ streamHostLine }}</div>
      <div ref="terminalRef" class="terminal-pre" v-html="streamHtml" />
    </div>
    <div v-if="result && cliRunnerLine" class="execution-context">
      <span class="execution-context__label">Execution</span>
      <span class="execution-context__meta">{{ cliRunnerLine }}</span>
    </div>
    <div v-if="result" class="test-out" :class="resultBoxClass">
      <strong>{{ verifyOk ? 'OK' : 'Failed' }}</strong>
      <span v-if="result.qos_level"> · QoS: {{ result.qos_level }}</span>
      <span v-if="result.latency_ms != null"> · {{ result.latency_ms }}ms</span>
      <span v-if="rawValueSummary"> · {{ rawValueSummary }}</span>
      <span v-if="artifactoryDownloadLine" class="muted"> · {{ artifactoryDownloadLine }}</span>
      <p v-if="result.error" class="err">{{ result.error }}</p>
      <div v-if="showResultTrace" class="trace trace--summary trace--ansi" v-html="traceHtml" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { AnsiUp } from 'ansi_up'
import { Sun } from 'lucide-vue-next'
import { computed, nextTick, ref, watch } from 'vue'
import type { AdapterType } from '~/types/telemetry'
import { stripAnsi } from '~/utils/stripAnsi'

function ansiToHtml(s: string): string {
  return new AnsiUp().ansi_to_html(s)
}

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

const streamText = ref('')
const streamHtml = computed(() => ansiToHtml(streamText.value))
const streamHostLine = ref('')
const streamIdle = ref(false)
const terminalRef = ref<HTMLElement | null>(null)
let lastLogAt = 0

const showTerminal = computed(() => props.busy || streamText.value.length > 0 || streamHostLine.value.length > 0)

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

const traceHtml = computed(() => {
  const r = result.value
  if (!r || r.trace == null) return ''
  return ansiToHtml(String(r.trace))
})

const cliQosAttempts = computed(() => {
  const raw = cliLogsObj.value?.cli_qos_attempts
  return Array.isArray(raw) ? raw : []
})

const cliRunnerLine = computed(() => {
  const o = cliLogsObj.value
  if (!o) return ''
  const runner = strVal(o.cli_runner)
  const image = strVal(o.cli_image)
  const job = strVal(o.job)
  const ns = strVal(o.namespace)
  const parts: string[] = []
  if (runner) parts.push(`runner=${runner}`)
  if (image) parts.push(`image=${image}`)
  if (job) parts.push(`job=${job}`)
  if (ns) parts.push(`ns=${ns}`)
  return parts.join(' · ')
})

/** True when API returned CLI shell metadata — suppress capped `result.trace` so it does not duplicate streamed output. */
const hasCliStructuredLogs = computed(() => {
  if (props.adapter !== 'cli' && props.adapter !== 'hlctl' && props.adapter !== 'kubernetes') return false
  return Boolean(
    cliStdout.value ||
      cliStderr.value ||
      cliQosAttempts.value.length ||
      cliRunnerLine.value,
  )
})

function metricShellSourceLabel(source: string): string {
  switch (source) {
    case 'json':
      return 'numeric, JSON field value'
    case 'first_number':
      return 'numeric, first number in stdout'
    case 'last_number':
      return 'numeric, last number in stdout'
    default:
      return 'numeric'
  }
}

/** QoS verify line: compared value + how it was parsed + configured bands (metric shell). */
const rawValueSummary = computed(() => {
  const r = result.value
  if (!r) return ''

  const o = cliLogsObj.value
  const vk = o ? String(o.value_kind || '').toLowerCase().trim() : ''

  if (vk === 'number' && o) {
    const src = String(o.metric_value_source || '').trim()
    const g = String(o.metric_green_band || '').trim()
    const y = String(o.metric_yellow_band || '').trim()
    const bands = [g && `green ${g}`, y && `yellow ${y}`].filter(Boolean).join(' · ')
    const v = r.raw_value
    if (src && typeof v === 'number' && Number.isFinite(v)) {
      const mode = metricShellSourceLabel(src)
      const parts = [`Compared (${mode}): ${v}`]
      if (bands) parts.push(`Bands: ${bands}`)
      return parts.join(' · ')
    }
    if (g || y) {
      const err = typeof r.error === 'string' ? r.error.trim() : ''
      const bandStr = bands ? `Bands: ${bands}` : ''
      if (err) return bandStr ? `${bandStr} · ${err}` : err
      if (bandStr) return bandStr
    }
  }

  if (vk === 'text' && o) {
    const t = String(o.cli_text_value ?? '').trim()
    const g = String(o.metric_green_band || '').trim()
    const y = String(o.metric_yellow_band || '').trim()
    const bands = [g && `green ${g}`, y && `yellow ${y}`].filter(Boolean).join(' · ')
    if (t !== '') {
      const parts = [`Compared (text, trimmed stdout): ${t}`]
      if (bands) parts.push(`Bands: ${bands}`)
      return parts.join(' · ')
    }
    if (g || y) {
      const err = typeof r.error === 'string' ? r.error.trim() : ''
      const bandStr = bands ? `Bands: ${bands}` : ''
      if (err) return bandStr ? `${bandStr} · ${err}` : err
      if (bandStr) return bandStr
    }
  }

  const v = r.raw_value
  if (v === undefined || v === null) return ''
  if (typeof v === 'number' && Number.isFinite(v)) return `value: ${v}`
  if (typeof v === 'string') {
    const t = v.trim()
    return t === '' ? '' : `value: ${t}`
  }
  try {
    return `value: ${JSON.stringify(v)}`
  } catch {
    return ''
  }
})

/** Combined trace duplicates prep + probe; CLI QoS card uses cli_stdout only. */
const showResultTrace = computed(() => {
  const r = result.value
  if (!r || r.trace == null || String(r.trace).trim() === '') return false
  if (hasCliStructuredLogs.value) return false
  return true
})

const copyableExecutionLog = computed(() => streamText.value.length > 0)

async function copyExecutionLog() {
  const parts: string[] = []
  if (streamHostLine.value.trim()) parts.push(streamHostLine.value)
  if (streamText.value) parts.push(streamText.value)
  const t = parts.join('\n\n')
  if (!t || !navigator.clipboard?.writeText) return
  try {
    await navigator.clipboard.writeText(stripAnsi(t))
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
  /** User clicked Stop — parent should abort the SSE request. */
  'abort-verify': []
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

function clearStream() {
  streamText.value = ''
  streamHostLine.value = ''
  streamIdle.value = false
  lastLogAt = 0
}

function setStreamHost(meta: Record<string, unknown>) {
  const parts: string[] = []
  if (meta.cluster_name) parts.push(`cluster=${String(meta.cluster_name)}`)
  if (meta.cluster_id && !meta.cluster_name) parts.push(`cluster_id=${String(meta.cluster_id)}`)
  if (meta.execution_target) parts.push(`target=${String(meta.execution_target)}`)
  if (meta.shell_host_kind) parts.push(`kind=${String(meta.shell_host_kind)}`)
  streamHostLine.value = parts.join(' · ')
}

function appendStreamLog(stream: string, chunk: string) {
  lastLogAt = Date.now()
  streamIdle.value = false
  const label = stream === 'combined' ? '' : `[${stream}] `
  streamText.value += label + chunk
  void nextTick(() => {
    const el = terminalRef.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

function markStreamHeartbeat() {
  if (!props.busy) return
  if (!streamText.value && lastLogAt === 0) {
    streamIdle.value = true
    return
  }
  if (streamText.value.length > 0 && Date.now() - lastLogAt > 1800) {
    streamIdle.value = true
  }
}

function onVerify() {
  result.value = null
  clearStream()
  const payload: Record<string, unknown> = { test_connect_only: false }
  if (requiresEnvironment.value && environmentId.value) payload.environment_id = environmentId.value
  emit('verify', payload)
}

function onConnectOnly() {
  result.value = null
  clearStream()
  const payload: Record<string, unknown> = { test_connect_only: true }
  if (requiresEnvironment.value && environmentId.value) payload.environment_id = environmentId.value
  emit('verify', payload)
}

defineExpose({
  setResult(r: Record<string, unknown> | null) {
    result.value = r
  },
  clearStream,
  setStreamHost,
  appendStreamLog,
  markStreamHeartbeat,
})
</script>

<style scoped>
.test-panel { margin-top: 16px; padding-top: 12px; border-top: 1px solid var(--color-border); }
.subheading { font-size: 0.95rem; margin: 0 0 8px; font-weight: 600; }
.form-group { margin-bottom: 8px; max-width: 400px; }
.form-label { font-size: 0.85rem; font-weight: 500; color: var(--color-form-label); }
.actions { display: flex; gap: 8px; margin-top: 8px; flex-wrap: wrap; align-items: center; }
.btn-stop-verify { border-color: var(--color-border-strong); color: var(--color-text-secondary); }
.btn-stop-verify:hover:not(:disabled) { color: var(--color-text); border-color: var(--color-text-secondary); }
.terminal-wrap {
  margin-top: 12px;
  border-radius: 8px;
  border: 1px solid #30363d;
  background: #0d1117;
  overflow: hidden;
}
.terminal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 10px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
  font-size: 0.72rem;
  font-weight: 600;
  color: #8b949e;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}
.terminal-head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
/* Dark chrome: global .btn uses page text color (dark) — force readable label on terminal header */
.terminal-head .terminal-copy.btn {
  color: #c9d1d9;
  border-color: #484f58;
  background: rgba(240, 246, 252, 0.06);
}
.terminal-head .terminal-copy.btn:hover:not(:disabled) {
  color: #f0f6fc;
  border-color: #8b949e;
  background: rgba(240, 246, 252, 0.12);
}
.terminal-head .terminal-copy.btn:disabled {
  color: #6e7681;
  border-color: #30363d;
  background: transparent;
  opacity: 1;
}
.terminal-copy {
  text-transform: none;
  letter-spacing: normal;
  font-weight: 500;
  font-size: 0.75rem;
}
.terminal-title { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }
.terminal-sun {
  color: #e3b341;
  animation: sun-spin 1.2s linear infinite;
}
@keyframes sun-spin {
  to { transform: rotate(360deg); }
}
.terminal-host {
  padding: 6px 10px;
  font-size: 0.7rem;
  color: #79c0ff;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  border-bottom: 1px solid #21262d;
  word-break: break-word;
}
.terminal-pre {
  margin: 0;
  padding: 10px;
  min-height: 100px;
  max-height: 320px;
  overflow: auto;
  font-size: 0.75rem;
  line-height: 1.45;
  color: #3fb950;
  background: #0d1117;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  white-space: pre-wrap;
  word-break: break-word;
}
.execution-context {
  margin-top: 10px;
  padding: 8px 10px;
  border-radius: 6px;
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary, #f4f4f5);
  font-size: 0.72rem;
  line-height: 1.45;
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px 10px;
}
.execution-context__label {
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.02em;
}
.execution-context__meta {
  flex: 1;
  min-width: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  color: var(--color-text-secondary);
  word-break: break-word;
}
.test-out { margin-top: 10px; padding: 10px; border-radius: 6px; font-size: 0.85rem; border: 1px solid transparent; }
.test-out--ok { background: #dcfce7; color: #166534; border-color: rgba(34, 197, 94, 0.35); }
.test-out--fail { background: #fef2f2; color: #991b1b; border-color: rgba(239, 68, 68, 0.35); }
.test-out--qos-green { background: rgba(34, 197, 94, 0.18); color: #14532d; border-color: rgba(34, 197, 94, 0.45); }
.test-out--qos-yellow { background: rgba(234, 179, 8, 0.22); color: #713f12; border-color: rgba(234, 179, 8, 0.5); }
.test-out--qos-red { background: rgba(239, 68, 68, 0.18); color: #7f1d1d; border-color: rgba(239, 68, 68, 0.45); }
.err { margin: 6px 0 0; }
.trace { margin-top: 8px; font-size: 0.75rem; white-space: pre-wrap; word-break: break-word; }
.trace--summary { max-height: 140px; overflow: auto; }
.trace--ansi {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  line-height: 1.45;
}
.btn-ghost { background: transparent; border: 1px solid var(--color-border-strong); }
.muted { color: var(--color-text-secondary); }
.help { font-size: 0.8rem; color: var(--color-text-secondary); margin: 0 0 8px; line-height: 1.4; }
</style>
