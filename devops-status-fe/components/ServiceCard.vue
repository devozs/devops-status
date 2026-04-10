<template>
  <div class="service-card" :class="{ 'service-card--nested': variant === 'nested' }">
    <div class="card-header">
      <span class="card-name">{{ item.name }}</span>
      <span class="card-status" :class="'status--' + item.status">{{ statusLabel }}</span>
    </div>
    <div
      ref="barRef"
      class="card-bar"
      @mouseleave="scheduleHidePopover"
    >
      <template v-for="(day, i) in timelineDays" :key="i">
        <button
          v-if="day.date"
          type="button"
          class="bar-segment bar-segment--interactive"
          :class="`bar--${day.level}`"
          :aria-describedby="popoverOpen && activeIndex === i ? tooltipId : undefined"
          @mouseenter="(e) => openPopover(i, day, e.currentTarget as HTMLElement)"
          @focus="(e) => openPopover(i, day, e.currentTarget as HTMLElement)"
          @blur="onSegmentBlur"
        />
        <div
          v-else
          class="bar-segment"
          :class="`bar--${day.level}`"
          aria-hidden="true"
        />
      </template>
    </div>
    <div class="card-footer">
      <span class="card-range">90 days ago</span>
      <span class="card-uptime">{{ uptimeLabel }}</span>
      <span class="card-range">Today</span>
    </div>
    <Teleport to="body">
      <div
        v-if="popoverOpen && popoverDay"
        :id="tooltipId"
        ref="popoverRef"
        role="tooltip"
        class="uptime-day-popover"
        :style="popoverStyle"
        @mouseenter="cancelHidePopover"
        @mouseleave="scheduleHidePopover"
      >
        <div class="uptime-day-popover__date">{{ formatUtcDateHeader(popoverDay.date) }}</div>
        <p class="uptime-day-popover__summary">{{ rollupSummaryLine(popoverDay) }}</p>
        <div v-if="popoverRelated.length" class="uptime-day-popover__related">
          <div class="uptime-day-popover__related-label">Related incidents</div>
          <ul class="uptime-day-popover__list">
            <li v-for="inc in popoverRelated" :key="inc.id" class="uptime-day-popover__li">
              <span class="uptime-day-popover__inc-title">{{ inc.title }}</span>
              <span class="uptime-day-popover__meta">{{ incidentMetaLine(inc, popoverDay.date) }}</span>
            </li>
          </ul>
        </div>
        <p v-else class="uptime-day-popover__none">No related incidents for this day.</p>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import type { PublicIncidentStub, StatusSummaryItem } from '~/types/status-summary'

interface TimelineDay {
  date: string
  pct: number
  level: string
  availability_pct: number
  qos_level: string
  total_samples: number
  failed_samples: number
}

const props = withDefaults(
  defineProps<{
    item: StatusSummaryItem
    /** Smaller padding for nested / per-telemetry rows */
    variant?: 'default' | 'nested'
    /** Incidents for this resource from status summary (parent + nested reuse). */
    recentIncidents?: PublicIncidentStub[] | null
    /** When set (nested telemetry row), filter incidents by source_telemetry_id or null (legacy). */
    telemetryId?: string | null
  }>(),
  {
    variant: 'default',
    recentIncidents: null,
    telemetryId: null,
  },
)

const barRef = ref<HTMLElement | null>(null)
const popoverRef = ref<HTMLElement | null>(null)
const tooltipId = `uptime-tt-${Math.random().toString(36).slice(2, 11)}`
const activeIndex = ref<number | null>(null)
const popoverAnchorEl = ref<HTMLElement | null>(null)
const popoverDay = ref<TimelineDay | null>(null)
const popoverStyle = ref<Record<string, string>>({})
const popoverRelated = ref<PublicIncidentStub[]>([])

const popoverOpen = computed(() => popoverDay.value != null && activeIndex.value != null)

let hideTimer: ReturnType<typeof setTimeout> | null = null

function cancelHidePopover() {
  if (hideTimer != null) {
    clearTimeout(hideTimer)
    hideTimer = null
  }
}

function scheduleHidePopover() {
  cancelHidePopover()
  hideTimer = setTimeout(() => {
    closePopover()
    hideTimer = null
  }, 180)
}

function closePopover() {
  popoverDay.value = null
  activeIndex.value = null
  popoverAnchorEl.value = null
  popoverRelated.value = []
}

function utcDayRangeMs(iso: string): { start: number; end: number } {
  const [y, m, d] = iso.split('-').map(Number)
  const start = Date.UTC(y, m - 1, d)
  return { start, end: start + 86400000 }
}

function incidentOverlapsUtcDay(inc: PublicIncidentStub, iso: string): boolean {
  const { start, end } = utcDayRangeMs(iso)
  const s = new Date(inc.started_at).getTime()
  const e = inc.resolved_at ? new Date(inc.resolved_at).getTime() : Date.now()
  return s < end && e > start
}

function incidentsForDay(iso: string): PublicIncidentStub[] {
  const list = props.recentIncidents ?? []
  const tel = props.telemetryId
  return list.filter((inc) => {
    if (!incidentOverlapsUtcDay(inc, iso)) return false
    if (tel == null || tel === '') return true
    const st = inc.source_telemetry_id
    return st == null || st === tel
  })
}

function formatUtcDateHeader(iso: string): string {
  const [y, m, d] = iso.split('-').map(Number)
  const dt = new Date(Date.UTC(y, m - 1, d))
  return dt.toLocaleDateString(undefined, {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    timeZone: 'UTC',
  })
}

