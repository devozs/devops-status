<template>
  <div>
    <div v-if="loadError" class="telemetry-editor-load-error">
      <p>{{ loadError }}</p>
      <NuxtLink to="/admin/telemetry" class="btn btn-sm">Back to telemetry</NuxtLink>
    </div>
    <TelemetryFormEditor
      v-else-if="ready"
      ref="editorRef"
      :initial="initial"
      :environments="environments"
      :clusters="clusters"
      :api-fetch="apiFetch"
      @saved="onSaved"
    />
  </div>
</template>

<script setup lang="ts">
import type { TelemetryFormInitial, TelemetryK8sClusterRow, TelemetryRow } from '~/types/telemetry'
import { suggestDuplicateName } from '~/utils/telemetryDuplicateName'

definePageMeta({ layout: 'admin' })

const { apiFetch } = useApi()
const route = useRoute()

const environments = ref<{ id: string; name: string }[]>([])
const clusters = ref<TelemetryK8sClusterRow[]>([])
const initial = ref<TelemetryFormInitial | null>(null)
const ready = ref(false)
const loadError = ref('')
const editorRef = ref<{
  isTelemetryFormDirty: () => boolean
  consumeNextUnsavedBypass: () => boolean
  promptDiscardThenNavigate: (perform: () => void) => void
} | null>(null)

async function loadMeta() {
  try {
    environments.value = await apiFetch<{ id: string; name: string }[]>('/api/admin/environments')
  } catch {
    environments.value = []
  }
  try {
    clusters.value = (await apiFetch<TelemetryK8sClusterRow[]>('/api/admin/k8s-clusters')) || []
  } catch {
    clusters.value = []
  }
}

async function resolveInitial() {
  const fromId = typeof route.query.from === 'string' ? route.query.from.trim() : ''
  if (!fromId) {
    initial.value = null
    return
  }
  try {
    const [row, all] = await Promise.all([
      apiFetch<TelemetryRow>(`/api/admin/telemetry/${fromId}`),
      apiFetch<TelemetryRow[]>('/api/admin/telemetry'),
    ])
    const names = (all || []).map((t) => t.name)
    initial.value = {
      name: suggestDuplicateName(row.name, names),
      adapter: row.adapter,
      config_json: { ...(row.config_json || {}) },
      qos_thresholds: { ...(row.qos_thresholds || {}) },
      execution_target: row.execution_target,
      timeout_ms: row.timeout_ms,
      retries: row.retries ?? 1,
    }
  } catch {
    loadError.value = 'Could not load telemetry to duplicate. Check the link or go back to the list.'
  }
}

async function onSaved() {
  await navigateTo('/admin/telemetry')
}

onBeforeRouteLeave((to, _from, next) => {
  const ed = editorRef.value
  if (!ed) {
    next()
    return
  }
  if (ed.consumeNextUnsavedBypass()) {
    next()
    return
  }
  if (!ed.isTelemetryFormDirty()) {
    next()
    return
  }
  next(false)
  ed.promptDiscardThenNavigate(() => void navigateTo(to.fullPath))
})

onMounted(async () => {
  await loadMeta()
  await resolveInitial()
  if (!loadError.value) {
    ready.value = true
  }
})
</script>

<style scoped>
.telemetry-editor-load-error {
  max-width: 480px;
  padding: 24px;
  border-radius: var(--radius);
  border: 1px solid var(--color-border-strong);
  background: var(--color-bg);
}
.telemetry-editor-load-error p {
  margin: 0 0 16px;
  color: var(--color-text);
}
</style>
