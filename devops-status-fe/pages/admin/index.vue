<template>
  <div>
    <h1 class="page-title">Admin Dashboard</h1>
    <div class="dashboard-cards">
      <AdminDashboardResourceCard
        to="/admin/services"
        title="Services"
        description="Manage monitored services"
        :loading="servicesLoading"
        :error="servicesError"
        :total="servicesTotal"
        :lines="services"
      />
      <AdminDashboardResourceCard
        to="/admin/environments"
        title="Environments"
        description="Manage environments and membership"
        :loading="environmentsLoading"
        :error="environmentsError"
        :total="environmentsTotal"
        :lines="environments"
      />
      <AdminDashboardResourceCard
        to="/admin/data-sources"
        title="Data Sources"
        description="Configure probe data sources"
        :loading="dataSourcesLoading"
        :error="dataSourcesError"
        :total="dataSourcesTotal"
        :lines="dataSources"
      />
      <AdminDashboardResourceCard
        to="/admin/bindings"
        title="Bindings"
        description="Map services/environments to probes"
        :loading="bindingsLoading"
        :error="bindingsError"
        :total="bindingsTotal"
        :lines="bindings"
        :summary="bindingsSummary || undefined"
      />
      <AdminDashboardResourceCard
        to="/admin/k8s-clusters"
        title="K8s Clusters"
        description="Onboard and manage Kubernetes clusters"
        :loading="k8sClustersLoading"
        :error="k8sClustersError"
        :total="k8sClustersTotal"
        :lines="k8sClusters"
      />
      <AdminDashboardResourceCard
        to="/admin/incidents"
        title="Incidents"
        description="View and update incidents"
        :loading="incidentsLoading"
        :error="incidentsError"
        :total="incidentsTotal"
        :lines="incidents"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'admin' })

const {
  services,
  servicesTotal,
  servicesError,
  servicesLoading,
  environments,
  environmentsTotal,
  environmentsError,
  environmentsLoading,
  dataSources,
  dataSourcesTotal,
  dataSourcesError,
  dataSourcesLoading,
  bindings,
  bindingsTotal,
  bindingsError,
  bindingsSummary,
  bindingsLoading,
  k8sClusters,
  k8sClustersTotal,
  k8sClustersError,
  k8sClustersLoading,
  incidents,
  incidentsTotal,
  incidentsError,
  incidentsLoading,
} = useAdminDashboardData()
</script>

<style scoped>
.page-title {
  font-size: 1.75rem;
  font-weight: 700;
  color: #1a202c;
  margin-bottom: 28px;
  letter-spacing: -0.02em;
}

.dashboard-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
  max-width: 1100px;
}

@media (max-width: 900px) {
  .dashboard-cards {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 520px) {
  .dashboard-cards {
    grid-template-columns: 1fr;
  }
}
</style>
