<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Service providers</h1>
      <button type="button" class="btn btn-primary btn-pill" @click="openCreate">
        <Plus class="btn-leading-icon" :size="18" :stroke-width="2" />
        Add provider
      </button>
    </div>
    <div class="table-panel">
      <div class="table-split-scroll">
        <table class="data-table data-table--head">
          <colgroup>
            <col style="width: 72%" />
            <col style="width: 28%" />
          </colgroup>
          <thead>
            <tr>
              <th>Provider</th>
              <th>Actions</th>
            </tr>
          </thead>
        </table>
        <div class="table-body-shell">
          <table class="data-table data-table--body">
            <colgroup>
              <col style="width: 72%" />
              <col style="width: 28%" />
            </colgroup>
            <tbody>
              <tr v-for="p in list" :key="p.id">
                <td>
                  <div class="table-resource-cell">
                    <div class="table-resource-icon table-resource-icon--provider" aria-hidden="true">
                      <img
                        v-if="p.image_url?.trim() && !rowImageFailed[p.id]"
                        :src="p.image_url"
                        alt=""
                        class="table-resource-img"
                        @error="onRowImageError(p.id)"
                      />
                      <Activity v-else-if="p.provider_type === 'prometheus'" :size="16" :stroke-width="2" />
                      <Globe v-else-if="p.provider_type === 'grafana'" :size="16" :stroke-width="2" />
                      <Database v-else-if="p.provider_type === 'elasticsearch'" :size="16" :stroke-width="2" />
                      <Wrench v-else-if="p.provider_type === 'jenkins'" :size="16" :stroke-width="2" />
                      <Package v-else-if="p.provider_type === 'artifactory'" :size="16" :stroke-width="2" />
                      <Search v-else-if="p.provider_type === 'dns'" :size="16" :stroke-width="2" />
                      <Share2 v-else :size="16" :stroke-width="2" />
                    </div>
                    <div class="table-resource-body">
                      <span class="table-resource-title">{{ p.name }}</span>
                      <span class="table-resource-meta">
                        <span class="type-pill">{{ providerTypeLabel(p.provider_type) }}</span>
                        {{ providerMetaLine(p) }}
                      </span>
                    </div>
                  </div>
                </td>
                <td>
                  <button type="button" class="btn btn-sm" @click="editRow(p)">Edit</button>
                  <button type="button" class="btn btn-sm btn-danger" @click="requestRemoveProvider(p)">Delete</button>
                </td>
              </tr>
              <tr v-if="!list.length"><td colspan="2" class="empty-row">No providers yet.</td></tr>
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

    <div v-if="modal" class="modal-overlay" @click.self="closeModal">
      <div class="modal-card modal-card--lg">
        <div class="modal-card__header">
          <h2 class="modal-title">{{ editingId ? 'Edit provider' : 'New provider' }}</h2>
          <button type="button" class="modal-card__close" aria-label="Close" @click="closeModal">
            <X :size="20" :stroke-width="2" />
          </button>
        </div>
        <p v-if="!editingId" class="modal-subtitle">Verify before saving. Name must be unique (case-insensitive).</p>
        <form class="modal-form" @submit.prevent="save">
          <h3 class="modal-section-title">General</h3>
          <div class="form-group">
            <label class="form-label">Provider type<span class="form-label-required" aria-hidden="true">*</span></label>
            <AdminSegmentedRadioGroup
              v-model="form.provider_type"
              name="provider-type"
              aria-label="Provider type"
              :options="providerOptions"
              @update:model-value="onTypeChange"
            />
          </div>
          <div class="form-group form-group--name">
            <label class="form-label">Name<span class="form-label-required" aria-hidden="true">*</span></label>
            <div class="name-field-row">
              <input
                v-model="form.name"
                class="form-input"
                :class="nameFieldClass"
                required
                minlength="2"
                autocomplete="off"
              />
              <span v-if="nameAvail === 'loading'" class="name-status name-status--loading">Checking…</span>
              <span
                v-else-if="nameTrimmed.length >= 3 && nameAvail === 'available'"
                class="name-status name-status--ok"
              >Available</span>
              <span
                v-else-if="nameTrimmed.length >= 3 && nameAvail === 'taken'"
                class="name-status name-status--bad"
              >In use</span>
              <span
                v-else-if="nameTrimmed.length >= 3 && nameAvail === 'error'"
                class="name-status name-status--err"
              >Check failed</span>
            </div>
            <p v-if="nameTrimmed.length > 0 && nameTrimmed.length < 3" class="hint name-avail-hint">
              Type at least 3 characters to check name availability.
            </p>
          </div>
          <div class="form-group form-group--image">
            <label class="form-label">Image URL (optional)</label>
            <div class="image-row">
              <div class="image-preview" aria-hidden="true">
                <img
                  v-if="imageUrlTrimmed && !imagePreviewBroken"
                  :src="imageUrlTrimmed"
                  alt=""
                  class="image-preview__img"
                  @error="imagePreviewBroken = true"
                />
                <Share2 v-else :size="28" :stroke-width="2" class="image-preview__fallback" />
              </div>
              <input
                v-model="form.image_url"
                type="url"
                class="form-input image-url-input"
                placeholder="https://…"
                autocomplete="off"
              />
            </div>
            <p class="hint">Leave empty to use the default icon.</p>
          </div>

          <template v-if="form.provider_type === 'tcp'">
            <h4 class="sub">{{ providerTypeLabel('tcp') }}</h4>
            <div class="form-group">
              <label class="form-label">Host<span class="form-label-required" aria-hidden="true">*</span></label>
              <input v-model="form.host" class="form-input" required />
            </div>
            <div class="form-group">
              <label class="form-label">Port<span class="form-label-required" aria-hidden="true">*</span></label>
              <input v-model.number="form.port" type="number" class="form-input" min="1" max="65535" required />
            </div>
          </template>

          <template v-else-if="form.provider_type === 'prometheus'">
            <h4 class="sub">{{ providerTypeLabel('prometheus') }}</h4>
            <div class="form-group">
              <label class="form-label">Endpoint<span class="form-label-required" aria-hidden="true">*</span></label>
              <input v-model="form.prom_endpoint" class="form-input" placeholder="https://prometheus.example.com:9090" />
            </div>
            <div class="form-group">
              <label class="form-label">Auth</label>
              <AdminSelect v-model="form.prom_auth" aria-label="Prometheus auth" :options="promAuthSelectOptions" />
            </div>
            <template v-if="form.prom_auth === 'basic'">
              <div class="form-group"><label class="form-label">Username</label><input v-model="credUser" class="form-input" autocomplete="off" /></div>
              <div class="form-group"><label class="form-label">Password</label><input v-model="credPass" type="password" class="form-input" autocomplete="new-password" /></div>
            </template>
            <template v-if="form.prom_auth === 'bearer'">
              <div class="form-group"><label class="form-label">Token</label><input v-model="credBearer" type="password" class="form-input" autocomplete="new-password" /></div>
            </template>
            <p v-if="editingId && hasStoredCreds" class="hint">{{ storedCredsHint }}</p>
          </template>

          <template v-else-if="form.provider_type === 'grafana'">
            <h4 class="sub">{{ providerTypeLabel('grafana') }}</h4>
            <div class="form-group">
              <label class="form-label">Base URL<span class="form-label-required" aria-hidden="true">*</span></label>
              <input v-model="form.grafana_base_url" class="form-input" placeholder="https://grafana.example.com" />
            </div>
            <div class="form-group">
              <label class="form-label">Auth</label>
              <AdminSelect v-model="form.grafana_auth" aria-label="Grafana auth" :options="basicAuthSelectOptions" />
            </div>
            <template v-if="form.grafana_auth === 'basic'">
              <div class="form-group"><label class="form-label">Username</label><input v-model="credUser" class="form-input" autocomplete="off" /></div>
              <div class="form-group"><label class="form-label">Password</label><input v-model="credPass" type="password" class="form-input" autocomplete="new-password" /></div>
            </template>
            <p v-if="editingId && hasStoredCreds" class="hint">{{ storedCredsHint }}</p>
          </template>

          <template v-else-if="form.provider_type === 'elasticsearch'">
            <h4 class="sub">{{ providerTypeLabel('elasticsearch') }}</h4>
            <div class="form-group">
              <label class="form-label">Cluster URL<span class="form-label-required" aria-hidden="true">*</span></label>
              <input v-model="form.es_url" class="form-input" placeholder="https://elastic.example.com:9243" />
            </div>
            <div class="form-group">
              <label class="form-label">Auth</label>
              <AdminSelect v-model="form.es_auth" aria-label="Elasticsearch auth" :options="basicAuthSelectOptions" />
            </div>
            <template v-if="form.es_auth === 'basic'">
              <div class="form-group"><label class="form-label">Username</label><input v-model="credUser" class="form-input" autocomplete="off" /></div>
              <div class="form-group"><label class="form-label">Password</label><input v-model="credPass" type="password" class="form-input" autocomplete="new-password" /></div>
            </template>
            <p v-if="editingId && hasStoredCreds" class="hint">{{ storedCredsHint }}</p>
          </template>

          <template v-else-if="form.provider_type === 'jenkins'">
            <h4 class="sub">{{ providerTypeLabel('jenkins') }}</h4>
            <div class="form-group">
              <label class="form-label">Jenkins URL<span class="form-label-required" aria-hidden="true">*</span></label>
              <input v-model="form.jenkins_url" class="form-input" placeholder="https://jenkins.example.com" />
            </div>
            <div class="form-group">
              <label class="form-label">Auth</label>
              <AdminSelect v-model="form.jenkins_auth" aria-label="Jenkins auth" :options="jenkinsAuthSelectOptions" />
            </div>
            <template v-if="form.jenkins_auth === 'basic'">
              <div class="form-group"><label class="form-label">Username</label><input v-model="credUser" class="form-input" autocomplete="off" /></div>
              <div class="form-group">
                <label class="form-label">Password or API token</label>
                <input v-model="credPass" type="password" class="form-input" autocomplete="new-password" />
              </div>
            </template>
            <p v-if="editingId && hasStoredCreds" class="hint">{{ storedCredsHint }}</p>
          </template>

          <template v-else-if="form.provider_type === 'artifactory'">
            <h4 class="sub">{{ providerTypeLabel('artifactory') }}</h4>
            <div class="form-group">
              <label class="form-label">Base URL<span class="form-label-required" aria-hidden="true">*</span></label>
              <input v-model="form.artifactory_base_url" class="form-input" placeholder="https://artifactory.example.com or …/artifactory" />
            </div>
            <p class="hint">Must resolve <code class="inline-code">/api/system/ping</code> on this base (add <code class="inline-code">/artifactory</code> to the path if needed).</p>
            <div class="form-group">
              <label class="form-label">Auth</label>
              <AdminSelect v-model="form.artifactory_auth" aria-label="Artifactory auth" :options="basicAuthSelectOptions" />
            </div>
            <template v-if="form.artifactory_auth === 'basic'">
              <div class="form-group"><label class="form-label">Username</label><input v-model="credUser" class="form-input" autocomplete="off" /></div>
              <div class="form-group"><label class="form-label">Password</label><input v-model="credPass" type="password" class="form-input" autocomplete="new-password" /></div>
            </template>
            <p v-if="editingId && hasStoredCreds" class="hint">{{ storedCredsHint }}</p>
          </template>

          <template v-else-if="form.provider_type === 'dns'">
            <h4 class="sub">{{ providerTypeLabel('dns') }}</h4>
            <div class="form-group">
              <label class="form-label">Hostname<span class="form-label-required" aria-hidden="true">*</span></label>
              <input v-model="form.dns_hostname" class="form-input" placeholder="example.com" />
            </div>
            <div class="form-group">
              <label class="form-label">Record type<span class="form-label-required" aria-hidden="true">*</span></label>
              <AdminSelect v-model="form.dns_record_type" aria-label="DNS record type" :options="dnsRecordTypeOptions" />
            </div>
            <div class="form-group">
              <label class="form-label">Nameserver (optional)</label>
              <input v-model="form.dns_nameserver" class="form-input" placeholder="8.8.8.8 or ns.example.com" />
            </div>
            <p class="hint">Leave nameserver empty to use the host resolver; otherwise queries that server on UDP port 53.</p>
          </template>

          <div v-if="verifyResult" class="verify-out" :class="verifyResult.ok ? 'verify-out--ok' : 'verify-out--fail'">
            <template v-if="verifyResult.ok">
              <strong>{{ verifyOkLabel }}</strong>
              <span v-if="verifyResult.latency_ms != null"> · {{ verifyResult.latency_ms }}ms</span>
            </template>
            <template v-else>
              <strong>Verification failed</strong>
              <span v-if="verifyResult.error"> — {{ verifyResult.error }}</span>
            </template>
          </div>
          <p v-if="saveBlockedHint" class="hint verify-hint">{{ saveBlockedHint }}</p>
          <p v-if="formError" class="field-error">{{ formError }}</p>

          <div class="modal-actions modal-actions--split">
            <div class="modal-actions-left">
              <button type="button" class="btn btn-sm" :disabled="verifyBusy || !canRunVerify" @click="runVerify">
                {{ verifyBusy ? 'Verifying…' : 'Verify' }}
              </button>
            </div>
            <div class="modal-actions-right">
              <button type="button" class="btn btn-modal-cancel" @click="closeModal">Cancel</button>
              <button type="submit" class="btn btn-primary" :disabled="saving || !canSave">Save</button>
            </div>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Activity, Database, Globe, Package, Plus, Search, Share2, Wrench, X } from 'lucide-vue-next'

