<template>
  <div class="shell-hint-insert">
    <span v-if="linkedTitle" class="shell-hint-insert__linked mono" :title="'Linked: ' + linkedTitle">{{ linkedTitle }}</span>
    <select
      class="form-input shell-hint-insert__select"
      :disabled="disabled || !hints.length"
      :aria-label="ariaLabel || 'Insert predefined shell hint'"
      @change="onSelectChange"
    >
      <option value="">{{ hints.length ? 'Insert hint…' : 'No hints' }}</option>
      <option v-for="h in hints" :key="h.id" :value="h.id">{{ h.title }}</option>
    </select>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TelemetryShellHintRow } from '~/types/telemetry'

const text = defineModel<string>({ required: true })
const hintId = defineModel<string>('hintId', { default: '' })

const props = withDefaults(
  defineProps<{
    hints: TelemetryShellHintRow[]
    disabled?: boolean
    ariaLabel?: string
  }>(),
  { disabled: false },
)

const linkedTitle = computed(() => {
  const id = hintId.value.trim()
  if (!id) return ''
  return props.hints.find((h) => h.id === id)?.title ?? ''
})

/** Normalize newlines; drop trailing newlines from a block before joining. */
function normalizeHintBlock(s: string): string {
  return s.replace(/\r\n/g, '\n').replace(/\n+$/u, '')
}

function onSelectChange(ev: Event) {
  const el = ev.target as HTMLSelectElement
  const id = el.value
  el.value = ''
  if (!id) return
  const h = props.hints.find((x) => x.id === id)
  if (!h) return

  const existing = text.value.replace(/\r\n/g, '\n')
  const existingTrimEnd = existing.trimEnd()
  const block = normalizeHintBlock(h.body.replace(/\r\n/g, '\n'))

  if (!existingTrimEnd.trim()) {
    text.value = h.body
    hintId.value = h.id
    return
  }

  text.value = block === '' ? existingTrimEnd : `${block}\n${existingTrimEnd}`
  hintId.value = ''
}
</script>

<style scoped>
.shell-hint-insert {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  justify-content: flex-end;
}
.shell-hint-insert__linked {
  font-size: 0.75rem;
  color: var(--admin-muted-fg, #64748b);
  max-width: 12rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.shell-hint-insert__select {
  width: auto;
  min-width: 9rem;
  max-width: 14rem;
  padding: 0.25rem 0.5rem;
  font-size: 0.8125rem;
}
</style>
