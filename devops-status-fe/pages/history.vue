<template>
  <div class="container history-page">
    <h1 class="page-title">History</h1>

    <div class="tabs-wrap">
      <div class="tabs">
        <button
          type="button"
          class="tab"
          :class="{ 'tab--active': activeTab === 'incidents' }"
          @click="activeTab = 'incidents'"
        >
          Incidents
        </button>
        <button
          type="button"
          class="tab"
          :class="{ 'tab--active': activeTab === 'uptime' }"
          @click="activeTab = 'uptime'"
        >
          Uptime
        </button>
      </div>
    </div>

    <div class="controls-row">
      <div class="controls-left">
        <AdminSelect
          v-model="targetFilter"
          aria-label="Filter by target type"
          :options="historyTargetFilterOptions"
        />
        <AdminSelect
          v-model="componentSlug"
          aria-label="Filter by component"
          empty-option-label="All components"
          placeholder="All components"
          :groups="historyComponentFilterGroups"
        />
      </div>
      <div class="controls-right">
        <button type="button" class="nav-btn" aria-label="Previous period" @click="shiftMonths(-3)">
          ‹
        </button>
        <span class="range-label">{{ threeMonthRangeLabelChrono }}</span>
        <button
          type="button"
          class="nav-btn"
          aria-label="Next period"
          :disabled="!canShiftNext"
          @click="shiftMonths(3)"
        >
          ›
        </button>
      </div>
    </div>

    <!-- Incidents tab -->
    <div v-if="activeTab === 'incidents'" class="panel incidents-panel">
      <p v-if="incidentsLoading" class="muted">Loading incidents…</p>
      <template v-else>
        <section v-for="group in incidentsByMonth" :key="group.key" class="month-section">
          <h2 class="month-heading">{{ group.label }}</h2>
          <div class="month-divider" />
          <template v-if="group.total === 0">
            <p class="muted month-empty">No incidents this month.</p>
          </template>
          <template v-else>
            <ul class="timeline">
              <li
                v-for="inc in group.visible"
                :key="inc.id"
                class="timeline-item"
              >
                <div class="timeline-marker" aria-hidden="true" />
                <div class="timeline-body">
                  <h3 class="entry-title" :class="'severity--' + inc.severity">{{ inc.title }}</h3>
                  <p v-if="inc.issue_message" class="entry-text">{{ inc.issue_message }}</p>
                  <p v-if="inc.resolution_message" class="entry-text entry-text--resolution">{{ inc.resolution_message }}</p>
                  <p class="entry-meta">{{ formatIncidentUtcRange(inc) }}</p>
                  <button
                    type="button"
                    class="details-toggle"
                    @click="toggleIncidentDetails(inc.id)"
                  >
                    {{ expandedIncidentId === inc.id ? 'Hide updates' : 'Updates / details' }}
                  </button>
                  <div v-if="expandedIncidentId === inc.id" class="details-embed">
                    <PastIncidentCard :incident="inc" />
                  </div>
                </div>
              </li>
            </ul>
            <button
              v-if="group.hiddenCount > 0 && !showAllMonths[group.key]"
              type="button"
              class="show-all-btn"
              @click="showAllMonths[group.key] = true"
            >
              + Show all {{ group.total }} incidents
            </button>
          </template>
        </section>
      </template>
    </div>

    <!-- Uptime tab -->
    <div v-else class="panel uptime-panel">
      <p v-if="uptimeLoading" class="muted">Loading uptime…</p>
      <template v-else-if="uptimePayload">
        <h2 class="uptime-name">{{ uptimeComponentName }}</h2>
        <div class="uptime-calendars">
          <div v-for="cal in uptimeCalendars" :key="cal.key" class="uptime-month">
            <div class="uptime-month-head">
              <span class="uptime-month-title">{{ cal.label }}</span>
              <span class="uptime-month-pct">{{ cal.monthPctLabel }}</span>
            </div>
            <div class="cal-grid" role="grid" :aria-label="`Daily uptime for ${cal.label}`">
              <div
                v-for="(cell, idx) in cal.cells"
                :key="idx"
                class="cal-cell-wrap"
              >
                <button
                  v-if="cell.date"
                  type="button"
                  class="cal-cell"
                  :class="'cal-cell--' + cell.level"
                  :aria-label="cellAriaLabel(cell)"
                  @mouseenter="(e) => openDayPopover(e, cell)"
                  @mouseleave="scheduleClosePopover"
                  @focus="(e) => openDayPopover(e, cell)"
                  @blur="onDayBlur"
                />
                <div v-else class="cal-cell cal-cell--pad" aria-hidden="true" />
              </div>
            </div>
          </div>
        </div>
      </template>
      <p v-else-if="uptimeNoTargetsMessage" class="placeholder-text">{{ uptimeNoTargetsMessage }}</p>
      <p v-else class="placeholder-text">Could not load uptime.</p>
    </div>

    <Teleport to="body">
      <div
        v-if="popoverCell?.date && popoverAnchor"
        ref="popoverEl"
        role="tooltip"
        class="day-popover"
        :style="popoverStyle"
        @mouseenter="cancelClosePopover"
        @mouseleave="scheduleClosePopover"
      >
        <div class="day-popover__date">{{ popoverDateHeader }}</div>
        <p class="day-popover__summary">{{ popoverSummary }}</p>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import PastIncidentCard from '~/components/PastIncidentCard.vue'
