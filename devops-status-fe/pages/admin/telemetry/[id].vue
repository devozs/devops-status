<template>
  <div>
    <div v-if="loadError" class="telemetry-editor-load-error">
      <p>{{ loadError }}</p>
      <NuxtLink to="/admin/telemetry" class="btn btn-sm">Back to telemetry</NuxtLink>
    </div>
    <TelemetryFormEditor
      v-else-if="ready && initial"
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

async function loadRow() {
  const id = typeof route.params.id === 'string' ? route.params.id.trim() : ''
  if (!id) {
    loadError.value = 'Missing telemetry id.'
    return
  }
  try {
    const row = await apiFetch<TelemetryRow>(`/api/admin/telemetry/${id}`)
    initial.value = {
      ...row,
      config_json: { ...(row.config_json || {}) },
      qos_thresholds: { ...(row.qos_thresholds || {}) },
    }
  } catch {
    loadError.value = 'Telemetry not found or you do not have access.'
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
  ready.value = false
  loadError.value = ''
  initial.value = null
  await loadMeta()
  await loadRow()
  if (!loadError.value) {
    ready.value = true
  }
})

watch(
  () => route.params.id,
  async () => {
    ready.value = false
    loadError.value = ''
    initial.value = null
    await loadRow()
    if (!loadError.value) {
      ready.value = true
    }
  },
)
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
