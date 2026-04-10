<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import type { NodeProps } from '@vue-flow/core'
import { Boxes, Cpu } from 'lucide-vue-next'
import { computed } from 'vue'

const props = defineProps<NodeProps>()

const Icon = computed(() => (props.data?.kind === 'environment' ? Boxes : Cpu))
const label = computed(() => (props.data?.kind === 'environment' ? 'Environment' : 'Service'))
</script>

<template>
  <div class="topo-card topo-card--root">
    <div class="topo-card__row">
      <div class="topo-card__icon">
        <component :is="Icon" :size="18" :stroke-width="2" aria-hidden="true" />
      </div>
      <div class="topo-card__body">
        <span class="topo-card__eyebrow">{{ label }}</span>
        <span class="topo-card__title">{{ data?.name }}</span>
        <span class="topo-card__meta">{{ data?.slug }}</span>
      </div>
    </div>
    <Handle id="b" type="source" :position="Position.Bottom" class="topo-handle" />
  </div>
</template>

<style scoped>
.topo-card {
  min-width: 200px;
  max-width: 260px;
  padding: 12px 14px;
  border-radius: var(--radius, 10px);
  border: 1px solid var(--color-border-strong, #e2e8f0);
  background: var(--color-bg, #fff);
  box-shadow: var(--shadow-card, 0 2px 8px rgba(15, 23, 42, 0.06));
}
.topo-card--root {
  border-top-width: 3px;
  border-top-color: #22d3ee;
}
.topo-card__row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}
.topo-card__icon {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #22d3ee, #0ea5e9);
  color: #fff;
}
.topo-card__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.topo-card__eyebrow {
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-secondary, #64748b);
}
.topo-card__title {
  font-weight: 600;
  font-size: 0.9rem;
  line-height: 1.3;
  color: var(--color-text, #0f172a);
}
.topo-card__meta {
  font-size: 0.75rem;
  color: var(--color-text-secondary, #64748b);
  word-break: break-all;
}
.topo-handle {
  width: 8px;
  height: 8px;
  background: #94a3b8;
  border: 2px solid var(--color-bg, #fff);
}
.topo-handle--hidden {
  opacity: 0;
  pointer-events: none;
}
</style>