import type { IncidentDisplayItem } from '~/types/incident-display'

const { apiFetch } = useApi()

const activeTab = ref<'incidents' | 'uptime'>('incidents')
const targetFilter = ref('')
const componentSlug = ref('')

const nowUtc = () => new Date()
const utcYmd = (y: number, m0: number, d: number) => {
  const iso = new Date(Date.UTC(y, m0, d)).toISOString()
  return iso.slice(0, 10)
}

function addUtcMonths(ym: { y: number; m: number }, delta: number) {
  const d = new Date(Date.UTC(ym.y, ym.m + delta, 1))
  return { y: d.getUTCFullYear(), m: d.getUTCMonth() }
}

const n0 = nowUtc()
/** Newest month in the 3-month strip (UTC), as { y, m } with m = 0..11 */
const newestYm = ref<{ y: number; m: number }>({
  y: n0.getUTCFullYear(),
  m: n0.getUTCMonth(),
})

const maxYm = computed(() => {
  const n = nowUtc()
  return { y: n.getUTCFullYear(), m: n.getUTCMonth() }
})

/** True if shifting the anchor forward by 3 months stays within or before the current calendar month (UTC). */
const canShiftNext = computed(() => {
  const n = addUtcMonths(newestYm.value, 3)
  const max = maxYm.value
  if (n.y < max.y) return true
  if (n.y > max.y) return false
  return n.m <= max.m
})

function shiftMonths(delta: number) {
  if (delta > 0 && !canShiftNext.value) return
  newestYm.value = addUtcMonths(newestYm.value, delta)
}

/** Oldest → newest (e.g. Feb, Mar, Apr). Used for API range, range label, and uptime calendars (left → right). */
const threeVisibleMonthsChrono = computed(() => {
  const { y, m } = newestYm.value
  const out: { y: number; m: number; label: string; key: string }[] = []
  for (let i = 2; i >= 0; i--) {
    const d = new Date(Date.UTC(y, m - i, 1))
    const yy = d.getUTCFullYear()
    const mm = d.getUTCMonth()
    out.push({
      y: yy,
      m: mm,
      label: d.toLocaleDateString('en-US', { month: 'long', year: 'numeric', timeZone: 'UTC' }),
      key: `${yy}-${String(mm + 1).padStart(2, '0')}`,
    })
  }
  return out
})

const threeMonthRangeLabelChrono = computed(() => {
  const m = threeVisibleMonthsChrono.value
  if (m.length < 3) return ''
  return `${m[0].label} – ${m[2].label}`
})

const incidentSinceUntil = computed(() => {
  const months = threeVisibleMonthsChrono.value
  if (!months.length) return { since: '', until: '' }
  const first = months[0]
  const last = months[2]
  const since = utcYmd(first.y, first.m, 1)
  const until = utcYmd(last.y, last.m + 1, 1)
  return { since, until }
})

/** Last calendar day (UTC) of the newest visible month, for history API `end=` */
const historyEndYmd = computed(() => {
  const last = threeVisibleMonthsChrono.value[2]
  if (!last) return ''
  const lastDay = new Date(Date.UTC(last.y, last.m + 1, 0)).getUTCDate()
  return utcYmd(last.y, last.m, lastDay)
})

const showServiceComponentOptions = computed(
  () => !targetFilter.value || targetFilter.value === 'service',
)
const showEnvironmentComponentOptions = computed(
  () => !targetFilter.value || targetFilter.value === 'environment',
)

