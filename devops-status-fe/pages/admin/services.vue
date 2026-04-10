<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Services</h1>
      <button type="button" class="btn btn-primary btn-pill" @click="startCreate">
        <Plus class="btn-leading-icon" :size="18" :stroke-width="2" />
        Add service
      </button>
    </div>

    <div class="table-panel">
      <div class="table-split-scroll">
        <table class="data-table data-table--head">
          <colgroup>
            <col style="width: 34%" />
            <col style="width: 16%" />
            <col style="width: 12%" />
            <col style="width: 38%" />
          </colgroup>
          <thead>
            <tr>
              <th>Service</th>
              <th>Criticality</th>
              <th>Public</th>
              <th class="th-actions">Actions</th>
            </tr>
          </thead>
        </table>
        <div class="table-body-shell">
          <table class="data-table data-table--body">
            <colgroup>
              <col style="width: 34%" />
              <col style="width: 16%" />
              <col style="width: 12%" />
              <col style="width: 38%" />
            </colgroup>
            <tbody>
              <tr v-for="svc in services" :key="svc.id">
                <td>
                  <div class="table-resource-cell">
                    <div class="table-resource-icon" aria-hidden="true">
                      <Cpu :size="16" :stroke-width="2" />
                    </div>
                    <div class="table-resource-body">
                      <span class="table-resource-title">{{ svc.name }}</span>
                      <span class="table-resource-meta">{{ svc.slug }}</span>
                    </div>
                  </div>
                </td>
                <td>
                  <span class="status-pill" :class="criticalityPillClass(svc.criticality)">{{ svc.criticality }}</span>
                </td>
                <td>
                  <span class="status-pill" :class="svc.is_public ? 'status-pill--success' : 'status-pill--neutral'">
                    {{ svc.is_public ? 'Yes' : 'No' }}
                  </span>
                </td>
                <td class="table-row-actions">
                  <button type="button" class="btn btn-sm" @click="editService(svc)">Edit</button>
                  <button type="button" class="btn btn-sm" @click="openHistory(svc)">History</button>
                  <button type="button" class="btn btn-sm btn-danger" @click="requestDeleteService(svc)">Delete</button>
                </td>
              </tr>
              <tr v-if="!services?.length">
                <td colspan="4" class="empty-row">No services configured.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <AdminDeleteConfirmDialog
      v-model="deleteOpen"
      :title="deleteForm.title"
      :warning="deleteForm.warning"
      :resource-kind="deleteForm.resourceKind"
      :resource-name="deleteForm.resourceName"
      :require-name-match="deleteForm.requireNameMatch"
      :confirm-label="deleteForm.confirmLabel"
      @confirm="onDeleteConfirm"
      @cancel="onDeleteDialogCancel"
    />

    <div v-if="historySvc" class="modal-overlay" @click.self="closeHistory">
      <div class="modal-card modal-card--wide history-panel" role="dialog" aria-labelledby="svc-history-title">
        <h2 id="svc-history-title" class="modal-title">Probe history — {{ historySvc.name }}</h2>
        <p class="modal-subtitle muted">Recent samples per linked HTTP, Prometheus, or CLI telemetry (operational probes).</p>
        <div v-if="historyLoading && !historyLinks.length" class="history-state">Loading links…</div>
        <div v-else-if="!historyLinks.length" class="history-state">No telemetry links. Attach probes in the service editor.</div>
        <template v-else>
          <div class="history-tabs" role="tablist">
            <button
              v-for="link in historyLinks"
              :key="link.telemetry_id"
              type="button"
              role="tab"
              class="history-tab"
              :class="{ 'history-tab--active': historyTabTelemetryId === link.telemetry_id }"
              :aria-selected="historyTabTelemetryId === link.telemetry_id"
              @click="selectHistoryTab(link.telemetry_id)"
            >
              {{ telemetryName(link.telemetry_id) }}
            </button>
          </div>
          <div v-if="historySamplesLoading" class="history-state">Loading samples…</div>
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
                <template v-for="s in historySamples" :key="s.id">
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
                        v-if="historyTelemetryAdapter() === 'cli' && sampleHasCliLogs(s)"
                        type="button"
                        class="btn btn-sm history-cli-toggle"
                        @click="toggleHistoryCliExpand(s.id)"
                      >
                        {{ historyCliExpanded[s.id] ? 'Hide CLI logs' : 'CLI logs' }}
                      </button>
                    </td>
                  </tr>
                  <tr v-if="historyTelemetryAdapter() === 'cli' && historyCliExpanded[s.id] && sampleHasCliLogs(s)" class="history-cli-detail-row">
                    <td colspan="4">
                      <pre class="history-cli-pre">{{ formatCliSampleDetail(s) }}</pre>
                    </td>
                  </tr>
                </template>
                <tr v-if="!historySamples.length">
                  <td colspan="4" class="empty-row">No samples with per-telemetry tagging yet, or none in range.</td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>
        <div class="modal-actions">
          <button type="button" class="btn" @click="closeHistory">Close</button>
        </div>
      </div>
    </div>

    <div v-if="showCreate || editing" class="modal-overlay" @click.self="closeModal">
      <div class="modal-card modal-card--wide">
        <h2 class="modal-title">{{ editing ? 'Edit service' : 'Create service' }}</h2>
        <p v-if="!editing" class="modal-subtitle">
          Set name and slug (slug must be unique), then save. You can attach HTTP, Prometheus, or CLI telemetry next.
        </p>
        <form class="modal-form" @submit.prevent="saveService">
          <h3 class="modal-section-title">General</h3>
          <div class="form-group">
            <label class="form-label">Name<span class="form-label-required" aria-hidden="true">*</span></label>
            <input v-model="form.name" class="form-input" required autocomplete="off" />
          </div>
          <div class="form-group form-group--slug">
            <label class="form-label">Slug<span class="form-label-required" aria-hidden="true">*</span></label>
            <div class="slug-field-row">
              <input
                v-model="form.slug"
                class="form-input"
                :class="slugFieldClass"
                required
                minlength="3"
                autocomplete="off"
              />
              <span v-if="slugAvail === 'loading'" class="name-status name-status--loading">Checking…</span>
              <span
                v-else-if="slugTrimmed.length >= 3 && slugMatchesBaseline"
                class="name-status name-status--ok"
              >Current</span>
              <span
                v-else-if="slugTrimmed.length >= 3 && slugAvail === 'available'"
                class="name-status name-status--ok"
              >Available</span>
              <span
                v-else-if="slugTrimmed.length >= 3 && slugAvail === 'taken'"
                class="name-status name-status--bad"
              >In use</span>
              <span
                v-else-if="slugTrimmed.length >= 3 && slugAvail === 'error'"
                class="name-status name-status--err"
              >Check failed</span>
            </div>
            <p v-if="slugTrimmed.length > 0 && slugTrimmed.length < 3" class="hint name-avail-hint">
              Slug must be at least 3 characters to check availability.
            </p>
          </div>
          <div class="form-group">
            <label class="form-label">Description</label>
            <textarea v-model="form.description" class="form-input form-input--multiline" rows="2" />
          </div>
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">Criticality</label>
              <select v-model="form.criticality" class="form-input">
                <option value="low">Low</option>
                <option value="standard">Standard</option>
                <option value="critical">Critical</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">
                <input v-model="form.is_public" type="checkbox" /> Public
              </label>
            </div>
          </div>
          <div class="form-group">
            <label class="form-label">Service provider (host / port)</label>
            <select v-model="form.service_provider_id" class="form-input">
              <option value="">None</option>
              <option v-for="p in providers" :key="p.id" :value="p.id">{{ providerLabel(p) }}</option>
            </select>
            <p class="form-hint">
              <NuxtLink to="/admin/service-providers">Manage service providers</NuxtLink>
            </p>
          </div>

          <template v-if="editing">
            <h3 class="modal-section-title">Service telemetry</h3>
            <p class="form-hint">Link HTTP, Prometheus, or CLI telemetry for this service.</p>
            <TelemetryLinkSchedulingHelp />
            <table class="mini-table">
              <thead>
                <tr>
                  <th>Telemetry</th>
                  <th title="Seconds between scheduled probe runs for this link">Interval</th>
                  <th title="Number of recent operational results considered for up/down">Window</th>
                  <th title="Consecutive failures required before marking down">Fail→down</th>
                  <th title="Consecutive successes required before marking recovered">Success→up</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                <tr v-for="link in svcLinks" :key="link.id">
                  <td>{{ telemetryName(link.telemetry_id) }}</td>
                  <td><input v-model.number="linkEdit[link.id].sample_interval_sec" type="number" class="form-input form-input--xs" min="1" /></td>
                  <td><input v-model.number="linkEdit[link.id].window_size" type="number" class="form-input form-input--xs" min="1" /></td>
                  <td><input v-model.number="linkEdit[link.id].consecutive_failures_to_down" type="number" class="form-input form-input--xs" min="1" /></td>
                  <td><input v-model.number="linkEdit[link.id].consecutive_success_to_recover" type="number" class="form-input form-input--xs" min="1" /></td>
                  <td class="mini-table__actions">
                    <button type="button" class="btn btn-xs" @click="patchSvcLink(link.id)">Save</button>
                    <button type="button" class="btn btn-xs btn-danger" @click="requestRemoveSvcLink(link.id)">Remove</button>
                  </td>
                </tr>
              </tbody>
            </table>
            <div class="add-link-row">
              <select v-model="newLink.telemetry_id" class="form-input">
                <option value="">Add telemetry…</option>
                <option v-for="t in serviceTelemetry" :key="t.id" :value="t.id">{{ t.name }} ({{ t.adapter }})</option>
              </select>
              <button type="button" class="btn btn-sm btn-primary" :disabled="!newLink.telemetry_id" @click="addSvcLink">Add</button>
            </div>
          </template>

          <p v-if="formError" class="field-error">{{ formError }}</p>

          <div class="modal-actions">
            <button type="button" class="btn" @click="closeModal">Cancel</button>
            <button type="submit" class="btn btn-primary" :disabled="!canSaveGeneral || saving">
              {{ saving ? 'Saving…' : 'Save' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Cpu, Plus } from 'lucide-vue-next'

definePageMeta({ layout: 'admin' })

const { apiFetch } = useApi()

function criticalityPillClass(c: string) {
  if (c === 'critical') return 'status-pill--critical'
  if (c === 'low') return 'status-pill--muted'
  return 'status-pill--neutral'
}

type SvcRow = {
  id: string
  name: string
  slug: string
  description?: string
  criticality: string
  is_public: boolean
  service_provider_id?: string | null
}

type SvcLink = {
  id: string
  service_id: string
  telemetry_id: string
  sample_interval_sec: number
  window_size: number
  consecutive_failures_to_down: number
  consecutive_success_to_recover: number
}

type TelemetrySampleRow = {
  id: string
  sampled_at: string
  success: boolean
  latency_ms?: number | null
  source_trace?: string
  metadata_json?: Record<string, unknown>
}

type Provider = {
  id: string
  name: string
  host: string
  port: number
  image_url?: string
  provider_type?: 'tcp' | 'prometheus'
  config_json?: { endpoint?: string }
}

const services = ref<SvcRow[]>([])
const historySvc = ref<SvcRow | null>(null)
const historyLinks = ref<SvcLink[]>([])
const historyTabTelemetryId = ref('')
const historyLoading = ref(false)
const historySamplesLoading = ref(false)
const historySamples = ref<TelemetrySampleRow[]>([])
/** Expanded rows for CLI sample metadata (key = sample id). */
const historyCliExpanded = ref<Record<string, boolean>>({})
const showCreate = ref(false)
const editing = ref<SvcRow | null>(null)
const saving = ref(false)
const formError = ref('')
/** Slug loaded when modal opened (edit) or after create; unchanged slug skips remote check. */
const loadedSlugBaseline = ref('')
const slugAvail = ref<'idle' | 'loading' | 'available' | 'taken' | 'error'>('idle')
let slugCheckTimer: ReturnType<typeof setTimeout> | null = null
let slugCheckSeq = 0

const form = ref({
  name: '',
  slug: '',
  description: '',
  criticality: 'standard',
  is_public: true,
  service_provider_id: '' as string,
})

const providers = ref<Provider[]>([])
const allTelemetry = ref<{ id: string; name: string; display_name?: string; adapter: string }[]>([])
const svcLinks = ref<SvcLink[]>([])
const linkEdit = ref<Record<string, {
  telemetry_id: string
  sample_interval_sec: number
  window_size: number
  consecutive_failures_to_down: number
  consecutive_success_to_recover: number
}>>({})
const newLink = ref({ telemetry_id: '' })

const deleteOpen = ref(false)
const deleteForm = ref({
  title: '',
  warning: '',
  resourceKind: 'resource',
  resourceName: '',
  requireNameMatch: true,
  confirmLabel: 'Delete',
})
const deleteAction = ref<null | (() => Promise<void>)>(null)

function onDeleteDialogCancel() {
  deleteAction.value = null
}

async function onDeleteConfirm() {
  const fn = deleteAction.value
  if (!fn) {
    deleteOpen.value = false
    return
  }
  try {
    await fn()
    deleteOpen.value = false
    deleteAction.value = null
  } catch {
    /* keep dialog open */
  }
}

function requestDeleteService(svc: SvcRow) {
  deleteForm.value = {
    title: 'Delete service',
    warning:
      'This service will be permanently deleted. Telemetry links and scheduled probes for it will stop. This action cannot be undone.',
    resourceKind: 'service',
    resourceName: svc.name,
    requireNameMatch: true,
    confirmLabel: 'Delete service',
  }
  deleteAction.value = async () => {
    await apiFetch(`/api/admin/services/${svc.id}`, { method: 'DELETE' })
    await loadServices()
  }
  deleteOpen.value = true
}

function requestRemoveSvcLink(linkId: string) {
  if (!editing.value) return
  const svcId = editing.value.id
  deleteForm.value = {
    title: 'Remove telemetry link',
    warning:
      'This removes the binding between this service and the telemetry probe. Scheduled probes for this link will stop. The telemetry definition is not deleted.',
    resourceKind: 'link',
    resourceName: '',
    requireNameMatch: false,
    confirmLabel: 'Remove link',
  }
  deleteAction.value = async () => {
    await apiFetch(`/api/admin/services/${svcId}/telemetry-links/${linkId}`, { method: 'DELETE' })
    const detail = await apiFetch<{ telemetry_links?: SvcLink[] }>(`/api/admin/services/${svcId}`)
    svcLinks.value = detail.telemetry_links || []
    syncLinkEdit()
  }
  deleteOpen.value = true
}

const slugTrimmed = computed(() => form.value.slug.trim())

const slugMatchesBaseline = computed(
  () =>
    !!editing.value
    && loadedSlugBaseline.value !== ''
    && slugTrimmed.value === loadedSlugBaseline.value,
)

const slugFieldClass = computed(() => {
  if (slugTrimmed.value.length < 3) return ''
  if (slugMatchesBaseline.value) return 'form-input--name-ok'
  if (slugAvail.value === 'available') return 'form-input--name-ok'
  if (slugAvail.value === 'taken') return 'form-input--name-bad'
  if (slugAvail.value === 'error') return 'form-input--name-warn'
  return ''
})

function validateGeneralFields(): string {
  if (!form.value.name.trim()) return 'Name is required'
  if (!slugTrimmed.value) return 'Slug is required'
  if (slugTrimmed.value.length < 3) return 'Slug must be at least 3 characters'
  return ''
}

const canSaveGeneral = computed(() => {
  if (validateGeneralFields() !== '') return false
  if (slugAvail.value === 'loading') return false
  if (slugMatchesBaseline.value) return true
  if (slugTrimmed.value.length < 3) return false
  return slugAvail.value === 'available'
})

async function runSlugAvailabilityCheck() {
  const s = slugTrimmed.value
  if (s.length < 3) {
    slugAvail.value = 'idle'
    return
  }
  if (editing.value && s === loadedSlugBaseline.value) {
    slugAvail.value = 'idle'
    return
  }
  const seq = ++slugCheckSeq
  slugAvail.value = 'loading'
  try {
    const params = new URLSearchParams({ slug: s })
    const ex = editing.value?.id?.trim()
    if (ex) params.set('exclude_id', ex)
    const res = (await apiFetch(`/api/admin/services/check-slug?${params.toString()}`)) as {
      available?: boolean
    }
    if (seq !== slugCheckSeq) return
    slugAvail.value = res.available === true ? 'available' : 'taken'
  } catch {
    if (seq !== slugCheckSeq) return
    slugAvail.value = 'error'
  }
}

function scheduleSlugAvailabilityCheck() {
  if (slugCheckTimer) clearTimeout(slugCheckTimer)
  const s = slugTrimmed.value
  if (s.length < 3) {
    slugAvail.value = 'idle'
    return
  }
  if (editing.value && s === loadedSlugBaseline.value) {
    slugAvail.value = 'idle'
    return
  }
  slugAvail.value = 'loading'
  slugCheckTimer = setTimeout(() => {
    slugCheckTimer = null
    void runSlugAvailabilityCheck()
  }, 320)
}

watch(() => form.value.slug, scheduleSlugAvailabilityCheck)

function resetSlugValidation() {
  if (slugCheckTimer) {
    clearTimeout(slugCheckTimer)
    slugCheckTimer = null
  }
  slugCheckSeq++
  slugAvail.value = 'idle'
  loadedSlugBaseline.value = ''
}

const serviceTelemetry = computed(() =>
  allTelemetry.value.filter((d) => ['http', 'prometheus', 'cli'].includes(d.adapter)),
)

function providerLabel(p: Provider) {
  const n = p.name.trim()
  if (p.provider_type === 'prometheus') {
    const ep = (p.config_json?.endpoint || '').trim()
    return ep ? `${n} — ${ep}` : `${n} — Prometheus`
  }
  return `${n} — ${p.host}:${p.port}`
}

function telemetryName(id: string) {
  const t = allTelemetry.value.find((d) => d.id === id)
  if (!t) return id
  const d = t.display_name?.trim()
  return d || t.name
}

function formatSampleTime(iso: string) {
  try {
    return new Date(iso).toISOString().replace('T', ' ').slice(0, 19)
  } catch {
    return iso
  }
}

function traceSnippet(trace: string | undefined) {
  const t = (trace || '').trim()
  if (!t) return '—'
  return t.length > 400 ? `${t.slice(0, 400)}…` : t
}

function historyTelemetryAdapter() {
  const tid = historyTabTelemetryId.value
  const t = allTelemetry.value.find((d) => d.id === tid)
  return t?.adapter || ''
}

function sampleHasCliLogs(s: TelemetrySampleRow) {
  const m = s.metadata_json
  if (!m || typeof m !== 'object') return false
  const q = m.cli_qos_attempts
  return Boolean(
    m.cli_stdout ||
      m.cli_stderr ||
      (Array.isArray(q) && q.length > 0),
  )
}

function formatCliSampleDetail(s: TelemetrySampleRow) {
  const m = s.metadata_json
  if (!m || typeof m !== 'object') return ''
  const parts: string[] = []
  const pick = (k: string) => {
    const v = m[k]
    if (v === undefined || v === null || v === '') return
    parts.push(`${k}: ${typeof v === 'string' ? v : JSON.stringify(v)}`)
  }
  pick('cli_runner')
  pick('cli_image')
  pick('job')
  pick('namespace')
  pick('exit_code')
  if (typeof m.cli_stdout === 'string' && m.cli_stdout) {
    parts.push('--- cli_stdout ---\n' + m.cli_stdout)
  }
  if (typeof m.cli_stderr === 'string' && m.cli_stderr) {
    parts.push('--- cli_stderr ---\n' + m.cli_stderr)
  }
  if (Array.isArray(m.cli_qos_attempts) && m.cli_qos_attempts.length) {
    parts.push('--- cli_qos_attempts ---\n' + JSON.stringify(m.cli_qos_attempts, null, 2))
  }
  return parts.join('\n\n')
}

function toggleHistoryCliExpand(id: string) {
  const next = { ...historyCliExpanded.value, [id]: !historyCliExpanded.value[id] }
  historyCliExpanded.value = next
}

async function fetchHistorySamples() {
  const s = historySvc.value
  const tid = historyTabTelemetryId.value
  if (!s || !tid) {
    historySamples.value = []
    return
  }
  historySamplesLoading.value = true
  try {
    const q = new URLSearchParams({ telemetry_id: tid, limit: '50', probe_kind: 'operational' })
    const res = await apiFetch<{ samples?: TelemetrySampleRow[] }>(
      `/api/admin/services/${s.id}/telemetry-samples?${q.toString()}`,
    )
    historySamples.value = res.samples || []
    historyCliExpanded.value = {}
  } catch {
    historySamples.value = []
  } finally {
    historySamplesLoading.value = false
  }
}

async function openHistory(svc: SvcRow) {
  historySvc.value = svc
  historyLinks.value = []
  historySamples.value = []
  historyTabTelemetryId.value = ''
  historyLoading.value = true
  try {
    const detail = await apiFetch<SvcRow & { telemetry_links?: SvcLink[] }>(`/api/admin/services/${svc.id}`)
    historyLinks.value = detail.telemetry_links || []
    const first = historyLinks.value[0]?.telemetry_id
    historyTabTelemetryId.value = first || ''
    if (first) await fetchHistorySamples()
  } catch {
    historyLinks.value = []
  } finally {
    historyLoading.value = false
  }
}

function selectHistoryTab(tid: string) {
  historyTabTelemetryId.value = tid
  void fetchHistorySamples()
}

function closeHistory() {
  historySvc.value = null
  historyLinks.value = []
  historyTabTelemetryId.value = ''
  historySamples.value = []
  historyCliExpanded.value = {}
}

function syncLinkEdit() {
  const m: typeof linkEdit.value = {}
  for (const l of svcLinks.value) {
    m[l.id] = {
      telemetry_id: l.telemetry_id,
      sample_interval_sec: l.sample_interval_sec,
      window_size: l.window_size,
      consecutive_failures_to_down: l.consecutive_failures_to_down,
      consecutive_success_to_recover: l.consecutive_success_to_recover,
    }
  }
  linkEdit.value = m
}

async function loadServices() {
  services.value = await apiFetch<SvcRow[]>('/api/admin/services')
}

async function loadProviders() {
  try {
    providers.value = await apiFetch<Provider[]>('/api/admin/service-providers')
  } catch {
    providers.value = []
  }
}

async function loadTelemetry() {
  try {
    allTelemetry.value = await apiFetch('/api/admin/telemetry')
  } catch {
    allTelemetry.value = []
  }
}

async function editService(svc: SvcRow) {
  showCreate.value = false
  if (slugCheckTimer) {
    clearTimeout(slugCheckTimer)
    slugCheckTimer = null
  }
  slugCheckSeq++
  slugAvail.value = 'idle'
  editing.value = svc
  const detail = await apiFetch<SvcRow & { telemetry_links?: SvcLink[] }>(`/api/admin/services/${svc.id}`)
  loadedSlugBaseline.value = detail.slug.trim()
  form.value = {
    name: detail.name,
    slug: detail.slug,
    description: detail.description || '',
    criticality: detail.criticality,
    is_public: detail.is_public,
    service_provider_id: detail.service_provider_id || '',
  }
  svcLinks.value = detail.telemetry_links || []
  syncLinkEdit()
  newLink.value = { telemetry_id: '' }
}

function startCreate() {
  showCreate.value = true
  editing.value = null
  svcLinks.value = []
  linkEdit.value = {}
  formError.value = ''
  form.value = {
    name: '',
    slug: '',
    description: '',
    criticality: 'standard',
    is_public: true,
    service_provider_id: '',
  }
  resetSlugValidation()
  newLink.value = { telemetry_id: '' }
}

function closeModal() {
  showCreate.value = false
  editing.value = null
  svcLinks.value = []
  linkEdit.value = {}
  formError.value = ''
  form.value = {
    name: '',
    slug: '',
    description: '',
    criticality: 'standard',
    is_public: true,
    service_provider_id: '',
  }
  resetSlugValidation()
}

async function saveService() {
  formError.value = ''
  const v = validateGeneralFields()
  if (v) {
    formError.value = v
    return
  }
  if (!canSaveGeneral.value) {
    formError.value = 'Wait for slug validation or fix slug conflicts.'
    return
  }

  const body = {
    name: form.value.name.trim(),
    slug: form.value.slug.trim(),
    description: form.value.description,
    criticality: form.value.criticality,
    is_public: form.value.is_public,
    service_provider_id: form.value.service_provider_id || null,
  }

  saving.value = true
  try {
    if (editing.value) {
      await apiFetch(`/api/admin/services/${editing.value.id}`, {
        method: 'PUT',
        body: JSON.stringify(body),
      })
      await loadServices()
      closeModal()
    } else {
      const created = (await apiFetch('/api/admin/services', {
        method: 'POST',
        body: JSON.stringify(body),
      })) as SvcRow
      showCreate.value = false
      editing.value = created
      loadedSlugBaseline.value = created.slug.trim()
      slugAvail.value = 'available'
      await loadServices()
      const detail = await apiFetch<SvcRow & { telemetry_links?: SvcLink[] }>(
        `/api/admin/services/${created.id}`,
      )
      form.value = {
        name: detail.name,
        slug: detail.slug,
        description: detail.description || '',
        criticality: detail.criticality,
        is_public: detail.is_public,
        service_provider_id: detail.service_provider_id || '',
      }
      svcLinks.value = detail.telemetry_links || []
      syncLinkEdit()
      newLink.value = { telemetry_id: '' }
    }
  } catch (e: unknown) {
    const msg = (e as { data?: { error?: string } })?.data?.error
    formError.value = msg || (e instanceof Error ? e.message : 'Save failed')
  } finally {
    saving.value = false
  }
}

async function addSvcLink() {
  if (!editing.value || !newLink.value.telemetry_id) return
  await apiFetch(`/api/admin/services/${editing.value.id}/telemetry-links`, {
    method: 'POST',
    body: JSON.stringify({
      telemetry_id: newLink.value.telemetry_id,
      sample_interval_sec: 60,
      window_size: 5,
      consecutive_failures_to_down: 3,
      consecutive_success_to_recover: 2,
    }),
  })
  newLink.value = { telemetry_id: '' }
  const detail = await apiFetch<{ telemetry_links?: SvcLink[] }>(`/api/admin/services/${editing.value.id}`)
  svcLinks.value = detail.telemetry_links || []
  syncLinkEdit()
}

async function patchSvcLink(linkId: string) {
  if (!editing.value) return
  const inp = linkEdit.value[linkId]
  if (!inp) return
  await apiFetch(`/api/admin/services/${editing.value.id}/telemetry-links/${linkId}`, {
    method: 'PATCH',
    body: JSON.stringify(inp),
  })
  const detail = await apiFetch<{ telemetry_links?: SvcLink[] }>(`/api/admin/services/${editing.value.id}`)
  svcLinks.value = detail.telemetry_links || []
  syncLinkEdit()
}

onUnmounted(() => {
  if (slugCheckTimer) clearTimeout(slugCheckTimer)
})

onMounted(async () => {
  await Promise.all([loadServices(), loadProviders(), loadTelemetry()])
})
</script>

<style scoped>
.mini-table { width: 100%; border-collapse: collapse; font-size: 0.8rem; margin-bottom: 12px; }
.mini-table th, .mini-table td { padding: 6px 8px; border-bottom: 1px solid var(--color-border); text-align: left; vertical-align: middle; }
.mini-table__actions {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 6px;
  flex-wrap: nowrap;
  white-space: nowrap;
  width: 1%;
}
.mini-table :deep(.btn-xs) {
  padding: 4px 10px;
  font-size: 0.75rem;
  min-height: 0;
  line-height: 1.2;
}
.add-link-row { display: flex; gap: 8px; align-items: center; margin-bottom: 8px; }
.add-link-row .form-input { flex: 1; max-width: 360px; }
.form-group--slug .slug-field-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.form-group--slug .slug-field-row .form-input {
  flex: 1;
  min-width: 12rem;
}
.name-status {
  font-size: 0.8rem;
  font-weight: 600;
  white-space: nowrap;
}
.name-status--loading { color: var(--color-text-secondary); }
.name-status--ok { color: #15803d; }
.name-status--bad { color: var(--color-red, #b91c1c); }
.name-status--err { color: #b45309; }
.name-avail-hint { margin: 4px 0 0; }
.form-input--name-ok { border-color: #15803d; }
.form-input--name-bad { border-color: var(--color-red, #b91c1c); }
.form-input--name-warn { border-color: #d97706; }
.field-error { color: #b91c1c; font-size: 0.85rem; margin: 0 0 8px; }
.th-actions { white-space: nowrap; }
.table-row-actions {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  vertical-align: middle;
}
.history-panel .modal-subtitle.muted { color: var(--color-text-secondary); margin-top: 4px; }
.history-state { padding: 16px 0; color: var(--color-text-secondary); font-size: 0.9rem; }
.history-tabs { display: flex; flex-wrap: wrap; gap: 8px; margin: 12px 0 8px; }
.history-tab {
  padding: 6px 12px;
  border-radius: var(--radius, 8px);
  border: 1px solid var(--color-border);
  background: var(--color-bg);
  font-size: 0.85rem;
  cursor: pointer;
}
.history-tab:hover { border-color: var(--color-border-strong); }
.history-tab--active {
  border-color: var(--color-primary);
  background: color-mix(in srgb, var(--color-primary) 12%, transparent);
  font-weight: 600;
}
.history-table-wrap { max-height: min(50vh, 420px); overflow: auto; margin-bottom: 8px; }
.history-samples-table { margin-bottom: 0; }
.history-trace-cell { max-width: 280px; }
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
.history-cli-toggle { margin-top: 6px; display: inline-block; }
.history-cli-detail-row td { background: var(--color-bg-muted, rgba(0,0,0,0.04)); vertical-align: top; }
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
