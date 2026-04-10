<script setup lang="ts">
import type { ResourceTopology } from '~/types/resource-topology'

withDefaults(
  defineProps<{
    open: boolean
    title: string
    loading: boolean
    error: string
    topology: ResourceTopology | null
    /** Bumps Vue Flow remount / fit when reopening the same resource. */
    sessionKey?: number
  }>(),
  { sessionKey: 0 },
)

const emit = defineEmits<{
  close: []
}>()

const titleId = 'resource-topology-modal-title'
</script>

<template>
  <div v-if="open" class="modal-overlay" @click.self="emit('close')">
    <div class="modal-card modal-card--wide topo-modal-card" role="dialog" :aria-labelledby="titleId">
      <h2 :id="titleId" class="modal-title">Resource map — {{ title }}</h2>
      <p class="modal-subtitle muted">Service or environment, linked infra, and probe telemetries.</p>

      <div v-if="loading" class="topo-modal__state">Loading map…</div>
      <div v-else-if="error" class="topo-modal__state topo-modal__state--err">{{ error }}</div>
      <ClientOnly v-else-if="topology">
        <ResourceTopologyFlow :topology="topology" :render-key="sessionKey" />
        <template #fallback>
          <div class="topo-modal__state">Loading diagram…</div>
        </template>
      </ClientOnly>

      <div class="modal-actions">
        <button type="button" class="btn" @click="emit('close')">Close</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.topo-modal-card {
  max-width: 960px;
}
.topo-modal__state {
  padding: 32px;
  text-align: center;
  color: var(--color-text-secondary);
  font-size: 0.9rem;
}
.topo-modal__state--err {
  color: var(--color-red, #dc2626);
}
</style>