const historyTargetFilterOptions = [
  { value: '', label: 'All' },
  { value: 'service', label: 'Services' },
  { value: 'environment', label: 'Environments' },
] as const

const historyComponentFilterGroups = computed(() => {
  const g: { label: string; options: { value: string; label: string }[] }[] = []
  if (showServiceComponentOptions.value && (services.value?.length ?? 0) > 0) {
    g.push({
      label: 'Services',
      options: (services.value ?? []).map((s: { slug: string; name: string }) => ({
        value: `service:${s.slug}`,
        label: s.name,
      })),
    })
  }
  if (showEnvironmentComponentOptions.value && (environments.value?.length ?? 0) > 0) {
    g.push({
      label: 'Environments',
      options: (environments.value ?? []).map((e: { slug: string; name: string }) => ({
        value: `environment:${e.slug}`,
        label: e.name,
      })),
    })
  }
  return g
})

watch(targetFilter, (tf) => {
  if (!componentSlug.value) return
  const prefix = componentSlug.value.split(':')[0]
  if (tf === 'service' && prefix === 'environment') componentSlug.value = ''
  if (tf === 'environment' && prefix === 'service') componentSlug.value = ''
})

const { data: services } = await useAsyncData('pub-services', () =>
  apiFetch<any[]>('/api/public/services'),
  { default: () => [] },
)

const { data: environments } = await useAsyncData('pub-environments', () =>
  apiFetch<any[]>('/api/public/environments'),
  { default: () => [] },
)

const incidents = ref<IncidentDisplayItem[]>([])
const incidentsLoading = ref(false)

async function loadIncidents() {
  const { since, until } = incidentSinceUntil.value
  if (!since || !until) return
  incidentsLoading.value = true
  try {
    const q = new URLSearchParams()
    q.set('limit', '200')
    q.set('since', since)
    q.set('until', until)
    if (targetFilter.value) q.set('target_type', targetFilter.value)
    incidents.value = await apiFetch<IncidentDisplayItem[]>(`/api/public/incidents?${q.toString()}`)
  } catch {
    incidents.value = []
  } finally {
    incidentsLoading.value = false
  }
}

watch([newestYm, targetFilter], () => {
  loadIncidents()
}, { deep: true })

onMounted(() => {
  loadIncidents()
})

const filteredIncidents = computed(() => {
  const list = incidents.value || []
  if (!componentSlug.value) return list
  const [tt, slug] = componentSlug.value.split(':')
  if (!tt || !slug) return list
  return list.filter((i) => i.target_type === tt && i.target_slug === slug)
})

const INCIDENTS_PREVIEW = 5
const showAllMonths = reactive<Record<string, boolean>>({})
const expandedIncidentId = ref<string | null>(null)

function toggleIncidentDetails(id: string) {
  expandedIncidentId.value = expandedIncidentId.value === id ? null : id
}

const incidentsByMonth = computed(() => {
  const byKey = new Map<string, IncidentDisplayItem[]>()
  for (const inc of filteredIncidents.value) {
    const d = new Date(inc.started_at)
    const y = d.getUTCFullYear()
    const m = d.getUTCMonth()
    const key = `${y}-${String(m + 1).padStart(2, '0')}`
    if (!byKey.has(key)) byKey.set(key, [])
    byKey.get(key)!.push(inc)
  }
  for (const arr of byKey.values()) {
    arr.sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime())
  }

  return [...threeVisibleMonthsChrono.value].reverse().map((vm) => {
    const list = byKey.get(vm.key) || []
    const total = list.length
    const showAll = showAllMonths[vm.key]
    const visible = showAll || total <= INCIDENTS_PREVIEW ? list : list.slice(0, INCIDENTS_PREVIEW)
    const hiddenCount = showAll || total <= INCIDENTS_PREVIEW ? 0 : total - INCIDENTS_PREVIEW
    return {
      key: vm.key,
      label: vm.label,
      visible,
      hiddenCount,
      total,
    }
  })
})

watch(threeVisibleMonthsChrono, (months) => {
  for (const m of months) {
    if (!(m.key in showAllMonths)) showAllMonths[m.key] = false
  }
}, { immediate: true })

