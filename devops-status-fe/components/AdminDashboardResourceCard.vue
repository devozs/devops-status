<script setup lang="ts">
defineProps<{
  to: string
  title: string
  description: string
  loading: boolean
  error: string
  total: number
  lines: Array<{ main: string; meta?: string }>
  summary?: string
}>()
</script>

<template>
  <NuxtLink :to="to" class="dash-card">
    <span class="dash-card-title">{{ title }}</span>
    <span class="dash-card-desc">{{ description }}</span>

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
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 22px 24px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  transition: box-shadow 0.15s, border-color 0.15s;
  align-items: stretch;
  text-align: left;
}

.dash-card:hover {
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.06);
  border-color: #cbd5e1;
}

.dash-card-title {
  font-weight: 700;
  font-size: 1rem;
  color: #1a202c;
}

.dash-card-desc {
  font-size: 0.875rem;
  line-height: 1.45;
  color: #718096;
}

.dash-card-summary {
  font-size: 0.75rem;
  font-weight: 600;
  color: #64748b;
  text-transform: none;
  letter-spacing: 0.01em;
}

.dash-card-state {
  font-size: 0.8125rem;
  line-height: 1.4;
  margin-top: 4px;
}

.dash-card-state--muted {
  color: #94a3b8;
}

.dash-card-state--error {
  color: #b91c1c;
}

.dash-card-count {
  font-size: 0.75rem;
  color: #64748b;
  margin-top: 4px;
}

.dash-card-list {
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
  background: #f8fafc;
  border-radius: 6px;
  border: 1px solid #f1f5f9;
}

.dash-card-li-main {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #1e293b;
  line-height: 1.35;
  word-break: break-word;
}

.dash-card-li-meta {
  font-size: 0.72rem;
  color: #64748b;
  line-height: 1.35;
  word-break: break-word;
}
</style>
