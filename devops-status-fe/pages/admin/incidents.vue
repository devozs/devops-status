<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Incidents</h1>
    </div>

    <div class="table-wrap">
      <table class="data-table">
        <thead><tr><th>Title</th><th>Target</th><th>Severity</th><th>Status</th><th>Started</th><th>Actions</th></tr></thead>
        <tbody>
          <tr v-for="inc in incidents" :key="inc.id">
            <td>{{ inc.title }}</td>
            <td>{{ inc.target_type }}</td>
            <td>{{ inc.severity }}</td>
            <td><span class="status-badge" :class="'status--' + inc.status">{{ inc.status }}</span></td>
            <td>{{ formatDate(inc.started_at) }}</td>
            <td><button class="btn btn-sm" @click="openUpdate(inc)">Update</button></td>
          </tr>
          <tr v-if="!incidents?.length"><td colspan="6" class="empty-row">No incidents.</td></tr>
        </tbody>
      </table>
    </div>

    <div v-if="editing" class="modal-overlay" @click.self="editing = null">
      <div class="modal-card">
        <h2 class="modal-title">Update Incident</h2>
        <form class="modal-form" @submit.prevent="submitUpdate">
          <div class="form-group"><label class="form-label">Status</label>
            <select v-model="updateForm.status" class="form-input">
              <option value="investigating">Investigating</option>
              <option value="identified">Identified</option>
              <option value="monitoring">Monitoring</option>
              <option value="resolved">Resolved</option>
            </select>
          </div>
          <div class="form-group"><label class="form-label">Message</label><textarea v-model="updateForm.message" class="form-input" rows="3" /></div>
          <div class="modal-actions">
            <button type="button" class="btn" @click="editing = null">Cancel</button>
            <button type="submit" class="btn btn-primary">Submit</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'admin' })
const { apiFetch } = useApi()

const incidents = ref<any[]>([])
const editing = ref<any>(null)
const updateForm = ref({ status: '', message: '' })

async function load() { incidents.value = await apiFetch<any[]>('/api/admin/incidents') }

function openUpdate(inc: any) {
  editing.value = inc
  updateForm.value = { status: inc.status, message: '' }
}

async function submitUpdate() {
  if (!editing.value) return
  await apiFetch(`/api/admin/incidents/${editing.value.id}`, { method: 'PUT', body: JSON.stringify(updateForm.value) })
  editing.value = null
  await load()
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' })
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
.empty-row { color: var(--color-text-secondary); text-align: center; padding: 24px; }
.status-badge { padding: 2px 8px; border-radius: 10px; font-size: 0.75rem; font-weight: 600; }
.status--investigating { background: #fef3c7; color: #92400e; }
.status--identified { background: #dbeafe; color: #1e40af; }
.status--monitoring { background: #e0e7ff; color: #3730a3; }
.status--resolved { background: #dcfce7; color: #166534; }
.btn { padding: 6px 14px; border: 1px solid var(--color-border); border-radius: 6px; font-size: 0.85rem; background: var(--color-bg); }
.btn-primary { background: #1f2937; color: #fff; border-color: #1f2937; }
.btn-sm { padding: 4px 10px; font-size: 0.8rem; }
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.3); display: flex; align-items: center; justify-content: center; z-index: 200; }
.modal-card { background: var(--color-bg); border-radius: var(--radius); padding: 28px; width: 100%; max-width: 480px; }
.modal-title { font-size: 1.15rem; font-weight: 700; margin-bottom: 20px; }
.modal-form { display: flex; flex-direction: column; gap: 14px; }
.form-group { display: flex; flex-direction: column; gap: 4px; }
.form-label { font-size: 0.85rem; font-weight: 500; color: var(--color-text-secondary); }
.form-input { padding: 8px 10px; border: 1px solid var(--color-border); border-radius: 6px; font-size: 0.9rem; }
.modal-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 8px; }
</style>
