<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Kubernetes Clusters</h1>
      <button class="btn btn-primary" @click="showCreate = true">Add Cluster</button>
    </div>

    <div class="table-wrap">
      <table class="data-table">
        <thead><tr><th>Name</th><th>Endpoint</th><th>Version</th><th>Status</th><th>Actions</th></tr></thead>
        <tbody>
          <tr v-for="c in clusters" :key="c.id">
            <td>{{ c.name }}</td>
            <td class="mono">{{ c.endpoint }}</td>
            <td>{{ c.k8s_version || '-' }}</td>
            <td><span class="status-badge" :class="'badge--' + c.status">{{ c.status }}</span></td>
            <td>
              <button class="btn btn-sm" @click="onboard(c)">Onboard</button>
              <button class="btn btn-sm btn-danger" @click="revoke(c.id)">Revoke</button>
              <button class="btn btn-sm btn-danger" @click="deleteCluster(c.id)">Delete</button>
            </td>
          </tr>
          <tr v-if="!clusters?.length"><td colspan="5" class="empty-row">No clusters registered.</td></tr>
        </tbody>
      </table>
    </div>

    <!-- Onboarding modal -->
    <div v-if="onboardData" class="modal-overlay" @click.self="onboardData = null">
      <div class="modal-card modal-card--wide">
        <h2 class="modal-title">Onboarding Instructions</h2>
        <ol class="instructions">
          <li v-for="(step, i) in onboardData.instructions" :key="i">{{ step }}</li>
        </ol>
        <div class="manifest-block">
          <label class="form-label">Kubernetes Manifest</label>
          <textarea class="form-input mono" rows="12" readonly :value="onboardData.manifest" />
        </div>
        <p class="token-info">Token expires at: {{ onboardData.expires_at }}</p>
        <div class="modal-actions"><button class="btn" @click="onboardData = null">Close</button></div>
      </div>
    </div>

    <!-- Create modal -->
    <div v-if="showCreate" class="modal-overlay" @click.self="showCreate = false">
      <div class="modal-card">
        <h2 class="modal-title">Register Cluster</h2>
        <form class="modal-form" @submit.prevent="createCluster">
          <div class="form-group"><label class="form-label">Name</label><input v-model="form.name" class="form-input" required /></div>
          <div class="form-group"><label class="form-label">API Endpoint</label><input v-model="form.endpoint" class="form-input" required placeholder="https://k8s-api:6443" /></div>
          <div class="form-group"><label class="form-label">Default Namespace</label><input v-model="form.default_namespace" class="form-input" /></div>
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

const clusters = ref<any[]>([])
const showCreate = ref(false)
const onboardData = ref<any>(null)
const form = ref({ name: '', endpoint: '', default_namespace: 'default' })

async function load() { clusters.value = await apiFetch<any[]>('/api/admin/k8s-clusters') }
async function createCluster() {
  await apiFetch('/api/admin/k8s-clusters', { method: 'POST', body: JSON.stringify(form.value) })
  showCreate.value = false
  form.value = { name: '', endpoint: '', default_namespace: 'default' }
  await load()
}
async function onboard(c: any) {
  onboardData.value = await apiFetch<any>(`/api/admin/k8s-clusters/${c.id}/onboard`, { method: 'POST' })
}
async function revoke(id: string) {
  if (!confirm('Revoke cluster credentials?')) return
  await apiFetch(`/api/admin/k8s-clusters/${id}/revoke`, { method: 'POST' })
  await load()
}
async function deleteCluster(id: string) {
  if (!confirm('Delete cluster?')) return
  await apiFetch(`/api/admin/k8s-clusters/${id}`, { method: 'DELETE' })
  await load()
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
.status-badge { padding: 2px 8px; border-radius: 10px; font-size: 0.75rem; font-weight: 600; }
.badge--connected { background: #dcfce7; color: #166534; }
.badge--pending_registration { background: #fef3c7; color: #92400e; }
.badge--revoked { background: #fef2f2; color: #991b1b; }
.badge--degraded { background: #fef2f2; color: #991b1b; }
.btn { padding: 6px 14px; border: 1px solid var(--color-border); border-radius: 6px; font-size: 0.85rem; background: var(--color-bg); }
.btn-primary { background: #1f2937; color: #fff; border-color: #1f2937; }
.btn-danger { color: var(--color-red); border-color: var(--color-red); }
.btn-sm { padding: 4px 10px; font-size: 0.8rem; }
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.3); display: flex; align-items: center; justify-content: center; z-index: 200; }
.modal-card { background: var(--color-bg); border-radius: var(--radius); padding: 28px; width: 100%; max-width: 480px; }
.modal-card--wide { max-width: 700px; }
.modal-title { font-size: 1.15rem; font-weight: 700; margin-bottom: 20px; }
.modal-form { display: flex; flex-direction: column; gap: 14px; }
.form-group { display: flex; flex-direction: column; gap: 4px; }
.form-label { font-size: 0.85rem; font-weight: 500; color: var(--color-text-secondary); }
.form-input { padding: 8px 10px; border: 1px solid var(--color-border); border-radius: 6px; font-size: 0.9rem; }
.modal-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 8px; }
.instructions { margin-bottom: 16px; padding-left: 20px; font-size: 0.9rem; }
.instructions li { margin-bottom: 6px; }
.manifest-block { margin-bottom: 12px; }
.manifest-block textarea { width: 100%; resize: vertical; }
.token-info { font-size: 0.8rem; color: var(--color-text-secondary); margin-bottom: 12px; }
</style>
