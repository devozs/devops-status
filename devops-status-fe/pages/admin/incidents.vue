<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Incidents</h1>
    </div>

    <AdminCallout variant="info" class="incidents-callout">
      <p>
        <strong>Target</strong> is either a <strong>service</strong> or an <strong>environment</strong>.
        <strong>Infra</strong> shows the linked service provider (services) or Kubernetes cluster (environments).
        <strong>QoS / probe</strong> uses stored degradation data when present; older rows may show pass rate parsed from system updates.
        Open <strong>Timeline</strong> for the full update history.
      </p>
    </AdminCallout>

    <div class="table-panel">
      <div class="table-split-scroll">
        <table class="data-table data-table--head">
          <colgroup>
            <col style="width: 16%" />
            <col style="width: 12%" />
            <col style="width: 12%" />
            <col style="width: 14%" />
            <col style="width: 10%" />
            <col style="width: 8%" />
            <col style="width: 10%" />
            <col style="width: 10%" />
            <col style="width: 8%" />
          </colgroup>
          <thead>
            <tr>
              <th>Title</th>
              <th>Target</th>
              <th>Infra</th>
              <th>QoS / probe</th>
              <th>Telemetry</th>
              <th>Status</th>
              <th>Resolution</th>
              <th>Started</th>
              <th class="th-actions">Actions</th>
            </tr>
          </thead>
        </table>
        <div class="table-body-shell">
          <table class="data-table data-table--body">
            <colgroup>
              <col style="width: 16%" />
              <col style="width: 12%" />
              <col style="width: 12%" />
              <col style="width: 14%" />
              <col style="width: 10%" />
              <col style="width: 8%" />
              <col style="width: 10%" />
              <col style="width: 10%" />
              <col style="width: 8%" />
            </colgroup>
            <tbody>
              <tr v-for="inc in incidents" :key="inc.id">
                <td class="cell-title">{{ inc.title }}</td>
                <td>
                  <span class="kind-badge" :class="'kind--' + inc.target_type">{{ targetKindLabel(inc.target_type) }}</span>
                  <div class="cell-strong">{{ inc.target_name || '—' }}</div>
                  <div v-if="inc.target_slug" class="table-cell-muted cell-mono">{{ inc.target_slug }}</div>
                </td>
                <td class="cell-infra">{{ formatInfra(inc) }}</td>
                <td class="cell-qos">{{ formatQoSLine(inc) }}</td>
                <td>
                  <div v-if="inc.linked_telemetry?.length" class="telemetry-chips">
                    <span v-for="t in inc.linked_telemetry" :key="t.id" class="chip" :title="t.display_name || t.name">{{ t.adapter }}</span>
                  </div>
                  <span v-else class="table-cell-muted">—</span>
                </td>
                <td>
                  <span class="status-badge" :class="'status--' + inc.status">{{ incidentStatusDisplay(inc) }}</span>
                </td>
                <td>
                  <span v-if="inc.resolution === 'open'" class="table-cell-muted">—</span>
                  <span v-else class="resolution-badge" :class="'resolution--' + inc.resolution">{{ resolutionLabel(inc.resolution) }}</span>
                </td>
                <td class="table-cell-muted">{{ formatDate(inc.started_at) }}</td>
                <td class="cell-actions">
                  <button type="button" class="btn btn-sm" @click="openDetail(inc)">Timeline</button>
                  <button type="button" class="btn btn-sm" @click="openUpdate(inc)">Update</button>
                </td>
              </tr>
              <tr v-if="!incidents?.length">
                <td colspan="9" class="empty-row">No incidents.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <div v-if="detail" class="modal-overlay" @click.self="detail = null">
      <div class="modal-card modal-card--wide">
        <h2 class="modal-title">Incident timeline</h2>
        <p class="modal-subtitle">
          {{ detail.incident.target_name }} · {{ targetKindLabel(detail.incident.target_type) }} · {{ formatInfra(detail.incident) }}
        </p>
        <div class="detail-section">
          <h3 class="modal-section-title">Context</h3>
          <ul class="detail-list">
            <li><strong>QoS / probe:</strong> {{ formatQoSLine(detail.incident) }}</li>
            <li v-if="detail.incident.source_telemetry_id">
              <strong>Source telemetry id:</strong> <span class="cell-mono">{{ detail.incident.source_telemetry_id }}</span>
            </li>
            <li><strong>Resolution:</strong> {{ resolutionLabel(detail.incident.resolution) }}</li>
          </ul>
        </div>
        <div class="detail-section">
          <h3 class="modal-section-title">Updates</h3>
          <ol class="timeline">
            <li v-for="u in detailUpdatesChrono" :key="u.id" class="timeline-item">
              <div class="timeline-meta">
                <span class="timeline-author">{{ u.author }}</span>
                <span class="table-cell-muted">{{ formatDate(u.created_at) }}</span>
                <span class="status-badge status-badge--sm" :class="'status--' + u.status">{{ incidentUpdateStatusDisplay(u.status) }}</span>
              </div>
              <p class="timeline-msg">{{ u.message }}</p>
            </li>
          </ol>
        </div>
        <div class="modal-actions">
          <button type="button" class="btn" @click="detail = null">Close</button>
        </div>
      </div>
    </div>

    <div v-if="editing" class="modal-overlay" @click.self="requestCloseIncidentEdit">
      <div class="modal-card">
        <div class="modal-card__header">
          <h2 class="modal-title">Update Incident</h2>
          <button type="button" class="modal-card__close" aria-label="Close" @click="requestCloseIncidentEdit">
            <X :size="20" :stroke-width="2" />
          </button>
        </div>
        <p class="modal-subtitle">Change status and add a public message for the status page.</p>
        <form class="modal-form" @submit.prevent="submitUpdate">
          <h3 class="modal-section-title">Details</h3>
          <div class="form-group">
            <label class="form-label">Status</label>
            <AdminSelect
              v-model="updateForm.status"
              aria-label="Incident status"
              :options="[...incidentStatusSelectOptions]"
            />
          </div>
          <div class="form-group">
            <label class="form-label">Message</label>
            <textarea v-model="updateForm.message" class="form-input form-input--multiline" rows="3" />
          </div>
          <div class="modal-actions">
            <button type="button" class="btn btn-modal-cancel" @click="requestCloseIncidentEdit">Cancel</button>
            <button type="submit" class="btn btn-primary btn-pill">Submit</button>
          </div>
        </form>
      </div>
    </div>

    <AdminUnsavedConfirmDialog
      v-model="unsavedDialogOpen"
      :title="unsavedDialogTitle"
      :warning="unsavedDialogMessage"
      :confirm-label="unsavedConfirmLabel"
      :discard-confirm="confirmUnsavedDialog"
      @cancel="cancelUnsavedDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { X } from 'lucide-vue-next'
