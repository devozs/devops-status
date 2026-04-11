<template>
  <div
    class="admin-segmented"
    role="radiogroup"
    :aria-label="ariaLabel"
    :aria-disabled="disabled ? 'true' : undefined"
  >
    <label
      v-for="opt in options"
      :key="String(opt.value)"
      class="admin-segmented__option"
      :class="{ 'admin-segmented__option--checked': isSelected(opt.value) }"
    >
      <input
        class="admin-segmented__control"
        type="radio"
        :name="nameAttr"
        :value="String(opt.value)"
        :checked="isSelected(opt.value)"
        :disabled="disabled || opt.disabled"
        @change="onPick(opt.value)"
      />
      <span class="admin-segmented__text">{{ opt.label }}</span>
    </label>
  </div>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue'

export type SegmentedOption = { value: string | number; label: string; disabled?: boolean }

const props = withDefaults(
  defineProps<{
    modelValue: string | number
    options: SegmentedOption[]
    name?: string
    disabled?: boolean
    ariaLabel?: string
  }>(),
  { name: undefined, disabled: false, ariaLabel: undefined },
)

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

const autoId = useId()
const nameAttr = computed(() => props.name ?? `segmented-${autoId}`)

function norm(v: string | number) {
  return String(v)
}

function isSelected(v: string | number) {
  return norm(props.modelValue) === norm(v)
}

function onPick(v: string | number) {
  if (props.disabled) return
  emit('update:modelValue', v)
}
</script>

<style scoped>
.admin-segmented {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  width: 100%;
  min-height: 44px;
  padding: 3px;
  row-gap: 4px;
  column-gap: 0;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-control);
  background: var(--nebius-segment-bg);
  box-sizing: border-box;
}

.admin-segmented__option {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0;
  padding: 8px 10px;
  border-radius: calc(var(--radius-control) - 4px);
  border: 1px solid transparent;
  background: transparent;
  color: var(--color-form-label);
  font-size: 0.875rem;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  transition:
    background 0.12s ease,
    border-color 0.12s ease,
    color 0.12s ease;
  user-select: none;
}

.admin-segmented__option:hover:not(:has(input:disabled)) {
  background: rgba(90, 93, 255, 0.06);
}

.admin-segmented__option--checked {
  background: var(--nebius-segment-selected-bg);
  border-color: var(--nebius-segment-active-border);
  color: var(--color-primary);
}

.admin-segmented__control {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.admin-segmented__text {
  text-align: center;
  width: 100%;
  line-height: 1.25;
  word-break: break-word;
  hyphens: auto;
}

.admin-segmented__option:has(.admin-segmented__control:disabled) {
  opacity: 0.45;
  cursor: not-allowed;
}
</style>
