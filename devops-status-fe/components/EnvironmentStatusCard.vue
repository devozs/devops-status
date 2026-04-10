<template>
  <div class="environment-status-card">
    <ServiceCard :item="item" />
    <div class="env-card__toolbar">
      <button
        type="button"
        class="env-card__toggle"
        :aria-expanded="expanded"
        :aria-controls="breakdownId"
        @click="toggle"
      >
        <span>{{ expanded ? 'Hide' : 'Show' }} per-telemetry breakdown</span>
        <span class="env-card__chev" :class="{ 'env-card__chev--open': expanded }" aria-hidden="true">▾</span>
      </button>
    </div>
    <div
      v-show="expanded"
      :id="breakdownId"
      class="env-card__breakdown"
      role="region"
      :aria-label="`Telemetry breakdown for ${item.name}`"
    >
      <p v-if="breakdownError" class="env-card__err">{{ breakdownError }}</p>
      <p v-else-if="breakdownLoading" class="env-card__muted">Loading…</p>
      <template v-else-if="breakdown">
        <div
          v-for="row in breakdown.telemetries"
          :key="row.telemetry_id"
          class="env-card__sub"
        >
          <template v-if="!row.has_per_telemetry_samples">
            <div class="env-card__sub-head">{{ telemetryLabel(row) }}</div>
            <p class="env-card__muted">No per-probe history yet for this telemetry.</p>
          </template>
          <ServiceCard v-else :item="telemetryRowToItem(row)" variant="nested" />
        </div>
      </template>
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

interface BreakdownTelemetry {
  telemetry_id: string
  name: string
  display_name?: string
  status: string
  uptime_pct: number
  days: DayData[]
  has_per_telemetry_samples: boolean
}

interface BreakdownResponse {
  telemetries: BreakdownTelemetry[]
}

const props = withDefaults(
  defineProps<{
    item: StatusItem
    slug: string
    /** Public breakdown API path: environments vs services */
    resource?: 'environment' | 'service'
  }>(),
  { resource: 'environment' },
)

const { apiFetch } = useApi()

const expanded = ref(false)
const breakdownLoading = ref(false)
const breakdown = ref<BreakdownResponse | null>(null)
const breakdownError = ref('')

const breakdownId = computed(() => {
  const p = props.resource === 'service' ? 'svc-breakdown' : 'env-breakdown'
  return `${p}-${props.slug.replace(/[^a-zA-Z0-9_-]/g, '-')}`
})

function telemetryLabel(row: BreakdownTelemetry): string {
  const d = row.display_name?.trim()
  if (d) return d
  return row.name
}

function telemetryRowToItem(row: BreakdownTelemetry): StatusItem {
  return {
    name: telemetryLabel(row),
    slug: row.telemetry_id,
    status: row.status,
    uptime_pct: row.uptime_pct,
    days: row.days,
  }
}

async function loadBreakdown() {
  breakdownLoading.value = true
  breakdownError.value = ''
  try {
    const base =
      props.resource === 'service'
        ? `/api/public/services/${encodeURIComponent(props.slug)}/telemetry-breakdown`
        : `/api/public/environments/${encodeURIComponent(props.slug)}/telemetry-breakdown`
    breakdown.value = await apiFetch<BreakdownResponse>(base)
  } catch {
    breakdownError.value = 'Could not load per-telemetry breakdown.'
    breakdown.value = null
  } finally {
    breakdownLoading.value = false
  }
}

async function toggle() {
  const next = !expanded.value
  expanded.value = next
  if (next) await loadBreakdown()
}
</script>

<style scoped>
.environment-status-card {
  display: flex;
  flex-direction: column;
  gap: 0;
}
.env-card__toolbar {
  margin-top: -4px;
  padding: 0 4px 8px;
}
.env-card__toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  margin: 0;
  border: none;
  background: transparent;
  color: var(--color-primary);
  font-size: 0.82rem;
  font-weight: 500;
  cursor: pointer;
  text-decoration: underline;
  text-underline-offset: 2px;
  border-radius: var(--radius, 8px);
}
.env-card__toggle:hover {
  color: var(--color-primary-hover);
}
.env-card__toggle:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
.env-card__chev {
  display: inline-block;
  transition: transform 0.15s ease;
  text-decoration: none;
  font-size: 0.7rem;
}
.env-card__chev--open {
  transform: rotate(180deg);
}
.env-card__breakdown {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 8px 0 4px;
  border-top: 1px solid var(--color-border);
  margin-top: 4px;
}
.env-card__sub {
  padding-left: 8px;
  border-left: 3px solid var(--color-border-strong);
}
.env-card__sub-head {
  font-weight: 600;
  font-size: 0.88rem;
  margin-bottom: 6px;
}
.env-card__muted {
  font-size: 0.82rem;
  color: var(--color-text-secondary);
  margin: 0;
}
.env-card__err {
  font-size: 0.85rem;
  color: var(--color-red);
  margin: 0;
}
</style>
