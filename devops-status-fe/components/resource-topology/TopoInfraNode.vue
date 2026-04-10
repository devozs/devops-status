<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import type { NodeProps } from '@vue-flow/core'
import { Container, Network } from 'lucide-vue-next'
import { computed } from 'vue'
import type { ResourceTopologyInfra } from '~/types/resource-topology'

type Data = {
  infra: ResourceTopologyInfra | null | undefined
  resourceKind: 'service' | 'environment'
}

const props = defineProps<NodeProps<Data>>()

const Icon = computed(() => {
  if (props.data?.infra?.type === 'k8s_cluster') return Container
  return Network
})

const title = computed(() => {
  const infra = props.data?.infra
  if (!infra) {
    return props.data?.resourceKind === 'environment' ? 'Kubernetes cluster' : 'Service provider'
  }
  if (infra.type === 'k8s_cluster') return 'Kubernetes cluster'
  return 'Service provider'
})

const muted = computed(() => !props.data?.infra)
</script>

<template>
  <div class="topo-card" :class="{ 'topo-card--muted': muted }">
    <Handle type="target" :position="Position.Top" class="topo-handle" />
    <div class="topo-card__row">
      <div class="topo-card__icon topo-card__icon--infra">
        <component :is="Icon" :size="16" :stroke-width="2" aria-hidden="true" />
      </div>
      <div class="topo-card__body">
        <span class="topo-card__eyebrow">{{ title }}</span>
        <span class="topo-card__title">{{ data?.infra?.name || '—' }}</span>
        <span v-if="data?.infra && data.infra.type === 'service_provider' && data.infra.provider_type" class="topo-card__meta">
          {{ data.infra.provider_type }}
        </span>
        <span v-else-if="muted" class="topo-card__meta">Link in admin to show infra</span>
      </div>
    </div>
    <Handle type="source" :position="Position.Bottom" class="topo-handle topo-handle--hidden" />
  </div>
</template>

<style scoped>
.topo-card {
  min-width: 188px;
  max-width: 240px;
  padding: 10px 12px;
  border-radius: var(--radius, 10px);
  border: 1px solid var(--color-border-strong, #e2e8f0);
  background: var(--color-bg, #fff);
  box-shadow: var(--shadow-card, 0 2px 8px rgba(15, 23, 42, 0.06));
}
.topo-card--muted {
  border-style: dashed;
  opacity: 0.92;
}
.topo-card__row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.topo-card__icon--infra {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(34, 211, 238, 0.15);
  color: #0e7490;
}
.topo-card__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
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
  line-height: 1.3;
  color: var(--color-text, #0f172a);
}
.topo-card__meta {
  font-size: 0.72rem;
  color: var(--color-text-secondary, #64748b);
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