definePageMeta({ layout: 'admin' })
const { apiFetch } = useApi()

type ProviderType =
  | 'tcp'
  | 'prometheus'
  | 'grafana'
  | 'elasticsearch'
  | 'jenkins'
  | 'artifactory'
  | 'dns'

const PROVIDER_TYPES: ProviderType[] = [
  'tcp',
  'prometheus',
  'grafana',
  'elasticsearch',
  'jenkins',
  'artifactory',
  'dns',
]

const providerOptions: { value: ProviderType; label: string }[] = [
  { value: 'tcp', label: 'TCP' },
  { value: 'prometheus', label: 'Prometheus' },
  { value: 'grafana', label: 'Grafana' },
  { value: 'elasticsearch', label: 'Elasticsearch' },
  { value: 'jenkins', label: 'Jenkins' },
  { value: 'artifactory', label: 'Artifactory' },
  { value: 'dns', label: 'DNS checker' },
]

const promAuthSelectOptions = [
  { value: 'none', label: 'None' },
  { value: 'basic', label: 'Basic' },
  { value: 'bearer', label: 'Bearer' },
]
const basicAuthSelectOptions = [
  { value: 'none', label: 'None' },
  { value: 'basic', label: 'Basic' },
]
const jenkinsAuthSelectOptions = [
  { value: 'none', label: 'None (anonymous read)' },
  { value: 'basic', label: 'Basic' },
]
const dnsRecordTypeOptions = [
  { value: 'a', label: 'A' },
  { value: 'aaaa', label: 'AAAA' },
  { value: 'cname', label: 'CNAME' },
  { value: 'txt', label: 'TXT' },
]

