<script setup lang="ts">
import {
  LayoutDashboard,
  Server,
  Boxes,
  Activity,
  Network,
  ChevronDown,
  ChevronUp,
  AlertTriangle,
  Download,
  LogOut,
  ArrowLeft,
} from 'lucide-vue-next'

const { apiFetch } = useApi()
const router = useRouter()
const route = useRoute()

const routeTitles: Record<string, string> = {
  '/admin': 'Dashboard',
  '/admin/services': 'Services',
  '/admin/environments': 'Environments',
  '/admin/telemetry': 'Telemetry',
  '/admin/service-providers': 'Service providers',
  '/admin/k8s-clusters': 'Kubernetes clusters',
  '/admin/incidents': 'Incidents',
  '/admin/export': 'Export',
}

const breadcrumbLabel = computed(() => {
  const p = route.path.replace(/\/$/, '') || '/'
  return routeTitles[p] ?? 'Admin'
})

function pathActive(target: string) {
  const p = route.path.replace(/\/$/, '') || '/'
  const t = target.replace(/\/$/, '') || '/'
  return p === t
}

async function logout() {
  try {
    await apiFetch('/api/admin/auth/logout', { method: 'POST' })
  } catch {}
  await router.push('/admin/login')
}

const navOverview = [{ to: '/admin', label: 'Dashboard', icon: LayoutDashboard }]

const navResources = [
  { to: '/admin/services', label: 'Services', icon: Server },
  { to: '/admin/environments', label: 'Environments', icon: Boxes },
  { to: '/admin/telemetry', label: 'Telemetry', icon: Activity },
]

const providerChildPaths = ['/admin/service-providers', '/admin/k8s-clusters'] as const

const navProvidersChildren = [
  { to: '/admin/service-providers', label: 'Service Providers' },
  { to: '/admin/k8s-clusters', label: 'K8s Clusters' },
]

const providersOpen = ref(false)

function providersGroupActive(): boolean {
  return providerChildPaths.some((p) => pathActive(p))
}

watch(
  () => route.path,
  () => {
    if (providersGroupActive()) providersOpen.value = true
  },
  { immediate: true },
)

const navOperations = [
  { to: '/admin/incidents', label: 'Incidents', icon: AlertTriangle },
  { to: '/admin/export', label: 'Export', icon: Download },
]
</script>

