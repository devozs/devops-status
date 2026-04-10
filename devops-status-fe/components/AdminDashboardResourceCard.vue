<script setup lang="ts">
import type { Component } from 'vue'

withDefaults(
  defineProps<{
    to: string
    title: string
    description: string
    loading: boolean
    error: string
    total: number
    lines: Array<{ main: string; meta?: string }>
    summary?: string
    /** Lucide icon component (e.g. import { Server } from 'lucide-vue-next'). */
    icon?: Component
  }>(),
  { icon: undefined },
)
</script>

<template>
  <NuxtLink :to="to" class="dash-card">
    <div class="dash-card-header-row">
      <div v-if="icon" class="dash-card-icon-box" aria-hidden="true">
        <component :is="icon" class="dash-card-icon-glyph" :size="22" :stroke-width="2" />
      </div>
      <div class="dash-card-header-text">
        <span class="dash-card-title">{{ title }}</span>
        <span class="dash-card-desc">{{ description }}</span>
      </div>
    </div>

    <div v-if="summary" class="dash-card-summary">{{ summary }}</div>

    <div v-if="loading" class="dash-card-state dash-card-state--muted">
      Loading…
    </div>
    <div v-else-if="error" class="dash-card-state dash-card-state--error">
      {{ error }}
    </div>
    <template v-else>
      <p v-if="total === 0" class="dash-card-state dash-card-state--muted">
        None yet
      </p>
      <template v-else>
        <p v-if="total > lines.length" class="dash-card-count">
          {{ total }} total · first {{ lines.length }} shown
        </p>
        <ul class="dash-card-list" aria-label="Preview">
          <li v-for="(line, i) in lines" :key="i" class="dash-card-li">
            <span class="dash-card-li-main">{{ line.main }}</span>
            <span v-if="line.meta" class="dash-card-li-meta">{{ line.meta }}</span>
          </li>
        </ul>
      </template>
    </template>
  </NuxtLink>
</template>

<style scoped>
.dash-card {
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 22px 24px;
  background: var(--color-bg);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius);
  transition: box-shadow 0.15s, border-color 0.15s;
  align-items: stretch;
  text-align: left;
  box-shadow: var(--shadow-card);
}

.dash-card-header-row {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  margin-bottom: 2px;
}

.dash-card-icon-box {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--gradient-resource-tile);
  box-shadow: 0 2px 10px rgba(34, 211, 238, 0.28);
}

.dash-card-icon-glyph {
  display: block;
  color: #ffffff;
}

.dash-card-header-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.dash-card:hover {
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.08);
  border-color: var(--color-border);
}

.dash-card-title {
  font-weight: 700;
  font-size: 1rem;
  color: var(--color-text);
  letter-spacing: -0.01em;
}

.dash-card-desc {
  font-size: 0.875rem;
  line-height: 1.45;
  color: var(--color-text-secondary);
}

.dash-card-summary {
  position: relative;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: none;
  letter-spacing: 0.01em;
}

.dash-card-state {
  position: relative;
  font-size: 0.8125rem;
  line-height: 1.4;
  margin-top: 4px;
}

.dash-card-state--muted {
  color: var(--color-text-muted);
}

.dash-card-state--error {
  color: var(--color-red);
}

.dash-card-count {
  position: relative;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin-top: 4px;
}

.dash-card-list {
  position: relative;
  list-style: none;
  margin: 6px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.dash-card-li {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 10px;
  background: var(--color-bg-secondary);
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
}

.dash-card-li-main {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--color-text);
  line-height: 1.35;
  word-break: break-word;
}

.dash-card-li-meta {
  font-size: 0.72rem;
  color: var(--color-text-secondary);
  line-height: 1.35;
  word-break: break-word;
}
</style>
