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

const { data: statusData } = await useAsyncData('status-summary', () =>
  apiFetch<StatusSummaryPayload>('/api/public/status/summary'),
  {
    default: () => ({
      status: 'operational',
      message: 'All Systems Operational',
      services: [],
      environments: [],
    }),
  },
)

const { data: incidents } = await useAsyncData('recent-incidents', () =>
  apiFetch<IncidentDisplayItem[]>('/api/public/incidents?limit=5'),
  { default: () => [] },
)
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