function formatIncidentUtcRange(inc: IncidentDisplayItem): string {
  const start = new Date(inc.started_at)
  const end = inc.resolved_at ? new Date(inc.resolved_at) : null
  const dateOpts: Intl.DateTimeFormatOptions = {
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC',
  }
  const timeOpts: Intl.DateTimeFormatOptions = {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'UTC',
  }
  const sameDay =
    end &&
    start.getUTCFullYear() === end.getUTCFullYear() &&
    start.getUTCMonth() === end.getUTCMonth() &&
    start.getUTCDate() === end.getUTCDate()
  if (!end) {
    const line = `${start.toLocaleString('en-US', { ...dateOpts, ...timeOpts, year: 'numeric' })} UTC`
    return `${line} – ongoing`
  }
  if (sameDay) {
    const dayPart = start.toLocaleString('en-US', { ...dateOpts, year: 'numeric' })
    const t0 = start.toLocaleString('en-US', timeOpts)
    const t1 = end.toLocaleString('en-US', timeOpts)
    return `${dayPart}, ${t0} – ${t1} UTC`
  }
  const a = start.toLocaleString('en-US', { ...dateOpts, ...timeOpts, year: 'numeric' })
  const b = end.toLocaleString('en-US', { ...dateOpts, ...timeOpts, year: 'numeric' })
  return `${a} – ${b} UTC`
}

/** Uptime */
interface DayRollup {
  date: string
  availability_pct: number
  qos_level: string
  total_samples: number
  failed_samples: number
}

interface UptimePayload {
  service?: { name: string }
  environment?: { name: string }
  /** When set, this row is an aggregate over many targets (no service/environment object). */
  aggregateLabel?: string
  days: DayRollup[]
}

const parsedComponent = computed(() => {
  if (!componentSlug.value) return { type: '' as const, slug: '' }
  const [type, slug] = componentSlug.value.split(':')
  if (type !== 'service' && type !== 'environment') return { type: '' as const, slug: '' }
  return { type, slug }
})

const uptimeTargetsForAggregate = computed((): { type: 'service' | 'environment'; slug: string }[] => {
  const tf = targetFilter.value
  const out: { type: 'service' | 'environment'; slug: string }[] = []
  const svcs = services.value ?? []
  const envs = environments.value ?? []
  if (!tf || tf === 'service') {
    for (const s of svcs) {
      if (s?.slug) out.push({ type: 'service', slug: s.slug })
    }
  }
  if (!tf || tf === 'environment') {
    for (const e of envs) {
      if (e?.slug) out.push({ type: 'environment', slug: e.slug })
    }
  }
  return out
})

const uptimeNoTargetsMessage = computed(() => {
  if (activeTab.value !== 'uptime') return ''
  if (componentSlug.value) return ''
  if (uptimeTargetsForAggregate.value.length === 0) {
    return 'No services or environments are configured for this filter.'
  }
  return ''
})

const uptimeLoading = ref(false)
const uptimePayload = ref<UptimePayload | null>(null)

const uptimeIsAggregate = computed(() => !!uptimePayload.value?.aggregateLabel)

const uptimeComponentName = computed(() => {
  const p = uptimePayload.value
  if (!p) return ''
  return p.aggregateLabel || p.service?.name || p.environment?.name || ''
})

const uptimeDays = computed(() => uptimePayload.value?.days || [])

function qosLevelFromAggregatedPct(pct: number): string {
  if (pct >= 99) return 'green'
  if (pct >= 95) return 'yellow'
  return 'red'
}

function mergeAggregatedRollups(series: DayRollup[][]): DayRollup[] {
  const map = new Map<string, { total: number; failed: number }>()
  for (const days of series) {
    if (!days?.length) continue
    for (const r of days) {
      const cur = map.get(r.date) ?? { total: 0, failed: 0 }
      cur.total += r.total_samples
      cur.failed += r.failed_samples
      map.set(r.date, cur)
    }
  }
  const out: DayRollup[] = []
  for (const date of [...map.keys()].sort((a, b) => a.localeCompare(b))) {
    const v = map.get(date)!
    if (v.total <= 0) continue
    const availability_pct = Math.round((10000 * (v.total - v.failed)) / v.total) / 100
    out.push({
      date,
      availability_pct,
      qos_level: qosLevelFromAggregatedPct(availability_pct),
      total_samples: v.total,
      failed_samples: v.failed,
    })
  }
  return out
}