function providerTypeLabel(pt: string | undefined): string {
  const o = providerOptions.find((x) => x.value === pt)
  return o?.label ?? String(pt ?? '')
}

type Row = {
  id: string
  name: string
  host: string
  port: number
  image_url?: string
  provider_type?: string
  config_json?: Record<string, unknown>
  has_stored_credentials?: boolean
  has_prometheus_credentials?: boolean
}

const list = ref<Row[]>([])

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

function requestRemoveProvider(p: Row) {
  deleteForm.value = {
    title: 'Delete service provider',
    warning:
      'This provider will be permanently deleted. Telemetry or probes that reference it may fail until reconfigured. This action cannot be undone.',
    resourceKind: 'provider',
    resourceName: p.name,
    requireNameMatch: true,
    confirmLabel: 'Delete provider',
  }
  deleteAction.value = async () => {
    await apiFetch(`/api/admin/service-providers/${p.id}`, { method: 'DELETE' })
    await load()
  }
  deleteOpen.value = true
}

type HttpAuth = 'none' | 'basic'

type FormState = {
  provider_type: ProviderType
  name: string
  host: string
  port: number
  image_url: string
  prom_endpoint: string
  prom_auth: 'none' | 'basic' | 'bearer'
  grafana_base_url: string
  grafana_auth: HttpAuth
  es_url: string
  es_auth: HttpAuth
  jenkins_url: string
  jenkins_auth: HttpAuth
  artifactory_base_url: string
  artifactory_auth: HttpAuth
  dns_hostname: string
  dns_record_type: 'a' | 'aaaa' | 'cname' | 'txt'
  dns_nameserver: string
}

