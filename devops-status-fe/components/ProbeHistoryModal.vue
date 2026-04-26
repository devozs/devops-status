<template>
  <div v-if="open" class="modal-overlay" @click.self="emitClose">
    <div
      class="modal-card modal-card--wide history-panel"
      role="dialog"
      :aria-labelledby="titleId"
    >
      <h2 :id="titleId" class="modal-title">Probe history — {{ displayName }}</h2>
      <p class="modal-subtitle muted">{{ subtitle }}</p>
      <div v-if="breakdownLoading && !links.length" class="history-state">Loading links…</div>
      <div v-else-if="!links.length" class="history-state">{{ emptyLinksMessage }}</div>
      <template v-else>
        <div class="history-tabs" role="tablist">
          <button
            v-for="row in links"
            :key="row.telemetry_id"
            type="button"
            role="tab"
            class="history-tab"
            :class="{ 'history-tab--active': tabTelemetryId === row.telemetry_id }"
            :aria-selected="tabTelemetryId === row.telemetry_id"
            @click="selectTab(row.telemetry_id)"
          >
            {{ tabLabel(row) }}
          </button>
        </div>
        <div v-if="samplesLoading" class="history-state">Loading samples…</div>
        <div v-else class="history-table-wrap">
          <table class="mini-table history-samples-table">
            <thead>
              <tr>
                <th>Time (UTC)</th>
                <th>OK</th>
                <th>Latency</th>
                <th>Trace / error</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="s in samples" :key="s.id">
                <tr>
                  <td>{{ formatSampleTime(s.sampled_at) }}</td>
                  <td>
                    <span class="status-pill" :class="s.success ? 'status-pill--success' : 'status-pill--critical'">
                      {{ s.success ? 'Yes' : 'No' }}
                    </span>
                  </td>
                  <td>{{ s.latency_ms != null ? `${s.latency_ms} ms` : '—' }}</td>
                  <td class="history-trace-cell">
                    <span class="history-trace">{{ traceSnippet(s.source_trace) }}</span>
                    <button
                      v-if="historyAdapterCliLike && sampleHasCliLogs(s)"
                      type="button"
                      class="btn btn-sm history-cli-toggle"
                      @click="toggleCliExpand(s.id)"
                    >
                      {{ cliExpanded[s.id] ? 'Hide CLI logs' : 'CLI logs' }}
                    </button>
                  </td>
                </tr>
                <tr
                  v-if="historyAdapterCliLike && cliExpanded[s.id] && sampleHasCliLogs(s)"
                  class="history-cli-detail-row"
                >
                  <td colspan="4">
                    <pre class="history-cli-pre">{{ formatCliSampleDetail(s) }}</pre>
                  </td>
                </tr>
              </template>
              <tr v-if="!samples.length">
                <td colspan="4" class="empty-row">No samples with per-telemetry tagging yet, or none in range.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
      <div class="modal-actions">
        <button type="button" class="btn" @click="emitClose">Close</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  formatCliSampleDetail,
  formatSampleTime,
  sampleHasCliLogs,
  traceSnippet,
  type TelemetrySampleRow,
} from '~/utils/probeHistoryFormat'

interface BreakdownTelemetry {
  telemetry_id: string
  name: string
  display_name?: string
  adapter?: string
}

const props = withDefaults(
  defineProps<{
    open: boolean
    resourceKind: 'service' | 'environment'
    slug: string
    displayName: string
    /** When set, select this telemetry tab after links load */
    initialTelemetryId?: string | null
  }>(),
  { initialTelemetryId: null },
)

const emit = defineEmits<{
  close: []
}>()

const { apiFetch } = useApi()

const titleId = `probe-history-title-${Math.random().toString(36).slice(2, 11)}`

const subtitle = computed(() =>
  props.resourceKind === 'service'
    ? 'Recent samples per linked HTTP, Prometheus, or CLI telemetry (operational probes).'
    : 'Recent samples per linked Kubernetes telemetry (operational probes).',
)

const emptyLinksMessage = computed(() =>
  props.resourceKind === 'service'
    ? 'No telemetry links. Attach probes in the service editor.'
    : 'No telemetry links. Attach probes in the environment editor.',
)

const breakdownLoading = ref(false)
const links = ref<BreakdownTelemetry[]>([])
const tabTelemetryId = ref('')
const samplesLoading = ref(false)
const samples = ref<TelemetrySampleRow[]>([])
const cliExpanded = ref<Record<string, boolean>>({})

