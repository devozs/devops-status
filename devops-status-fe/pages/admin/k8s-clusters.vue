<template>
  <div>
    <div class="page-header page-header--compact">
      <h1 class="page-title">Kubernetes Clusters</h1>
      <button type="button" class="btn btn-primary btn-pill" @click="showCreate = true">
        <Plus class="btn-leading-icon" :size="18" :stroke-width="2" />
        Add cluster
      </button>
    </div>
    <AdminCallout variant="warning">
      <p>
        <strong>Revoke</strong> marks the cluster disconnected in this app: it invalidates the onboarding handshake and
        <em>deactivates the stored API token</em> so DevOps Status stops calling your Kubernetes API. It does not remove
        RBAC or namespaces in your cluster; delete those with kubectl if you want them gone.
      </p>
    </AdminCallout>

    <div v-if="backendContext" class="backend-url-card">
      <div class="backend-url-row">
        <span class="backend-url-label">Backend public URL (STATUS_API_URL in onboarding)</span>
        <span class="url-kind-pill" :class="'url-kind-pill--' + publicUrlKind">{{ publicUrlKindLabel }}</span>
      </div>
      <p class="backend-url-value mono">{{ backendContext.external_url }}</p>
      <p class="backend-url-caption">
        In-cluster Jobs and agents call this URL. It updates when you restart the backend with a new Cloudflare tunnel or change
        <code class="ping-code">EXTERNAL_URL</code> in production. The admin UI still talks to
        <code class="ping-code">NUXT_PUBLIC_API_BASE</code> (e.g. localhost:8080).
      </p>
    </div>

    <p v-if="loadError" class="load-error">
      {{ loadError }}
      <span v-if="loadErrorHint" class="load-error-hint">{{ loadErrorHint }}</span>
    </p>

    <div class="table-panel">
      <div class="table-split-scroll">
        <table class="data-table data-table--head">
          <colgroup>
            <col style="width: 14%" />
            <col style="width: 22%" />
            <col style="width: 9%" />
            <col style="width: 12%" />
            <col style="width: 8%" />
            <col style="width: 23%" />
            <col style="width: 12%" />
          </colgroup>
          <thead>
            <tr>
              <th>Name</th>
              <th>Endpoint</th>
              <th>Version</th>
              <th>Status</th>
              <th>API creds</th>
              <th>API live check</th>
              <th>Actions</th>
            </tr>
          </thead>
        </table>
        <div class="table-body-shell">
          <table class="data-table data-table--body">
            <colgroup>
              <col style="width: 14%" />
              <col style="width: 22%" />
              <col style="width: 9%" />
              <col style="width: 12%" />
              <col style="width: 8%" />
              <col style="width: 23%" />
              <col style="width: 12%" />
            </colgroup>
            <tbody>
              <tr v-for="c in clusters" :key="c.id">
                <td>{{ c.name }}</td>
                <td class="mono">{{ c.endpoint }}</td>
                <td>{{ c.k8s_version || '-' }}</td>
                <td>
                  <span class="status-badge" :class="statusBadgeClass(c)">{{ statusBadgeLabel(c) }}</span>
                </td>
                <td>
                  <span v-if="c.has_api_credential" class="cred-yes" title="Stored token for Kubernetes probes">Yes</span>
                  <span v-else class="cred-no" title="No token — probes using cluster_id will not authenticate">No</span>
                </td>
                <td class="ping-cell">
                  <template v-if="c.status === 'connected' && c.has_api_credential">
                    <div class="ping-box" :class="pingClass(c.id)">
                      <button type="button" class="btn btn-sm" :disabled="pingById[c.id]?.loading" @click="verifyCluster(c)">
                        {{ pingById[c.id]?.loading ? 'Checking…' : 'Verify API' }}
                      </button>
                      <div v-if="pingById[c.id] && !pingById[c.id]?.loading && pingById[c.id]?.at" class="ping-result">
                        <span v-if="pingById[c.id]?.ok" class="ping-ok">{{ pingById[c.id]?.k8s_version || 'OK' }}</span>
                        <span v-else class="ping-bad">{{ pingById[c.id]?.message }}</span>
                        <span class="ping-time">{{ formatPingTime(pingById[c.id]?.at) }}</span>
                      </div>
                      <p class="ping-caption">
                        Verify uses <code class="ping-code">NUXT_PUBLIC_API_BASE</code> → this app then runs <code class="ping-code">GET /version</code> on the cluster API with the stored token (not the public URL above).
                        Private CA / TLS: <code class="ping-code">K8S_INSECURE_SKIP_TLS_VERIFY=true</code> in <code class="ping-code">.env.dev</code> and restart the backend.
                      </p>
                      <p v-if="isRancherStyleEndpoint(c.endpoint)" class="ping-caption ping-caption--rancher">
                        <strong>Rancher:</strong> tokens from <code>kubectl create token</code> are usually bound to the in-cluster API audience and return HTTP 401 on the Rancher proxy URL. Use <strong>Add API token</strong> and paste the bearer from your Rancher kubeconfig for this cluster (same token <code>kubectl</code> uses against that server URL).
                      </p>
                    </div>
                  </template>
                  <span v-else class="ping-na">—</span>
                </td>
                <td class="actions-cell">
                  <button v-if="c.status !== 'connected'" class="btn btn-sm" @click="onboard(c)">Onboard</button>
                  <button v-if="c.status === 'pending_registration'" class="btn btn-sm btn-success" @click="openManualModal(c)">Manual Register</button>
                  <button v-if="c.status === 'connected' && !c.has_api_credential" class="btn btn-sm btn-success" @click="openTokenModal(c)">Add API token</button>
                  <button v-if="c.status === 'connected'" class="btn btn-sm btn-danger" @click="requestRevokeCluster(c)">Revoke</button>
                  <button class="btn btn-sm btn-danger" @click="requestDeleteCluster(c)">Delete</button>
                </td>
              </tr>
              <tr v-if="!clusters?.length"><td colspan="7" class="empty-row">No clusters registered.</td></tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Onboarding modal -->
    <div v-if="onboardData" class="modal-overlay" @click.self="closeOnboardModal">
      <div class="modal-card modal-card--wide">
        <AdminModalStepper
          v-if="onboardTwoSteps"
          :steps="['Instructions', 'Run command']"
          :current-step="onboardUiStep"
        />
        <h2 class="modal-title">Cluster onboarding</h2>
        <p class="modal-subtitle">Follow the steps to register the agent in your cluster.</p>

        <div v-show="!onboardTwoSteps || onboardUiStep === 0">
          <div v-if="onboardData.is_dev" class="tunnel-status" :class="tunnelIsActive ? 'tunnel-status--ok' : 'tunnel-status--warn'">
            <span v-if="tunnelIsActive" class="tunnel-dot tunnel-dot--green" />
            <span v-else class="tunnel-dot tunnel-dot--red" />
            <span v-if="tunnelIsActive">Tunnel active: {{ onboardData.external_url }}</span>
            <span v-else>Backend URL is {{ onboardData.external_url }} &mdash; K8s pods cannot reach it</span>
          </div>

          <ol class="instructions">
            <li v-for="(step, i) in onboardData.instructions" :key="i">{{ step }}</li>
          </ol>
        </div>

        <div v-show="onboardTwoSteps && onboardUiStep === 1">
          <div class="command-block">
            <div class="command-header">
              <label class="form-label">Run on target cluster</label>
              <button type="button" class="btn btn-sm" @click="copyCommand">{{ copied ? 'Copied!' : 'Copy' }}</button>
            </div>
            <pre class="command-pre" ref="commandRef">{{ onboardData.command }}</pre>
          </div>
          <p class="token-info">Handshake expires at: {{ formatExpiry(onboardData.expires_at) }}</p>
          <p v-if="onboardData.is_dev" class="help-note">Command uses Cloudflare tunnel. If the Job still fails, use <strong>Manual Register</strong> and paste a token from <code>kubectl create token</code>.</p>
        </div>

        <div v-show="!onboardTwoSteps && onboardData.is_dev && !tunnelIsActive" class="localhost-warn">
          <p>The kubectl command is not shown because pods cannot reach <code>{{ onboardData.external_url }}</code>.</p>
          <p>Start the tunnel: <code>make dev-infra-up && make be-run</code> &mdash; then re-open this modal.</p>
          <p>Or use <strong>Manual Register</strong> with a token from: <code>kubectl create token devops-status-agent -n devops-status-system --duration=24h</code></p>
        </div>

        <div class="modal-actions">
          <template v-if="onboardTwoSteps">
            <button v-if="onboardUiStep > 0" type="button" class="btn" @click="onboardUiStep--">Back</button>
            <button v-if="onboardUiStep === 0" type="button" class="btn btn-primary btn-pill" @click="onboardUiStep++">
              Next
              <ChevronRight class="btn-leading-icon" :size="18" :stroke-width="2" />
            </button>
          </template>
          <button type="button" class="btn" @click="closeOnboardModal">Close</button>
        </div>
      </div>
    </div>

    <!-- Manual register / add token -->
    <div v-if="manualModalCluster" class="modal-overlay" @click.self="manualModalCluster = null">
      <div class="modal-card modal-card--wide">
        <h2 class="modal-title">{{ manualModalMode === 'register' ? 'Manual cluster registration' : 'Add Kubernetes API token' }}</h2>
        <p class="modal-subtitle">Paste a bearer token the app will use for <code class="ping-code">GET /version</code> against your stored API endpoint.</p>
        <p v-if="manualModalMode === 'register'" class="help-note">
          Completes the handshake. For a normal API server URL, use the service account token (same as the onboarding Job sends). For a Rancher proxy URL, use your kubeconfig bearer instead — see below.
        </p>
        <p v-else class="help-note">
          You already completed the handshake without a token. For Rancher endpoints, paste the kubeconfig bearer; otherwise a token for <code>devops-status-agent</code> in <code>devops-status-system</code> works when the endpoint is the apiserver directly.
        </p>
        <p v-if="manualModalCluster && isRancherStyleEndpoint(manualModalCluster.endpoint)" class="help-note help-note--rancher">
          This cluster&apos;s endpoint looks like a <strong>Rancher</strong> proxy (<code>…/k8s/clusters/…</code>). <code>kubectl create token</code> usually produces HTTP 401 here (token audience). Paste the token from your kubeconfig for this context, e.g.
          <code class="mono hint-cmd hint-cmd--block">kubectl config view --minify --raw -o jsonpath='{.users[0].user.token}{"\n"}'</code>
          with the correct context selected (<code>kubectl config use-context …</code>).
        </p>
        <p v-else class="mono hint-cmd">kubectl create token devops-status-agent -n devops-status-system --duration=24h</p>
        <div class="form-group">
          <label class="form-label">Bearer token (paste full JWT)</label>
          <textarea v-model="manualSaToken" class="form-input token-area" rows="4" placeholder="eyJhbGciOiJSUzI1NiIs..." />
        </div>
        <p v-if="manualError" class="form-error">{{ manualError }}</p>
        <div class="modal-actions">
          <button type="button" class="btn" @click="manualModalCluster = null">Cancel</button>
          <button type="button" class="btn btn-primary" :disabled="manualSubmitting" @click="submitManualModal">
            {{ manualSubmitting ? 'Saving…' : 'Save' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Create modal -->
    <div v-if="showCreate" class="modal-overlay" @click.self="showCreate = false">
      <div class="modal-card">
        <h2 class="modal-title">Register cluster</h2>
        <p class="modal-subtitle">Create a cluster record before running onboarding in the target cluster.</p>
        <form class="modal-form" @submit.prevent="createCluster">
          <h3 class="modal-section-title">General</h3>
          <div class="form-group"><label class="form-label">Name<span class="form-label-required" aria-hidden="true">*</span></label><input v-model="form.name" class="form-input" required /></div>
          <div class="form-group">
            <label class="form-label">API endpoint<span class="form-label-required" aria-hidden="true">*</span></label>
            <input v-model="form.endpoint" class="form-input" required placeholder="https://k8s-api:6443" />
            <span class="form-hint">Use the server URL from <code>kubectl cluster-info</code> or kubeconfig. Rancher: <code>https://…/k8s/clusters/&lt;id&gt;</code>. Verify and probes call this URL with the stored bearer token.</span>
          </div>
          <div class="form-group">
            <label class="form-label">DevOps Status namespace</label>
            <input :value="form.default_namespace" class="form-input form-input--locked" readonly />
            <span class="form-hint">Agent and RBAC resources will be deployed in this namespace.</span>
          </div>
          <div class="modal-actions">
            <button type="button" class="btn" @click="showCreate = false">Cancel</button>
            <button type="submit" class="btn btn-primary">Create</button>
          </div>
        </form>
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
  </div>
</template>

<script setup lang="ts">
import { watch } from 'vue'
import { ChevronRight, Plus } from 'lucide-vue-next'

definePageMeta({ layout: 'admin' })
const { apiFetch } = useApi()

const clusters = ref<any[]>([])
const backendContext = ref<{ external_url: string; is_dev: boolean } | null>(null)
const loadError = ref('')
const loadErrorHint = ref('')
const showCreate = ref(false)
const onboardData = ref<any>(null)
const onboardUiStep = ref(0)
const copied = ref(false)

const manualModalCluster = ref<any>(null)
const manualModalMode = ref<'register' | 'token'>('register')
const manualSaToken = ref('')
const manualError = ref('')
const manualSubmitting = ref(false)

type PingState = { loading?: boolean; ok?: boolean; message?: string; k8s_version?: string; at?: number }

const pingById = ref<Record<string, PingState>>({})

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

function requestRevokeCluster(c: { id: string; name: string }) {
  deleteForm.value = {
    title: 'Revoke cluster',
    warning:
      'Stored API tokens will be deactivated and the onboarding handshake invalidated. DevOps Status will stop calling your Kubernetes API. This does not remove RBAC or namespaces in your cluster.',
    resourceKind: 'cluster',
    resourceName: c.name,
    requireNameMatch: true,
    confirmLabel: 'Revoke cluster',
  }
  deleteAction.value = async () => {
    await apiFetch(`/api/admin/k8s-clusters/${c.id}/revoke`, { method: 'POST' })
    const next = { ...pingById.value }
    delete next[c.id]
    pingById.value = next
    await load()
  }
  deleteOpen.value = true
}

function requestDeleteCluster(c: { id: string; name: string }) {
  deleteForm.value = {
    title: 'Delete cluster',
    warning:
      'This cluster record will be permanently removed from DevOps Status. Onboarding state and stored credentials for this record will be deleted. This action cannot be undone.',
    resourceKind: 'cluster',
    resourceName: c.name,
    requireNameMatch: true,
    confirmLabel: 'Delete cluster',
  }
  deleteAction.value = async () => {
    await apiFetch(`/api/admin/k8s-clusters/${c.id}`, { method: 'DELETE' })
    await load()
  }
  deleteOpen.value = true
}

const tunnelIsActive = computed(() => {
  if (!onboardData.value) return false
  const url: string = onboardData.value.external_url || ''
  return !!url && !url.includes('localhost') && !url.includes('127.0.0.1')
})

/** Step 2 shows kubectl command (prod or dev with reachable tunnel). */
const onboardTwoSteps = computed(() => {
  if (!onboardData.value) return false
  return tunnelIsActive.value || !onboardData.value.is_dev
})

watch(onboardData, (v) => {
  onboardUiStep.value = 0
  if (v) copied.value = false
})

function closeOnboardModal() {
  onboardData.value = null
  onboardUiStep.value = 0
}

type PublicUrlKind = 'tunnel' | 'prod' | 'localhost'

const publicUrlKind = computed<PublicUrlKind>(() => {
  const u = backendContext.value?.external_url || ''
  if (!u) return 'localhost'
  if (u.includes('trycloudflare.com')) return 'tunnel'
  if (u.includes('localhost') || u.includes('127.0.0.1')) return 'localhost'
  return 'prod'
})

const publicUrlKindLabel = computed(() => {
  switch (publicUrlKind.value) {
    case 'tunnel':
      return 'Cloudflare quick tunnel'
    case 'prod':
      return 'Production'
    default:
      return 'Local (remote clusters cannot reach)'
  }
})

async function loadBackendMeta() {
  try {
    backendContext.value = await apiFetch<{ external_url: string; is_dev: boolean }>('/api/admin/meta')
  } catch {
    backendContext.value = null
  }
}

function pingClass(id: string) {
  const p = pingById.value[id]
  if (!p?.at || p.loading) return ''
  return p.ok ? 'ping-box--ok' : 'ping-box--bad'
}

function formatPingTime(at?: number) {
  if (!at) return ''
  return new Date(at).toLocaleTimeString()
}

/** Rancher exposes the Kubernetes API under .../k8s/clusters/<id> — SA tokens from kubectl create token often 401 here. */
function isRancherStyleEndpoint(endpoint: string | undefined | null) {
  if (!endpoint) return false
  return endpoint.toLowerCase().includes('/k8s/clusters/')
}

function statusBadgeLabel(c: any) {
  if (c.status === 'connected' && c.has_api_credential && c.api_verify_ok === false) {
    return 'connected · API not verified'
  }
  return c.status
}

function statusBadgeClass(c: any) {
  if (c.status === 'connected' && c.has_api_credential && c.api_verify_ok === false) {
    return 'badge--connected_api_unverified'
  }
  return 'badge--' + c.status
}

async function verifyCluster(c: { id: string }, opts?: { skipListRefresh?: boolean }) {
  const id = c.id
  pingById.value = { ...pingById.value, [id]: { ...pingById.value[id], loading: true } }
  try {
    const r = await apiFetch<{ ok: boolean; message: string; k8s_version?: string }>(
      `/api/admin/k8s-clusters/${id}/verify`,
    )
    pingById.value = {
      ...pingById.value,
      [id]: {
        loading: false,
        ok: r.ok,
        message: r.message,
        k8s_version: r.k8s_version,
        at: Date.now(),
      },
    }
    if (r.ok && r.k8s_version) {
      const row = clusters.value.find(x => x.id === id)
      if (row && !row.k8s_version) row.k8s_version = r.k8s_version
    }
  } catch (e: any) {
    pingById.value = {
      ...pingById.value,
      [id]: {
        loading: false,
        ok: false,
        message: e?.data?.error || e?.message || 'Request failed',
        at: Date.now(),
      },
    }
  } finally {
    if (!opts?.skipListRefresh) await load()
  }
}

function copyCommand() {
  if (!onboardData.value?.command) return
  navigator.clipboard.writeText(onboardData.value.command)
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

function formatExpiry(iso: string) {
  return new Date(iso).toLocaleString()
}
const form = ref({ name: '', endpoint: '', default_namespace: 'devops-status-system' })

async function load() {
  loadError.value = ''
  loadErrorHint.value = ''
  try {
    const list = await apiFetch<any[]>('/api/admin/k8s-clusters')
    clusters.value = list || []
  } catch (e: any) {
    clusters.value = []
    const status = e?.statusCode ?? e?.status ?? e?.response?.status
    const msg = e?.data?.error || e?.message || 'Failed to load clusters'
    loadError.value = status === 500 || String(msg).includes('500')
      ? 'Could not load clusters (server error).'
      : msg
    if (status === 500 || String(msg).includes('500')) {
      loadErrorHint.value = 'Run: make db-migrate from the repo root (Postgres must be up). If you skipped migrations, columns such as api_verify_ok are missing and the API returns 500. ./run.sh runs db-migrate after dev-infra-up by default.'
    }
  }
}

async function loadAndVerifyConnected() {
  await load()
  if (loadError.value) return
  const toCheck = clusters.value.filter(c => c.status === 'connected' && c.has_api_credential)
  await Promise.all(toCheck.map(c => verifyCluster(c, { skipListRefresh: true })))
  await load()
}
async function createCluster() {
  await apiFetch('/api/admin/k8s-clusters', { method: 'POST', body: JSON.stringify(form.value) })
  showCreate.value = false
  form.value = { name: '', endpoint: '', default_namespace: 'devops-status-system' }
  await load()
}
async function onboard(c: any) {
  onboardData.value = await apiFetch<any>(`/api/admin/k8s-clusters/${c.id}/onboard`, { method: 'POST' })
  onboardData.value._clusterId = c.id
}

function openManualModal(c: any) {
  manualModalCluster.value = c
  manualModalMode.value = 'register'
  manualSaToken.value = ''
  manualError.value = ''
}

function openTokenModal(c: any) {
  manualModalCluster.value = c
  manualModalMode.value = 'token'
  manualSaToken.value = ''
  manualError.value = ''
}

async function submitManualModal() {
  if (!manualModalCluster.value) return
  manualError.value = ''
  const tok = manualSaToken.value.trim()
  if (manualModalMode.value === 'token' && !tok) {
    manualError.value = 'Paste the bearer token.'
    return
  }
  manualSubmitting.value = true
  try {
    if (manualModalMode.value === 'register') {
      const ob = await apiFetch<any>(`/api/admin/k8s-clusters/${manualModalCluster.value.id}/onboard`, { method: 'POST' })
      const body: Record<string, string> = { token: ob.token }
      if (tok) body.cluster_bearer_token = tok
      await apiFetch(`/api/admin/k8s-clusters/${manualModalCluster.value.id}/register`, {
        method: 'POST',
        body: JSON.stringify(body),
      })
    } else {
      await apiFetch(`/api/admin/k8s-clusters/${manualModalCluster.value.id}/api-token`, {
        method: 'POST',
        body: JSON.stringify({ bearer_token: tok }),
      })
    }
    const savedId = manualModalCluster.value.id
    manualModalCluster.value = null
    await load()
    const row = clusters.value.find(x => x.id === savedId)
    if (row?.status === 'connected' && row?.has_api_credential) await verifyCluster(row)
  } catch (e: any) {
    manualError.value = e?.data?.error || e?.message || 'Request failed'
  } finally {
    manualSubmitting.value = false
  }
}
async function boot() {
  await loadBackendMeta()
  await loadAndVerifyConnected()
}
onMounted(boot)
</script>

<style scoped>
.backend-url-card { border: 1px solid var(--color-border-strong); border-radius: var(--radius); padding: 14px 16px; margin-bottom: 20px; max-width: 960px; background: var(--color-bg); box-shadow: var(--shadow-card); }
.backend-url-row { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; margin-bottom: 8px; }
.backend-url-label { font-size: 0.8rem; font-weight: 600; color: var(--color-text-secondary); text-transform: uppercase; letter-spacing: 0.02em; }
.backend-url-value { margin: 0 0 10px; font-size: 0.9rem; word-break: break-all; color: #0f172a; }
.backend-url-caption { margin: 0; font-size: 0.75rem; color: #64748b; line-height: 1.5; }
.url-kind-pill { font-size: 0.7rem; font-weight: 600; padding: 3px 10px; border-radius: 999px; text-transform: none; letter-spacing: 0; }
.url-kind-pill--tunnel { background: #dbeafe; color: #1e40af; }
.url-kind-pill--prod { background: #ecfdf5; color: #047857; }
.url-kind-pill--localhost { background: #fef3c7; color: #92400e; }
.load-error { background: #fef2f2; border: 1px solid #fecaca; color: #991b1b; padding: 12px 14px; border-radius: 8px; font-size: 0.85rem; margin-bottom: 16px; max-width: 960px; line-height: 1.5; }
.load-error-hint { display: block; margin-top: 8px; color: #7f1d1d; font-size: 0.8rem; }
.ping-cell { vertical-align: top; min-width: 200px; }
.ping-box { border: 1px solid var(--color-border); border-radius: 8px; padding: 10px; background: #fafafa; font-size: 0.78rem; }
.ping-box--ok { border-color: #bbf7d0; background: #f0fdf4; }
.ping-box--bad { border-color: #fecaca; background: #fef2f2; }
.ping-result { margin-top: 8px; line-height: 1.4; }
.ping-ok { font-weight: 600; color: #166534; font-family: monospace; font-size: 0.8rem; }
.ping-bad { color: #991b1b; }
.ping-time { display: block; font-size: 0.72rem; color: var(--color-text-secondary); margin-top: 4px; }
.ping-caption { margin: 8px 0 0; font-size: 0.72rem; color: #6b7280; line-height: 1.45; }
.ping-caption--rancher { margin-top: 6px; padding: 8px 10px; background: #eff6ff; border: 1px solid #bfdbfe; border-radius: 6px; color: #1e3a5f; }
.ping-code { font-size: 0.68rem; background: #f3f4f6; padding: 0 4px; border-radius: 3px; }
.ping-na { color: var(--color-text-secondary); }
.mono { font-family: monospace; font-size: 0.85rem; }
.status-badge { padding: 2px 8px; border-radius: 10px; font-size: 0.75rem; font-weight: 600; }
.badge--connected { background: #dcfce7; color: #166534; }
.badge--connected_api_unverified { background: #ffedd5; color: #9a3412; }
.badge--pending_registration { background: #fef3c7; color: #92400e; }
.badge--revoked { background: #fef2f2; color: #991b1b; }
.badge--degraded { background: #fef2f2; color: #991b1b; }
.actions-cell { display: flex; gap: 4px; flex-wrap: wrap; }
.instructions { margin-bottom: 16px; padding-left: 20px; font-size: 0.9rem; }
.instructions li { margin-bottom: 6px; }
.command-block { margin-bottom: 16px; }
.command-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.command-pre { background: #1f2937; color: #e5e7eb; padding: 16px; border-radius: 8px; font-family: monospace; font-size: 0.8rem; line-height: 1.5; white-space: pre; overflow-x: auto; max-height: 400px; overflow-y: auto; }
.token-info { font-size: 0.8rem; color: var(--color-text-secondary); margin-bottom: 12px; }
.help-note { font-size: 0.85rem; color: var(--color-text-secondary); margin-bottom: 12px; line-height: 1.45; }
.hint-cmd { font-size: 0.8rem; background: #f3f4f6; padding: 8px 10px; border-radius: 6px; margin-bottom: 12px; word-break: break-all; }
.hint-cmd--block { display: block; white-space: pre-wrap; }
.help-note--rancher { background: #eff6ff; border: 1px solid #bfdbfe; padding: 10px 12px; border-radius: 8px; color: #1e3a5f; }
.token-area { font-family: monospace; font-size: 0.8rem; resize: vertical; }
.cred-yes { color: #166534; font-weight: 600; font-size: 0.85rem; }
.cred-no { color: #b45309; font-weight: 600; font-size: 0.85rem; }
.tunnel-status { display: flex; align-items: center; gap: 8px; padding: 10px 14px; border-radius: 8px; font-size: 0.85rem; margin-bottom: 16px; }
.tunnel-status--ok { background: #dcfce7; color: #166534; border: 1px solid #bbf7d0; }
.tunnel-status--warn { background: #fef2f2; color: #991b1b; border: 1px solid #fecaca; }
.tunnel-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.tunnel-dot--green { background: #22c55e; }
.tunnel-dot--red { background: #ef4444; }
.localhost-warn { background: #fffbeb; border: 1px solid #fde68a; border-radius: 8px; padding: 16px; font-size: 0.85rem; color: #92400e; line-height: 1.6; margin-bottom: 16px; }
.localhost-warn code { background: #fef3c7; padding: 2px 6px; border-radius: 4px; font-size: 0.8rem; }
.localhost-warn p { margin-bottom: 8px; }
.localhost-warn p:last-child { margin-bottom: 0; }
</style>
