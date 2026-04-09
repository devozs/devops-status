<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Data Sources</h1>
      <button class="btn btn-primary" @click="showCreate = true">Add Data Source</button>
    </div>

    <div class="table-wrap">
      <table class="data-table">
        <thead><tr><th>Name</th><th>Type</th><th>Adapter</th><th>Timeout</th><th>Actions</th></tr></thead>
        <tbody>
          <tr v-for="ds in sources" :key="ds.id">
            <td>{{ ds.name }}</td>
            <td>{{ ds.ds_type }}</td>
            <td class="mono">{{ ds.adapter }}</td>
            <td>{{ ds.timeout_ms }}ms</td>
            <td>
              <button class="btn btn-sm" @click="testSource(ds)">Test</button>
              <button class="btn btn-sm btn-danger" @click="deleteSource(ds.id)">Delete</button>
            </td>
          </tr>
          <tr v-if="!sources?.length"><td colspan="5" class="empty-row">No data sources configured.</td></tr>
        </tbody>
      </table>
    </div>

    <div v-if="testResult" class="test-result" :class="testResult.success ? 'test--ok' : 'test--fail'">
      <strong>Test Result:</strong> {{ testResult.success ? 'Success' : 'Failed' }}
      <span v-if="testResult.error"> - {{ testResult.error }}</span>
      <span v-if="testResult.latency_ms"> ({{ testResult.latency_ms }}ms)</span>
      <button class="btn btn-sm" @click="testResult = null">Dismiss</button>
    </div>

    <div v-if="showCreate" class="modal-overlay" @click.self="showCreate = false">
      <div class="modal-card">
        <h2 class="modal-title">Create Data Source</h2>
        <form class="modal-form" @submit.prevent="createSource">
          <div class="form-group"><label class="form-label">Name</label><input v-model="form.name" class="form-input" required /></div>
          <div class="form-row">
            <div class="form-group"><label class="form-label">Type</label>
              <select v-model="form.ds_type" class="form-input"><option value="operational">Operational</option><option value="qos">QoS</option></select>
            </div>
            <div class="form-group"><label class="form-label">Adapter</label>
              <select v-model="form.adapter" class="form-input"><option value="http">HTTP</option><option value="prometheus">Prometheus</option><option value="kubernetes">Kubernetes</option><option value="cli">CLI</option></select>
            </div>
          </div>
          <div class="form-group"><label class="form-label">Config (JSON)</label><textarea v-model="configStr" class="form-input" rows="4" /></div>
          <div class="form-group"><label class="form-label">Timeout (ms)</label><input v-model.number="form.timeout_ms" type="number" class="form-input" /></div>
          <div class="modal-actions">
            <button type="button" class="btn" @click="showCreate = false">Cancel</button>
            <button type="submit" class="btn btn-primary">Save</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'admin' })
const { apiFetch } = useApi()

const sources = ref<any[]>([])
const showCreate = ref(false)
const testResult = ref<any>(null)
const form = ref({ name: '', ds_type: 'operational', adapter: 'http', timeout_ms: 30000, retries: 1 })
const configStr = ref('{}')

async function load() { sources.value = await apiFetch<any[]>('/api/admin/data-sources') }
async function createSource() {
  let configJSON = {}
  try { configJSON = JSON.parse(configStr.value) } catch {}
  await apiFetch('/api/admin/data-sources', { method: 'POST', body: JSON.stringify({ ...form.value, config_json: configJSON }) })
  showCreate.value = false
  await load()
}
async function deleteSource(id: string) {
  if (!confirm('Delete?')) return
  await apiFetch(`/api/admin/data-sources/${id}`, { method: 'DELETE' })
  await load()
}
async function testSource(ds: any) {
  testResult.value = await apiFetch<any>('/api/admin/probes/test', {
    method: 'POST', body: JSON.stringify({ adapter: ds.adapter, config_json: ds.config_json })
  })
}
onMounted(load)
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-title { font-size: 1.5rem; font-weight: 700; }
.table-wrap { overflow-x: auto; }
.data-table { width: 100%; border-collapse: collapse; background: var(--color-bg); border: 1px solid var(--color-border); border-radius: var(--radius); }
.data-table th, .data-table td { padding: 10px 14px; text-align: left; border-bottom: 1px solid var(--color-border); font-size: 0.9rem; }
.data-table th { font-weight: 600; font-size: 0.8rem; color: var(--color-text-secondary); text-transform: uppercase; }
.mono { font-family: monospace; font-size: 0.85rem; }
.empty-row { color: var(--color-text-secondary); text-align: center; padding: 24px; }
.btn { padding: 6px 14px; border: 1px solid var(--color-border); border-radius: 6px; font-size: 0.85rem; background: var(--color-bg); }
.btn-primary { background: #1f2937; color: #fff; border-color: #1f2937; }
.btn-danger { color: var(--color-red); border-color: var(--color-red); }
.btn-sm { padding: 4px 10px; font-size: 0.8rem; }
.test-result { margin-top: 16px; padding: 12px 16px; border-radius: var(--radius); display: flex; gap: 8px; align-items: center; font-size: 0.9rem; }
.test--ok { background: #dcfce7; color: #166534; }
.test--fail { background: #fef2f2; color: #991b1b; }
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.3); display: flex; align-items: center; justify-content: center; z-index: 200; }
.modal-card { background: var(--color-bg); border-radius: var(--radius); padding: 28px; width: 100%; max-width: 520px; }
.modal-title { font-size: 1.15rem; font-weight: 700; margin-bottom: 20px; }
.modal-form { display: flex; flex-direction: column; gap: 14px; }
.form-group { display: flex; flex-direction: column; gap: 4px; }
.form-label { font-size: 0.85rem; font-weight: 500; color: var(--color-text-secondary); }
.form-input { padding: 8px 10px; border: 1px solid var(--color-border); border-radius: 6px; font-size: 0.9rem; }
.form-row { display: flex; gap: 16px; }
.modal-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 8px; }
</style>
