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
        <ServiceCard v-for="env in statusData.environments" :key="env.slug" :item="env" />
      </div>
    </section>

    <section v-if="statusData.services?.length" class="status-section">
      <h2 class="section-title">Services</h2>
      <div class="card-list">
        <ServiceCard v-for="svc in statusData.services" :key="svc.slug" :item="svc" />
      </div>
    </section>

    <section v-if="incidents?.length" class="status-section">
      <h2 class="section-title">Past Incidents</h2>
      <div class="incident-list">
        <div v-for="inc in incidents" :key="inc.id" class="incident-item">
          <div class="incident-header">
            <span class="incident-title" :class="'severity--' + inc.severity">{{ inc.title }}</span>
            <span class="incident-date">{{ formatDate(inc.started_at) }}</span>
          </div>
          <span class="incident-status">{{ inc.status }}</span>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
const { apiFetch } = useApi()

const { data: statusData } = await useAsyncData('status-summary', () =>
  apiFetch<any>('/api/public/status/summary'),
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
  apiFetch<any[]>('/api/public/incidents?limit=5'),
  { default: () => [] },
)

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}
</script>

<style scoped>
.home-page { padding-top: 16px; }
.section-header { display: flex; justify-content: flex-end; gap: 8px; margin-bottom: 24px; font-size: 0.85rem; color: var(--color-text-secondary); }
.section-link { font-weight: 500; text-decoration: underline; }
.status-section { margin-bottom: 40px; }
.section-title { font-size: 1.15rem; font-weight: 600; margin-bottom: 16px; }
.card-list { display: flex; flex-direction: column; gap: 12px; }
.incident-list { display: flex; flex-direction: column; gap: 12px; }
.incident-item { border-left: 3px solid var(--color-orange); padding: 12px 16px; background: var(--color-bg-secondary); border-radius: 0 var(--radius) var(--radius) 0; }
.incident-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
.incident-title { font-weight: 600; font-size: 0.9rem; }
.severity--major { color: var(--color-red); }
.severity--minor { color: var(--color-orange); }
.incident-date { font-size: 0.8rem; color: var(--color-text-secondary); }
.incident-status { font-size: 0.8rem; color: var(--color-text-secondary); text-transform: capitalize; }
</style>