import type { AdminIncidentDetailResponse, AdminIncidentListItem, IncidentDegradation, IncidentUpdate } from '~/types/admin-incidents'

definePageMeta({ layout: 'admin' })
const { apiFetch } = useApi()
const {
  requestClose: requestModalClose,
  unsavedDialogOpen,
  unsavedDialogMessage,
  unsavedDialogTitle,
  unsavedConfirmLabel,
  confirmUnsavedDialog,
  cancelUnsavedDialog,
} = useModalUnsavedGuard()

const incidents = ref<AdminIncidentListItem[]>([])
const editing = ref<AdminIncidentListItem | null>(null)
const updateForm = ref({ status: '', message: '' })
const incidentUpdateBaseline = ref('')

const incidentStatusSelectOptions = [
  { value: 'investigating', label: 'Investigating' },
  { value: 'identified', label: 'Identified' },
  { value: 'monitoring', label: 'Monitoring' },
  { value: 'manually_resolved', label: 'Manually resolved' },
  { value: 'auto_resolved', label: 'Recovered (automatic)' },
] as const

const detail = ref<{ incident: AdminIncidentListItem; updates: IncidentUpdate[] } | null>(null)

const detailUpdatesChrono = computed(() => {
  if (!detail.value) return []
  return [...detail.value.updates].sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
  )
})

async function load() {
  incidents.value = await apiFetch<AdminIncidentListItem[]>('/api/admin/incidents')
}

function targetKindLabel(t: string) {
  if (t === 'service') return 'Service'
  if (t === 'environment') return 'Environment'
  return t
}

