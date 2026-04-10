<template>
  <div class="service-card" :class="{ 'service-card--nested': variant === 'nested' }">
    <div class="card-header">
      <span class="card-name">{{ item.name }}</span>
      <span class="card-status" :class="'status--' + item.status">{{ statusLabel }}</span>
    </div>
    <div class="card-bar">
      <div
        v-for="(day, i) in timelineDays"
        :key="i"
        class="bar-segment"
        :class="`bar--${day.level}`"
        :title="day.date + ': ' + day.pct + '% uptime'"
      />
    </div>
    <div class="card-footer">
      <span class="card-range">90 days ago</span>
      <span class="card-uptime">{{ uptimeLabel }}</span>
      <span class="card-range">Today</span>
    </div>
  </div>
</template>

<script setup lang="ts">
interface DayData {
  date: string
  availability_pct: number
  qos_level: string
}

interface StatusItem {
  name: string
  slug: string
  description?: string
  status?: string
  uptime_pct?: number
  days?: DayData[]
}

const props = defineProps<{
  item: StatusItem
  /** Smaller padding for nested / per-telemetry rows */
  variant?: 'default' | 'nested'
}>()

const statusLabel = computed(() => {
  if (props.item.status === 'disruption') return 'Disruption'
  return 'Operational'
})

const uptimeLabel = computed(() => {
  const pct = props.item.uptime_pct ?? 100
  return `${pct.toFixed(2)} % uptime`
})

const timelineDays = computed(() => {
  const days = props.item.days || []
  const mapped = days.map(d => ({
    date: d.date,
    pct: d.availability_pct,
    level: d.qos_level || 'green',
  }))
  const padCount = 90 - mapped.length
  const padding = Array.from({ length: Math.max(0, padCount) }, () => ({
    date: '',
    pct: 100,
    level: 'empty',
  }))
  return [...padding, ...mapped]
})
</script>

<style scoped>
.service-card {
  background: var(--color-bg);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius);
  padding: 16px 20px;
  box-shadow: var(--shadow-card);
}
.service-card--nested {
  padding: 12px 14px;
  box-shadow: none;
  border-color: var(--color-border);
}
.service-card--nested .card-bar { height: 22px; }
.service-card--nested .card-name { font-size: 0.88rem; }
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.card-name { font-weight: 600; font-size: 0.95rem; }
.card-status { font-size: 0.8rem; font-weight: 500; }
.status--operational { color: var(--color-green); }
.status--disruption { color: var(--color-red); }
.card-bar { display: flex; gap: 1px; height: 28px; margin-bottom: 4px; }
.bar-segment { flex: 1; border-radius: 2px; min-width: 2px; }
.bar--green { background: var(--color-green); }
.bar--yellow { background: var(--color-yellow); }
.bar--red { background: var(--color-red); }
.bar--empty { background: #e8eaef; }
.card-footer { display: flex; justify-content: space-between; font-size: 0.75rem; color: var(--color-text-secondary); }
.card-uptime { font-weight: 500; }
</style>
