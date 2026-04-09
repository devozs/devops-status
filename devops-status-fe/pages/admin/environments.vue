<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Environments</h1>
      <button class="btn btn-primary" @click="showCreate = true">Add Environment</button>
    </div>

    <div class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Slug</th>
            <th>Type</th>
            <th>Criticality</th>
            <th>Public</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="env in environments" :key="env.id">
            <td>{{ env.name }}</td>
            <td class="mono">{{ env.slug }}</td>
            <td>{{ env.env_type }}</td>
            <td>{{ env.criticality }}</td>
            <td>{{ env.is_public ? 'Yes' : 'No' }}</td>
            <td>
              <button class="btn btn-sm" @click="editEnv(env)">Edit</button>
              <button class="btn btn-sm" @click="manageMembers(env)">Members</button>
              <button class="btn btn-sm btn-danger" @click="deleteEnv(env.id)">Delete</button>
            </td>
          </tr>
          <tr v-if="!environments?.length">
            <td colspan="6" class="empty-row">No environments configured.</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create/Edit modal -->
    <div v-if="showCreate || editing" class="modal-overlay" @click.self="closeModal">
      <div class="modal-card">
        <h2 class="modal-title">{{ editing ? 'Edit Environment' : 'Create Environment' }}</h2>
        <form class="modal-form" @submit.prevent="saveEnv">
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
          <div class="form-group">
            <label class="form-label">
              <input v-model="form.is_public" type="checkbox" /> Public
            </label>
          </div>
          <div class="modal-actions">
            <button type="button" class="btn" @click="closeModal">Cancel</button>
            <button type="submit" class="btn btn-primary">Save</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Members modal -->
    <div v-if="membersEnv" class="modal-overlay" @click.self="membersEnv = null">
      <div class="modal-card modal-card--wide">
        <h2 class="modal-title">Members of {{ membersEnv.name }}</h2>
        <div class="members-list">
          <div v-for="svc in members" :key="svc.id" class="member-row">
            <span>{{ svc.name }}</span>
            <button class="btn btn-sm btn-danger" @click="removeMember(svc.id)">Remove</button>
          </div>
          <p v-if="!members?.length" class="empty-text">No services assigned.</p>
        </div>
        <div class="add-member-row">
          <select v-model="addServiceId" class="form-input">
            <option value="">Select service to add...</option>
            <option v-for="svc in availableServices" :key="svc.id" :value="svc.id">{{ svc.name }}</option>
          </select>
          <button class="btn btn-primary btn-sm" :disabled="!addServiceId" @click="addMember">Add</button>
        </div>
        <div class="modal-actions">
          <button class="btn" @click="membersEnv = null">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'admin' })

const { apiFetch } = useApi()

const environments = ref<any[]>([])
const showCreate = ref(false)
const editing = ref<any>(null)
const form = ref({ name: '', slug: '', description: '', env_type: 'custom', criticality: 'standard', is_public: true })

const membersEnv = ref<any>(null)
const members = ref<any[]>([])
const allServices = ref<any[]>([])
const addServiceId = ref('')

const availableServices = computed(() => {
  const memberIds = new Set(members.value.map((m: any) => m.id))
  return allServices.value.filter((s: any) => !memberIds.has(s.id))
})

async function loadEnvironments() {
  environments.value = await apiFetch<any[]>('/api/admin/environments')
}

async function loadAllServices() {
  allServices.value = await apiFetch<any[]>('/api/admin/services')
}

function editEnv(env: any) {
  editing.value = env
  form.value = { ...env }
}

function closeModal() {
  showCreate.value = false
  editing.value = null
  form.value = { name: '', slug: '', description: '', env_type: 'custom', criticality: 'standard', is_public: true }
}

async function saveEnv() {
  if (editing.value) {
    await apiFetch(`/api/admin/environments/${editing.value.id}`, {
      method: 'PUT',
      body: JSON.stringify(form.value),
    })
  } else {
    await apiFetch('/api/admin/environments', {
      method: 'POST',
      body: JSON.stringify(form.value),
    })
  }
  closeModal()
  await loadEnvironments()
}

async function deleteEnv(id: string) {
  if (!confirm('Delete this environment?')) return
  await apiFetch(`/api/admin/environments/${id}`, { method: 'DELETE' })
  await loadEnvironments()
}

async function manageMembers(env: any) {
  membersEnv.value = env
  await loadAllServices()
  members.value = await apiFetch<any[]>(`/api/admin/environments/${env.id}/members`)
}

async function addMember() {
  if (!addServiceId.value || !membersEnv.value) return
  await apiFetch(`/api/admin/environments/${membersEnv.value.id}/members`, {
    method: 'POST',
    body: JSON.stringify({ service_id: addServiceId.value }),
  })
  addServiceId.value = ''
  members.value = await apiFetch<any[]>(`/api/admin/environments/${membersEnv.value.id}/members`)
}

async function removeMember(svcId: string) {
  if (!membersEnv.value) return
  await apiFetch(`/api/admin/environments/${membersEnv.value.id}/members/${svcId}`, { method: 'DELETE' })
  members.value = await apiFetch<any[]>(`/api/admin/environments/${membersEnv.value.id}/members`)
}

onMounted(loadEnvironments)
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
.modal-card--wide { max-width: 560px; }
.modal-title { font-size: 1.15rem; font-weight: 700; margin-bottom: 20px; }
.modal-form { display: flex; flex-direction: column; gap: 14px; }
.form-group { display: flex; flex-direction: column; gap: 4px; }
.form-label { font-size: 0.85rem; font-weight: 500; color: var(--color-text-secondary); }
.form-input { padding: 8px 10px; border: 1px solid var(--color-border); border-radius: 6px; font-size: 0.9rem; }
.form-row { display: flex; gap: 16px; }
.modal-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 8px; }
.members-list { margin-bottom: 16px; }
.member-row { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; border-bottom: 1px solid var(--color-border); }
.empty-text { color: var(--color-text-secondary); font-size: 0.9rem; }
.add-member-row { display: flex; gap: 8px; margin-bottom: 16px; }
.add-member-row .form-input { flex: 1; }
</style>
