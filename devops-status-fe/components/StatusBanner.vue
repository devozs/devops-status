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
  box-shadow: var(--shadow-card);
}

.banner--ok {
  background: linear-gradient(135deg, #15803d 0%, #166534 100%);
  color: #ffffff;
}

.banner--disruption {
  background: linear-gradient(135deg, #c2410c 0%, #9a3412 100%);
  color: #ffffff;
}

.banner-text {
  font-size: 1rem;
  font-weight: 600;
  letter-spacing: -0.01em;
}
</style>