function emptyForm(): FormState {
  return {
    provider_type: 'tcp',
    name: '',
    host: '',
    port: 443,
    image_url: '',
    prom_endpoint: '',
    prom_auth: 'none',
    grafana_base_url: '',
    grafana_auth: 'none',
    es_url: '',
    es_auth: 'none',
    jenkins_url: '',
    jenkins_auth: 'none',
    artifactory_base_url: '',
    artifactory_auth: 'none',
    dns_hostname: '',
    dns_record_type: 'a',
    dns_nameserver: '',
  }
}

const modal = ref(false)
const editingId = ref<string | null>(null)
const hasStoredCreds = ref(false)
const form = ref<FormState>(emptyForm())
const credUser = ref('')
const credPass = ref('')
const credBearer = ref('')
const formError = ref('')
const saving = ref(false)
const verifyBusy = ref(false)
const verifyResult = ref<{ ok: boolean; error?: string; latency_ms?: number } | null>(null)
const lastVerifyFingerprint = ref<string | null>(null)
const rowImageFailed = reactive<Record<string, boolean>>({})
const imagePreviewBroken = ref(false)

const storedCredsHint =
  'Credentials are stored. Enter new values here to replace them, or leave blank and verify still uses stored credentials when you pass the saved provider id.'

const nameTrimmed = computed(() => form.value.name.trim())
const imageUrlTrimmed = computed(() => form.value.image_url.trim())

const nameAvail = ref<'idle' | 'loading' | 'available' | 'taken' | 'error'>('idle')
let nameCheckTimer: ReturnType<typeof setTimeout> | null = null
let nameCheckSeq = 0

const nameFieldClass = computed(() => {
  if (nameTrimmed.value.length < 3) return ''
  if (nameAvail.value === 'available') return 'form-input--name-ok'
  if (nameAvail.value === 'taken') return 'form-input--name-bad'
  if (nameAvail.value === 'error') return 'form-input--name-warn'
  return ''
})

const verifyOkLabel = computed(() => {
  const pt = form.value.provider_type
  if (pt === 'tcp') return 'Reachable'
  if (pt === 'dns') return 'DNS OK'
  return `${providerTypeLabel(pt)} OK`
})

function rowProviderType(p: Row): ProviderType {
  const t = (p.provider_type || 'tcp').toLowerCase()
  return (PROVIDER_TYPES.includes(t as ProviderType) ? t : 'tcp') as ProviderType
}

function providerMetaLine(p: Row) {
  const cfg = p.config_json || {}
  const pt = rowProviderType(p)
  switch (pt) {
    case 'tcp':
      return `${p.host}:${p.port}`
    case 'prometheus':
      return String(cfg.endpoint || '').trim() || '—'
    case 'grafana':
    case 'artifactory':
      return String(cfg.base_url || '').trim() || '—'
    case 'elasticsearch':
    case 'jenkins':
      return String(cfg.url || '').trim() || '—'
    case 'dns': {
      const h = String(cfg.hostname || '').trim()
      const rt = String(cfg.record_type || 'a').trim()
      return h ? `${h} (${rt.toUpperCase()})` : '—'
    }
    default:
      return '—'
  }
}

function isHttpsURL(s: string): boolean {
  const u = s.trim()
  if (!u) return false
  return /^https?:\/\//i.test(u)
}

