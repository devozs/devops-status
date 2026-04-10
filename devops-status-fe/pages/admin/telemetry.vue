<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Telemetry</h1>
      <button type="button" class="btn btn-primary btn-pill" @click="openCreate">
        <Plus class="btn-leading-icon" :size="18" :stroke-width="2" />
        Add telemetry
      </button>
    </div>

    <AdminCallout variant="info">
      <p>
        Telemetry definitions are reusable probes (HTTP, Prometheus, Kubernetes, CLI). Use <strong>Verify</strong> in the editor to test
        connectivity and QoS before attaching them to services or environments.
      </p>
    </AdminCallout>

    <div class="table-panel">
      <div class="table-split-scroll">
        <table class="data-table data-table--head">
          <colgroup>
            <col style="width: 72%" />
            <col style="width: 28%" />
          </colgroup>
          <thead>
            <tr>
              <th>Telemetry</th>
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
              <tr v-for="t in sources" :key="t.id">
                <td>
                  <div class="table-resource-cell">
                    <div class="table-resource-icon" aria-hidden="true">
                      <Activity :size="16" :stroke-width="2" />
                    </div>
                    <div class="table-resource-body">
                      <span class="table-resource-title">{{ t.name }}</span>
                      <span class="table-resource-meta">{{ t.adapter }} · {{ t.execution_target || 'backend' }}</span>
                    </div>
                  </div>
                </td>
                <td>
                  <button type="button" class="btn btn-sm" @click="openEdit(t)">Edit</button>
                  <button type="button" class="btn btn-sm" @click="openDuplicate(t)">Duplicate</button>
                  <button type="button" class="btn btn-sm" @click="openTest(t)">Test</button>
                  <button type="button" class="btn btn-sm btn-danger" @click="requestDeleteRow(t)">Delete</button>
                </td>
              </tr>
              <tr v-if="!sources?.length"><td colspan="2" class="empty-row">No telemetry configured.</td></tr>
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

    <div v-if="drawerTest" class="drawer">
      <div class="drawer-header">
        <strong>Test: {{ drawerTest.name }}</strong>
        <button type="button" class="btn btn-sm" @click="drawerTest = null">Close</button>
      </div>
      <TelemetryTestPanel
        ref="drawerTestRef"
        :adapter="drawerTest.adapter as AdapterType"
        :environments="environments"
        :clusters="clusters"
        :k8s-version="drawerK8sVersion"
        :busy="drawerBusy"
        @verify="runSavedTest"
      />
    </div>

    <TelemetryFormModal
      v-if="showModal"
      :initial="editing"
      :environments="environments"
      :clusters="clusters"
      :api-fetch="apiFetch"
      @close="closeModal"
      @saved="onSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Activity, Plus } from 'lucide-vue-next'
import type { AdapterType, TelemetryFormInitial, TelemetryK8sClusterRow, TelemetryRow } from '~/types/telemetry'

definePageMeta({ layout: 'admin' })
const { apiFetch } = useApi()

const sources = ref<TelemetryRow[]>([])
const environments = ref<{ id: string; name: string }[]>([])
const clusters = ref<TelemetryK8sClusterRow[]>([])
const showModal = ref(false)
const editing = ref<TelemetryFormInitial | null>(null)
const drawerTest = ref<TelemetryRow | null>(null)
const drawerBusy = ref(false)
const drawerTestRef = ref<{ setResult: (r: Record<string, unknown> | null) => void } | null>(null)

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

function requestDeleteRow(t: TelemetryRow) {
  deleteForm.value = {
    title: 'Delete telemetry',
    warning:
      'This telemetry definition will be permanently deleted. Service and environment links that use it may stop working. This action cannot be undone.',
    resourceKind: 'telemetry',
    resourceName: t.name,
    requireNameMatch: true,
    confirmLabel: 'Delete telemetry',
  }
  deleteAction.value = async () => {
    await apiFetch(`/api/admin/telemetry/${t.id}`, { method: 'DELETE' })
    await load()
  }
  deleteOpen.value = true
}

async function load() {
  sources.value = (await apiFetch<TelemetryRow[]>('/api/admin/telemetry')) || []
}

async function loadMeta() {
  try {
    environments.value = await apiFetch<{ id: string; name: string }[]>('/api/admin/environments')
  } catch {
    environments.value = []
  }
  try {
    clusters.value =
      (await apiFetch<TelemetryK8sClusterRow[]>('/api/admin/k8s-clusters')) || []
  } catch {
    clusters.value = []
  }
}

function openCreate() {
  editing.value = null
  showModal.value = true
}

function openEdit(t: TelemetryRow) {
  editing.value = { ...t, config_json: { ...(t.config_json || {}) }, qos_thresholds: { ...(t.qos_thresholds || {}) } }
  showModal.value = true
}

function suggestDuplicateName(sourceName: string): string {
  const base = sourceName.replace(/( \(copy(?: \d+)?\))+$/i, '').trim() || sourceName.trim()
  let candidate = `${base} (copy)`
  let n = 2
  while (sources.value.some((s) => s.name === candidate)) {
    candidate = `${base} (copy ${n})`
    n++
  }
  return candidate
}

function openDuplicate(t: TelemetryRow) {
  editing.value = {
    name: suggestDuplicateName(t.name),
    adapter: t.adapter,
    config_json: { ...(t.config_json || {}) },
    qos_thresholds: { ...(t.qos_thresholds || {}) },
    execution_target: t.execution_target,
    timeout_ms: t.timeout_ms,
    retries: t.retries ?? 1,
  }
  showModal.value = true
}

const drawerK8sVersion = computed(() => {
  const t = drawerTest.value
  if (!t) return ''
  const c = (t.config_json || {}) as Record<string, unknown>
  if (t.adapter === 'kubernetes') return String(c.k8s_version || '')
  if (t.adapter === 'cli' && t.execution_target === 'k8s_cluster') {
    return String(c.k8s_version || '')
  }
  return ''
})

function closeModal() {
  showModal.value = false
  editing.value = null
}

async function onSaved() {
  await load()
}

function openTest(t: TelemetryRow) {
  drawerTest.value = t
}

async function runSavedTest(payload: Record<string, unknown>) {
  if (!drawerTest.value) return
  drawerBusy.value = true
  drawerTestRef.value?.setResult(null)
  try {
    const t = drawerTest.value
    const body: Record<string, unknown> = {
      adapter: t.adapter,
      config_json: t.config_json,
      qos_thresholds: t.qos_thresholds || {},
      execution_target: t.execution_target || 'backend',
      telemetry_id: t.id,
    }
    if (payload.environment_id) body.environment_id = payload.environment_id
    if (payload.test_connect_only) body.test_connect_only = true
    const res = (await apiFetch('/api/admin/probes/test', {
      method: 'POST',
      body: JSON.stringify(body),
    })) as Record<string, unknown>
    drawerTestRef.value?.setResult(res)
  } catch (e: unknown) {
    drawerTestRef.value?.setResult({ success: false, error: e instanceof Error ? e.message : 'Failed' })
  } finally {
    drawerBusy.value = false
  }
}

onMounted(async () => {
  await Promise.all([load(), loadMeta()])
})
</script>

<style scoped>
.drawer { margin-top: 24px; padding: 16px; border: 1px solid var(--color-border-strong); border-radius: var(--radius); background: var(--color-bg); box-shadow: var(--shadow-card); }
.drawer-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
</style>
