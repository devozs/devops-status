<script setup lang="ts">
import { useVueFlow } from '@vue-flow/core'
import { nextTick, watch } from 'vue'

const props = defineProps<{ trigger: number }>()

const { fitView } = useVueFlow()

watch(
  () => props.trigger,
  () => {
    nextTick(() => {
      fitView({ padding: 0.2, duration: 220 })
    })
  },
  { immediate: true },
)
</script>

<template>
  <span class="topo-fit-sentinel" aria-hidden="true" />
</template>

<style scoped>
.topo-fit-sentinel {
  display: none;
}
</style>
