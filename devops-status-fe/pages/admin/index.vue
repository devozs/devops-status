<template>
  <div>
    <h1 class="page-title page-title--dashboard">Admin dashboard</h1>
    <p class="dashboard-lead">
      Configure telemetry, link it to services and environments, and onboard Kubernetes clusters—similar to how a cloud
      console groups compute, storage, and observability.
    </p>
    <div class="dashboard-cards">
      <AdminDashboardResourceCard
        to="/admin/services"
        title="Services"
        description="Manage monitored services"
        :icon="Cpu"
        :loading="servicesLoading"
        :error="servicesError"
        :total="servicesTotal"
        :lines="services"
      />
      <AdminDashboardResourceCard
        to="/admin/environments"
        title="Environments"
        description="Manage environments and Kubernetes telemetry links"
        :icon="Boxes"
        :loading="environmentsLoading"
        :error="environmentsError"
        :total="environmentsTotal"
        :lines="environments"
      />
      <AdminDashboardResourceCard
        to="/admin/telemetry"
        title="Telemetry"
        description="Configure probe telemetry"
        :icon="Activity"
        :loading="telemetryLoading"
        :error="telemetryError"
        :total="telemetryTotal"
        :lines="telemetry"
      />
      <AdminDashboardResourceCard
        to="/admin/environments"
        title="Telemetry links"
        description="Service and environment probe links"
        :icon="Link2"
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
        :icon="Container"
        :loading="k8sClustersLoading"
        :error="k8sClustersError"
        :total="k8sClustersTotal"
        :lines="k8sClusters"
      />
      <AdminDashboardResourceCard
        to="/admin/incidents"
        title="Incidents"
        description="View and update incidents"
        :icon="AlertTriangle"
        :loading="incidentsLoading"
        :error="incidentsError"
        :total="incidentsTotal"
        :lines="incidents"
      />
    </div>

    <section class="dashboard-help" aria-label="Help topics">
      <h2 class="dashboard-help-heading">Help</h2>
      <div class="dashboard-help-stack">
        <article class="help-card help-card--a">
          <div class="help-card-title-row">
            <BookOpen class="help-card-icon" :size="18" :stroke-width="2" aria-hidden="true" />
            <h3 class="help-card-title">Telemetry</h3>
          </div>
          <p class="help-card-text">
            Each telemetry row defines how we reach a URL or API. Verify from the editor to confirm connectivity and QoS latency before
            linking to a service or environment.
          </p>
        </article>
        <article class="help-card help-card--b">
          <div class="help-card-title-row">
            <BookOpen class="help-card-icon" :size="18" :stroke-width="2" aria-hidden="true" />
            <h3 class="help-card-title">Kubernetes</h3>
          </div>
          <p class="help-card-text">
            Onboarding runs a Job in your cluster to register the control plane. Use the backend public URL for in-cluster
            traffic; use <strong>Revoke</strong> to stop API calls without deleting RBAC in the cluster.
          </p>
        </article>
        <article class="help-card help-card--c">
          <div class="help-card-title-row">
            <BookOpen class="help-card-icon" :size="18" :stroke-width="2" aria-hidden="true" />
            <h3 class="help-card-title">Incidents</h3>
          </div>
          <p class="help-card-text">
            Incidents drive the public status page. Keep statuses updated while you investigate so subscribers see accurate
            messaging on the Status and History views.
          </p>
        </article>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { Activity, AlertTriangle, BookOpen, Boxes, Container, Cpu, Link2 } from 'lucide-vue-next'

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
  telemetry,
  telemetryTotal,
  telemetryError,
  telemetryLoading,
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
.page-title--dashboard {
  font-size: 1.75rem;
  font-weight: 700;
  margin-bottom: 8px;
}

.dashboard-lead {
  font-size: 0.9rem;
  color: var(--color-text-secondary);
  line-height: 1.55;
  max-width: 720px;
  margin-bottom: 28px;
}

.dashboard-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
  max-width: 1100px;
}

.dashboard-help {
  margin-top: 40px;
  padding-top: 28px;
  border-top: 1px solid var(--color-border-strong);
  max-width: 1100px;
}

.dashboard-help-heading {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-secondary);
  margin: 0 0 16px;
}

.dashboard-help-stack {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.help-card {
  padding: 18px 20px;
  padding-top: 16px;
  background: var(--color-bg);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius);
  box-shadow: var(--shadow-card);
  border-top-width: 3px;
  border-top-style: solid;
}

.help-card--a {
  border-top-color: #84cc16;
}

.help-card--b {
  border-top-color: #14b8a6;
}

.help-card--c {
  border-top-color: #38bdf8;
}

.help-card-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.help-card-icon {
  flex-shrink: 0;
  color: var(--color-text-secondary);
}

.help-card-title {
  font-size: 0.9rem;
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.help-card-text {
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--color-text-secondary);
  margin: 0;
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
