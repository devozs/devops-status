<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Environments</h1>
      <button type="button" class="btn btn-primary btn-pill" @click="startCreate">
        <Plus class="btn-leading-icon" :size="18" :stroke-width="2" />
        Add environment
      </button>
    </div>

    <div class="table-panel">
      <div class="table-split-scroll">
        <table class="data-table data-table--head">
            <colgroup>
              <col style="width: 32%" />
              <col style="width: 12%" />
              <col style="width: 14%" />
              <col style="width: 11%" />
              <col style="width: 31%" />
            </colgroup>
          <thead>
            <tr>
              <th>Environment</th>
              <th>Type</th>
              <th>Criticality</th>
              <th>Public</th>
              <th class="th-actions">Actions</th>
            </tr>
          </thead>
        </table>
        <div class="table-body-shell">
          <table class="data-table data-table--body">
            <colgroup>
              <col style="width: 32%" />
              <col style="width: 12%" />
              <col style="width: 14%" />
              <col style="width: 11%" />
              <col style="width: 31%" />
            </colgroup>
            <tbody>
              <tr v-for="env in environments" :key="env.id">
                <td>
                  <div class="table-resource-cell">
                    <div class="table-resource-icon" aria-hidden="true">
                      <Boxes :size="16" :stroke-width="2" />
                    </div>
                    <div class="table-resource-body">
                      <span class="table-resource-title">{{ env.name }}</span>
                      <span class="table-resource-meta">{{ env.slug }}</span>
                    </div>
                  </div>
                </td>
                <td><span class="status-pill status-pill--neutral">{{ envTypeLabel(env.env_type) }}</span></td>
                <td>
                  <span class="status-pill" :class="criticalityPillClass(env.criticality)">{{ env.criticality }}</span>
                </td>
                <td>
                  <span class="status-pill" :class="env.is_public ? 'status-pill--success' : 'status-pill--neutral'">
                    {{ env.is_public ? 'Yes' : 'No' }}
                  </span>
                </td>
                <td class="table-row-actions">
                  <button type="button" class="btn btn-sm" @click="editEnv(env)">Edit</button>
                  <button type="button" class="btn btn-sm" @click="openHistory(env)">History</button>
                  <button type="button" class="btn btn-sm" @click="openResourceMap(env)">Map</button>
                  <button type="button" class="btn btn-sm btn-danger" @click="requestDeleteEnv(env)">Delete</button>
                </td>
              </tr>
              <tr v-if="!environments?.length">
                <td colspan="5" class="empty-row">No environments configured.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <ResourceTopologyModal
      :open="topologyOpen"
      :title="topologyTitle"
      :loading="topologyLoading"
      :error="topologyError"
      :topology="topologyData"
      :session-key="topologySessionKey"
      @close="closeResourceMap"
    />

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

    <div v-if="historyEnv" class="modal-overlay" @click.self="closeHistory">
      <div class="modal-card modal-card--wide history-panel" role="dialog" aria-labelledby="history-title">
        <h2 id="history-title" class="modal-title">Probe history — {{ historyEnv.name }}</h2>
        <p class="modal-subtitle muted">Recent samples per linked Kubernetes telemetry (operational probes).</p>
        <div v-if="historyLoading && !historyLinks.length" class="history-state">Loading links…</div>
        <div v-else-if="!historyLinks.length" class="history-state">No telemetry links. Attach probes in the environment editor.</div>
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
        <h2 class="modal-title">{{ editing ? 'Edit environment' : 'Create environment' }}</h2>
        <p v-if="!editing" class="modal-subtitle">Set name and slug (slug must be unique), then save. You can attach a cluster and Kubernetes telemetry next.</p>
        <form class="modal-form" @submit.prevent="saveEnv">
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
              <label class="form-label">Type</label>
              <select v-model="form.env_type" class="form-input">
                <option value="dev">Development</option>
                <option value="staging">Staging</option>
                <option value="prod">Production</option>
                <option value="custom">Custom</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">Criticality</label>
              <select v-model="form.criticality" class="form-input">
                <option value="low">Low</option>
                <option value="standard">Standard</option>
                <option value="critical">Critical</option>
              </select>
            </div>
          </div>
          <div v-if="editing" class="form-group">
            <label class="form-label">Primary Kubernetes cluster</label>
            <select v-model="form.k8s_cluster_id" class="form-input">
              <option value="">None</option>
              <option
                v-for="c in clustersForEnv"
                :key="c.id"
                :value="c.id"
              >
                {{ c.name }} ({{ c.status }})
              </option>
            </select>
            <p class="form-hint">
              Connected clusters with API credentials. Unassigned clusters appear here; choosing one attaches it to this environment for probes. Used as the API target for Kubernetes telemetry below.
            </p>
          </div>
          <div class="form-group">
            <label class="form-label">
              <input v-model="form.is_public" type="checkbox" /> Public
            </label>
          </div>

          <template v-if="editing">
            <h3 class="modal-section-title">Kubernetes telemetry</h3>
            <p class="form-hint">Link Kubernetes telemetry definitions to this environment. Scheduling and status thresholds are per link.</p>
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
                <tr v-for="link in envLinks" :key="link.id">
                  <td>{{ telemetryName(link.telemetry_id) }}</td>
                  <td><input v-model.number="linkEdit[link.id].sample_interval_sec" type="number" class="form-input form-input--xs" min="1" /></td>
                  <td><input v-model.number="linkEdit[link.id].window_size" type="number" class="form-input form-input--xs" min="1" /></td>
                  <td><input v-model.number="linkEdit[link.id].consecutive_failures_to_down" type="number" class="form-input form-input--xs" min="1" /></td>
                  <td><input v-model.number="linkEdit[link.id].consecutive_success_to_recover" type="number" class="form-input form-input--xs" min="1" /></td>
                  <td class="mini-table__actions">
                    <button type="button" class="btn btn-xs" @click="patchEnvLink(link.id)">Save</button>
                    <button type="button" class="btn btn-xs btn-danger" @click="requestRemoveEnvLink(link.id)">Remove</button>
                  </td>
                </tr>
              </tbody>
            </table>
            <div class="add-link-row">
              <select v-model="newLink.telemetry_id" class="form-input">
                <option value="">Add telemetry…</option>
                <option v-for="t in k8sTelemetry" :key="t.id" :value="t.id">{{ t.name }}</option>
              </select>
              <button type="button" class="btn btn-sm btn-primary" :disabled="!newLink.telemetry_id" @click="addEnvLink">Add</button>
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
import { Boxes, Plus } from 'lucide-vue-next'
import type { ResourceTopology } from '~/types/resource-topology'