function formatInfra(inc: AdminIncidentListItem) {
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

function formatQoSLine(inc: AdminIncidentListItem) {
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

function incidentStatusDisplay(inc: AdminIncidentListItem) {
  if (inc.status === 'auto_resolved') return 'Recovered'
  if (inc.status === 'manually_resolved') return 'Manually resolved'
  return inc.status
}

function incidentUpdateStatusDisplay(status: string) {
  if (status === 'auto_resolved') return 'auto resolved'
  if (status === 'manually_resolved') return 'manually resolved'
  return status
}

async function openDetail(inc: AdminIncidentListItem) {
  const data = await apiFetch<AdminIncidentDetailResponse>(`/api/admin/incidents/${inc.id}`)
  detail.value = { incident: data.incident, updates: data.updates }
}

function captureIncidentUpdateSnapshot(): string {
  return JSON.stringify({
    status: updateForm.value.status,
    message: updateForm.value.message,
  })
}

function openUpdate(inc: AdminIncidentListItem) {
  editing.value = inc
  updateForm.value = { status: inc.status, message: '' }
  incidentUpdateBaseline.value = captureIncidentUpdateSnapshot()
}

function performCloseIncidentEdit() {
  editing.value = null
  updateForm.value = { status: '', message: '' }
}

function requestCloseIncidentEdit() {
  requestModalClose(performCloseIncidentEdit, {
    isDirty: () => captureIncidentUpdateSnapshot() !== incidentUpdateBaseline.value,
    message: 'You have unsaved changes to this incident update. Discard them?',
  })
}

async function submitUpdate() {
  if (!editing.value) return
  await apiFetch(`/api/admin/incidents/${editing.value.id}`, { method: 'PUT', body: JSON.stringify(updateForm.value) })
  performCloseIncidentEdit()
  await load()
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

onMounted(load)
</script>

<style scoped>
.incidents-callout {
  margin-bottom: 1rem;
}

.cell-title {
  font-weight: 500;
}

.cell-strong {
  font-weight: 600;
  margin-top: 0.25rem;
}

.cell-mono {
  font-family: ui-monospace, monospace;
  font-size: 0.75rem;
}

.cell-infra,
.cell-qos {
  font-size: 0.875rem;
  line-height: 1.35;
}

.th-actions,
.cell-actions {
  text-align: right;
  white-space: nowrap;
}

.cell-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  justify-content: flex-end;
}

.kind-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 0.65rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.kind--service {
  background: #e0f2fe;
  color: #0369a1;
}

.kind--environment {
  background: #f3e8ff;
  color: #7e22ce;
}

.telemetry-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
}

.chip {
  font-size: 0.65rem;
  padding: 2px 6px;
  border-radius: 4px;
  background: #f1f5f9;
  color: #475569;
  font-weight: 600;
}

.resolution-badge {
  font-size: 0.75rem;
  font-weight: 600;
}

.resolution--automatic {
  color: #166534;
}

.resolution--manual {
  color: #1e40af;
}

.resolution--unknown {
  color: #92400e;
}

.status-badge {
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 0.75rem;
  font-weight: 600;
}

.status-badge--sm {
  font-size: 0.65rem;
  padding: 1px 6px;
}

.status--investigating {
  background: #fef3c7;
  color: #92400e;
}

.status--identified {
  background: #dbeafe;
  color: #1e40af;
}

.status--monitoring {
  background: #e0e7ff;
  color: #3730a3;
}

.status--resolved,
.status--auto_resolved,
.status--manually_resolved {
  background: #dcfce7;
  color: #166534;
}

.status--update {
  background: #f1f5f9;
  color: #475569;
}

.modal-card--wide {
  max-width: 42rem;
}

.detail-section {
  margin-bottom: 1.25rem;
}

.detail-list {
  margin: 0;
  padding-left: 1.2rem;
  font-size: 0.9rem;
  line-height: 1.5;
}

.timeline {
  list-style: none;
  margin: 0;
  padding: 0;
  border-left: 2px solid #e2e8f0;
}

.timeline-item {
  margin-left: 0.75rem;
  padding-bottom: 1rem;
  padding-left: 0.75rem;
}

.timeline-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.35rem;
}

.timeline-author {
  font-weight: 600;
  font-size: 0.8rem;
}

.timeline-msg {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.45;
  white-space: pre-wrap;
}
</style>
