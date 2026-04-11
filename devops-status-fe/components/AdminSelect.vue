<template>
  <div ref="rootRef" class="admin-select" :class="{ 'admin-select--open': open, 'admin-select--disabled': disabled }">
    <button
      :id="buttonId"
      ref="triggerRef"
      type="button"
      class="admin-select__trigger"
      :class="{ 'admin-select__trigger--placeholder': !hasValue }"
      :disabled="disabled"
      :aria-expanded="open"
      aria-haspopup="listbox"
      :aria-controls="listboxId"
      :aria-invalid="invalid ? 'true' : undefined"
      :aria-required="required ? 'true' : undefined"
      :aria-label="ariaLabel"
      @click="toggle"
      @keydown="onTriggerKeydown"
    >
      <span class="admin-select__value">{{ displayLabel }}</span>
      <ChevronDown class="admin-select__chevron" :size="18" :stroke-width="2" aria-hidden="true" />
    </button>
    <Teleport to="body">
      <div
        v-show="open"
        ref="panelRef"
        class="admin-select__panel"
        :style="panelStyle"
        role="listbox"
        :id="listboxId"
        :aria-labelledby="ariaLabelledBy || undefined"
        tabindex="-1"
        @keydown="onPanelKeydown"
      >
        <template v-if="panelRows.length">
          <template v-for="(row, rowIdx) in panelRows" :key="row.key + '-' + rowIdx">
            <div v-if="row.kind === 'heading'" class="admin-select__heading" role="presentation">
              {{ row.label }}
            </div>
            <button
              v-else
              :id="`${listboxId}-opt-${row.flatIndex}`"
              type="button"
              role="option"
              class="admin-select__option"
              :class="{
                'admin-select__option--active': row.flatIndex === highlightedFlatIndex,
                'admin-select__option--selected': isSelected(row.value),
              }"
              :disabled="row.disabled"
              :aria-selected="isSelected(row.value)"
              @click="choose(row.value)"
              @mouseenter="highlightedFlatIndex = row.flatIndex"
            >
              {{ row.label }}
            </button>
          </template>
        </template>
        <div v-else class="admin-select__empty" role="presentation">No options</div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ChevronDown } from 'lucide-vue-next'
import {
  computed,
  nextTick,
  onMounted,
  onUnmounted,
  ref,
  useId,
  watch,
} from 'vue'

export type AdminSelectOption = { value: string | number; label: string; disabled?: boolean }
export type AdminSelectGroup = { label: string; options: AdminSelectOption[] }

const props = withDefaults(
  defineProps<{
    modelValue: string | number | null
    options?: AdminSelectOption[]
    groups?: AdminSelectGroup[]
    /** When set, a first row with value "" appears (e.g. "All" / "All components"). */
    emptyOptionLabel?: string
    placeholder?: string
    disabled?: boolean
    required?: boolean
    invalid?: boolean
    id?: string
    ariaLabel?: string
    ariaLabelledBy?: string
  }>(),
  {
    options: undefined,
    groups: undefined,
    emptyOptionLabel: undefined,
    placeholder: 'Select…',
    disabled: false,
    required: false,
    invalid: false,
    id: undefined,
    ariaLabel: undefined,
    ariaLabelledBy: undefined,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string | number | '']
}>()

const uid = useId()
const buttonId = computed(() => props.id ?? `admin-select-btn-${uid}`)
const listboxId = computed(() => `admin-select-list-${uid}`)

const rootRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const panelRef = ref<HTMLElement | null>(null)
const open = ref(false)
const panelStyle = ref<Record<string, string>>({})
const highlightedFlatIndex = ref(0)

type PanelRow =
  | { kind: 'heading'; label: string; key: string }
  | {
      kind: 'option'
      key: string
      value: string | number
      label: string
      disabled?: boolean
      flatIndex: number
    }

const flatSelectable = computed(() => {
  const out: { value: string | number; label: string; disabled?: boolean }[] = []
  if (props.emptyOptionLabel != null && props.emptyOptionLabel !== '') {
    out.push({ value: '', label: props.emptyOptionLabel })
  }
  if (props.groups?.length) {
    for (const g of props.groups) {
      for (const o of g.options) {
        out.push({ ...o })
      }
    }
    return out
  }
  return [...out, ...(props.options?.map((o) => ({ ...o })) ?? [])]
})