definePageMeta({ layout: 'admin' })

const { apiFetch } = useApi()
const { fetchAdmin: fetchTopologyAdmin } = useResourceTopology()

type ClusterRow = {
  id: string
  name?: string
  environment_id?: string | null
  status: string
  has_api_credential?: boolean
}

function criticalityPillClass(c: string) {
  if (c === 'critical') return 'status-pill--critical'
  if (c === 'low') return 'status-pill--muted'
  return 'status-pill--neutral'
}

function envTypeLabel(t: string) {
  const m: Record<string, string> = {
    dev: 'Development',
    staging: 'Staging',
    prod: 'Production',
    custom: 'Custom',
  }
  return m[t] || t
}

type EnvRow = {
  id: string
  name: string
  slug: string
  description?: string
  env_type: string
  criticality: string
  is_public: boolean
  k8s_cluster_id?: string | null
}

type EnvLink = {
  id: string
  environment_id: string
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

const environments = ref<EnvRow[]>([])
const historyEnv = ref<EnvRow | null>(null)
const topologyOpen = ref(false)
const topologyTitle = ref('')
const topologyLoading = ref(false)
const topologyError = ref('')
const topologyData = ref<ResourceTopology | null>(null)
const topologySessionKey = ref(0)
const historyLinks = ref<EnvLink[]>([])
const historyTabTelemetryId = ref('')
const historyLoading = ref(false)
const historySamplesLoading = ref(false)
const historySamples = ref<TelemetrySampleRow[]>([])
const historyCliExpanded = ref<Record<string, boolean>>({})
const showCreate = ref(false)
const editing = ref<EnvRow | null>(null)
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
  env_type: 'custom',
  criticality: 'standard',
  is_public: true,
  k8s_cluster_id: '' as string,
})

const clusters = ref<ClusterRow[]>([])
const allTelemetry = ref<{ id: string; name: string; display_name?: string; adapter: string }[]>([])
const envLinks = ref<EnvLink[]>([])
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

