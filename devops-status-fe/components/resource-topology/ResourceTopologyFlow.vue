<script setup lang="ts">
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'

import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import type { Edge, Node } from '@vue-flow/core'
import { VueFlow } from '@vue-flow/core'
import { computed, markRaw, ref, watch } from 'vue'
import type { ResourceTopology } from '~/types/resource-topology'
import TopoInfraNode from './TopoInfraNode.vue'
import TopoRootNode from './TopoRootNode.vue'
import TopoTelemetryHubNode from './TopoTelemetryHubNode.vue'
import TopoTelemetryLeafNode from './TopoTelemetryLeafNode.vue'
import TopologyFitView from './TopologyFitView.vue'

const props = withDefaults(
  defineProps<{
    topology: ResourceTopology
    /** Increment to re-fit the viewport (e.g. each modal open). */
    renderKey?: number
  }>(),
  { renderKey: 0 },
)

const nodeTypes = {
  topoRoot: markRaw(TopoRootNode),
  topoInfra: markRaw(TopoInfraNode),
  topoHub: markRaw(TopoTelemetryHubNode),
  topoTel: markRaw(TopoTelemetryLeafNode),
}

function buildGraph(topo: ResourceTopology): { nodes: Node[]; edges: Edge[] } {
  const nodes: Node[] = []
  const edges: Edge[] = []
  const rootId = 'topo-root'
  const infraId = 'topo-infra'
  const hubId = 'topo-hub'
  const cx = 400

  nodes.push({
    id: rootId,
    type: 'topoRoot',
    position: { x: cx - 100, y: 8 },
    data: {
      kind: topo.kind,
      name: topo.resource.name,
      slug: topo.resource.slug,
    },
  })

  nodes.push({
    id: infraId,
    type: 'topoInfra',
    position: { x: cx - 300, y: 140 },
    data: {
      infra: topo.infra ?? null,
      resourceKind: topo.kind,
    },
  })
  edges.push({
    id: 'e-root-infra',
    source: rootId,
    target: infraId,
    sourceHandle: 'b',
    type: 'smoothstep',
  })

  nodes.push({
    id: hubId,
    type: 'topoHub',
    position: { x: cx + 20, y: 140 },
    data: { count: topo.telemetries.length },
  })
  edges.push({
    id: 'e-root-hub',
    source: rootId,
    target: hubId,
    sourceHandle: 'b',
    type: 'smoothstep',
  })

  const cardW = 220
  const gap = 20
  const n = topo.telemetries.length
  const rowY = 310
  const totalW = n > 0 ? n * cardW + (n - 1) * gap : 0
  const firstX = n > 0 ? cx - totalW / 2 : cx - cardW / 2

  topo.telemetries.forEach((row, i) => {
    const id = `topo-tel-${row.telemetry.id}`
    nodes.push({
      id,
      type: 'topoTel',
      position: { x: firstX + i * (cardW + gap), y: rowY },
      data: { row },
    })
    edges.push({
      id: `e-hub-${id}`,
      source: hubId,
      target: id,
      type: 'smoothstep',
    })
  })

  return { nodes, edges }
}

const nodes = ref<Node[]>([])
const edges = ref<Edge[]>([])

const fitTick = ref(0)

watch(
  () => [props.topology, props.renderKey] as const,
  () => {
    const g = buildGraph(props.topology)
    nodes.value = g.nodes
    edges.value = g.edges
    fitTick.value += 1
  },
  { immediate: true, deep: true },
)
</script>

<template>
  <div class="resource-topology-flow">
    <VueFlow
      :id="`topology-${renderKey}`"
      v-model:nodes="nodes"
      v-model:edges="edges"
      :node-types="nodeTypes"
      :nodes-draggable="false"
      :nodes-connectable="false"
      :elements-selectable="false"
      :zoom-on-scroll="true"
      :pan-on-scroll="false"
      :min-zoom="0.35"
      :max-zoom="1.5"
      fit-view-on-init
      class="topology-vue-flow"
    >
      <TopologyFitView :trigger="fitTick" />
      <Background :gap="20" pattern-color="#cbd5e1" />
      <Controls position="bottom-right" />
    </VueFlow>
  </div>
</template>

<style scoped>
.resource-topology-flow {
  width: 100%;
  height: min(62vh, 520px);
  min-height: 360px;
  border-radius: var(--radius, 10px);
  border: 1px solid var(--color-border-strong, #e2e8f0);
  overflow: hidden;
  background: var(--color-bg-muted, #f8fafc);
}

.topology-vue-flow {
  width: 100%;
  height: 100%;
}

:deep(.vue-flow__controls) {
  box-shadow: var(--shadow-card, 0 2px 8px rgba(15, 23, 42, 0.08));
}

:deep(.vue-flow__controls-button) {
  background: var(--color-bg, #fff);
  border-color: var(--color-border-strong, #e2e8f0);
  color: var(--color-text, #0f172a);
}

:deep(.vue-flow__edge-path) {
  stroke: #94a3b8;
  stroke-width: 2;
}
</style>