function verifyFingerprint(): string {
  const f = form.value
  const pt = f.provider_type
  switch (pt) {
    case 'tcp':
      return JSON.stringify({
        provider_type: 'tcp',
        host: f.host.trim(),
        port: Number(f.port),
      })
    case 'prometheus':
      return JSON.stringify({
        provider_type: 'prometheus',
        endpoint: f.prom_endpoint.trim(),
        auth_method: f.prom_auth,
      })
    case 'grafana':
      return JSON.stringify({
        provider_type: 'grafana',
        base_url: f.grafana_base_url.trim(),
        auth_method: f.grafana_auth,
      })
    case 'elasticsearch':
      return JSON.stringify({
        provider_type: 'elasticsearch',
        url: f.es_url.trim(),
        auth_method: f.es_auth,
      })
    case 'jenkins':
      return JSON.stringify({
        provider_type: 'jenkins',
        url: f.jenkins_url.trim(),
        auth_method: f.jenkins_auth,
      })
    case 'artifactory':
      return JSON.stringify({
        provider_type: 'artifactory',
        base_url: f.artifactory_base_url.trim(),
        auth_method: f.artifactory_auth,
      })
    case 'dns':
      return JSON.stringify({
        provider_type: 'dns',
        dns_hostname: f.dns_hostname.trim(),
        dns_record_type: f.dns_record_type,
        dns_nameserver: f.dns_nameserver.trim(),
      })
    default:
      return ''
  }
}

