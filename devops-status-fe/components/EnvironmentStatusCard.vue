<template>
  <div class="environment-status-card">
    <ServiceCard
      :item="item"
      :recent-incidents="item.recent_incidents ?? []"
      :history-target-type="resource"
      :history-parent-slug="slug"
    >
      <template #title-actions>
        <button
          type="button"
          class="btn btn-sm env-card__history-btn"
          @click="openResourceProbeHistory"
        >
          History
        </button>
        <button
          type="button"
          class="env-card__head-action env-card__head-action--telemetry"
          :aria-expanded="expanded"
          :aria-controls="breakdownId"
          :aria-label="telemetryAriaLabel"
          @click="toggle"
        >
          <span class="env-card__telemetry-count">{{ telemetryCountLabel }}</span>
          <span class="env-card__chev" :class="{ 'env-card__chev--open': expanded }" aria-hidden="true">▾</span>
        </button>
        <button
          type="button"
          class="env-card__head-action env-card__head-action--map"
          aria-label="Resource map"
          title="Resource map"
          @click="openResourceMap"
        >
          <Map :size="16" :stroke-width="2" aria-hidden="true" />
        </button>
      </template>
    </ServiceCard>

    <ResourceTopologyModal
      :open="topologyOpen"
      :title="topologyTitle"
      :loading="topologyLoading"
      :error="topologyError"
      :topology="topologyData"
      :session-key="topologySessionKey"
      @close="closeResourceMap"
    />
    <ProbeHistoryModal
      :open="probeHistoryOpen"
      :resource-kind="resource === 'service' ? 'service' : 'environment'"
      :slug="slug"
      :display-name="item.name"
      :initial-telemetry-id="probeHistoryInitialTelemetryId"
      @close="closeProbeHistory"
    />
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
            <div class="env-card__sub-head-row">
              <div class="env-card__sub-head">{{ telemetryLabel(row) }}</div>
              <button type="button" class="btn btn-sm" @click="openTelemetryProbeHistory(row.telemetry_id)">
                History
              </button>
            </div>
            <p class="env-card__muted">No per-probe history yet for this telemetry.</p>
          </template>
          <ServiceCard
            v-else
            :item="telemetryRowToItem(row)"
            variant="nested"
            :recent-incidents="item.recent_incidents ?? []"
            :telemetry-id="row.telemetry_id"
            :history-target-type="resource"
            :history-parent-slug="slug"
            show-probe-history
            @probe-history="openTelemetryProbeHistory(row.telemetry_id)"
          />
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Map } from 'lucide-vue-next'
import ProbeHistoryModal from '~/components/ProbeHistoryModal.vue'
import type { ResourceTopology } from '~/types/resource-topology'
import type { StatusSummaryDay, StatusSummaryItem } from '~/types/status-summary'

interface BreakdownTelemetry {
  telemetry_id: string
  name: string
  display_name?: string
  status: string
  uptime_pct: number
  days: StatusSummaryDay[]
  has_per_telemetry_samples: boolean
}

interface BreakdownResponse {
  telemetries: BreakdownTelemetry[]
}

const props = withDefaults(
  defineProps<{
    item: StatusSummaryItem
    slug: string
    /** Public breakdown API path: environments vs services */
    resource?: 'environment' | 'service'
  }>(),
  { resource: 'environment' },
)

const { apiFetch } = useApi()
const { fetchPublic: fetchTopologyPublic } = useResourceTopology()

const expanded = ref(false)
const topologyOpen = ref(false)
const topologyTitle = ref('')
const topologyLoading = ref(false)
const topologyError = ref('')
const topologyData = ref<ResourceTopology | null>(null)
const topologySessionKey = ref(0)
const breakdownLoading = ref(false)
const breakdown = ref<BreakdownResponse | null>(null)
const breakdownError = ref('')

const probeHistoryOpen = ref(false)
const probeHistoryInitialTelemetryId = ref<string | null>(null)

