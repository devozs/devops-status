<template>
  <div>
    <h1 class="page-title">Admin Dashboard</h1>
    <div class="dashboard-cards">
      <NuxtLink to="/admin/services" class="dash-card">
        <span class="dash-card-title">Services</span>
        <span class="dash-card-desc">Manage monitored services</span>
      </NuxtLink>
      <NuxtLink to="/admin/environments" class="dash-card">
        <span class="dash-card-title">Environments</span>
        <span class="dash-card-desc">Manage environments and membership</span>
      </NuxtLink>
      <NuxtLink to="/admin/data-sources" class="dash-card">
        <span class="dash-card-title">Data Sources</span>
        <span class="dash-card-desc">Configure probe data sources</span>
      </NuxtLink>
      <NuxtLink to="/admin/bindings" class="dash-card">
        <span class="dash-card-title">Bindings</span>
        <span class="dash-card-desc">Map services/environments to probes</span>
      </NuxtLink>
      <NuxtLink to="/admin/k8s-clusters" class="dash-card">
        <span class="dash-card-title">K8s Clusters</span>
        <span class="dash-card-desc">Onboard and manage Kubernetes clusters</span>
      </NuxtLink>
      <NuxtLink to="/admin/incidents" class="dash-card">
        <span class="dash-card-title">Incidents</span>
        <span class="dash-card-desc">View and update incidents</span>
      </NuxtLink>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'admin' })
</script>

<style scoped>
.page-title {
  font-size: 1.5rem;
  font-weight: 700;
  margin-bottom: 24px;
}

.dashboard-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 16px;
}

.dash-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 20px;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  transition: box-shadow 0.15s;
}

.dash-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.dash-card-title {
  font-weight: 600;
  font-size: 1rem;
}

.dash-card-desc {
  font-size: 0.85rem;
  color: var(--color-text-secondary);
}
</style>
