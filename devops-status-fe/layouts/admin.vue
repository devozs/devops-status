<script setup lang="ts">
const { apiFetch } = useApi()
const router = useRouter()

async function logout() {
  try {
    await apiFetch('/api/admin/auth/logout', { method: 'POST' })
  } catch {}
  await router.push('/admin/login')
}
</script>

<template>
  <div class="admin-layout">
    <aside class="admin-sidebar">
      <div class="sidebar-header">
        <NuxtLink
          to="/admin"
          class="admin-logo"
          title="DevOpsStatus"
        >
          <span class="admin-logo-mark">DOS</span>
          <span class="admin-logo-sub">Admin</span>
        </NuxtLink>
        <BackendIndicator />
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
        <button class="sidebar-link sidebar-link--logout" @click="logout">Logout</button>
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
  width: 260px;
  background: #1a1f2e;
  color: #f9fafb;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar-header {
  padding: 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  align-items: center;
  gap: 10px;
}

.admin-logo {
  display: flex;
  flex-direction: column;
  line-height: 1.15;
  color: #f9fafb;
  flex: 1;
  min-width: 0;
}

.admin-logo-mark {
  font-size: 1.15rem;
  font-weight: 800;
  letter-spacing: -0.02em;
}

.admin-logo-sub {
  font-size: 0.7rem;
  font-weight: 600;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  padding: 12px 0;
}

.sidebar-link {
  padding: 10px 20px;
  font-size: 0.9rem;
  color: #cbd5e1;
  transition: background 0.15s, color 0.15s;
}

.sidebar-link:hover,
.sidebar-link.router-link-active {
  background: #2d3748;
  color: #f9fafb;
}

.sidebar-link--back {
  color: #9ca3af;
  font-size: 0.8rem;
}

.sidebar-link--logout {
  background: none;
  border: none;
  text-align: left;
  cursor: pointer;
  color: #ef4444;
  font-size: 0.8rem;
}

.sidebar-link--logout:hover {
  background: #2d3748;
  color: #f87171;
}

.sidebar-divider {
  height: 1px;
  background: rgba(255, 255, 255, 0.08);
  margin: 12px 0;
}

.admin-main {
  flex: 1;
  padding: 32px 40px;
  background: #f9fafb;
}
</style>