const panelRows = computed<PanelRow[]>(() => {
  const rows: PanelRow[] = []
  let fi = 0
  if (props.emptyOptionLabel != null && props.emptyOptionLabel !== '') {
    rows.push({
      kind: 'option',
      key: 'empty',
      value: '',
      label: props.emptyOptionLabel,
      flatIndex: fi,
    })
    fi++
  }
  if (props.groups?.length) {
    props.groups.forEach((g, gi) => {
      rows.push({ kind: 'heading', label: g.label, key: `h-${gi}` })
      for (const o of g.options) {
        rows.push({
          kind: 'option',
          key: `o-${fi}-${String(o.value)}`,
          value: o.value,
          label: o.label,
          disabled: o.disabled,
          flatIndex: fi,
        })
        fi++
      }
    })
    return rows
  }
  for (const o of props.options ?? []) {
    rows.push({
      kind: 'option',
      key: `o-${fi}-${String(o.value)}`,
      value: o.value,
      label: o.label,
      disabled: o.disabled,
      flatIndex: fi,
    })
    fi++
  }
  return rows
})

function norm(v: string | number | null | undefined) {
  if (v === null || v === undefined) return ''
  return String(v)
}

const hasValue = computed(() => {
  const v = norm(props.modelValue)
  if (v === '') {
    return (
      flatSelectable.value.some((o) => norm(o.value) === '') || Boolean(props.emptyOptionLabel)
    )
  }
  return true
})

function isSelected(v: string | number) {
  return norm(props.modelValue) === norm(v)
}

const displayLabel = computed(() => {
  const v = norm(props.modelValue)
  if (v === '' && props.emptyOptionLabel) return props.emptyOptionLabel
  if (v === '') {
    const z = flatSelectable.value.find((o) => norm(o.value) === '')
    if (z) return z.label
    return props.placeholder
  }
  const f = flatSelectable.value.find((o) => norm(o.value) === v)
  return f?.label ?? String(props.modelValue)
})

function syncHighlightToValue() {
  const idx = flatSelectable.value.findIndex((o) => norm(o.value) === norm(props.modelValue))
  highlightedFlatIndex.value = idx >= 0 ? idx : 0
}

watch(
  () => props.modelValue,
  () => {
    if (!open.value) syncHighlightToValue()
  },
)

watch(flatSelectable, () => syncHighlightToValue(), { immediate: true })

function positionPanel() {
  const el = triggerRef.value
  if (!el) return
  const r = el.getBoundingClientRect()
  const gap = 4
  panelStyle.value = {
    position: 'fixed',
    left: `${r.left}px`,
    top: `${r.bottom + gap}px`,
    width: `${r.width}px`,
    zIndex: '260',
    maxHeight: `min(320px, calc(100vh - ${r.bottom + gap + 16}px))`,
  }
}

function toggle() {
  if (props.disabled) return
  open.value = !open.value
  if (open.value) {
    syncHighlightToValue()
    nextTick(() => {
      positionPanel()
      panelRef.value?.focus({ preventScroll: true })
    })
  }
}

function close() {
  open.value = false
  triggerRef.value?.focus({ preventScroll: true })
}

function choose(v: string | number) {
  emit('update:modelValue', v)
  close()
}

function onDocPointerDown(e: MouseEvent) {
  if (!open.value) return
  const t = e.target as Node
  if (rootRef.value?.contains(t)) return
  if (panelRef.value?.contains(t)) return
  open.value = false
}

function onWinScrollResize() {
  if (open.value) positionPanel()
}

onMounted(() => {
  document.addEventListener('pointerdown', onDocPointerDown, true)
  window.addEventListener('scroll', onWinScrollResize, true)
  window.addEventListener('resize', onWinScrollResize)
})

onUnmounted(() => {
  document.removeEventListener('pointerdown', onDocPointerDown, true)
  window.removeEventListener('scroll', onWinScrollResize, true)
  window.removeEventListener('resize', onWinScrollResize)
})

const optionRowsOnly = computed(() =>
  panelRows.value.filter((r): r is Extract<PanelRow, { kind: 'option' }> => r.kind === 'option'),
)

