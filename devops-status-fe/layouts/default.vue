<script setup lang="ts">
const { backendStatus, startPolling } = useBackendHealth()
onMounted(() => startPolling())
</script>

<template>
  <div class="layout">
    <header class="site-header">
      <div class="container header-inner">
        <div class="logo-group">
          <NuxtLink to="/" class="logo">DevOps Status</NuxtLink>
          <BackendIndicator />
        </div>
        <nav class="header-nav">
          <NuxtLink to="/" class="nav-link">Status</NuxtLink>
          <NuxtLink to="/history" class="nav-link">History</NuxtLink>
        </nav>
      </div>
    </header>
    <main class="site-main">
      <slot />
    </main>
    <footer class="site-footer">
      <div class="container footer-inner">
        <span class="footer-text">&copy; {{ new Date().getFullYear() }} DevOps Status</span>
        <NuxtLink to="/admin" class="admin-link">Admin</NuxtLink>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.site-header {
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg);
  position: sticky;
  top: 0;
  z-index: 100;
}

.header-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 60px;
}

.logo-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.logo {
  font-size: 1.25rem;
  font-weight: 700;
}

.header-nav {
  display: flex;
  gap: 24px;
}

.nav-link {
  font-size: 0.9rem;
  color: var(--color-text-secondary);
  transition: color 0.15s;
}

.nav-link:hover,
.nav-link.router-link-active {
  color: var(--color-text);
}

.site-main {
  flex: 1;
  padding: 32px 0;
}

.site-footer {
  border-top: 1px solid var(--color-border);
  padding: 24px 0;
}

.footer-inner {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
}

.footer-text {
  color: var(--color-text-secondary);
  font-size: 0.85rem;
}

.admin-link {
  color: var(--color-text-secondary);
  font-size: 0.8rem;
  opacity: 0.5;
  transition: opacity 0.15s;
}

.admin-link:hover {
  opacity: 1;
}
</style>