<template>
  <div class="admin-layout">
    <aside class="admin-sidebar">
      <div class="sidebar-header">
        <NuxtLink to="/admin" class="admin-logo" title="DevOps Status Admin">
          <span class="admin-logo-mark">DEVOPS STATUS</span>
          <span class="admin-logo-sub">Console</span>
        </NuxtLink>
        <BackendIndicator />
      </div>
      <nav class="sidebar-nav" aria-label="Admin navigation">
        <p class="sidebar-group-label">Overview</p>
        <NuxtLink
          v-for="item in navOverview"
          :key="item.to"
          :to="item.to"
          class="sidebar-link"
          :class="{ 'sidebar-link--active': pathActive(item.to) }"
        >
          <span class="sidebar-nav-marker" aria-hidden="true">
            <span v-if="pathActive(item.to)" class="sidebar-active-dot" />
          </span>
          <component :is="item.icon" class="sidebar-icon" :size="18" :stroke-width="1.75" />
          {{ item.label }}
        </NuxtLink>

        <p class="sidebar-group-label">Resources</p>
        <NuxtLink
          v-for="item in navResources"
          :key="item.to"
          :to="item.to"
          class="sidebar-link"
          :class="{ 'sidebar-link--active': pathActive(item.to) }"
        >
          <span class="sidebar-nav-marker" aria-hidden="true">
            <span v-if="pathActive(item.to)" class="sidebar-active-dot" />
          </span>
          <component :is="item.icon" class="sidebar-icon" :size="18" :stroke-width="1.75" />
          {{ item.label }}
        </NuxtLink>

        <div class="sidebar-nav-group">
          <button
            type="button"
            class="sidebar-disclosure"
            :class="{ 'sidebar-disclosure--active': providersGroupActive() }"
            :aria-expanded="providersOpen"
            aria-controls="sidebar-providers-subnav"
            @click="providersOpen = !providersOpen"
          >
            <span class="sidebar-nav-marker" aria-hidden="true">
              <span v-if="providersGroupActive() && !providersOpen" class="sidebar-active-dot" />
            </span>
            <Network class="sidebar-icon" :size="18" :stroke-width="1.75" />
            <span class="sidebar-disclosure-label">Providers</span>
            <ChevronDown v-if="!providersOpen" class="sidebar-chevron" :size="18" :stroke-width="1.75" aria-hidden="true" />
            <ChevronUp v-else class="sidebar-chevron" :size="18" :stroke-width="1.75" aria-hidden="true" />
          </button>
          <div
            v-show="providersOpen"
            id="sidebar-providers-subnav"
            class="sidebar-subnav"
            role="region"
            aria-label="Providers navigation"
          >
            <NuxtLink
              v-for="child in navProvidersChildren"
              :key="child.to"
              :to="child.to"
              class="sidebar-sublink"
              :class="{ 'sidebar-sublink--active': pathActive(child.to) }"
            >
              <span class="sidebar-nav-marker sidebar-nav-marker--sub" aria-hidden="true">
                <span v-if="pathActive(child.to)" class="sidebar-active-dot" />
              </span>
              {{ child.label }}
            </NuxtLink>
          </div>
        </div>

        <p class="sidebar-group-label">Operations</p>
        <NuxtLink
          v-for="item in navOperations"
          :key="item.to"
          :to="item.to"
          class="sidebar-link"
          :class="{ 'sidebar-link--active': pathActive(item.to) }"
        >
          <span class="sidebar-nav-marker" aria-hidden="true">
            <span v-if="pathActive(item.to)" class="sidebar-active-dot" />
          </span>
          <component :is="item.icon" class="sidebar-icon" :size="18" :stroke-width="1.75" />
          {{ item.label }}
        </NuxtLink>

        <div class="sidebar-divider" />

        <p class="sidebar-group-label">Account</p>
        <NuxtLink to="/" class="sidebar-link sidebar-link--muted">
          <span class="sidebar-nav-marker" aria-hidden="true" />
          <ArrowLeft class="sidebar-icon" :size="18" :stroke-width="1.75" />
          Back to status
        </NuxtLink>
        <button type="button" class="sidebar-link sidebar-link--logout" @click="logout">
          <span class="sidebar-nav-marker" aria-hidden="true" />
          <LogOut class="sidebar-icon" :size="18" :stroke-width="1.75" />
          Logout
        </button>
      </nav>
    </aside>
    <div class="admin-workspace">
      <header class="admin-topbar">
        <span class="admin-breadcrumb">
          <span class="admin-breadcrumb-root">Admin</span>
          <span class="admin-breadcrumb-sep" aria-hidden="true">/</span>
          <span class="admin-breadcrumb-current">{{ breadcrumbLabel }}</span>
        </span>
      </header>
      <main class="admin-main">
        <slot />
      </main>
    </div>
  </div>
</template>

<style>
@import '~/assets/css/admin-shared.css';
</style>

<style scoped>
.admin-layout {
  min-height: 100vh;
  display: flex;
}

.admin-sidebar {
  width: 260px;
  background: var(--admin-sidebar-bg);
  color: var(--admin-sidebar-text-active);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  border-right: 1px solid var(--admin-sidebar-border);
}

.sidebar-header {
  padding: 20px 16px;
  border-bottom: 1px solid var(--admin-sidebar-border);
  display: flex;
  align-items: center;
  gap: 10px;
}

.admin-logo {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
  color: var(--admin-sidebar-text-active);
  flex: 1;
  min-width: 0;
}

