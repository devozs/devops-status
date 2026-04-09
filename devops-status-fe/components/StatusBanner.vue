<template>
  <div class="status-banner" :class="bannerClass">
    <span class="banner-text">{{ message }}</span>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  status: string
  message: string
}>()

const bannerClass = computed(() => ({
  'banner--ok': props.status === 'operational',
  'banner--disruption': props.status !== 'operational',
}))
</script>

<style scoped>
.status-banner {
  padding: 16px 24px;
  border-radius: var(--radius);
  margin-bottom: 32px;
}

.banner--ok {
  background: var(--color-banner-ok);
  color: #ffffff;
}

.banner--disruption {
  background: var(--color-banner-disruption);
  color: #ffffff;
}

.banner-text {
  font-size: 1rem;
  font-weight: 600;
}
</style>
