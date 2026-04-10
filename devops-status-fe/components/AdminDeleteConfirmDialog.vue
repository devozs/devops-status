<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="modal-overlay admin-delete-dialog-overlay"
      @click.self="dismiss"
    >
      <div
        class="modal-card modal-card--confirm-danger"
        role="alertdialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        @keydown.escape.stop.prevent="dismiss"
      >
        <div class="admin-delete-dialog__header">
          <h2 :id="titleId" class="modal-title admin-delete-dialog__title">{{ title }}</h2>
          <button
            type="button"
            class="admin-delete-dialog__close"
            aria-label="Close"
            @click="dismiss"
          >
            <X :size="22" :stroke-width="2" />
          </button>
        </div>

        <AdminCallout variant="warning" class="admin-delete-dialog__callout">
          <p class="admin-delete-dialog__warning-text">{{ warning }}</p>
        </AdminCallout>

        <template v-if="requireNameMatch && nameToMatch">
          <p class="admin-delete-dialog__hint">
            To confirm, enter the {{ resourceKind }} name below.
          </p>
          <div class="admin-delete-dialog__name-row">
            <strong class="admin-delete-dialog__name" :title="nameToMatch">{{ nameToMatch }}</strong>
            <button
              type="button"
              class="btn btn-sm admin-delete-dialog__copy"
              :aria-label="copied ? 'Copied' : 'Copy name'"
              @click="copyName"
            >
              <Check v-if="copied" :size="16" :stroke-width="2" />
              <Copy v-else :size="16" :stroke-width="2" />
              {{ copied ? 'Copied' : 'Copy' }}
            </button>
          </div>
          <div class="form-group admin-delete-dialog__input-group">
            <label class="form-label" :for="inputId">Confirmation</label>
            <input
              :id="inputId"
              ref="inputRef"
              v-model="typed"
              type="text"
              class="form-input"
              :placeholder="`Enter the ${resourceKind} name`"
              autocomplete="off"
              @keydown.enter.prevent="maybeConfirm"
            />
          </div>
        </template>

        <div class="modal-actions admin-delete-dialog__actions">
          <button type="button" class="btn btn-ghost-pill" @click="dismiss">Cancel</button>
          <button
            type="button"
            class="btn btn-sm btn-danger"
            :disabled="!canConfirm"
            @click="onConfirm"
          >
            {{ confirmLabel }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { Check, Copy, X } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title: string
    warning: string
    /** Shown in placeholder / hint, e.g. "service", "cluster". */
    resourceKind?: string
    /** Exact string the user must type when requireNameMatch is true. */
    resourceName?: string
    requireNameMatch?: boolean
    confirmLabel: string
  }>(),
  {
    resourceKind: 'resource',
    resourceName: '',
    requireNameMatch: true,
  },
)

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  confirm: []
  cancel: []
}>()

const typed = ref('')
const copied = ref(false)
const inputRef = ref<HTMLInputElement | null>(null)
let copyTimer: ReturnType<typeof setTimeout> | null = null

const titleId = useId()
const inputId = useId()

const nameToMatch = computed(() => (props.resourceName ?? '').trim())

const canConfirm = computed(() => {
  if (!props.requireNameMatch) return true
  if (!nameToMatch.value) return false
  return typed.value === nameToMatch.value
})

function resetLocal() {
  typed.value = ''
  copied.value = false
  if (copyTimer) {
    clearTimeout(copyTimer)
    copyTimer = null
  }
}

function dismiss() {
  resetLocal()
  emit('update:modelValue', false)
  emit('cancel')
}

function onConfirm() {
  if (!canConfirm.value) return
  emit('confirm')
}

/** Parent should close the dialog after a successful API call; we still reset typing when it closes. */
function maybeConfirm() {
  if (canConfirm.value) onConfirm()
}

async function copyName() {
  if (!nameToMatch.value) return
  try {
    await navigator.clipboard.writeText(nameToMatch.value)
    copied.value = true
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => {
      copied.value = false
      copyTimer = null
    }, 2000)
  } catch {
    copied.value = false
  }
}

watch(
  () => props.modelValue,
  (open) => {
    if (!open) {
      resetLocal()
      return
    }
    nextTick(() => {
      if (props.requireNameMatch && nameToMatch.value) {
        inputRef.value?.focus()
      }
    })
  },
)
</script>

<style scoped>
.admin-delete-dialog-overlay {
  z-index: 220;
}

.admin-delete-dialog__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 4px;
}

.admin-delete-dialog__title {
  margin-bottom: 0;
  flex: 1;
  min-width: 0;
}

.admin-delete-dialog__close {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  margin: -8px -8px 0 0;
  border: none;
  border-radius: var(--radius-pill, 999px);
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
}

.admin-delete-dialog__close:hover {
  background: var(--color-bg-secondary);
  color: var(--color-text);
}

.admin-delete-dialog__callout {
  margin-top: 12px;
  margin-bottom: 16px;
}

.admin-delete-dialog__warning-text {
  margin: 0;
  white-space: pre-line;
  line-height: 1.5;
}

.admin-delete-dialog__hint {
  margin: 0 0 8px;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.admin-delete-dialog__name-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.admin-delete-dialog__name {
  font-size: 0.95rem;
  word-break: break-word;
  flex: 1;
  min-width: 0;
}

.admin-delete-dialog__copy {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.admin-delete-dialog__input-group {
  margin-bottom: 20px;
}

.admin-delete-dialog__actions {
  margin-top: 8px;
  justify-content: flex-end;
}
</style>
