<template>
  <div
    class="admin-radio-group"
    :class="{
      'admin-radio-group--vertical': direction === 'vertical',
      'admin-radio-group--horizontal': direction === 'horizontal',
    }"
    role="radiogroup"
    :aria-label="ariaLabel"
    :aria-disabled="disabled ? 'true' : undefined"
  >
    <label
      v-for="opt in options"
      :key="String(opt.value)"
      class="admin-radio-group__option"
      :class="{
        'admin-radio-group__option--checked': isSelected(opt.value),
        'admin-radio-group__option--disabled': disabled || opt.disabled,
      }"
    >
      <input
        class="admin-radio-group__input"
        type="radio"
        :name="nameAttr"
        :value="String(opt.value)"
        :checked="isSelected(opt.value)"
        :disabled="disabled || opt.disabled"
        @change="onPick(opt.value)"
      />
      <span class="admin-radio-group__indicator" aria-hidden="true">
        <span class="admin-radio-group__disc" />
      </span>
      <span class="admin-radio-group__text">{{ opt.label }}</span>
    </label>
  </div>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue'

export type AdminRadioOption = { value: string | number; label: string; disabled?: boolean }

const props = withDefaults(
  defineProps<{
    modelValue: string | number
    options: AdminRadioOption[]
    name?: string
    disabled?: boolean
    direction?: 'vertical' | 'horizontal'
    ariaLabel?: string
  }>(),
  {
    name: undefined,
    disabled: false,
    direction: 'vertical',
    ariaLabel: undefined,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

const autoId = useId()
const nameAttr = computed(() => props.name ?? `radio-${autoId}`)

function norm(v: string | number) {
  return String(v)
}

function isSelected(v: string | number) {
  return norm(props.modelValue) === norm(v)
}

function onPick(v: string | number) {
  if (props.disabled) return
  const opt = props.options.find((o) => norm(o.value) === norm(v))
  if (opt?.disabled) return
  emit('update:modelValue', v)
}
</script>

<style scoped>
.admin-radio-group {
  display: flex;
  gap: 12px;
}

.admin-radio-group--vertical {
  flex-direction: column;
  align-items: flex-start;
}

.admin-radio-group--horizontal {
  flex-direction: row;
  flex-wrap: wrap;
  align-items: center;
}

.admin-radio-group__option {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  margin: 0;
  cursor: pointer;
  font-size: 0.9375rem;
  font-weight: 500;
  color: var(--color-form-label);
  font-family: inherit;
  user-select: none;
  transition: color 0.12s ease;
}

.admin-radio-group__option:hover:not(.admin-radio-group__option--disabled) {
  color: var(--color-text);
}

.admin-radio-group__input {
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

.admin-radio-group__input:focus-visible + .admin-radio-group__indicator .admin-radio-group__disc {
  box-shadow: 0 0 0 3px var(--color-primary-muted);
}

.admin-radio-group__indicator {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
}

.admin-radio-group__disc {
  box-sizing: border-box;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 2px solid #c5cad3;
  background: var(--color-bg);
  transition:
    border-color 0.12s ease,
    background 0.12s ease,
    box-shadow 0.12s ease;
}

.admin-radio-group__option--checked .admin-radio-group__disc {
  border-color: var(--color-primary);
  background: var(--color-primary);
  box-shadow: inset 0 0 0 3px var(--color-bg);
}

.admin-radio-group__option--disabled {
  cursor: not-allowed;
  color: var(--color-text-muted);
}

.admin-radio-group__option--disabled .admin-radio-group__disc {
  border-color: #e5e7eb;
  background: #eceff3;
  box-shadow: none;
}

.admin-radio-group__option--disabled.admin-radio-group__option--checked .admin-radio-group__disc {
  border-color: #d1d5db;
  background: #d1d5db;
  box-shadow: inset 0 0 0 3px #f3f4f6;
}

.admin-radio-group__text {
  line-height: 1.35;
}
</style>
