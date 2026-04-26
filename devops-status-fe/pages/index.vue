<template>
  <div class="container home-page">
    <StatusBanner :status="statusData.status" :message="statusData.message" />

    <div class="section-header">
      <span class="section-label">Uptime over the past 90 days.</span>
      <NuxtLink to="/history" class="section-link">View historical uptime.</NuxtLink>
    </div>

    <section v-if="statusData.environments?.length" class="status-section">
      <h2 class="section-title">Environments</h2>
      <div class="card-list">
        <EnvironmentStatusCard
          v-for="env in statusData.environments"
          :key="env.slug"
          :item="env"
          :slug="env.slug"
        />
      </div>
    </section>

    <section v-if="statusData.services?.length" class="status-section">
      <h2 class="section-title">Services</h2>
      <div class="card-list">
        <EnvironmentStatusCard
          v-for="svc in statusData.services"
          :key="svc.slug"
          resource="service"
          :item="svc"
          :slug="svc.slug"
        />
      </div>
    </section>

    <section v-if="incidents?.length" class="status-section">
      <h2 class="section-title">Past Incidents</h2>
      <p class="incidents-hint">Select an incident to see target, infra, telemetry, and update history.</p>
      <div class="incident-list">
        <PastIncidentCard v-for="inc in incidents" :key="inc.id" :incident="inc" />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import PastIncidentCard from '~/components/PastIncidentCard.vue'
import type { IncidentDisplayItem } from '~/types/incident-display'
import type { StatusSummaryItem } from '~/types/status-summary'

const { apiFetch } = useApi()

interface StatusSummaryPayload {
  status: string
  message: string
  services: StatusSummaryItem[]
  environments: StatusSummaryItem[]
}

// Home data must load in the browser after mount. Reasons:
// 1) SSR to the public API from the pod is unreliable without NUXT_API_BASE_INTERNAL.
// 2) useAsyncData({ server: false }) still embeds default empty values in the Nuxt payload on SSR;
//    on hydration Nuxt treats that as "already fetched" and skips running the handler — so a hard
//    refresh shows no summary/incidents XHR, while client-side navigation runs the fetch.
const defaultSummary = (): StatusSummaryPayload => ({
  status: 'operational',
  message: 'All Systems Operational',
  services: [],
  environments: [],
})

const statusData = ref<StatusSummaryPayload>(defaultSummary())
const incidents = ref<IncidentDisplayItem[]>([])

onMounted(async () => {
  try {
    const [summary, inc] = await Promise.all([
      apiFetch<StatusSummaryPayload>('/api/public/status/summary'),
      apiFetch<IncidentDisplayItem[]>('/api/public/incidents?limit=5'),
    ])
    statusData.value = summary
    incidents.value = inc
  } catch {
    // Leave defaults; user still sees the banner.
  }
})
</script>

<style scoped>
.home-page { padding-top: 16px; }
.section-header { display: flex; justify-content: flex-end; gap: 8px; margin-bottom: 24px; font-size: 0.85rem; color: var(--color-text-secondary); }
.section-link { font-weight: 500; color: var(--color-primary); text-decoration: underline; text-underline-offset: 2px; }
.section-link:hover { color: var(--color-primary-hover); }
.status-section { margin-bottom: 40px; }
.section-title { font-size: 1.15rem; font-weight: 600; margin-bottom: 16px; }
.card-list { display: flex; flex-direction: column; gap: 12px; }
.incident-list { display: flex; flex-direction: column; gap: 12px; }
.incidents-hint { font-size: 0.85rem; color: var(--color-text-secondary); margin: -8px 0 12px; }
</style>