const activeAdapter = computed(() => {
  const row = links.value.find((r) => r.telemetry_id === tabTelemetryId.value)
  return row?.adapter || ''
})

const historyAdapterCliLike = computed(
  () => activeAdapter.value === 'cli' || activeAdapter.value === 'hlctl',
)

function tabLabel(row: BreakdownTelemetry): string {
  const d = row.display_name?.trim()
  if (d) return d
  return row.name
}

function emitClose() {
  emit('close')
}

function toggleCliExpand(id: string) {
  cliExpanded.value = { ...cliExpanded.value, [id]: !cliExpanded.value[id] }
}

function resetState() {
  links.value = []
  tabTelemetryId.value = ''
  samples.value = []
  cliExpanded.value = {}
}

async function fetchBreakdown(): Promise<BreakdownTelemetry[]> {
  const enc = encodeURIComponent(props.slug)
  const path =
    props.resourceKind === 'service'
      ? `/api/public/services/${enc}/telemetry-breakdown`
      : `/api/public/environments/${enc}/telemetry-breakdown`
  const res = await apiFetch<{ telemetries?: BreakdownTelemetry[] }>(path)
  return res.telemetries || []
}

async function fetchSamples() {
  const tid = tabTelemetryId.value
  if (!tid) {
    samples.value = []
    return
  }
  samplesLoading.value = true
  try {
    const enc = encodeURIComponent(props.slug)
    const base =
      props.resourceKind === 'service'
        ? `/api/public/services/${enc}/telemetry-samples`
        : `/api/public/environments/${enc}/telemetry-samples`
    const q = new URLSearchParams({ telemetry_id: tid, limit: '50', probe_kind: 'operational' })
    const res = await apiFetch<{ samples?: TelemetrySampleRow[] }>(`${base}?${q.toString()}`)
    samples.value = res.samples || []
    cliExpanded.value = {}
  } catch {
    samples.value = []
  } finally {
    samplesLoading.value = false
  }
}

function pickInitialTab(list: BreakdownTelemetry[]) {
  const want = props.initialTelemetryId?.trim()
  if (want && list.some((r) => r.telemetry_id === want)) {
    tabTelemetryId.value = want
    return
  }
  tabTelemetryId.value = list[0]?.telemetry_id || ''
}

async function loadWhenOpen() {
  resetState()
  breakdownLoading.value = true
  try {
    const list = await fetchBreakdown()
    links.value = list
    pickInitialTab(list)
    if (tabTelemetryId.value) await fetchSamples()
  } catch {
    links.value = []
  } finally {
    breakdownLoading.value = false
  }
}

function selectTab(tid: string) {
  tabTelemetryId.value = tid
  void fetchSamples()
}

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) void loadWhenOpen()
    else resetState()
  },
)
</script>

<style scoped>
.history-panel .modal-subtitle.muted {
  color: var(--color-text-secondary);
  margin-top: 4px;
}
.history-state {
  padding: 16px 0;
  color: var(--color-text-secondary);
  font-size: 0.9rem;
}
.history-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 12px 0 8px;
}
.history-tab {
  padding: 6px 12px;
  border-radius: var(--radius, 8px);
  border: 1px solid var(--color-border);
  background: var(--color-bg);
  font-size: 0.85rem;
  cursor: pointer;
}
.history-tab:hover {
  border-color: var(--color-border-strong);
}
.history-tab--active {
  border-color: var(--color-primary);
  background: color-mix(in srgb, var(--color-primary) 12%, transparent);
  font-weight: 600;
}
.history-table-wrap {
  max-height: min(50vh, 420px);
  overflow: auto;
  margin-bottom: 8px;
}
.history-samples-table {
  margin-bottom: 0;
}
.mini-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8rem;
  margin-bottom: 12px;
}
.mini-table th,
.mini-table td {
  padding: 6px 8px;
  border-bottom: 1px solid var(--color-border);
  text-align: left;
  vertical-align: middle;
}
.history-trace-cell {
  max-width: 280px;
}
.history-trace {
  display: block;
  font-size: 0.72rem;
  font-family: ui-monospace, monospace;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 5.5em;
  overflow: auto;
  line-height: 1.35;
}
.history-cli-toggle {
  margin-top: 6px;
  display: inline-block;
}
.history-cli-detail-row td {
  background: var(--color-bg-muted, rgba(0, 0, 0, 0.04));
  vertical-align: top;
}
.history-cli-pre {
  margin: 0;
  padding: 8px;
  font-size: 0.72rem;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 320px;
  overflow: auto;
  font-family: ui-monospace, monospace;
}
</style>
