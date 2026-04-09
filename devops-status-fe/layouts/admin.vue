<template>
  <div class="admin-layout">
    <aside class="admin-sidebar">
      <div class="sidebar-header">
        <NuxtLink to="/admin" class="admin-logo">Admin</NuxtLink>
      </div>
      <nav class="sidebar-nav">
        <NuxtLink to="/admin" class="sidebar-link">Dashboard</NuxtLink>
        <NuxtLink to="/admin/services" class="sidebar-link">Services</NuxtLink>
        <NuxtLink to="/admin/environments" class="sidebar-link">Environments</NuxtLink>
        <NuxtLink to="/admin/data-sources" class="sidebar-link">Data Sources</NuxtLink>
        <NuxtLink to="/admin/bindings" class="sidebar-link">Bindings</NuxtLink>
        <NuxtLink to="/admin/k8s-clusters" class="sidebar-link">K8s Clusters</NuxtLink>
        <NuxtLink to="/admin/incidents" class="sidebar-link">Incidents</NuxtLink>
        <div class="sidebar-divider" />
        <NuxtLink to="/" class="sidebar-link sidebar-link--back">Back to Status</NuxtLink>
      </nav>
    </aside>
    <main class="admin-main">
      <slot />
    </main>
  </div>
</template>

<style scoped>
.admin-layout {
  min-height: 100vh;
  display: flex;
}

.admin-sidebar {
  width: 220px;
  background: #1f2937;
  color: #f9fafb;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar-header {
  padding: 20px;
  border-bottom: 1px solid #374151;
}

.admin-logo {
  font-size: 1.1rem;
  font-weight: 700;
  color: #f9fafb;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  padding: 12px 0;
}

.sidebar-link {
  padding: 10px 20px;
  font-size: 0.9rem;
  color: #d1d5db;
  transition: background 0.15s, color 0.15s;
}

.sidebar-link:hover,
.sidebar-link.router-link-active {
  background: #374151;
  color: #f9fafb;
}

.sidebar-link--back {
  color: #9ca3af;
  font-size: 0.8rem;
}

.sidebar-divider {
  height: 1px;
  background: #374151;
  margin: 12px 0;
}

.admin-main {
  flex: 1;
  padding: 32px;
  background: var(--color-bg-secondary);
}
</style>
