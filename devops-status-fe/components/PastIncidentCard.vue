<template>
  <article class="past-incident">
    <button
      :id="headerId"
      type="button"
      class="past-incident__header"
      :aria-expanded="expanded"
      :aria-controls="panelId"
      @click="toggle"
    >
      <span class="past-incident__header-main">
        <ChevronDown v-if="!expanded" class="past-incident__chev" :size="18" aria-hidden="true" />
        <ChevronUp v-else class="past-incident__chev" :size="18" aria-hidden="true" />
        <span class="past-incident__title" :class="'severity--' + incident.severity">{{ incident.title }}</span>
      </span>
      <span class="past-incident__meta">
        <span v-if="incident.target_name" class="past-incident__target">{{ targetKindLabel(incident.target_type) }} · {{ incident.target_name }}</span>
        <span class="past-incident__date">{{ formatDate(incident.started_at) }}</span>
      </span>
      <span class="past-incident__status" :class="pastIncidentStatusPillClass(incident)">{{ pastIncidentStatusPillLabel(incident) }}</span>
    </button>

    <div
      v-show="expanded"
      :id="panelId"
      class="past-incident__panel"
      role="region"
      :aria-labelledby="headerId"
    >
      <dl class="past-incident__dl">
        <template v-if="incident.target_name || incident.target_slug">
          <dt>Target</dt>
          <dd>
            <span class="kind-tag" :class="'kind--' + incident.target_type">{{ targetKindLabel(incident.target_type) }}</span>
            {{ incident.target_name || '—' }}
            <span v-if="incident.target_slug" class="past-incident__slug">{{ incident.target_slug }}</span>
          </dd>
        </template>
        <dt>Infra</dt>
        <dd>{{ formatInfra(incident) }}</dd>
        <dt>QoS / probe</dt>
        <dd>{{ formatQoSLine(incident) }}</dd>
        <dt v-if="incident.resolution !== 'open'">Resolution</dt>
        <dd v-if="incident.resolution !== 'open'">{{ resolutionLabel(incident.resolution) }}</dd>
        <dt>Telemetry</dt>
        <dd>
          <span v-if="incident.linked_telemetry?.length" class="past-incident__chips">
            <span v-for="t in incident.linked_telemetry" :key="t.id" class="chip" :title="t.display_name || t.name">{{ t.adapter }}</span>
          </span>
          <span v-else class="past-incident__muted">—</span>
        </dd>
      </dl>

      <div v-if="updatesLoading" class="past-incident__muted past-incident__updates-loading">Loading updates…</div>
      <div v-else-if="updatesError" class="past-incident__updates-err">{{ updatesError }}</div>
      <ol v-else-if="updatesChrono.length" class="past-incident__timeline">
        <li v-for="u in updatesChrono" :key="u.id" class="past-incident__tl-item">
          <div class="past-incident__tl-meta">
            <span class="past-incident__tl-author">{{ u.author }}</span>
            <span class="past-incident__muted">{{ formatDateTime(u.created_at) }}</span>
            <span class="past-incident__tl-status">{{ pastIncidentUpdateStatusLabel(u.status) }}</span>
          </div>
          <p class="past-incident__tl-msg">{{ u.message }}</p>
        </li>
      </ol>
    </div>
  </article>
</template>

<script setup lang="ts">
import { ChevronDown, ChevronUp } from 'lucide-vue-next'
import type { IncidentDisplayItem, IncidentUpdate, IncidentDegradation, PublicIncidentDetailResponse } from '~/types/incident-display'

const props = defineProps<{ incident: IncidentDisplayItem }>()

const { apiFetch } = useApi()

const expanded = ref(false)
const updates = ref<IncidentUpdate[] | null>(null)
const updatesLoading = ref(false)
const updatesError = ref('')

const headerId = computed(() => `past-incident-h-${props.incident.id}`)
const panelId = computed(() => `past-incident-p-${props.incident.id}`)

const updatesChrono = computed(() => {
  if (!updates.value?.length) return []
  return [...updates.value].sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
  )
})

async function loadUpdates() {
  if (updates.value !== null || updatesLoading.value) return
  updatesLoading.value = true
  updatesError.value = ''
  try {
    const data = await apiFetch<PublicIncidentDetailResponse>(`/api/public/incidents/${props.incident.id}`)
    updates.value = data.updates || []
  } catch (e: any) {
    updatesError.value = e?.message || 'Could not load updates'
    updates.value = []
  } finally {
    updatesLoading.value = false
  }
}

async function toggle() {
  expanded.value = !expanded.value
  if (expanded.value) {
    await loadUpdates()
  }
}

function targetKindLabel(t: string) {
  if (t === 'service') return 'Service'
  if (t === 'environment') return 'Environment'
  return t
}

function formatInfra(inc: IncidentDisplayItem) {
  const infra = inc.infra
  if (!infra) return '—'
  if (infra.service_provider) {
    const p = infra.service_provider
    return `${p.name} (${p.provider_type})`
  }
  if (infra.k8s_cluster) return infra.k8s_cluster.name
  return '—'
}

function formatDegradation(d: IncidentDegradation | null | undefined) {
  if (!d || typeof d !== 'object') return ''
  const o = d as Record<string, unknown>
  if (o.kind === 'qos' && typeof o.pass_rate === 'number') {
    const lvl = typeof o.qos_level === 'string' ? o.qos_level : ''
    const ad = typeof o.adapter === 'string' ? o.adapter : ''
    return `Pass rate ${o.pass_rate.toFixed(1)}% · level ${lvl || '—'} · ${ad || 'probe'}`
  }
  if (o.kind === 'operational' && typeof o.adapter === 'string') {
    return `Operational · ${o.adapter}`
  }
  return ''
}

