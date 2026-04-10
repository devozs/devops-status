<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Probe Bindings</h1>
      <button class="btn btn-primary" @click="openCreate">Add Binding</button>
    </div>

    <h2 class="section-title">Service Bindings</h2>
    <div class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th>Service</th>
            <th>Probe Kind</th>
            <th>Data Source</th>
            <th>Interval</th>
            <th>Window</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="b in serviceBindings" :key="b.id">
            <td>{{ serviceName(b.service_id) }}</td>
            <td>{{ b.probe_kind }}</td>
            <td>{{ dsName(b.data_source_id) }}</td>
            <td>{{ b.sample_interval_sec }}s</td>
            <td>{{ b.window_size }}</td>
            <td><button class="btn btn-sm btn-danger" @click="deleteBinding('services', b.id)">Delete</button></td>
          </tr>
          <tr v-if="!serviceBindings.length"><td colspan="6" class="empty-row">No service bindings.</td></tr>
        </tbody>
      </table>
    </div>

    <h2 class="section-title" style="margin-top:32px">Environment Bindings</h2>
    <div class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th>Environment</th>
            <th>Probe Kind</th>
            <th>Data Source</th>
            <th>Interval</th>
            <th>Window</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="b in envBindings" :key="b.id">
            <td>{{ envName(b.environment_id) }}</td>
            <td>{{ b.probe_kind }}</td>
            <td>{{ dsName(b.data_source_id) }}</td>
            <td>{{ b.sample_interval_sec }}s</td>
            <td>{{ b.window_size }}</td>
            <td><button class="btn btn-sm btn-danger" @click="deleteBinding('environments', b.id)">Delete</button></td>
          </tr>
          <tr v-if="!envBindings.length"><td colspan="6" class="empty-row">No environment bindings.</td></tr>
        </tbody>
      </table>
    </div>

    <!-- Create modal -->
    <div v-if="showCreate" class="modal-overlay" @click.self="showCreate = false">
      <div class="modal-card">
        <h2 class="modal-title">Create Binding</h2>
        <form class="modal-form" @submit.prevent="createBinding">
          <div class="form-group">
            <label class="form-label">Target Type</label>
            <select v-model="form.target_type" class="form-input">
              <option value="service">Service</option>
              <option value="environment">Environment</option>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">Target</label>
            <select v-model="form.target_id" class="form-input" required>
              <option value="">Select...</option>
              <template v-if="form.target_type === 'service'">
                <option v-for="s in services" :key="s.id" :value="s.id">{{ s.name }}</option>
              </template>
              <template v-else>
                <option v-for="e in environments" :key="e.id" :value="e.id">{{ e.name }}</option>
              </template>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">Probe Kind</label>
            <select v-model="form.probe_kind" class="form-input">
              <option value="operational">Operational</option>
              <option value="qos">QoS</option>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">Data Source</label>
            <select v-model="form.data_source_id" class="form-input" required>
              <option value="">Select...</option>
              <option v-for="ds in dataSources" :key="ds.id" :value="ds.id">{{ ds.name }} ({{ ds.adapter }})</option>
            </select>
          </div>
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">Interval (sec)</label>
              <input v-model.number="form.sample_interval_sec" type="number" min="10" class="form-input" />
            </div>
            <div class="form-group">
              <label class="form-label">Window</label>
              <input v-model.number="form.window_size" type="number" min="1" class="form-input" />
            </div>
          </div>
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">Failures to Down</label>
              <input v-model.number="form.consecutive_failures_to_down" type="number" min="1" class="form-input" />
            </div>
            <div class="form-group">
              <label class="form-label">Success to Recover</label>
              <input v-model.number="form.consecutive_success_to_recover" type="number" min="1" class="form-input" />
            </div>
          </div>
          <div class="modal-actions">
            <button type="button" class="btn" @click="showCreate = false">Cancel</button>
            <button type="submit" class="btn btn-primary">Create</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'admin' })

const { apiFetch } = useApi()

const serviceBindings = ref<any[]>([])
const envBindings = ref<any[]>([])
const services = ref<any[]>([])
const environments = ref<any[]>([])
const dataSources = ref<any[]>([])
const showCreate = ref(false)

const form = ref({
  target_type: 'service',
  target_id: '',
  probe_kind: 'operational',
  data_source_id: '',
  sample_interval_sec: 60,
  window_size: 5,
  consecutive_failures_to_down: 3,
  consecutive_success_to_recover: 2,
})

async function loadAll() {
  const [sb, eb, svc, env, ds] = await Promise.all([
    apiFetch<any[]>('/api/admin/bindings/services'),
    apiFetch<any[]>('/api/admin/bindings/environments'),
    apiFetch<any[]>('/api/admin/services'),
    apiFetch<any[]>('/api/admin/environments'),
    apiFetch<any[]>('/api/admin/data-sources'),
  ])
  serviceBindings.value = sb || []
  envBindings.value = eb || []
  services.value = svc || []
  environments.value = env || []
  dataSources.value = ds || []
}

function serviceName(id: string) {
  return services.value.find((s: any) => s.id === id)?.name || id
}

function envName(id: string) {
  return environments.value.find((e: any) => e.id === id)?.name || id
}

function dsName(id: string) {
  return dataSources.value.find((d: any) => d.id === id)?.name || id
}

function openCreate() {
  form.value = {
    target_type: 'service', target_id: '', probe_kind: 'operational', data_source_id: '',
    sample_interval_sec: 60, window_size: 5, consecutive_failures_to_down: 3, consecutive_success_to_recover: 2,
  }
  showCreate.value = true
}

async function createBinding() {
  const payload: any = { ...form.value }
  const tt = payload.target_type
  delete payload.target_type

  if (tt === 'service') {
    payload.service_id = payload.target_id
  } else {
    payload.environment_id = payload.target_id
  }
  delete payload.target_id

  await apiFetch(`/api/admin/bindings/${tt === 'service' ? 'services' : 'environments'}`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  showCreate.value = false
  await loadAll()
}

async function deleteBinding(scope: string, id: string) {
  if (!confirm('Delete this binding?')) return
  await apiFetch(`/api/admin/bindings/${scope}/${id}`, { method: 'DELETE' })
  await loadAll()
}

onMounted(loadAll)
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-title { font-size: 1.5rem; font-weight: 700; }
.section-title { font-size: 1.1rem; font-weight: 600; margin-bottom: 12px; }
.table-wrap { overflow-x: auto; }
.data-table { width: 100%; border-collapse: collapse; background: var(--color-bg); border: 1px solid var(--color-border); border-radius: var(--radius); }
.data-table th, .data-table td { padding: 10px 14px; text-align: left; border-bottom: 1px solid var(--color-border); font-size: 0.9rem; }
.data-table th { font-weight: 600; font-size: 0.8rem; color: var(--color-text-secondary); text-transform: uppercase; letter-spacing: 0.04em; }
.empty-row { color: var(--color-text-secondary); text-align: center; padding: 24px; }
.btn { padding: 6px 14px; border: 1px solid var(--color-border); border-radius: 6px; font-size: 0.85rem; background: var(--color-bg); }
.btn-primary { background: #1f2937; color: #fff; border-color: #1f2937; }
.btn-danger { color: var(--color-red); border-color: var(--color-red); }
.btn-sm { padding: 4px 10px; font-size: 0.8rem; }
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
