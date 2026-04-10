<script setup lang="ts">
import { AlertTriangle, Info } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    variant?: 'info' | 'warning'
  }>(),
  { variant: 'info' },
)

const Icon = computed(() => (props.variant === 'warning' ? AlertTriangle : Info))
</script>

<template>
  <div class="admin-callout" :class="'admin-callout--' + variant" role="note">
    <Icon class="admin-callout__icon" :size="20" :stroke-width="2" aria-hidden="true" />
    <div class="admin-callout__content">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.admin-callout {
  display: flex;
  gap: 12px;
  padding: 14px 16px;
  border-radius: var(--radius);
  border: 1px solid transparent;
  margin-bottom: 20px;
  max-width: 960px;
  line-height: 1.5;
  font-size: 0.875rem;
}

.admin-callout--info {
  background: var(--callout-info-bg);
  border-color: var(--callout-info-border);
  color: var(--callout-info-text);
}

.admin-callout--warning {
  background: var(--callout-warning-bg);
  border-color: var(--callout-warning-border);
  color: var(--callout-warning-text);
}

.admin-callout__icon {
  flex-shrink: 0;
  margin-top: 1px;
}

.admin-callout__content {
  min-width: 0;
}

.admin-callout__content :deep(p) {
  margin: 0 0 8px;
}

.admin-callout__content :deep(p:last-child) {
  margin-bottom: 0;
}

.admin-callout__content :deep(code) {
  font-size: 0.8em;
  padding: 1px 5px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.06);
}

.admin-callout--warning .admin-callout__content :deep(code) {
  background: rgba(146, 64, 14, 0.12);
}
</style>
