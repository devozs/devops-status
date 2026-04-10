<template>
  <div class="container history-page">
    <h1 class="page-title">History</h1>

    <div class="tabs">
      <button class="tab" :class="{ 'tab--active': activeTab === 'incidents' }" @click="activeTab = 'incidents'">Incidents</button>
      <button class="tab" :class="{ 'tab--active': activeTab === 'uptime' }" @click="activeTab = 'uptime'">Uptime</button>
    </div>

    <div class="filter-row">
      <select v-model="targetFilter" class="filter-select">
        <option value="">All</option>
        <option value="service">Services</option>
        <option value="environment">Environments</option>
      </select>
      <select v-if="activeTab === 'uptime'" v-model="selectedSlug" class="filter-select">
        <option value="">Select component...</option>
        <optgroup label="Services">
          <option v-for="s in services" :key="s.slug" :value="'service:' + s.slug">{{ s.name }}</option>
        </optgroup>
        <optgroup label="Environments">
          <option v-for="e in environments" :key="e.slug" :value="'environment:' + e.slug">{{ e.name }}</option>
        </optgroup>
      </select>
    </div>

    <!-- Incidents tab -->
    <div v-if="activeTab === 'incidents'" class="incidents-panel">
      <div v-if="filteredIncidents?.length" class="incident-list">
        <div v-for="inc in filteredIncidents" :key="inc.id" class="incident-item">
          <div class="incident-header">
            <span class="incident-title" :class="'severity--' + inc.severity">{{ inc.title }}</span>
            <span class="incident-meta">{{ inc.target_type }} &middot; {{ formatDate(inc.started_at) }}</span>
          </div>
          <span class="incident-status">{{ inc.status }}</span>
        </div>
      </div>
      <p v-else class="placeholder-text">No incidents recorded yet.</p>
    </div>

    <!-- Uptime tab -->
    <div v-else class="uptime-panel">
      <div v-if="!selectedSlug" class="placeholder-text">Select a service or environment above to view uptime.</div>
      <div v-else-if="uptimeData">
        <div class="uptime-header">
          <h2 class="uptime-component-name">{{ uptimeData.name }}</h2>
        </div>
        <div class="months-grid">
          <div v-for="month in uptimeMonths" :key="month.label" class="month-block">
            <div class="month-label">
              <span>{{ month.label }}</span>
              <span class="month-pct">{{ month.pct }}%</span>
            </div>
            <div class="day-grid">
              <div
                v-for="(day, i) in month.days"
                :key="i"
                class="day-cell"
                :class="'day--' + day.level"
                :title="day.date + ': ' + day.pct + '% uptime'"
              />
            </div>
          </div>
        </div>
      </div>
      <p v-else class="placeholder-text">Loading uptime data...</p>
    </div>
  </div>
</template>

<script setup lang="ts">
const { apiFetch } = useApi()

const activeTab = ref<'incidents' | 'uptime'>('incidents')
const targetFilter = ref('')
const selectedSlug = ref('')

const { data: incidents } = await useAsyncData('all-incidents', () =>
  apiFetch<any[]>('/api/public/incidents?limit=50'),
  { default: () => [] },
)

const { data: services } = await useAsyncData('pub-services', () =>
  apiFetch<any[]>('/api/public/services'),
  { default: () => [] },
)

const { data: environments } = await useAsyncData('pub-environments', () =>
  apiFetch<any[]>('/api/public/environments'),
  { default: () => [] },
)

const filteredIncidents = computed(() => {
  if (!targetFilter.value) return incidents.value
  return incidents.value?.filter((i: any) => i.target_type === targetFilter.value) || []
})

const uptimeData = ref<any>(null)

watch(selectedSlug, async (val) => {
  if (!val) { uptimeData.value = null; return }
  const [type, slug] = val.split(':')
  try {
    const data = await apiFetch<any>(`/api/public/${type}s/${slug}/history`)
    uptimeData.value = data
  } catch {
    uptimeData.value = null
  }
})

