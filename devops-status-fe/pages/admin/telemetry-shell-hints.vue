<template>
  <div>
    <div class="page-header page-header--compact">
      <h1 class="page-title">Telemetry shell hints</h1>
      <button type="button" class="btn btn-primary btn-pill" @click="openCreate">
        <Plus class="btn-leading-icon" :size="18" :stroke-width="2" />
        Add hint
      </button>
    </div>

    <AdminCallout variant="info">
      <p>
        Reusable snippets for <strong>Job env</strong>, <strong>node selector</strong>, <strong>container prep</strong>, and
        <strong>probe shell</strong> when editing telemetry. Admins pick these from the telemetry form; optional hint links are
        stored on the telemetry row and must match the snippet text on save.
      </p>
      <p>
        When you change a hint’s <strong>body</strong> and save, any telemetry that references that hint is updated in the
        background so stored <code class="mono">env</code>, <code class="mono">node_selector</code>, prep, and
        <code class="mono">cli_shell</code> stay aligned (rows that fail validation after the update are skipped and listed).
      </p>
    </AdminCallout>

    <AdminCallout v-if="syncBanner" :variant="syncBanner.errors.length ? 'warning' : 'info'" class="sync-banner-callout">
      <p>
        After saving the hint, <strong>{{ syncBanner.updated }}</strong> linked telemetry row(s) were updated so their stored
        shell text matches the new body.
      </p>
      <ul v-if="syncBanner.errors.length" class="sync-banner-errors">
        <li v-for="(msg, i) in syncBanner.errors" :key="i">{{ msg }}</li>
      </ul>
      <p v-if="syncBanner.errors.length" class="hint sync-banner-hint">
        Rows listed above were left unchanged because validation failed after applying the new body.
      </p>
      <button type="button" class="btn btn-sm" @click="syncBanner = null">Dismiss</button>
    </AdminCallout>

    <p v-if="loadError" class="load-error">{{ loadError }}</p>

    <div class="hint-kind-tabs" role="tablist" aria-label="Hint kind">
      <button
        v-for="tab in kindTabs"
        :key="tab.kind"
        type="button"
        class="hint-kind-tab"
        :class="{ 'hint-kind-tab--active': activeKind === tab.kind }"
        role="tab"
        :aria-selected="activeKind === tab.kind"
        @click="activeKind = tab.kind"
      >
        {{ tab.label }}
      </button>
    </div>

    <div class="table-panel">
      <div class="table-split-scroll">
        <table class="data-table data-table--head">
          <colgroup>
            <col style="width: 3%" />
            <col style="width: 22%" />
            <col style="width: 40%" />
            <col style="width: 20%" />
            <col style="width: 15%" />
          </colgroup>
          <thead>
            <tr>
              <th aria-label="Reorder" />
              <th>Title</th>
              <th>Body preview</th>
              <th>Description</th>
              <th>Actions</th>
            </tr>
          </thead>
        </table>
        <div class="table-body-shell">
          <table class="data-table data-table--body">
            <colgroup>
              <col style="width: 3%" />
              <col style="width: 22%" />
              <col style="width: 40%" />
              <col style="width: 20%" />
              <col style="width: 15%" />
            </colgroup>
            <tbody>
              <tr
                v-for="(h, idx) in rowsForKind"
                :key="h.id"
                draggable="true"
                class="hint-dnd-row"
                @dragstart="onDragStart($event, idx)"
                @dragover.prevent="onDragOver"
                @drop="onDrop(idx)"
                @dragend="dragSourceIndex = null"
              >
                <td class="hint-dnd-handle" title="Drag to reorder">
                  <GripVertical :size="18" :stroke-width="1.75" aria-hidden="true" />
                </td>
                <td>{{ h.title }}</td>
                <td class="mono hint-body-preview">{{ previewBody(h.body) }}</td>
                <td class="hint-desc-preview">{{ h.description || '—' }}</td>
                <td>
                  <button type="button" class="btn btn-sm" @click="openEdit(h)">Edit</button>
                  <button type="button" class="btn btn-sm btn-danger" @click="requestDelete(h)">Delete</button>
                </td>
              </tr>
              <tr v-if="!rowsForKind.length">
                <td colspan="5" class="empty-row">No hints for this category yet.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <p v-if="reorderError" class="field-error">{{ reorderError }}</p>
    <p v-if="reorderSaving" class="hint">Saving order…</p>

    <div v-if="editorOpen" class="modal-overlay" @click.self="closeEditor">
      <div class="modal-card modal-card--md">
        <div class="modal-card__header">
          <h2 class="modal-title">{{ editorId ? 'Edit hint' : 'New hint' }}</h2>
          <button type="button" class="modal-card__close" aria-label="Close" @click="closeEditor">
            <X :size="20" :stroke-width="2" />
          </button>
        </div>
        <form class="modal-form" @submit.prevent="saveEditor">
          <div class="form-group">
            <label class="form-label">Kind</label>
            <AdminSelect
              v-model="editorKind"
              aria-label="Hint kind"
              :options="kindSelectOptions"
              :disabled="Boolean(editorId)"
            />
          </div>
          <div class="form-group">
            <label class="form-label">Title</label>
            <input v-model="editorTitle" class="form-input" required maxlength="256" autocomplete="off" />
          </div>
          <div class="form-group">
            <label class="form-label">Body</label>
            <textarea v-model="editorBody" class="form-input mono" rows="10" required />
          </div>
          <div class="form-group">
            <label class="form-label">Description (optional)</label>
            <textarea v-model="editorDescription" class="form-input" rows="2" />
          </div>
          <p v-if="editorError" class="field-error">{{ editorError }}</p>
          <div class="modal-actions">
            <button type="button" class="btn btn-modal-cancel" @click="closeEditor">Cancel</button>
            <button type="submit" class="btn btn-save btn-pill" :disabled="editorSaving">
              {{ editorSaving ? 'Saving…' : 'Save' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <AdminDeleteConfirmDialog
      v-model="deleteOpen"
      :title="deleteDialogTitle"
      warning="Telemetry rows may still reference this hint by id; the UI will show unknown links until edited."
      resource-kind="shell hint"
      :resource-name="deleteTargetName"
      :require-name-match="false"
      confirm-label="Delete"
      @confirm="onDeleteConfirm"
      @cancel="deleteOpen = false"
    />
  </div>
</template>

<script setup lang="ts">
import { GripVertical, Plus, X } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import type { ShellHintKind, TelemetryShellHintRow, TelemetryShellHintSaveResponse } from '~/types/telemetry'
import { SHELL_HINT_KINDS } from '~/types/telemetry'

definePageMeta({ layout: 'admin' })

const { apiFetch } = useApi()

const kindTabs: { kind: ShellHintKind; label: string }[] = [
  { kind: 'job_env', label: 'Job / container env' },
  { kind: 'job_node_selector', label: 'Job node selector' },
  { kind: 'container_prep', label: 'Container prep' },
  { kind: 'probe_shell', label: 'Probe shell' },
]

const kindSelectOptions = SHELL_HINT_KINDS.map((k) => ({
  value: k,
  label: kindTabs.find((t) => t.kind === k)?.label ?? k,
}))

const activeKind = ref<ShellHintKind>('job_env')
const allRows = ref<TelemetryShellHintRow[]>([])
const syncBanner = ref<{ updated: number; errors: string[] } | null>(null)
const loadError = ref('')
const reorderError = ref('')
const reorderSaving = ref(false)
const dragSourceIndex = ref<number | null>(null)

const rowsForKind = computed(() => allRows.value.filter((r) => r.kind === activeKind.value))

async function loadAll() {
  loadError.value = ''
  try {
    allRows.value = (await apiFetch('/api/admin/telemetry-shell-hints')) as TelemetryShellHintRow[]
  } catch (e: unknown) {
    loadError.value = e instanceof Error ? e.message : 'Failed to load hints'
  }
}

watch(activeKind, () => {
  reorderError.value = ''
})

onMounted(() => {
  void loadAll()
})

function previewBody(s: string) {
  const t = s.replace(/\s+/g, ' ').trim()
  return t.length > 120 ? `${t.slice(0, 120)}…` : t || '—'
}

function onDragStart(ev: DragEvent, idx: number) {
  dragSourceIndex.value = idx
  ev.dataTransfer?.setData('text/plain', String(idx))
  ev.dataTransfer!.effectAllowed = 'move'
}

function onDragOver(ev: DragEvent) {
  ev.dataTransfer!.dropEffect = 'move'
}

async function onDrop(targetIdx: number) {
  const from = dragSourceIndex.value
  dragSourceIndex.value = null
  if (from === null || from === targetIdx) return
  const kind = activeKind.value
  const list = allRows.value.filter((r) => r.kind === kind)
  if (from < 0 || from >= list.length) return
  const reordered = [...list]
  const [moved] = reordered.splice(from, 1)
  reordered.splice(targetIdx, 0, moved)
  reorderError.value = ''
  reorderSaving.value = true
  try {
    await apiFetch('/api/admin/telemetry-shell-hints/reorder', {
      method: 'PATCH',
      body: JSON.stringify({
        kind,
        ordered_ids: reordered.map((r) => r.id),
      }),
    })
    await loadAll()
  } catch (e: unknown) {
    reorderError.value = e instanceof Error ? e.message : 'Reorder failed'
  } finally {
    reorderSaving.value = false
  }
}

const editorOpen = ref(false)
const editorId = ref<string | null>(null)
const editorKind = ref<ShellHintKind>('job_env')
const editorTitle = ref('')
const editorBody = ref('')
const editorDescription = ref('')
const editorError = ref('')
const editorSaving = ref(false)

function openCreate() {
  editorId.value = null
  syncBanner.value = null
  editorKind.value = activeKind.value
  editorTitle.value = ''
  editorBody.value = ''
  editorDescription.value = ''
  editorError.value = ''
  editorOpen.value = true
}

function openEdit(h: TelemetryShellHintRow) {
  editorId.value = h.id
  editorKind.value = h.kind
  editorTitle.value = h.title
  editorBody.value = h.body
  editorDescription.value = h.description || ''
  editorError.value = ''
  editorOpen.value = true
}

function closeEditor() {
  editorOpen.value = false
}

async function saveEditor() {
  editorError.value = ''
  editorSaving.value = true
  try {
    if (editorId.value) {
      const res = await apiFetch<TelemetryShellHintSaveResponse>(`/api/admin/telemetry-shell-hints/${editorId.value}`, {
        method: 'PUT',
        body: JSON.stringify({
          kind: editorKind.value,
          title: editorTitle.value.trim(),
          body: editorBody.value,
          description: editorDescription.value.trim(),
        }),
      })
      if (typeof res.telemetry_rows_updated === 'number') {
        syncBanner.value = {
          updated: res.telemetry_rows_updated,
          errors: res.telemetry_sync_errors ?? [],
        }
      } else {
        syncBanner.value = null
      }
    } else {
      syncBanner.value = null
      await apiFetch('/api/admin/telemetry-shell-hints', {
        method: 'POST',
        body: JSON.stringify({
          kind: editorKind.value,
          title: editorTitle.value.trim(),
          body: editorBody.value,
          description: editorDescription.value.trim(),
        }),
      })
    }
    closeEditor()
    await loadAll()
  } catch (e: unknown) {
    editorError.value =
      e && typeof e === 'object' && 'data' in e
        ? String((e as { data?: { error?: string } }).data?.error || 'Save failed')
        : e instanceof Error
          ? e.message
          : 'Save failed'
  } finally {
    editorSaving.value = false
  }
}

const deleteOpen = ref(false)
const deleteTarget = ref<TelemetryShellHintRow | null>(null)
const deleteDialogTitle = computed(() => (deleteTarget.value ? `Delete “${deleteTarget.value.title}”?` : 'Delete hint'))
const deleteTargetName = computed(() => deleteTarget.value?.title ?? '')

function requestDelete(h: TelemetryShellHintRow) {
  deleteTarget.value = h
  deleteOpen.value = true
}

async function onDeleteConfirm() {
  const h = deleteTarget.value
  deleteOpen.value = false
  if (!h) return
  try {
    await apiFetch(`/api/admin/telemetry-shell-hints/${h.id}`, { method: 'DELETE' })
    await loadAll()
  } catch {
    loadError.value = 'Delete failed'
  }
  deleteTarget.value = null
}
</script>

<style scoped>
.hint-kind-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-bottom: 1rem;
}
.hint-kind-tab {
  border: 1px solid var(--admin-border, #e2e8f0);
  background: var(--admin-surface-2, #f8fafc);
  padding: 0.35rem 0.75rem;
  border-radius: 999px;
  font-size: 0.8125rem;
  cursor: pointer;
}
.hint-kind-tab--active {
  background: var(--admin-accent-soft, #e0e7ff);
  border-color: var(--admin-accent, #6366f1);
}
.hint-dnd-handle {
  cursor: grab;
  color: var(--admin-muted-fg, #94a3b8);
  text-align: center;
}
.hint-dnd-row:active .hint-dnd-handle {
  cursor: grabbing;
}
.hint-body-preview {
  font-size: 0.75rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 48vw;
}
.hint-desc-preview {
  font-size: 0.8125rem;
  color: var(--admin-muted-fg, #64748b);
}
.modal-card--md {
  max-width: 36rem;
}
.sync-banner-callout {
  margin-bottom: 1rem;
}
.sync-banner-errors {
  margin: 0.5rem 0 0 1rem;
  font-size: 0.85rem;
}
.sync-banner-hint {
  margin-top: 0.35rem;
}
</style>
