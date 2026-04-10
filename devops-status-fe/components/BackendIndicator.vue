<script setup lang="ts">
const { backendStatus, startPolling } = useBackendHealth()
onMounted(() => startPolling())
</script>

<template>
  <span
    class="backend-indicator"
    :class="'indicator--' + backendStatus"
    :title="backendStatus === 'connected' ? 'Backend connected' : backendStatus === 'disconnected' ? 'Backend unreachable' : 'Checking...'"
  />
</template>

<style scoped>
.backend-indicator {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.indicator--connected {
  background: #22c55e;
  box-shadow: 0 0 4px #22c55e80;
}

.indicator--disconnected {
  background: #ef4444;
  box-shadow: 0 0 4px #ef444480;
}

.indicator--checking {
  background: #a3a3a3;
}
</style>
