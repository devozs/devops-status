<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import type { NodeProps } from '@vue-flow/core'
import { Activity } from 'lucide-vue-next'
import type { ResourceTopologyTelemetryRow } from '~/types/resource-topology'

defineProps<NodeProps<{ row: ResourceTopologyTelemetryRow }>>()

function label(row: ResourceTopologyTelemetryRow) {
  const d = row.telemetry.display_name?.trim()
  if (d) return d
  return row.telemetry.name
}
</script>

<template>
  <div class="topo-card topo-card--leaf">
    <Handle type="target" :position="Position.Top" class="topo-handle" />
    <div class="topo-card__row">
      <div class="topo-card__icon topo-card__icon--tel">
        <Activity :size="16" :stroke-width="2" aria-hidden="true" />
      </div>
      <div class="topo-card__body">
        <span class="topo-card__title">{{ data?.row ? label(data.row) : '' }}</span>
        <span class="topo-card__meta">Adapter: {{ data?.row?.telemetry.adapter }}</span>
        <span class="topo-card__detail">
          Interval {{ data?.row?.link.sample_interval_sec }}s · window {{ data?.row?.link.window_size }}
        </span>
        <span class="topo-card__detail">
          Fail→down {{ data?.row?.link.consecutive_failures_to_down }} · recover {{ data?.row?.link.consecutive_success_to_recover }}
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.topo-card {
  width: 212px;
  padding: 10px 12px;
  border-radius: var(--radius, 10px);
  border: 1px solid var(--color-border-strong, #e2e8f0);
  background: var(--color-bg, #fff);
  box-shadow: var(--shadow-card, 0 2px 8px rgba(15, 23, 42, 0.06));
}
.topo-card--leaf {
  border-left: 3px solid #84cc16;
}
.topo-card__row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.topo-card__icon--tel {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(132, 204, 22, 0.18);
  color: #3f6212;
}
.topo-card__body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.topo-card__title {
  font-weight: 600;
  font-size: 0.8rem;
  line-height: 1.25;
  color: var(--color-text, #0f172a);
}
.topo-card__meta {
  font-size: 0.72rem;
  font-weight: 500;
  color: var(--color-text-secondary, #64748b);
}
.topo-card__detail {
  font-size: 0.68rem;
  color: var(--color-text-secondary, #64748b);
  line-height: 1.35;
}
.topo-handle {
  width: 8px;
  height: 8px;
  background: #94a3b8;
  border: 2px solid var(--color-bg, #fff);
}
</style>