interface DayCell { date: string; pct: number; level: string }
interface MonthBlock { label: string; pct: string; days: DayCell[] }

const uptimeMonths = computed((): MonthBlock[] => {
  if (!uptimeData.value?.rollups) return []
  const rollups: any[] = uptimeData.value.rollups || []
  const byMonth = new Map<string, DayCell[]>()

  for (const r of rollups) {
    const d = new Date(r.bucket_start)
    const monthKey = d.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })
    const pct = r.availability_pct ?? 100
    const level = pct >= 99 ? 'green' : pct >= 95 ? 'yellow' : pct >= 0.01 ? 'red' : 'empty'
    if (!byMonth.has(monthKey)) byMonth.set(monthKey, [])
    byMonth.get(monthKey)!.push({
      date: d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
      pct, level,
    })
  }

  const months: MonthBlock[] = []
  for (const [label, days] of byMonth) {
    const avg = days.length > 0 ? days.reduce((s, d) => s + d.pct, 0) / days.length : 100
    months.push({ label, pct: avg.toFixed(2), days })
  }
  return months
})

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}
</script>

<style scoped>
.history-page { padding-top: 16px; }
.page-title { font-size: 1.5rem; font-weight: 700; margin-bottom: 24px; letter-spacing: -0.02em; color: var(--color-text); }
.tabs { display: flex; gap: 0; border-bottom: 1px solid var(--color-border-strong); margin-bottom: 16px; }
.tab { padding: 10px 20px; font-size: 0.875rem; font-weight: 500; background: none; border: none; border-bottom: 2px solid transparent; margin-bottom: -1px; color: var(--color-text-secondary); transition: color 0.15s, border-color 0.15s; }
.tab:hover { color: var(--color-text); }
.tab--active { color: var(--color-primary); border-bottom-color: var(--color-primary); font-weight: 600; }
.filter-row { display: flex; gap: 12px; margin-bottom: 20px; flex-wrap: wrap; }
.filter-select { padding: 8px 12px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); font-size: 0.875rem; font-family: inherit; background: var(--color-bg); color: var(--color-text); }
.filter-select:focus { outline: none; border-color: var(--color-primary); box-shadow: 0 0 0 3px var(--color-primary-muted); }
.incident-list { display: flex; flex-direction: column; gap: 12px; }
.incident-item { border-left: 3px solid var(--color-orange); padding: 12px 16px; background: var(--color-bg); border: 1px solid var(--color-border-strong); border-radius: 0 var(--radius) var(--radius) 0; box-shadow: var(--shadow-card); }
.incident-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
.incident-title { font-weight: 600; font-size: 0.9rem; }
.severity--major { color: var(--color-red); }
.severity--minor { color: var(--color-orange); }
.incident-meta { font-size: 0.8rem; color: var(--color-text-secondary); }
.incident-status { font-size: 0.8rem; color: var(--color-text-secondary); text-transform: capitalize; }
.placeholder-text { color: var(--color-text-secondary); font-size: 0.9rem; padding: 24px 0; }

.uptime-header { margin-bottom: 20px; }
.uptime-component-name { font-size: 1.15rem; font-weight: 600; }
.months-grid { display: flex; gap: 24px; flex-wrap: wrap; }
.month-block { min-width: 200px; }
.month-label { display: flex; justify-content: space-between; margin-bottom: 8px; font-size: 0.9rem; font-weight: 500; }
.month-pct { color: var(--color-text-secondary); font-size: 0.85rem; }
.day-grid { display: grid; grid-template-columns: repeat(7, 24px); gap: 3px; }
.day-cell { width: 24px; height: 24px; border-radius: 3px; }
.day--green { background: var(--color-green); }
.day--yellow { background: var(--color-yellow); }
.day--red { background: var(--color-red); }
.day--empty { background: #e5e7eb; }
</style>