function moveHighlight(delta: number) {
  const opts = optionRowsOnly.value.filter((o) => !o.disabled)
  if (!opts.length) return
  const current = highlightedFlatIndex.value
  const idxInEnabled = opts.findIndex((o) => o.flatIndex === current)
  const start = idxInEnabled >= 0 ? idxInEnabled : 0
  const next = (start + delta + opts.length) % opts.length
  highlightedFlatIndex.value = opts[next].flatIndex
}

function onTriggerKeydown(e: KeyboardEvent) {
  if (props.disabled) return
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    if (!open.value) {
      open.value = true
      nextTick(() => {
        positionPanel()
        if (e.key === 'ArrowDown') moveHighlight(1)
        else moveHighlight(-1)
        panelRef.value?.focus({ preventScroll: true })
      })
    } else {
      moveHighlight(e.key === 'ArrowDown' ? 1 : -1)
    }
  } else if (e.key === 'Enter' || e.key === ' ') {
    if (!open.value) {
      e.preventDefault()
      toggle()
    }
  } else if (e.key === 'Escape' && open.value) {
    e.preventDefault()
    close()
  }
}

function onPanelKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    close()
    return
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    moveHighlight(1)
    return
  }
  if (e.key === 'ArrowUp') {
    e.preventDefault()
    moveHighlight(-1)
    return
  }
  if (e.key === 'Home') {
    e.preventDefault()
    const first = optionRowsOnly.value.find((o) => !o.disabled)
    if (first) highlightedFlatIndex.value = first.flatIndex
    return
  }
  if (e.key === 'End') {
    e.preventDefault()
    const enabled = optionRowsOnly.value.filter((o) => !o.disabled)
    const last = enabled[enabled.length - 1]
    if (last) highlightedFlatIndex.value = last.flatIndex
    return
  }
  if (e.key === 'Enter') {
    e.preventDefault()
    const row = optionRowsOnly.value.find((o) => o.flatIndex === highlightedFlatIndex.value && !o.disabled)
    if (row) choose(row.value)
    return
  }
}

watch(open, (o) => {
  if (o) {
    nextTick(() => positionPanel())
  }
})
</script>

<style scoped>
.admin-select {
  position: relative;
  width: 100%;
}

.admin-select__trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  width: 100%;
  min-height: 42px;
  padding: 10px 12px 10px 14px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-control);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: 0.9375rem;
  font-family: inherit;
  text-align: left;
  cursor: pointer;
  transition:
    border-color 0.15s,
    box-shadow 0.15s,
    background 0.15s;
}

.admin-select__trigger:hover:not(:disabled) {
  background: var(--nebius-input-bg);
  border-color: #d1d5db;
}

.admin-select--open .admin-select__trigger:not(:disabled) {
  background: var(--nebius-input-bg);
  border-color: #d1d5db;
}

.admin-select__trigger:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px var(--color-primary-muted);
}

.admin-select__trigger:focus:not(:disabled) {
  background: var(--color-bg);
}

.admin-select--open .admin-select__trigger:focus:not(:disabled) {
  background: var(--nebius-input-bg);
}

.admin-select__trigger--placeholder .admin-select__value {
  color: var(--color-text-muted);
}

.admin-select__trigger:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.admin-select__value {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.admin-select__chevron {
  flex-shrink: 0;
  color: var(--color-text-secondary);
  transition: transform 0.15s ease;
}

.admin-select--open .admin-select__chevron {
  transform: rotate(180deg);
}

.admin-select__panel {
  overflow-y: auto;
  padding: 6px;
  margin: 0;
  list-style: none;
  background: var(--color-bg);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-control);
  box-shadow: var(--shadow-select-panel);
}

.admin-select__heading {
  padding: 8px 10px 4px;
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--color-table-header-text);
}

.admin-select__option {
  display: block;
  width: 100%;
  padding: 10px 12px;
  margin: 0;
  border: none;
  border-radius: calc(var(--radius-control) - 4px);
  background: transparent;
  color: var(--color-text);
  font-size: 0.875rem;
  font-family: inherit;
  text-align: left;
  cursor: pointer;
  transition: background 0.1s ease;
}

.admin-select__option:hover:not(:disabled),
.admin-select__option--active {
  background: var(--nebius-select-hover-bg);
}

.admin-select__option--selected {
  color: var(--color-primary);
  font-weight: 600;
}

.admin-select__option:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.admin-select__empty {
  padding: 12px;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}
</style>