async function loadUptime() {
  const end = historyEndYmd.value
  const q = end ? `?end=${encodeURIComponent(end)}` : ''

  if (componentSlug.value) {
    const { type, slug } = parsedComponent.value
    if (!type || !slug) {
      uptimePayload.value = null
      return
    }
    uptimeLoading.value = true
    try {
      uptimePayload.value = await apiFetch<UptimePayload>(
        `/api/public/${type}s/${encodeURIComponent(slug)}/history${q}`,
      )
    } catch {
      uptimePayload.value = null
    } finally {
      uptimeLoading.value = false
    }
    return
  }

  const targets = uptimeTargetsForAggregate.value
  if (targets.length === 0) {
    uptimePayload.value = null
    return
  }

  uptimeLoading.value = true
  try {
    const results = await Promise.all(
      targets.map((t) =>
        apiFetch<UptimePayload>(`/api/public/${t.type}s/${encodeURIComponent(t.slug)}/history${q}`),
      ),
    )
    const merged = mergeAggregatedRollups(results.map((r) => r.days || []))
    const tf = targetFilter.value
    const n = targets.length
    let aggregateLabel = `All components (${n})`
    if (tf === 'service') aggregateLabel = `All services (${n})`
    else if (tf === 'environment') aggregateLabel = `All environments (${n})`
    uptimePayload.value = { days: merged, aggregateLabel }
  } catch {
    uptimePayload.value = null
  } finally {
    uptimeLoading.value = false
  }
}

watch(
  [componentSlug, newestYm, historyEndYmd, targetFilter, services, environments],
  () => {
    if (activeTab.value === 'uptime') loadUptime()
  },
  { deep: true },
)

watch(activeTab, (t) => {
  if (t === 'uptime') loadUptime()
})

interface CalendarCell {
  date: string | null
  level: string
  rollup: DayRollup | null
}

function todayUtcYmd(): string {
  return new Date().toISOString().slice(0, 10)
}

function rollupByDateMap(days: DayRollup[]): Map<string, DayRollup> {
  const m = new Map<string, DayRollup>()
  for (const d of days) m.set(d.date, d)
  return m
}

function cellLevelForDate(iso: string, rollup: DayRollup | undefined, today: string): string {
  if (iso > today) return 'future'
  if (!rollup) return 'empty'
  return rollup.qos_level || 'green'
}

function monthAggregatedPct(y: number, m0: number, byDate: Map<string, DayRollup>, today: string): string {
  const dim = new Date(Date.UTC(y, m0 + 1, 0)).getUTCDate()
  let tot = 0
  let fail = 0
  for (let d = 1; d <= dim; d++) {
    const key = utcYmd(y, m0, d)
    if (key > today) continue
    const r = byDate.get(key)
    if (!r || r.total_samples <= 0) continue
    tot += r.total_samples
    fail += r.failed_samples
  }
  if (tot <= 0) return '—'
  const pct = (100 * (tot - fail)) / tot
  return `${pct.toFixed(pct >= 99.95 ? 1 : 2)}%`
}

function rollupSummaryLine(day: DayRollup): string {
  const failed = day.failed_samples
  const total = day.total_samples
  if (day.availability_pct >= 100 && failed === 0) return 'No downtime recorded on this day.'
  const pct = day.availability_pct.toFixed(1)
  const qos = day.qos_level === 'green' ? 'Healthy' : day.qos_level === 'yellow' ? 'Degraded' : 'Outage'
  if (total > 0) return `${qos} — ${pct}% successful checks (${failed} failed of ${total}).`
  return `${qos} — ${pct}% successful checks.`
}

const uptimeCalendars = computed(() => {
  const byDate = rollupByDateMap(uptimeDays.value)
  const today = todayUtcYmd()
  return threeVisibleMonthsChrono.value.map((vm) => {
    const dim = new Date(Date.UTC(vm.y, vm.m + 1, 0)).getUTCDate()
    const firstDow = new Date(Date.UTC(vm.y, vm.m, 1)).getUTCDay()
    const cells: CalendarCell[] = []
    for (let i = 0; i < firstDow; i++) cells.push({ date: null, level: 'pad', rollup: null })
    for (let d = 1; d <= dim; d++) {
      const iso = utcYmd(vm.y, vm.m, d)
      const rollup = byDate.get(iso)
      cells.push({
        date: iso,
        level: cellLevelForDate(iso, rollup, today),
        rollup: rollup ?? null,
      })
    }
    return {
      key: vm.key,
      label: vm.label,
      monthPctLabel: monthAggregatedPct(vm.y, vm.m, byDate, today),
      cells,
    }
  })
})