function requestDeleteEnv(env: EnvRow) {
  deleteForm.value = {
    title: 'Delete environment',
    warning:
      'This environment will be permanently deleted. Kubernetes telemetry links and scheduled probes for it will stop. This action cannot be undone.',
    resourceKind: 'environment',
    resourceName: env.name,
    requireNameMatch: true,
    confirmLabel: 'Delete environment',
  }
  deleteAction.value = async () => {
    await apiFetch(`/api/admin/environments/${env.id}`, { method: 'DELETE' })
    await loadEnvironments()
  }
  deleteOpen.value = true
}

function requestRemoveEnvLink(linkId: string) {
  if (!editing.value) return
  const envId = editing.value.id
  deleteForm.value = {
    title: 'Remove telemetry link',
    warning:
      'This removes the binding between this environment and the telemetry probe. Scheduled probes for this link will stop. The telemetry definition is not deleted.',
    resourceKind: 'link',
    resourceName: '',
    requireNameMatch: false,
    confirmLabel: 'Remove link',
  }
  deleteAction.value = async () => {
    await apiFetch(`/api/admin/environments/${envId}/telemetry-links/${linkId}`, { method: 'DELETE' })
    const detail = await apiFetch<{ telemetry_links?: EnvLink[] }>(`/api/admin/environments/${envId}`)
    envLinks.value = detail.telemetry_links || []
    syncLinkEdit()
  }
  deleteOpen.value = true
}

function clusterUsable(c: ClusterRow): boolean {
  if (c.status !== 'connected') return false
  if (c.has_api_credential === false) return false
  return true
}

const clustersForEnv = computed(() => {
  const envId = editing.value?.id
  if (!envId) return []
  const primaryId = (form.value.k8s_cluster_id || '').trim()
  return clusters.value.filter((c) => {
    const keepAsCurrentPrimary = primaryId !== '' && c.id === primaryId
    if (!clusterUsable(c) && !keepAsCurrentPrimary) return false
    if (keepAsCurrentPrimary) return true
    const eid = c.environment_id
    return !eid || eid === envId
  })
})

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
    const res = (await apiFetch(`/api/admin/environments/check-slug?${params.toString()}`)) as {
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

const k8sTelemetry = computed(() => allTelemetry.value.filter((d) => d.adapter === 'kubernetes'))

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
  historyCliExpanded.value = { ...historyCliExpanded.value, [id]: !historyCliExpanded.value[id] }
}

async function fetchHistorySamples() {
  const e = historyEnv.value
  const tid = historyTabTelemetryId.value
  if (!e || !tid) {
    historySamples.value = []
    return
  }
  historySamplesLoading.value = true
  try {
    const q = new URLSearchParams({ telemetry_id: tid, limit: '50', probe_kind: 'operational' })
    const res = await apiFetch<{ samples?: TelemetrySampleRow[] }>(
      `/api/admin/environments/${e.id}/telemetry-samples?${q.toString()}`,
    )
    historySamples.value = res.samples || []
    historyCliExpanded.value = {}
  } catch {
    historySamples.value = []
  } finally {
    historySamplesLoading.value = false
  }
}