const breakdownId = computed(() => {
  const p = props.resource === 'service' ? 'svc-breakdown' : 'env-breakdown'
  return `${p}-${props.slug.replace(/[^a-zA-Z0-9_-]/g, '-')}`
})

const telemetryCountLabel = computed(() => {
  if (breakdownLoading.value && breakdown.value === null) return '(…)'
  if (breakdownError.value && breakdown.value === null) return '(?)'
  if (breakdown.value) return `(${breakdown.value.telemetries.length})`
  return '(…)'
})

const telemetryAriaLabel = computed(() => {
  if (breakdown.value) {
    const n = breakdown.value.telemetries.length
    return expanded.value
      ? `Hide per-telemetry breakdown, ${n} ${n === 1 ? 'item' : 'items'}`
      : `Show per-telemetry breakdown, ${n} ${n === 1 ? 'item' : 'items'}`
  }
  if (breakdownError.value) return 'Per-telemetry breakdown unavailable, click to retry'
  return expanded.value ? 'Hide per-telemetry breakdown' : 'Show per-telemetry breakdown'
})

function telemetryLabel(row: BreakdownTelemetry): string {
  const d = row.display_name?.trim()
  if (d) return d
  return row.name
}

function telemetryRowToItem(row: BreakdownTelemetry): StatusSummaryItem {
  return {
    name: telemetryLabel(row),
    slug: row.telemetry_id,
    status: row.status,
    uptime_pct: row.uptime_pct,
    days: row.days,
  }
}

async function loadBreakdown() {
  if (breakdown.value !== null) return
  if (breakdownLoading.value) return
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

onMounted(() => {
  void loadBreakdown()
})

function closeResourceMap() {
  topologyOpen.value = false
  topologyData.value = null
  topologyError.value = ''
}

async function openResourceMap() {
  topologySessionKey.value += 1
  topologyTitle.value = props.item.name
  topologyOpen.value = true
  topologyLoading.value = true
  topologyError.value = ''
  topologyData.value = null
  try {
    const kind = props.resource === 'service' ? 'service' : 'environment'
    topologyData.value = await fetchTopologyPublic(kind, props.slug)
  } catch {
    topologyError.value = 'Could not load resource map. It may only be available for public resources.'
  } finally {
    topologyLoading.value = false
  }
}

function openResourceProbeHistory() {
  probeHistoryInitialTelemetryId.value = null
  probeHistoryOpen.value = true
}

function openTelemetryProbeHistory(telemetryId: string) {
  probeHistoryInitialTelemetryId.value = telemetryId
  probeHistoryOpen.value = true
}

function closeProbeHistory() {
  probeHistoryOpen.value = false
  probeHistoryInitialTelemetryId.value = null
}
</script>

<style scoped>
.environment-status-card {
  display: flex;
  flex-direction: column;
  gap: 0;
}
.env-card__head-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin: 0;
  padding: 2px 4px;
  border: none;
  background: transparent;
  color: var(--color-primary);
  border-radius: var(--radius, 8px);
  cursor: pointer;
  flex-shrink: 0;
}
.env-card__head-action:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
.env-card__head-action--telemetry {
  gap: 4px;
  font-size: 0.82rem;
  font-weight: 500;
  text-decoration: none;
}
.env-card__head-action--telemetry:hover {
  color: var(--color-primary-hover);
}
.env-card__telemetry-count {
  font-variant-numeric: tabular-nums;
  text-decoration: underline;
  text-underline-offset: 2px;
}
.env-card__head-action--map {
  padding: 4px;
  color: var(--color-primary);
}
.env-card__head-action--map:hover {
  color: var(--color-primary-hover);
}
.env-card__chev {
  display: inline-block;
  transition: transform 0.15s ease;
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
.env-card__history-btn {
  flex-shrink: 0;
  margin-right: 4px;
}
.env-card__sub-head-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 10px;
  margin-bottom: 6px;
}
.env-card__sub-head {
  font-weight: 600;
  font-size: 0.88rem;
  margin-bottom: 0;
  min-width: 0;
  flex: 1;
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