/** Popover */
const popoverCell = ref<CalendarCell | null>(null)
const popoverAnchor = ref<HTMLElement | null>(null)
const popoverEl = ref<HTMLElement | null>(null)
const popoverStyle = ref<Record<string, string>>({})
let popTimer: ReturnType<typeof setTimeout> | null = null

function positionPopover(el: HTMLElement) {
  const r = el.getBoundingClientRect()
  const x = r.left + r.width / 2
  const y = r.bottom + 6
  popoverStyle.value = {
    position: 'fixed',
    left: `${Math.round(x)}px`,
    top: `${Math.round(y)}px`,
    transform: 'translateX(-50%)',
    zIndex: '10050',
  }
}

function openDayPopover(e: Event, cell: CalendarCell) {
  if (!cell.date) return
  cancelClosePopover()
  popoverCell.value = cell
  popoverAnchor.value = e.currentTarget as HTMLElement
  nextTick(() => {
    const el = popoverAnchor.value
    if (el) positionPopover(el)
  })
}

function cancelClosePopover() {
  if (popTimer) {
    clearTimeout(popTimer)
    popTimer = null
  }
}

function scheduleClosePopover() {
  cancelClosePopover()
  popTimer = setTimeout(() => {
    popoverCell.value = null
    popoverAnchor.value = null
    popTimer = null
  }, 150)
}

function onDayBlur(ev: FocusEvent) {
  const next = ev.relatedTarget as Node | null
  if (popoverEl.value && next && popoverEl.value.contains(next)) return
  scheduleClosePopover()
}

const popoverDateHeader = computed(() => {
  const c = popoverCell.value
  if (!c?.date) return ''
  const [y, m, d] = c.date.split('-').map(Number)
  const dt = new Date(Date.UTC(y, m - 1, d))
  return dt.toLocaleDateString(undefined, {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    timeZone: 'UTC',
  })
})

const popoverSummary = computed(() => {
  const c = popoverCell.value
  if (!c?.rollup) {
    if (c?.level === 'future') return 'Future date.'
    return 'No probe data for this day.'
  }
  let line = rollupSummaryLine(c.rollup)
  if (uptimeIsAggregate.value) {
    line += ' Aggregated across all selected targets (probe checks summed per day).'
  }
  return line
})

function cellAriaLabel(cell: CalendarCell): string {
  if (!cell.date) return ''
  const r = cell.rollup
  if (!r) return `${cell.date}, no data`
  return `${cell.date}, ${r.availability_pct}% availability`
}

onBeforeUnmount(() => {
  cancelClosePopover()
})
</script>

<style scoped>
.history-page {
  padding-top: 16px;
  padding-bottom: 48px;
}
.page-title {
  font-size: 1.5rem;
  font-weight: 700;
  margin-bottom: 20px;
  letter-spacing: -0.02em;
  color: var(--color-text);
}

.tabs-wrap {
  border-bottom: 1px solid var(--color-border-strong);
  margin-bottom: 16px;
}
.tabs {
  display: flex;
  gap: 0;
}
.tab {
  padding: 10px 20px;
  font-size: 0.875rem;
  font-weight: 500;
  background: color-mix(in srgb, var(--color-text) 6%, var(--color-bg));
  border: 1px solid var(--color-border-strong);
  border-bottom: none;
  margin: 0 0 -1px -1px;
  color: var(--color-text-secondary);
  cursor: pointer;
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
}
.tab:first-child {
  margin-left: 0;
}
.tab:hover {
  color: var(--color-text);
}
.tab--active {
  background: var(--color-bg);
  color: var(--color-text);
  font-weight: 600;
  border-bottom: 1px solid var(--color-bg);
  position: relative;
  z-index: 1;
}

.controls-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 28px;
}
.controls-left {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
.controls-right {
  display: flex;
  align-items: center;
  gap: 10px;
}
.nav-btn {
  width: 36px;
  height: 36px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  font-size: 1.25rem;
  line-height: 1;
  cursor: pointer;
  color: var(--color-text);
}
.nav-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.range-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text);
  min-width: 200px;
  text-align: center;
}

.controls-left :deep(.admin-select) {
  min-width: 160px;
  max-width: min(320px, 100%);
}