async function openHistory(env: EnvRow) {
  historyEnv.value = env
  historyLinks.value = []
  historySamples.value = []
  historyTabTelemetryId.value = ''
  historyLoading.value = true
  try {
    const detail = await apiFetch<EnvRow & { telemetry_links?: EnvLink[] }>(`/api/admin/environments/${env.id}`)
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
  historyEnv.value = null
  historyLinks.value = []
  historyTabTelemetryId.value = ''
  historySamples.value = []
  historyCliExpanded.value = {}
}

function closeResourceMap() {
  topologyOpen.value = false
  topologyData.value = null
  topologyError.value = ''
}

async function openResourceMap(env: EnvRow) {
  topologySessionKey.value += 1
  topologyTitle.value = env.name
  topologyOpen.value = true
  topologyLoading.value = true
  topologyError.value = ''
  topologyData.value = null
  try {
    topologyData.value = await fetchTopologyAdmin('environment', env.id)
  } catch {
    topologyError.value = 'Could not load resource map.'
  } finally {
    topologyLoading.value = false
  }
}

async function loadEnvironments() {
  environments.value = await apiFetch<EnvRow[]>('/api/admin/environments')
}

async function loadClusters() {
  try {
    clusters.value = await apiFetch('/api/admin/k8s-clusters')
  } catch {
    clusters.value = []
  }
}

async function loadTelemetry() {
  try {
    allTelemetry.value = await apiFetch('/api/admin/telemetry')
  } catch {
    allTelemetry.value = []
  }
}

function syncLinkEdit() {
  const m: typeof linkEdit.value = {}
  for (const l of envLinks.value) {
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

async function editEnv(env: EnvRow) {
  showCreate.value = false
  if (slugCheckTimer) {
    clearTimeout(slugCheckTimer)
    slugCheckTimer = null
  }
  slugCheckSeq++
  slugAvail.value = 'idle'
  editing.value = env
  const detail = await apiFetch<EnvRow & { telemetry_links?: EnvLink[] }>(`/api/admin/environments/${env.id}`)
  loadedSlugBaseline.value = detail.slug.trim()
  form.value = {
    name: detail.name,
    slug: detail.slug,
    description: detail.description || '',
    env_type: detail.env_type,
    criticality: detail.criticality,
    is_public: detail.is_public,
    k8s_cluster_id: detail.k8s_cluster_id || '',
  }
  envLinks.value = detail.telemetry_links || []
  syncLinkEdit()
  newLink.value = { telemetry_id: '' }
}

function startCreate() {
  showCreate.value = true
  editing.value = null
  envLinks.value = []
  linkEdit.value = {}
  formError.value = ''
  form.value = {
    name: '',
    slug: '',
    description: '',
    env_type: 'custom',
    criticality: 'standard',
    is_public: true,
    k8s_cluster_id: '',
  }
  resetSlugValidation()
  newLink.value = { telemetry_id: '' }
}

function closeModal() {
  showCreate.value = false
  editing.value = null
  envLinks.value = []
  linkEdit.value = {}
  formError.value = ''
  form.value = {
    name: '',
    slug: '',
    description: '',
    env_type: 'custom',
    criticality: 'standard',
    is_public: true,
    k8s_cluster_id: '',
  }
  resetSlugValidation()
}

async function saveEnv() {
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
    env_type: form.value.env_type,
    criticality: form.value.criticality,
    is_public: form.value.is_public,
    k8s_cluster_id: form.value.k8s_cluster_id || null,
  }

  saving.value = true
  try {
    if (editing.value) {
      await apiFetch(`/api/admin/environments/${editing.value.id}`, {
        method: 'PUT',
        body: JSON.stringify(body),
      })
      await loadEnvironments()
      await loadClusters()
      closeModal()
    } else {
      const created = (await apiFetch('/api/admin/environments', {
        method: 'POST',
        body: JSON.stringify({ ...body, k8s_cluster_id: null }),
      })) as EnvRow
      showCreate.value = false
      editing.value = created
      loadedSlugBaseline.value = created.slug.trim()
      slugAvail.value = 'available'
      await loadEnvironments()
      await loadClusters()
      const detail = await apiFetch<EnvRow & { telemetry_links?: EnvLink[] }>(
        `/api/admin/environments/${created.id}`,
      )
      form.value = {
        name: detail.name,
        slug: detail.slug,
        description: detail.description || '',
        env_type: detail.env_type,
        criticality: detail.criticality,
        is_public: detail.is_public,
        k8s_cluster_id: detail.k8s_cluster_id || '',
      }
      envLinks.value = detail.telemetry_links || []
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

onUnmounted(() => {
  if (slugCheckTimer) clearTimeout(slugCheckTimer)
})

async function addEnvLink() {
  if (!editing.value || !newLink.value.telemetry_id) return
  await apiFetch(`/api/admin/environments/${editing.value.id}/telemetry-links`, {
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
  const detail = await apiFetch<{ telemetry_links?: EnvLink[] }>(`/api/admin/environments/${editing.value.id}`)
  envLinks.value = detail.telemetry_links || []
  syncLinkEdit()
}

async function patchEnvLink(linkId: string) {
  if (!editing.value) return
  const inp = linkEdit.value[linkId]
  if (!inp) return
  await apiFetch(`/api/admin/environments/${editing.value.id}/telemetry-links/${linkId}`, {
    method: 'PATCH',
    body: JSON.stringify(inp),
  })
  const detail = await apiFetch<{ telemetry_links?: EnvLink[] }>(`/api/admin/environments/${editing.value.id}`)
  envLinks.value = detail.telemetry_links || []
  syncLinkEdit()
}


onMounted(async () => {
  await Promise.all([loadEnvironments(), loadClusters(), loadTelemetry()])
})
</script>

<style scoped>
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
.add-link-row .form-input { flex: 1; max-width: 320px; }
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
