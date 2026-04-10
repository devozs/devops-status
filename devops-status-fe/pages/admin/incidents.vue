<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Incidents</h1>
    </div>

    <div class="table-panel">
      <div class="table-split-scroll">
        <table class="data-table data-table--head">
          <colgroup>
            <col style="width: 26%" />
            <col style="width: 12%" />
            <col style="width: 10%" />
            <col style="width: 12%" />
            <col style="width: 22%" />
            <col style="width: 18%" />
          </colgroup>
          <thead>
            <tr>
              <th>Title</th>
              <th>Target</th>
              <th>Severity</th>
              <th>Status</th>
              <th>Started</th>
              <th>Actions</th>
            </tr>
          </thead>
        </table>
        <div class="table-body-shell">
          <table class="data-table data-table--body">
            <colgroup>
              <col style="width: 26%" />
              <col style="width: 12%" />
              <col style="width: 10%" />
              <col style="width: 12%" />
              <col style="width: 22%" />
              <col style="width: 18%" />
            </colgroup>
            <tbody>
              <tr v-for="inc in incidents" :key="inc.id">
                <td>{{ inc.title }}</td>
                <td>{{ inc.target_type }}</td>
                <td>{{ inc.severity }}</td>
                <td><span class="status-badge" :class="'status--' + inc.status">{{ inc.status }}</span></td>
                <td class="table-cell-muted">{{ formatDate(inc.started_at) }}</td>
                <td><button type="button" class="btn btn-sm" @click="openUpdate(inc)">Update</button></td>
              </tr>
              <tr v-if="!incidents?.length"><td colspan="6" class="empty-row">No incidents.</td></tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <div v-if="editing" class="modal-overlay" @click.self="editing = null">
      <div class="modal-card">
        <h2 class="modal-title">Update Incident</h2>
        <p class="modal-subtitle">Change status and add a public message for the status page.</p>
        <form class="modal-form" @submit.prevent="submitUpdate">
          <h3 class="modal-section-title">Details</h3>
          <div class="form-group"><label class="form-label">Status</label>
            <select v-model="updateForm.status" class="form-input">
              <option value="investigating">Investigating</option>
              <option value="identified">Identified</option>
              <option value="monitoring">Monitoring</option>
              <option value="resolved">Resolved</option>
            </select>
          </div>
          <div class="form-group"><label class="form-label">Message</label><textarea v-model="updateForm.message" class="form-input form-input--multiline" rows="3" /></div>
          <div class="modal-actions">
            <button type="button" class="btn" @click="editing = null">Cancel</button>
            <button type="submit" class="btn btn-primary btn-pill">Submit</button>
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
.status-badge { padding: 2px 8px; border-radius: 10px; font-size: 0.75rem; font-weight: 600; }
.status--investigating { background: #fef3c7; color: #92400e; }
.status--identified { background: #dbeafe; color: #1e40af; }
.status--monitoring { background: #e0e7ff; color: #3730a3; }
.status--resolved { background: #dcfce7; color: #166534; }
</style>
