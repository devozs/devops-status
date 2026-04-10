<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import type { NodeProps } from '@vue-flow/core'
import { Link2 } from 'lucide-vue-next'

defineProps<NodeProps<{ count: number }>>()
</script>

<template>
  <div class="topo-card topo-card--hub">
    <Handle type="target" :position="Position.Top" class="topo-handle" />
    <div class="topo-card__row">
      <div class="topo-card__icon topo-card__icon--hub">
        <Link2 :size="16" :stroke-width="2" aria-hidden="true" />
      </div>
      <div class="topo-card__body">
        <span class="topo-card__eyebrow">Telemetries</span>
        <span class="topo-card__title">{{ data?.count ?? 0 }} linked</span>
      </div>
    </div>
    <Handle type="source" :position="Position.Bottom" class="topo-handle" />
  </div>
</template>

<style scoped>
.topo-card {
  min-width: 160px;
  padding: 10px 12px;
  border-radius: var(--radius, 10px);
  border: 1px solid var(--color-border-strong, #e2e8f0);
  background: var(--color-bg, #fff);
  box-shadow: var(--shadow-card, 0 2px 8px rgba(15, 23, 42, 0.06));
}
.topo-card--hub {
  border-top-width: 2px;
  border-top-color: #a3e635;
}
.topo-card__row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.topo-card__icon--hub {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(163, 230, 53, 0.2);
  color: #4d7c0f;
}
.topo-card__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.topo-card__eyebrow {
  font-size: 0.6rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-secondary, #64748b);
}
.topo-card__title {
  font-weight: 600;
  font-size: 0.82rem;
  color: var(--color-text, #0f172a);
}
.topo-handle {
  width: 8px;
  height: 8px;
  background: #94a3b8;
  border: 2px solid var(--color-bg, #fff);
}
</style>