.muted {
  color: var(--color-text-secondary);
  font-size: 0.9rem;
}
.placeholder-text {
  color: var(--color-text-secondary);
  font-size: 0.9rem;
  padding: 24px 0;
}

.month-section {
  margin-bottom: 40px;
}
.month-heading {
  font-size: 1.35rem;
  font-weight: 700;
  margin: 0 0 8px;
  color: var(--color-text);
}
.month-divider {
  height: 1px;
  background: var(--color-border-strong);
  margin-bottom: 20px;
}
.month-empty {
  margin: 0 0 8px;
}

.timeline {
  list-style: none;
  margin: 0;
  padding: 0;
  position: relative;
  border-left: 2px solid var(--color-border);
  margin-left: 11px;
}
.timeline-item {
  position: relative;
  padding: 0 0 28px 24px;
}
.timeline-item:last-child {
  padding-bottom: 8px;
}
.timeline-marker {
  position: absolute;
  left: -24px;
  top: 4px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--color-bg);
  border: 2px solid var(--color-border-strong);
  box-sizing: border-box;
}
.timeline-marker::after {
  content: 'i';
  display: block;
  text-align: center;
  font-size: 0.65rem;
  font-weight: 700;
  line-height: 16px;
  color: var(--color-text-secondary);
  font-style: italic;
  font-family: serif;
}

.entry-title {
  font-size: 1.05rem;
  font-weight: 600;
  margin: 0 0 8px;
}
.severity--major {
  color: var(--color-red);
}
.severity--minor {
  color: var(--color-orange);
}
.entry-text {
  margin: 0 0 10px;
  font-size: 0.9rem;
  line-height: 1.55;
  color: var(--color-text);
}
.entry-text--resolution {
  color: var(--color-text-secondary);
}
.entry-meta {
  margin: 0 0 10px;
  font-size: 0.8rem;
  color: var(--color-text-secondary);
}
.details-toggle {
  background: none;
  border: none;
  padding: 0;
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--color-primary);
  cursor: pointer;
  text-decoration: underline;
  text-underline-offset: 2px;
  margin-bottom: 8px;
}
.details-embed {
  margin-top: 8px;
}

.show-all-btn {
  display: block;
  width: 100%;
  margin-top: 8px;
  padding: 12px 16px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-primary);
  cursor: pointer;
}
.show-all-btn:hover {
  background: color-mix(in srgb, var(--color-text) 4%, var(--color-bg));
}

.uptime-name {
  font-size: 1.15rem;
  font-weight: 600;
  margin: 0 0 20px;
}
.uptime-calendars {
  display: flex;
  flex-wrap: wrap;
  gap: 28px;
  align-items: flex-start;
}
.uptime-month {
  flex: 1 1 220px;
  max-width: 320px;
}
.uptime-month-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 10px;
}
.uptime-month-title {
  font-weight: 600;
  font-size: 0.95rem;
}
.uptime-month-pct {
  font-size: 0.85rem;
  color: var(--color-text-secondary);
  font-weight: 500;
}
.cal-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 4px;
}
.cal-cell-wrap {
  aspect-ratio: 1;
  min-width: 0;
}
.cal-cell {
  width: 100%;
  height: 100%;
  min-height: 22px;
  border: none;
  border-radius: 3px;
  padding: 0;
  cursor: pointer;
  display: block;
}
.cal-cell:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 1px;
}
.cal-cell--green {
  background: var(--color-green);
}
.cal-cell--yellow {
  background: var(--color-yellow);
}
.cal-cell--red {
  background: var(--color-red);
}
.cal-cell--empty,
.cal-cell--future,
.cal-cell--pad {
  background: #e8eaef;
  cursor: default;
}
.cal-cell--pad {
  visibility: hidden;
}
</style>

<style>
.day-popover {
  min-width: 200px;
  max-width: 300px;
  padding: 12px 14px;
  background: var(--color-bg, #fff);
  color: var(--color-text, #1a1a1a);
  border: 1px solid var(--color-border-strong, #d0d4dc);
  border-radius: var(--radius, 8px);
  box-shadow: var(--shadow-card, 0 4px 24px rgba(0, 0, 0, 0.12));
  font-size: 0.82rem;
  line-height: 1.45;
  pointer-events: auto;
}
.day-popover__date {
  font-weight: 600;
  font-size: 0.88rem;
  margin-bottom: 6px;
}
.day-popover__summary {
  margin: 0;
  color: var(--color-text-secondary, #5c6370);
}
</style>
