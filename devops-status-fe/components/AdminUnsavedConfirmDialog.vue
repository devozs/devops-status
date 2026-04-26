<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="modal-overlay admin-unsaved-dialog-overlay"
      @click.self="dismiss"
    >
      <div
        class="modal-card modal-card--confirm-warn"
        role="alertdialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        @keydown.escape.stop.prevent="dismiss"
      >
        <div class="admin-unsaved-dialog__header">
          <h2 :id="titleId" class="modal-title admin-unsaved-dialog__title">{{ title }}</h2>
          <button
            type="button"
            class="admin-unsaved-dialog__close"
            aria-label="Close"
            @click="dismiss"
          >
            <X :size="22" :stroke-width="2" />
          </button>
        </div>

        <AdminCallout variant="warning" class="admin-unsaved-dialog__callout">
          <p class="admin-unsaved-dialog__warning-text">{{ warning }}</p>
        </AdminCallout>

        <div class="modal-actions admin-unsaved-dialog__actions">
          <button type="button" class="btn btn-ghost-pill" @click="dismiss">Cancel</button>
          <button type="button" class="btn btn-primary btn-pill" @click.stop="onConfirm">
            {{ confirmLabel }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { X } from 'lucide-vue-next'

const props = defineProps<{
  modelValue: boolean
  title: string
  warning: string
  /** e.g. "Discard" — primary action to leave without saving */
  confirmLabel: string
  /**
   * Direct callback when user confirms discard. Prefer this over emits: Teleport/multi-root parents
   * can fail to deliver custom events reliably in some Vue/Nuxt setups.
   */
  discardConfirm?: () => void
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  unsavedConfirm: []
  cancel: []
}>()

const titleId = useId()

function dismiss() {
  emit('update:modelValue', false)
  emit('cancel')
}

function onConfirm() {
  if (props.discardConfirm) {
    props.discardConfirm()
    return
  }
  emit('unsavedConfirm')
}
</script>

<style scoped>
.admin-unsaved-dialog-overlay {
  z-index: 230;
}

.admin-unsaved-dialog__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 4px;
}

.admin-unsaved-dialog__title {
  margin-bottom: 0;
  flex: 1;
  min-width: 0;
}

.admin-unsaved-dialog__close {
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

.admin-unsaved-dialog__close:hover {
  background: var(--color-bg-secondary);
  color: var(--color-text);
}

.admin-unsaved-dialog__callout {
  margin-top: 12px;
  margin-bottom: 8px;
}

.admin-unsaved-dialog__warning-text {
  margin: 0;
  white-space: pre-line;
  line-height: 1.5;
}

.admin-unsaved-dialog__actions {
  margin-top: 16px;
  justify-content: flex-end;
}
</style>