function validateLocal(): string {
  const n = nameTrimmed.value
  if (n.length < 2) return 'Name is required (at least 2 characters)'
  if (n.length >= 3) {
    if (nameAvail.value === 'loading') return 'Checking whether this name is available…'
    if (nameAvail.value === 'taken') return 'This name is already in use'
    if (nameAvail.value === 'error') return 'Could not verify name availability; try again'
  }
  const img = imageUrlTrimmed.value
  if (img && !/^https?:\/\//i.test(img)) return 'Image URL must start with http:// or https://'

  const f = form.value
  switch (f.provider_type) {
    case 'tcp': {
      const h = f.host.trim()
      if (!h) return 'Host is required'
      const port = Number(f.port)
      if (!Number.isInteger(port) || port < 1 || port > 65535) return 'Port must be between 1 and 65535'
      break
    }
    case 'prometheus': {
      if (!isHttpsURL(f.prom_endpoint)) return 'Prometheus endpoint must be a valid http(s) URL'
      if (f.prom_auth === 'basic') {
        const need = !editingId.value || !hasStoredCreds.value
        if (need && (!credUser.value.trim() || !credPass.value)) {
          return 'Basic auth requires username and password for new providers'
        }
      }
      if (f.prom_auth === 'bearer') {
        const need = !editingId.value || !hasStoredCreds.value
        if (need && !credBearer.value.trim()) return 'Bearer auth requires a token for new providers'
      }
      break
    }
    case 'grafana': {
      if (!isHttpsURL(f.grafana_base_url)) return 'Grafana base URL must be a valid http(s) URL'
      if (f.grafana_auth === 'basic') {
        const need = !editingId.value || !hasStoredCreds.value
        if (need && (!credUser.value.trim() || !credPass.value)) {
          return 'Basic auth requires username and password for new providers'
        }
      }
      break
    }
    case 'elasticsearch': {
      if (!isHttpsURL(f.es_url)) return 'Elasticsearch URL must be a valid http(s) URL'
      if (f.es_auth === 'basic') {
        const need = !editingId.value || !hasStoredCreds.value
        if (need && (!credUser.value.trim() || !credPass.value)) {
          return 'Basic auth requires username and password for new providers'
        }
      }
      break
    }
    case 'jenkins': {
      if (!isHttpsURL(f.jenkins_url)) return 'Jenkins URL must be a valid http(s) URL'
      if (f.jenkins_auth === 'basic') {
        const need = !editingId.value || !hasStoredCreds.value
        if (need && (!credUser.value.trim() || !credPass.value)) {
          return 'Basic auth requires username and password or API token for new providers'
        }
      }
      break
    }
    case 'artifactory': {
      if (!isHttpsURL(f.artifactory_base_url)) return 'Artifactory base URL must be a valid http(s) URL'
      if (f.artifactory_auth === 'basic') {
        const need = !editingId.value || !hasStoredCreds.value
        if (need && (!credUser.value.trim() || !credPass.value)) {
          return 'Basic auth requires username and password for new providers'
        }
      }
      break
    }
    case 'dns': {
      if (!f.dns_hostname.trim()) return 'Hostname is required'
      break
    }
    default:
      break
  }
  return ''
}

const canSave = computed(() => {
  if (validateLocal() !== '') return false
  const fp = lastVerifyFingerprint.value
  if (fp == null) return false
  return fp === verifyFingerprint()
})

const canRunVerify = computed(() => {
  const f = form.value
  const stored = Boolean(editingId.value && hasStoredCreds.value)
  switch (f.provider_type) {
    case 'tcp': {
      const h = f.host.trim()
      const port = Number(f.port)
      return Boolean(h) && Number.isInteger(port) && port >= 1 && port <= 65535
    }
    case 'prometheus': {
      if (!isHttpsURL(f.prom_endpoint)) return false
      if (f.prom_auth === 'basic') {
        if (credUser.value.trim() && credPass.value) return true
        return stored
      }
      if (f.prom_auth === 'bearer') {
        if (credBearer.value.trim()) return true
        return stored
      }
      return true
    }
    case 'grafana': {
      if (!isHttpsURL(f.grafana_base_url)) return false
      if (f.grafana_auth === 'basic') {
        if (credUser.value.trim() && credPass.value) return true
        return stored
      }
      return true
    }
    case 'elasticsearch': {
      if (!isHttpsURL(f.es_url)) return false
      if (f.es_auth === 'basic') {
        if (credUser.value.trim() && credPass.value) return true
        return stored
      }
      return true
    }
    case 'jenkins': {
      if (!isHttpsURL(f.jenkins_url)) return false
      if (f.jenkins_auth === 'basic') {
        if (credUser.value.trim() && credPass.value) return true
        return stored
      }
      return true
    }
    case 'artifactory': {
      if (!isHttpsURL(f.artifactory_base_url)) return false
      if (f.artifactory_auth === 'basic') {
        if (credUser.value.trim() && credPass.value) return true
        return stored
      }
      return true
    }
    case 'dns':
      return Boolean(f.dns_hostname.trim())
    default:
      return false
  }
})

const saveBlockedHint = computed(() => {
  if (canSave.value || validateLocal() !== '') return ''
  if (lastVerifyFingerprint.value == null) return 'Run Verify successfully before saving.'
  return 'Connection settings changed since the last successful verify — run Verify again.'
})

async function runNameAvailabilityCheck() {
  const t = nameTrimmed.value
  if (t.length < 3) {
    nameAvail.value = 'idle'
    return
  }
  const seq = ++nameCheckSeq
  nameAvail.value = 'loading'
  try {
    const params = new URLSearchParams({ name: t })
    const ex = editingId.value?.trim()
    if (ex) params.set('exclude_id', ex)
    const res = (await apiFetch(`/api/admin/service-providers/check-name?${params.toString()}`)) as {
      available?: boolean
    }
    if (seq !== nameCheckSeq) return
    nameAvail.value = res.available === true ? 'available' : 'taken'
  } catch {
    if (seq !== nameCheckSeq) return
    nameAvail.value = 'error'
  }
}

function scheduleNameAvailabilityCheck() {
  if (nameCheckTimer) clearTimeout(nameCheckTimer)
  const t = nameTrimmed.value
  if (t.length < 3) {
    nameAvail.value = 'idle'
    return
  }
  nameAvail.value = 'loading'
  nameCheckTimer = setTimeout(() => {
    nameCheckTimer = null
    void runNameAvailabilityCheck()
  }, 320)
}

watch(() => form.value.name, scheduleNameAvailabilityCheck)

watch(imageUrlTrimmed, () => {
  imagePreviewBroken.value = false
})

function clearVerifyOnConnectionChange() {
  lastVerifyFingerprint.value = null
  verifyResult.value = null
}

watch(
  () =>
    [
      form.value.provider_type,
      form.value.host,
      form.value.port,
      form.value.prom_endpoint,
      form.value.prom_auth,
      form.value.grafana_base_url,
      form.value.grafana_auth,
      form.value.es_url,
      form.value.es_auth,
      form.value.jenkins_url,
      form.value.jenkins_auth,
      form.value.artifactory_base_url,
      form.value.artifactory_auth,
      form.value.dns_hostname,
      form.value.dns_record_type,
      form.value.dns_nameserver,
    ] as const,
  () => clearVerifyOnConnectionChange(),
)

watch([credUser, credPass, credBearer], () => clearVerifyOnConnectionChange())

async function load() {
  list.value = await apiFetch<Row[]>('/api/admin/service-providers')
}

function resetModalState() {
  formError.value = ''
  verifyResult.value = null
  lastVerifyFingerprint.value = null
  imagePreviewBroken.value = false
  credUser.value = ''
  credPass.value = ''
  credBearer.value = ''
  if (nameCheckTimer) clearTimeout(nameCheckTimer)
  nameCheckTimer = null
  nameCheckSeq++
  nameAvail.value = 'idle'
}

function onTypeChange() {
  clearVerifyOnConnectionChange()
}

function openCreate() {
  editingId.value = null
  hasStoredCreds.value = false
  form.value = emptyForm()
  resetModalState()
  modal.value = true
}

function editRow(p: Row) {
  editingId.value = p.id
  const pt = rowProviderType(p)
  const cfg = p.config_json || {}
  const next = emptyForm()
  next.provider_type = pt
  next.name = p.name || ''
  next.image_url = p.image_url?.trim() || ''
  switch (pt) {
    case 'tcp':
      next.host = p.host
      next.port = p.port
      break
    case 'prometheus':
      next.prom_endpoint = String(cfg.endpoint || '')
      next.prom_auth = (String(cfg.auth_method || 'none').toLowerCase() as 'none' | 'basic' | 'bearer') || 'none'
      break
    case 'grafana':
      next.grafana_base_url = String(cfg.base_url || '')
      next.grafana_auth = (String(cfg.auth_method || 'none').toLowerCase() as HttpAuth) === 'basic' ? 'basic' : 'none'
      break
    case 'elasticsearch':
      next.es_url = String(cfg.url || '')
      next.es_auth = (String(cfg.auth_method || 'none').toLowerCase() as HttpAuth) === 'basic' ? 'basic' : 'none'
      break
    case 'jenkins':
      next.jenkins_url = String(cfg.url || '')
      next.jenkins_auth = (String(cfg.auth_method || 'none').toLowerCase() as HttpAuth) === 'basic' ? 'basic' : 'none'
      break
    case 'artifactory':
      next.artifactory_base_url = String(cfg.base_url || '')
      next.artifactory_auth = (String(cfg.auth_method || 'none').toLowerCase() as HttpAuth) === 'basic' ? 'basic' : 'none'
      break
    case 'dns':
      next.dns_hostname = String(cfg.hostname || '')
      {
        const rt = String(cfg.record_type || 'a').toLowerCase()
        if (rt === 'aaaa' || rt === 'cname' || rt === 'txt') next.dns_record_type = rt
        else next.dns_record_type = 'a'
      }
      next.dns_nameserver = String(cfg.nameserver || '')
      break
    default:
      break
  }
  form.value = next
  resetModalState()
  hasStoredCreds.value = Boolean(p.has_stored_credentials ?? p.has_prometheus_credentials)
  modal.value = true
  if (nameTrimmed.value.length >= 3) scheduleNameAvailabilityCheck()
}

function closeModal() {
  modal.value = false
  editingId.value = null
  hasStoredCreds.value = false
  resetModalState()
}

function onRowImageError(id: string) {
  rowImageFailed[id] = true
}

function appendCreds(
  body: Record<string, unknown>,
  auth: 'none' | 'basic' | 'bearer' | HttpAuth,
  includeBearer: boolean,
) {
  const creds: Record<string, string> = {}
  if (auth === 'basic') {
    if (credUser.value.trim()) creds.username = credUser.value
    if (credPass.value) creds.password = credPass.value
  }
  if (includeBearer && auth === 'bearer' && credBearer.value) {
    creds.bearer_token = credBearer.value
  }
  if (Object.keys(creds).length) body.credentials = creds
}

function buildVerifyBody(): Record<string, unknown> {
  const f = form.value
  const pt = f.provider_type
  if (pt === 'tcp') {
    return {
      provider_type: 'tcp',
      host: f.host.trim(),
      port: Number(f.port),
    }
  }
  const body: Record<string, unknown> = { provider_type: pt }
  if (editingId.value) body.service_provider_id = editingId.value

  switch (pt) {
    case 'prometheus':
      body.endpoint = f.prom_endpoint.trim()
      body.auth_method = f.prom_auth
      appendCreds(body, f.prom_auth, true)
      break
    case 'grafana':
      body.base_url = f.grafana_base_url.trim()
      body.auth_method = f.grafana_auth
      appendCreds(body, f.grafana_auth, false)
      break
    case 'elasticsearch':
      body.url = f.es_url.trim()
      body.auth_method = f.es_auth
      appendCreds(body, f.es_auth, false)
      break
    case 'jenkins':
      body.url = f.jenkins_url.trim()
      body.auth_method = f.jenkins_auth
      appendCreds(body, f.jenkins_auth, false)
      break
    case 'artifactory':
      body.base_url = f.artifactory_base_url.trim()
      body.auth_method = f.artifactory_auth
      appendCreds(body, f.artifactory_auth, false)
      break
    case 'dns':
      body.dns_hostname = f.dns_hostname.trim()
      body.dns_record_type = f.dns_record_type
      body.dns_nameserver = f.dns_nameserver.trim()
      break
    default:
      break
  }
  return body
}

async function runVerify() {
  formError.value = ''
  verifyResult.value = null
  verifyBusy.value = true
  try {
    const res = (await apiFetch('/api/admin/service-providers/verify', {
      method: 'POST',
      body: JSON.stringify(buildVerifyBody()),
    })) as { ok?: boolean; error?: string; latency_ms?: number }
    verifyResult.value = {
      ok: res.ok === true,
      error: typeof res.error === 'string' ? res.error : undefined,
      latency_ms: typeof res.latency_ms === 'number' ? res.latency_ms : undefined,
    }
    if (res.ok === true) {
      lastVerifyFingerprint.value = verifyFingerprint()
    } else {
      lastVerifyFingerprint.value = null
    }
  } catch (e: unknown) {
    verifyResult.value = { ok: false, error: e instanceof Error ? e.message : 'Verify failed' }
    lastVerifyFingerprint.value = null
  } finally {
    verifyBusy.value = false
  }
}

function buildConfigJsonForSave(pt: ProviderType): Record<string, unknown> {
  const f = form.value
  switch (pt) {
    case 'tcp':
      return {}
    case 'prometheus':
      return { endpoint: f.prom_endpoint.trim(), auth_method: f.prom_auth }
    case 'grafana':
      return { base_url: f.grafana_base_url.trim(), auth_method: f.grafana_auth }
    case 'elasticsearch':
      return { url: f.es_url.trim(), auth_method: f.es_auth }
    case 'jenkins':
      return { url: f.jenkins_url.trim(), auth_method: f.jenkins_auth }
    case 'artifactory':
      return { base_url: f.artifactory_base_url.trim(), auth_method: f.artifactory_auth }
    case 'dns': {
      const o: Record<string, unknown> = {
        hostname: f.dns_hostname.trim(),
        record_type: f.dns_record_type,
      }
      if (f.dns_nameserver.trim()) o.nameserver = f.dns_nameserver.trim()
      return o
    }
    default:
      return {}
  }
}

function buildCredentialsPayload(): Record<string, string> | undefined {
  const f = form.value
  const pt = f.provider_type
  const out: Record<string, string> = {}

  if (pt === 'prometheus') {
    if (f.prom_auth === 'none') return undefined
    if (f.prom_auth === 'basic') {
      if (credUser.value.trim()) out.username = credUser.value.trim()
      if (credPass.value) out.password = credPass.value
    }
    if (f.prom_auth === 'bearer' && credBearer.value.trim()) {
      out.bearer_token = credBearer.value.trim()
    }
    return Object.keys(out).length ? out : undefined
  }

  let basic = false
  if (pt === 'grafana' && f.grafana_auth === 'basic') basic = true
  if (pt === 'elasticsearch' && f.es_auth === 'basic') basic = true
  if (pt === 'jenkins' && f.jenkins_auth === 'basic') basic = true
  if (pt === 'artifactory' && f.artifactory_auth === 'basic') basic = true
  if (!basic) return undefined
  if (credUser.value.trim()) out.username = credUser.value.trim()
  if (credPass.value) out.password = credPass.value
  return Object.keys(out).length ? out : undefined
}

async function save() {
  formError.value = ''
  const ve = validateLocal()
  if (ve) {
    formError.value = ve
    return
  }
  if (!canSave.value) {
    formError.value = saveBlockedHint.value || 'Run Verify successfully before saving.'
    return
  }
  saving.value = true
  try {
    const pt = form.value.provider_type
    const body: Record<string, unknown> = {
      name: nameTrimmed.value,
      image_url: imageUrlTrimmed.value,
      provider_type: pt,
    }
    if (pt === 'tcp') {
      body.host = form.value.host.trim()
      body.port = Number(form.value.port)
      body.config_json = {}
    } else {
      body.host = ''
      body.port = 0
      body.config_json = buildConfigJsonForSave(pt)
      const creds = buildCredentialsPayload()
      if (creds) body.credentials = creds
    }
    if (editingId.value) {
      await apiFetch(`/api/admin/service-providers/${editingId.value}`, { method: 'PUT', body: JSON.stringify(body) })
    } else {
      await apiFetch('/api/admin/service-providers', { method: 'POST', body: JSON.stringify(body) })
    }
    closeModal()
    await load()
  } catch (e: unknown) {
    const apiErr = (e as { data?: { error?: string } })?.data?.error
    formError.value = apiErr || (e instanceof Error ? e.message : 'Save failed')
  } finally {
    saving.value = false
  }
}

onMounted(load)

onUnmounted(() => {
  if (nameCheckTimer) clearTimeout(nameCheckTimer)
})
</script>

<style scoped>
.type-pill {
  display: inline-block;
  margin-right: 6px;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  background: var(--color-surface-muted, #e5e7eb);
  color: var(--color-text-secondary);
}
.sub {
  font-size: 0.9rem;
  margin: 12px 0 6px;
  font-weight: 600;
}
.form-group--name .name-field-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.form-group--name .name-field-row .form-input {
  flex: 1;
  min-width: 12rem;
}
.name-status {
  font-size: 0.8rem;
  font-weight: 600;
  white-space: nowrap;
}
.name-status--loading {
  color: var(--color-text-secondary);
}
.name-status--ok {
  color: #15803d;
}
.name-status--bad {
  color: var(--color-red, #b91c1c);
}
.name-status--err {
  color: #b45309;
}
.name-avail-hint {
  margin: 4px 0 0;
}
.form-input--name-ok {
  border-color: #15803d;
}
.form-input--name-bad {
  border-color: var(--color-red, #b91c1c);
}
.form-input--name-warn {
  border-color: #d97706;
}
.form-group--image .image-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.image-preview {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  border: 1px solid var(--color-border-strong);
  background: var(--color-surface-muted, #f3f4f6);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  overflow: hidden;
}
.image-preview__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.image-preview__fallback {
  color: var(--color-text-secondary);
}
.image-url-input {
  flex: 1;
  min-width: 12rem;
}
.table-resource-icon--provider {
  overflow: hidden;
}
.table-resource-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 6px;
}
.verify-out {
  margin-top: 8px;
  padding: 10px;
  border-radius: 6px;
  font-size: 0.85rem;
  border: 1px solid transparent;
}
.verify-out--ok {
  background: #dcfce7;
  color: #166534;
  border-color: rgba(34, 197, 94, 0.35);
}
.verify-out--fail {
  background: #fef2f2;
  color: #991b1b;
  border-color: rgba(239, 68, 68, 0.35);
}
.verify-hint {
  margin-top: 6px;
}
.field-error {
  color: #b91c1c;
  font-size: 0.85rem;
  margin: 6px 0 0;
}
.hint {
  font-size: 0.8rem;
  color: var(--color-text-secondary);
  margin: 4px 0 0;
}
.inline-code {
  font-family: ui-monospace, monospace;
  font-size: 0.85em;
  padding: 1px 4px;
  border-radius: 4px;
  background: var(--color-surface-muted, #f3f4f6);
}
.modal-actions--split {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.modal-actions-right {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.modal-card--lg {
  max-width: 32rem;
}
</style>