.admin-logo-mark {
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.admin-logo-sub {
  font-size: 0.68rem;
  font-weight: 500;
  color: var(--admin-sidebar-text);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  margin-top: 2px;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  padding: 8px 0 16px;
  flex: 1;
  overflow-y: auto;
}

.sidebar-group-label {
  font-size: 0.62rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: rgba(255, 255, 255, 0.38);
  padding: 16px 20px 6px;
  margin: 0;
}

.sidebar-group-label:first-of-type {
  padding-top: 8px;
}

.sidebar-nav-marker {
  width: 12px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.sidebar-active-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--admin-nav-active-dot);
  box-shadow: 0 0 8px var(--admin-nav-active-dot-glow);
}

.sidebar-link {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 12px;
  margin: 2px 8px;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--admin-sidebar-text);
  border-radius: 8px;
  transition: background 0.15s, color 0.15s;
}

.sidebar-link:hover {
  background: var(--admin-sidebar-hover-bg);
  color: var(--admin-sidebar-text-active);
}

.sidebar-link--active {
  background: var(--admin-sidebar-active-bg);
  color: var(--admin-sidebar-text-active);
  font-weight: 600;
}

.sidebar-icon {
  flex-shrink: 0;
  opacity: 0.9;
}

.sidebar-link--muted {
  font-size: 0.8125rem;
  font-weight: 500;
}

.sidebar-link--logout {
  background: none;
  border: none;
  width: calc(100% - 16px);
  cursor: pointer;
  text-align: left;
  font: inherit;
  color: #f87171;
}

.sidebar-link--logout:hover {
  background: var(--admin-sidebar-hover-bg);
  color: #fca5a5;
}

.sidebar-divider {
  height: 1px;
  background: var(--admin-sidebar-border);
  margin: 12px 16px;
}

.admin-workspace {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: var(--admin-main-bg);
}

.admin-topbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  min-height: 52px;
  padding: 0 40px;
  background: var(--color-bg);
  border-bottom: 1px solid var(--color-border-strong);
}

.admin-breadcrumb {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}

.admin-breadcrumb-root {
  font-weight: 500;
}

.admin-breadcrumb-sep {
  margin: 0 8px;
  color: var(--color-border-strong);
}

.admin-breadcrumb-current {
  font-weight: 600;
  color: var(--color-text);
}

.admin-main {
  flex: 1;
  padding: 28px 40px 40px;
  background: var(--admin-main-bg);
}

.sidebar-nav-group {
  margin: 2px 8px 0;
}

.sidebar-disclosure {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 9px 12px;
  margin: 0;
  font-size: 0.875rem;
  font-weight: 500;
  font-family: inherit;
  color: var(--admin-sidebar-text);
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s, color 0.15s;
}

.sidebar-disclosure:hover {
  background: var(--admin-sidebar-hover-bg);
  color: var(--admin-sidebar-text-active);
}

.sidebar-disclosure--active {
  color: var(--admin-sidebar-text-active);
  font-weight: 600;
}

.sidebar-disclosure-label {
  flex: 1;
  min-width: 0;
}

.sidebar-chevron {
  flex-shrink: 0;
  opacity: 0.75;
  color: var(--admin-sidebar-text);
}

.sidebar-disclosure:hover .sidebar-chevron {
  color: var(--admin-sidebar-text-active);
  opacity: 0.95;
}

.sidebar-subnav {
  display: flex;
  flex-direction: column;
  padding: 2px 0 4px;
}

.sidebar-sublink {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 12px 7px 0;
  margin-left: 38px;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--admin-sidebar-text);
  border-radius: 8px;
  text-decoration: none;
  transition: background 0.15s, color 0.15s;
}

.sidebar-sublink:hover {
  background: var(--admin-sidebar-hover-bg);
  color: var(--admin-sidebar-text-active);
}

.sidebar-sublink--active {
  background: var(--admin-sidebar-active-bg);
  color: var(--admin-sidebar-text-active);
  font-weight: 600;
}

.sidebar-nav-marker--sub {
  width: 10px;
}
</style>
