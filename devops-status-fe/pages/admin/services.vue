<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Services</h1>
      <button class="btn btn-primary" @click="showCreate = true">Add Service</button>
    </div>

    <div class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Slug</th>
            <th>Criticality</th>
            <th>Public</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="svc in services" :key="svc.id">
            <td>{{ svc.name }}</td>
            <td class="mono">{{ svc.slug }}</td>
            <td>{{ svc.criticality }}</td>
            <td>{{ svc.is_public ? 'Yes' : 'No' }}</td>
            <td>
              <button class="btn btn-sm" @click="editService(svc)">Edit</button>
              <button class="btn btn-sm btn-danger" @click="deleteService(svc.id)">Delete</button>
            </td>
          </tr>
          <tr v-if="!services?.length">
            <td colspan="5" class="empty-row">No services configured.</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="showCreate || editing" class="modal-overlay" @click.self="closeModal">
      <div class="modal-card">
        <h2 class="modal-title">{{ editing ? 'Edit Service' : 'Create Service' }}</h2>
        <form class="modal-form" @submit.prevent="saveService">
          <div class="form-group">
            <label class="form-label">Name</label>
            <input v-model="form.name" class="form-input" required />
          </div>
          <div class="form-group">
            <label class="form-label">Slug</label>
            <input v-model="form.slug" class="form-input" required />
          </div>
          <div class="form-group">
            <label class="form-label">Description</label>
            <textarea v-model="form.description" class="form-input" rows="2" />
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
          <div class="modal-actions">
            <button type="button" class="btn" @click="closeModal">Cancel</button>
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

const services = ref<any[]>([])
const showCreate = ref(false)
const editing = ref<any>(null)
const form = ref({ name: '', slug: '', description: '', criticality: 'standard', is_public: true })

async function loadServices() {
  services.value = await apiFetch<any[]>('/api/admin/services')
}

function editService(svc: any) {
  editing.value = svc
  form.value = { ...svc }
}

function closeModal() {
  showCreate.value = false
  editing.value = null
  form.value = { name: '', slug: '', description: '', criticality: 'standard', is_public: true }
}

async function saveService() {
  if (editing.value) {
    await apiFetch(`/api/admin/services/${editing.value.id}`, {
      method: 'PUT',
      body: JSON.stringify(form.value),
    })
  } else {
    await apiFetch('/api/admin/services', {
      method: 'POST',
      body: JSON.stringify(form.value),
    })
  }
  closeModal()
  await loadServices()
}

async function deleteService(id: string) {
  if (!confirm('Delete this service?')) return
  await apiFetch(`/api/admin/services/${id}`, { method: 'DELETE' })
  await loadServices()
}

onMounted(loadServices)
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-title { font-size: 1.5rem; font-weight: 700; }
.table-wrap { overflow-x: auto; }
.data-table { width: 100%; border-collapse: collapse; background: var(--color-bg); border: 1px solid var(--color-border); border-radius: var(--radius); }
.data-table th, .data-table td { padding: 10px 14px; text-align: left; border-bottom: 1px solid var(--color-border); font-size: 0.9rem; }
.data-table th { font-weight: 600; font-size: 0.8rem; color: var(--color-text-secondary); text-transform: uppercase; letter-spacing: 0.04em; }
.mono { font-family: monospace; font-size: 0.85rem; }
.empty-row { color: var(--color-text-secondary); text-align: center; padding: 24px; }
.btn { padding: 6px 14px; border: 1px solid var(--color-border); border-radius: 6px; font-size: 0.85rem; background: var(--color-bg); }
.btn-primary { background: #1f2937; color: #fff; border-color: #1f2937; }
.btn-danger { color: var(--color-red); border-color: var(--color-red); }
.btn-sm { padding: 4px 10px; font-size: 0.8rem; }
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.3); display: flex; align-items: center; justify-content: center; z-index: 200; }
.modal-card { background: var(--color-bg); border-radius: var(--radius); padding: 28px; width: 100%; max-width: 480px; }
.modal-title { font-size: 1.15rem; font-weight: 700; margin-bottom: 20px; }
.modal-form { display: flex; flex-direction: column; gap: 14px; }
.form-group { display: flex; flex-direction: column; gap: 4px; }
.form-label { font-size: 0.85rem; font-weight: 500; color: var(--color-text-secondary); }
.form-input { padding: 8px 10px; border: 1px solid var(--color-border); border-radius: 6px; font-size: 0.9rem; }
.form-row { display: flex; gap: 16px; }
.modal-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 8px; }
</style>