function formatQoSLine(inc: IncidentDisplayItem) {
  const fromDeg = formatDegradation(inc.degradation ?? undefined)
  if (fromDeg) return fromDeg
  if (typeof inc.qos_pass_rate_percent === 'number') {
    return `Pass rate ${inc.qos_pass_rate_percent.toFixed(1)}% (from updates)`
  }
  return '—'
}

function resolutionLabel(r: string) {
  if (r === 'automatic') return 'Auto (probe recovered)'
  if (r === 'manual') return 'Manual (admin)'
  if (r === 'unknown') return 'Unknown'
  return '—'
}

function isClosedIncidentStatus(status: string) {
  return status === 'auto_resolved' || status === 'manually_resolved' || status === 'resolved'
}

function pastIncidentStatusPillLabel(inc: IncidentDisplayItem) {
  if (!isClosedIncidentStatus(inc.status)) return inc.status
  if (inc.resolution === 'manual') return 'Manually resolved'
  if (inc.resolution === 'automatic') return 'Recovered'
  return inc.status.replaceAll('_', ' ')
}

function pastIncidentStatusPillClass(inc: IncidentDisplayItem) {
  if (isClosedIncidentStatus(inc.status)) return 'status-pill--resolved'
  return 'status-pill--' + inc.status
}

function pastIncidentUpdateStatusLabel(status: string) {
  if (status === 'auto_resolved') return 'auto resolved'
  if (status === 'manually_resolved') return 'manually resolved'
  return status
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

function formatDateTime(iso: string) {
  return new Date(iso).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<style scoped>
.past-incident {
  border-left: 3px solid var(--color-orange);
  background: var(--color-bg);
  border: 1px solid var(--color-border-strong);
  border-radius: 0 var(--radius) var(--radius) 0;
  box-shadow: var(--shadow-card);
  overflow: hidden;
}

.past-incident__header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  width: 100%;
  padding: 12px 16px;
  margin: 0;
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: none;
  font: inherit;
  color: inherit;
}

.past-incident__header:hover {
  background: color-mix(in srgb, var(--color-text-primary) 4%, transparent);
}

.past-incident__header-main {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1 1 200px;
  min-width: 0;
}

.past-incident__chev {
  flex-shrink: 0;
  color: var(--color-text-secondary);
}

.past-incident__title {
  font-weight: 600;
  font-size: 0.9rem;
}

.severity--major {
  color: var(--color-red);
}

.severity--minor {
  color: var(--color-orange);
}

.past-incident__meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  font-size: 0.8rem;
  color: var(--color-text-secondary);
}

.past-incident__target {
  font-weight: 500;
  color: var(--color-text-primary);
  max-width: 220px;
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.past-incident__date {
  font-size: 0.75rem;
}

.past-incident__status {
  font-size: 0.75rem;
  text-transform: capitalize;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--color-bg-muted, #f1f5f9);
  color: var(--color-text-secondary);
}

.status-pill--resolved {
  background: #dcfce7;
  color: #166534;
}

.status-pill--identified {
  background: #dbeafe;
  color: #1e40af;
}

.status-pill--monitoring {
  background: #e0e7ff;
  color: #3730a3;
}

.status-pill--investigating {
  background: #fef3c7;
  color: #92400e;
}

.past-incident__panel {
  padding: 0 16px 14px 44px;
  border-top: 1px solid var(--color-border);
  background: color-mix(in srgb, var(--color-text-primary) 2%, var(--color-bg));
}

.past-incident__dl {
  display: grid;
  grid-template-columns: 7rem 1fr;
  gap: 6px 12px;
  margin: 12px 0;
  font-size: 0.85rem;
}

.past-incident__dl dt {
  margin: 0;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.past-incident__dl dd {
  margin: 0;
  line-height: 1.4;
}

.past-incident__slug {
  display: block;
  font-family: ui-monospace, monospace;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin-top: 2px;
}

.kind-tag {
  display: inline-block;
  margin-right: 6px;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 0.65rem;
  font-weight: 700;
  text-transform: uppercase;
}

.kind--service {
  background: #e0f2fe;
  color: #0369a1;
}

.kind--environment {
  background: #f3e8ff;
  color: #7e22ce;
}

.past-incident__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.chip {
  font-size: 0.65rem;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--color-bg-muted, #f1f5f9);
  color: var(--color-text-secondary);
  font-weight: 600;
}

.past-incident__muted {
  color: var(--color-text-secondary);
  font-size: 0.85rem;
}

.past-incident__updates-loading,
.past-incident__updates-err {
  margin-top: 8px;
  font-size: 0.85rem;
}

.past-incident__updates-err {
  color: var(--color-red, #b91c1c);
}

.past-incident__timeline {
  list-style: none;
  margin: 12px 0 0;
  padding: 0;
  border-left: 2px solid var(--color-border-strong);
}

.past-incident__tl-item {
  margin-left: 10px;
  padding: 0 0 12px 12px;
}

.past-incident__tl-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 10px;
  align-items: center;
  margin-bottom: 4px;
  font-size: 0.75rem;
}

.past-incident__tl-author {
  font-weight: 600;
}

.past-incident__tl-status {
  text-transform: capitalize;
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.past-incident__tl-msg {
  margin: 0;
  font-size: 0.85rem;
  line-height: 1.45;
  white-space: pre-wrap;
}
</style>