function rollupSummaryLine(day: TimelineDay): string {
  const failed = day.failed_samples
  const total = day.total_samples
  if (day.availability_pct >= 100 && failed === 0) {
    return 'No downtime recorded on this day.'
  }
  const pct = day.availability_pct.toFixed(1)
  const qos =
    day.level === 'green' ? 'Healthy' : day.level === 'yellow' ? 'Degraded' : 'Outage'
  if (total > 0) {
    return `${qos} — ${pct}% successful checks (${failed} failed of ${total}).`
  }
  return `${qos} — ${pct}% successful checks.`
}

function formatDurationShort(ms: number): string {
  if (ms < 60000) return ''
  const m = Math.floor(ms / 60000)
  if (m < 60) return `${m}m`
  const h = Math.floor(m / 60)
  const rm = m % 60
  return rm ? `${h}h ${rm}m` : `${h}h`
}

function incidentMetaLine(inc: PublicIncidentStub, iso: string): string {
  const parts = [inc.severity]
  const s = new Date(inc.started_at).getTime()
  const e = inc.resolved_at ? new Date(inc.resolved_at).getTime() : Date.now()
  const { start, end } = utcDayRangeMs(iso)
  const overlapStart = Math.max(s, start)
  const overlapEnd = Math.min(e, end)
  const dur = formatDurationShort(overlapEnd - overlapStart)
  if (dur) parts.push(dur)
  if (inc.status === 'resolved') parts.push('resolved')
  else parts.push(inc.status)
  return parts.join(' · ')
}

function positionPopover(anchor: HTMLElement) {
  const r = anchor.getBoundingClientRect()
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

function openPopover(i: number, day: TimelineDay, el: HTMLElement) {
  if (!day.date) return
  cancelHidePopover()
  activeIndex.value = i
  popoverAnchorEl.value = el
  popoverDay.value = day
  popoverRelated.value = incidentsForDay(day.date)
  nextTick(() => positionPopover(el))
}

function onSegmentBlur(ev: FocusEvent) {
  const next = ev.relatedTarget as Node | null
  if (popoverRef.value && next && popoverRef.value.contains(next)) return
  scheduleHidePopover()
}

function onReposition() {
  const el = popoverAnchorEl.value
  if (el && popoverDay.value) positionPopover(el)
}

function onGlobalKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    cancelHidePopover()
    closePopover()
  }
}

onMounted(() => {
  window.addEventListener('scroll', onReposition, true)
  window.addEventListener('resize', onReposition)
  window.addEventListener('keydown', onGlobalKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', onReposition, true)
  window.removeEventListener('resize', onReposition)
  window.removeEventListener('keydown', onGlobalKeydown)
  cancelHidePopover()
  closePopover()
})

const statusLabel = computed(() => {
  if (props.item.status === 'disruption') return 'Disruption'
  return 'Operational'
})

const uptimeLabel = computed(() => {
  const pct = props.item.uptime_pct ?? 100
  return `${pct.toFixed(2)} % uptime`
})

const timelineDays = computed((): TimelineDay[] => {
  const days = props.item.days || []
  const mapped: TimelineDay[] = days.map((d) => ({
    date: d.date,
    pct: d.availability_pct,
    level: d.qos_level || 'green',
    availability_pct: d.availability_pct,
    qos_level: d.qos_level || 'green',
    total_samples: d.total_samples ?? 0,
    failed_samples: d.failed_samples ?? 0,
  }))
  const padCount = 90 - mapped.length
  const padding: TimelineDay[] = Array.from({ length: Math.max(0, padCount) }, () => ({
    date: '',
    pct: 100,
    level: 'empty',
    availability_pct: 100,
    qos_level: 'green',
    total_samples: 0,
    failed_samples: 0,
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
.bar-segment--interactive {
  margin: 0;
  padding: 0;
  border: none;
  cursor: pointer;
  display: block;
}
.bar-segment--interactive:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 1px;
  z-index: 1;
}
.bar--green { background: var(--color-green); }
.bar--yellow { background: var(--color-yellow); }
.bar--red { background: var(--color-red); }
.bar--empty { background: #e8eaef; }
.card-footer { display: flex; justify-content: space-between; font-size: 0.75rem; color: var(--color-text-secondary); }
.card-uptime { font-weight: 500; }
</style>

<style>
.uptime-day-popover {
  min-width: 220px;
  max-width: 320px;
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
.uptime-day-popover__date {
  font-weight: 600;
  font-size: 0.88rem;
  margin-bottom: 6px;
}
.uptime-day-popover__summary {
  margin: 0 0 10px;
  color: var(--color-text-secondary, #5c6370);
}
.uptime-day-popover__related-label {
  font-size: 0.7rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--color-text-secondary, #5c6370);
  margin-bottom: 6px;
}
.uptime-day-popover__list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.uptime-day-popover__li {
  padding: 6px 0;
  border-top: 1px solid var(--color-border, #e8eaef);
}
.uptime-day-popover__li:first-child {
  border-top: none;
  padding-top: 0;
}
.uptime-day-popover__inc-title {
  display: block;
  font-weight: 500;
}
.uptime-day-popover__meta {
  display: block;
  font-size: 0.78rem;
  color: var(--color-text-secondary, #5c6370);
  margin-top: 2px;
}
.uptime-day-popover__none {
  margin: 0;
  font-size: 0.8rem;
  color: var(--color-text-secondary, #5c6370);
}
</style>
