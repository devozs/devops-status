<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-card modal-card--lg">
      <h2 class="modal-title">{{ title }}</h2>
      <form class="modal-form" @submit.prevent="submit">
        <h3 class="modal-section-title">General</h3>
        <div class="form-group form-group--name">
          <label class="form-label">Name</label>
          <div class="name-field-row">
            <input
              v-model="name"
              class="form-input"
              :class="nameFieldClass"
              required
              minlength="2"
              autocomplete="off"
            />
            <span v-if="nameAvail === 'loading'" class="name-status name-status--loading">Checking…</span>
            <span
              v-else-if="nameTrimmed.length >= 3 && nameAvail === 'available'"
              class="name-status name-status--ok"
            >Available</span>
            <span
              v-else-if="nameTrimmed.length >= 3 && nameAvail === 'taken'"
              class="name-status name-status--bad"
            >In use</span>
            <span
              v-else-if="nameTrimmed.length >= 3 && nameAvail === 'error'"
              class="name-status name-status--err"
            >Check failed</span>
          </div>
          <p v-if="nameTrimmed.length > 0 && nameTrimmed.length < 3" class="hint name-avail-hint">
            Type at least 3 characters to check availability with the server.
          </p>
        </div>
        <div class="form-group">
          <label class="form-label">Display name</label>
          <input v-model="displayName" class="form-input" maxlength="255" autocomplete="off" placeholder="e.g. Liveness" />
          <p class="hint">
            Optional. Shown when this probe is listed under an environment or service. <strong>Name</strong> stays the unique id.
          </p>
        </div>
        <div class="form-group">
          <label class="form-label">Adapter</label>
          <select v-model="adapter" class="form-input">
            <option value="http">HTTP</option>
            <option value="prometheus">Prometheus</option>
            <option value="kubernetes">Kubernetes</option>
            <option value="cli">CLI</option>
          </select>
        </div>

        <!-- HTTP -->
        <template v-if="adapter === 'http'">
          <div class="form-group">
            <label class="form-label">Source</label>
            <select v-model="http.service_provider_id" class="form-input" required>
              <option value="">Select TCP service provider…</option>
              <option v-for="p in tcpProviders" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">Path</label>
            <input v-model="http.path" class="form-input mono" placeholder="/health" required />
          </div>
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">Method</label>
              <select v-model="http.method" class="form-input">
                <option v-for="m in HTTP_METHODS" :key="m" :value="m">{{ m }}</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">Expected status</label>
              <input v-model.number="http.expected_status" type="number" class="form-input" />
            </div>
          </div>
          <div v-if="http.method === 'POST' || http.method === 'PUT'" class="form-group">
            <label class="form-label">Body</label>
            <textarea v-model="http.body" class="form-input" rows="2" />
          </div>
        </template>

        <!-- Prometheus -->
        <template v-else-if="adapter === 'prometheus'">
          <div class="form-group">
            <label class="form-label">Source</label>
            <select v-model="prom.service_provider_id" class="form-input" required>
              <option value="">Select Prometheus service provider…</option>
              <option v-for="p in promProviders" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </div>
          <p
            v-if="selectedPromProvider && selectedPromProvider.has_prometheus_credentials === false"
            class="hint"
          >
            This provider has no stored credentials — basic/bearer probes may fail until you add them under Service providers.
          </p>
          <div class="form-group">
            <label class="form-label">PromQL</label>
            <textarea v-model="prom.query" class="form-input mono" rows="3" required />
          </div>
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">Threshold</label>
              <input v-model.number="prom.threshold" type="number" step="any" class="form-input" />
            </div>
            <div class="form-group">
              <label class="form-label">Operator</label>
              <select v-model="prom.operator" class="form-input">
                <option v-for="o in PROM_OPERATORS" :key="o" :value="o">{{ o }}</option>
              </select>
            </div>
          </div>
        </template>

        <!-- Kubernetes -->
        <template v-else-if="adapter === 'kubernetes'">
          <div class="form-group">
            <label class="form-label">K8s version</label>
            <select v-model="k8s.k8s_version" class="form-input" required>
              <option v-for="v in K8S_VERSIONS" :key="v" :value="v">{{ v }}</option>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">Cluster</label>
            <select v-model="k8s.cluster_id" class="form-input">
              <option value="">Select cluster…</option>
              <option v-for="c in k8sFilteredClusters" :key="c.id" :value="c.id">{{ clusterLabel(c) }}</option>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">Check type</label>
            <select v-model="k8s.check_type" class="form-input">
              <option v-for="c in K8S_CHECK_TYPES" :key="c" :value="c">{{ c }}</option>
            </select>
          </div>
          <div v-if="k8s.check_type !== 'api_health' && k8s.check_type !== 'node_status'" class="form-group">
            <label class="form-label">Namespace</label>
            <input v-model="k8s.namespace" class="form-input" />
          </div>
          <div v-if="k8s.check_type === 'deployment_ready'" class="form-group">
            <label class="form-label">Deployment name</label>
            <input v-model="k8s.resource_name" class="form-input" required />
          </div>
          <div v-if="k8s.check_type === 'pod_status' || k8s.check_type === 'node_status'" class="form-group">
            <label class="form-label">Label selector</label>
            <input v-model="k8s.label_selector" class="form-input" />
          </div>
        </template>

        <!-- CLI -->
        <template v-else-if="adapter === 'cli'">
          <div class="form-group">
            <label class="form-label">Execution</label>
            <div class="radio-col">
              <label><input v-model="cli.execution_target" type="radio" value="backend" /> Backend</label>
              <label><input v-model="cli.execution_target" type="radio" value="k8s_cluster" /> Kubernetes Job</label>
            </div>
            <p class="hint">
              <strong>Backend</strong> runs the probe in a disposable Linux container via <code class="mono">docker run</code> (see server
              <code class="mono">CLI_BACKEND_EXECUTOR</code> / <code class="mono">CLI_DOCKER_NETWORK</code>), or as a Job on a
              <strong>designated cluster</strong> when <code class="mono">CLI_BACKEND_EXECUTOR=k8s</code>.
              <strong>Kubernetes Job</strong> uses the cluster you select below; both paths share the same shell + image presets.
            </p>
          </div>
          <div class="form-group">
            <label class="form-label">Runner image</label>
            <select v-model="cli.runner" class="form-input">
              <option v-for="p in CLI_RUNNER_PRESETS" :key="p" :value="p">
                {{ p === 'alpine' ? 'Alpine (stable 3.x)' : 'Ubuntu 24.04' }}
              </option>
            </select>
            <p class="hint">Mapped on the server to <code class="mono">CLI_RUNNER_IMAGE_ALPINE</code> or <code class="mono">CLI_RUNNER_IMAGE_UBUNTU_24</code>.</p>
          </div>
          <template v-if="cli.execution_target === 'k8s_cluster'">
            <div class="form-group">
              <label class="form-label">K8s version</label>
              <select v-model="cli.k8s_version" class="form-input" required>
                <option v-for="v in K8S_VERSIONS" :key="v" :value="v">{{ v }}</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">Cluster</label>
              <select v-model="cli.cluster_id" class="form-input" required>
                <option value="">Select cluster…</option>
                <option v-for="c in cliFilteredClusters" :key="c.id" :value="c.id">{{ clusterLabel(c) }}</option>
              </select>
            </div>
          </template>
          <div class="form-group">
            <label class="form-label">Container prep (shell)</label>
            <textarea
              v-model="cli.container_prep"
              class="form-input mono"
              rows="4"
              placeholder="Optional. Runs before the probe in the same shell (e.g. apk add --no-cache curl)."
            />
            <p class="hint">
              Runs first in the runner container (subshell). Install tools here (e.g. <code class="mono">apk add</code> or <code class="mono">apt-get</code>). Prep <strong>stdout</strong> is sent to <strong>stderr</strong> so you still see it when verifying (combined log / stderr), but it is <strong>not</strong> used for QoS or parsed probe values — only the probe command or probe shell writes to stdout for that.
            </p>
          </div>

          <template v-if="cliFormMode === 'legacy'">
            <p class="hint hint-legacy">
              Legacy <strong>three-command</strong> QoS (allowlisted binaries only). Prefer <strong>metric shell</strong> for new probes.
            </p>
            <p class="hint">
              <button type="button" class="btn btn-sm" @click="switchCliToMetric">Switch to metric shell QoS</button>
            </p>
            <h4 class="sub">CLI QoS (three commands)</h4>
            <p class="hint">
              <strong>Allowlisted binaries:</strong> <code class="mono">curl</code>, <code class="mono">kubectl</code>, <code class="mono">go</code>, <code class="mono">echo</code>.
              Put <strong>only the program name</strong> in each cmd field (e.g. <code class="mono">echo</code>), and arguments in args (e.g. <code class="mono">hi</code>) — not <code class="mono">echo "hi"</code> as a single cmd.
              The binary must exist in the image or be installed in <strong>Container prep</strong>.
            </p>
            <div class="qos-band qos-band--green" :class="{ 'qos-band--blink': verifyBlinkTier === 'green' }">
              <div class="form-group"><label>Green cmd</label><input v-model="cq.green_command" class="form-input" required /></div>
              <div class="form-group"><label>Green args</label><input v-model="cqGreenArgs" class="form-input" /></div>
            </div>
            <div class="qos-band qos-band--yellow" :class="{ 'qos-band--blink': verifyBlinkTier === 'yellow' }">
              <div class="form-group"><label>Yellow cmd</label><input v-model="cq.yellow_command" class="form-input" required /></div>
              <div class="form-group"><label>Yellow args</label><input v-model="cqYellowArgs" class="form-input" /></div>
            </div>
            <div class="qos-band qos-band--red" :class="{ 'qos-band--blink': verifyBlinkTier === 'red' }">
              <div class="form-group"><label>Red cmd</label><input v-model="cq.red_command" class="form-input" required /></div>
              <div class="form-group"><label>Red args</label><input v-model="cqRedArgs" class="form-input" /></div>
            </div>
            <label class="chk"><input v-model="cli.parse_json" type="checkbox" /> Parse JSON stdout</label>
            <p class="hint">Execution target and cluster apply to all three commands.</p>
          </template>

          <template v-else>
            <h4 class="sub">Probe shell</h4>
            <p class="hint">
              Runs after container prep. The server does <strong>not</strong> apply the allowlist to this body. Only this script’s <strong>stdout</strong> is used for parsing and QoS (prep output is on stderr for debugging). Use exit code 0 (default) for success; for <strong>numeric</strong> QoS the first number in this stdout (or JSON <code class="mono">value</code> when parse JSON is on) is compared to thresholds; for <strong>text</strong> QoS the whole trimmed stdout is compared to your reference strings.
            </p>
            <div class="form-group">
              <label class="form-label">Shell script</label>
              <textarea
                v-model="cli.cli_shell"
                class="form-input mono"
                rows="10"
                placeholder="e.g. echo 42"
                :required="cliFormMode === 'metric'"
              />
            </div>
            <label class="chk"><input v-model="cli.parse_json" type="checkbox" /> Parse JSON stdout (numeric mode: use number field <code class="mono">value</code>)</label>

            <h4 class="sub">QoS thresholds</h4>
            <div class="form-group">
              <label class="form-label">Value type</label>
              <select v-model="cm.value_kind" class="form-input">
                <option value="number">Numeric</option>
                <option value="text">Text (trimmed stdout)</option>
              </select>
            </div>

            <template v-if="cm.value_kind === 'number'">
              <div
                class="form-row qos-band qos-band--green"
                :class="{ 'qos-band--blink': verifyBlinkTier === 'green' }"
              >
                <div class="form-group">
                  <label>Green op</label>
                  <select v-model="cm.green_operator" class="form-input">
                    <option v-for="o in PROM_OPERATORS" :key="'cg'+o" :value="o">{{ o }}</option>
                  </select>
                </div>
                <div class="form-group">
                  <label>Green value</label>
                  <input v-model.number="cm.green_threshold" type="number" step="any" class="form-input" />
                </div>
              </div>
              <div
                class="form-row qos-band qos-band--yellow"
                :class="{ 'qos-band--blink': verifyBlinkTier === 'yellow' }"
              >
                <div class="form-group">
                  <label>Yellow op</label>
                  <select v-model="cm.yellow_operator" class="form-input">
                    <option v-for="o in PROM_OPERATORS" :key="'cy'+o" :value="o">{{ o }}</option>
                  </select>
                </div>
                <div class="form-group">
                  <label>Yellow value</label>
                  <input v-model.number="cm.yellow_threshold" type="number" step="any" class="form-input" />
                </div>
              </div>
              <div class="form-row qos-band qos-band--red">
                <div class="form-group hint-only">
                  <span>Red band</span>
                  <p class="hint">Neither green nor yellow match → red (same as Prometheus).</p>
                </div>
              </div>
            </template>

            <template v-else>
              <div
                class="form-row qos-band qos-band--green"
                :class="{ 'qos-band--blink': verifyBlinkTier === 'green' }"
              >
                <div class="form-group">
                  <label>Green op</label>
                  <select v-model="cm.green_operator" class="form-input">
                    <option v-for="o in CLI_TEXT_OPERATORS" :key="'tg'+o" :value="o">{{ o }}</option>
                  </select>
                </div>
                <div class="form-group">
                  <label>Green reference</label>
                  <input v-model="cm.green_text" class="form-input mono" />
                </div>
              </div>
              <div
                class="form-row qos-band qos-band--yellow"
                :class="{ 'qos-band--blink': verifyBlinkTier === 'yellow' }"
              >
                <div class="form-group">
                  <label>Yellow op</label>
                  <select v-model="cm.yellow_operator" class="form-input">
                    <option v-for="o in CLI_TEXT_OPERATORS" :key="'ty'+o" :value="o">{{ o }}</option>
                  </select>
                </div>
                <div class="form-group">
                  <label>Yellow reference</label>
                  <input v-model="cm.yellow_text" class="form-input mono" />
                </div>
              </div>
              <div class="form-row qos-band qos-band--red">
                <div class="form-group hint-only">
                  <span>Red band</span>
                  <p class="hint">Neither band matches → red.</p>
                </div>
              </div>
            </template>
          </template>
        </template>

        <!-- QoS thresholds -->
        <template v-if="adapter === 'http' || adapter === 'kubernetes'">
          <h4 class="sub">QoS latency (ms)</h4>
          <p class="hint">Whole milliseconds only. Each max must be a different value, with Green &lt; Yellow &lt; Red. Verify colors the result box to match QoS.</p>
          <div class="form-row form-row--triple">
            <div
              class="form-group qos-band qos-band--green"
              :class="{ 'qos-band--blink': verifyBlinkTier === 'green' }"
            >
              <label>Green max (ms)</label>
              <input
                v-model.number="latQos.green_max_ms"
                type="number"
                class="form-input"
                min="1"
                step="1"
                inputmode="numeric"
                required
              />
            </div>
            <div
              class="form-group qos-band qos-band--yellow"
              :class="{ 'qos-band--blink': verifyBlinkTier === 'yellow' }"
            >
              <label>Yellow max (ms)</label>
              <input
                v-model.number="latQos.yellow_max_ms"
                type="number"
                class="form-input"
                min="1"
                step="1"
                inputmode="numeric"
                required
              />
            </div>
            <div
              class="form-group qos-band qos-band--red"
              :class="{ 'qos-band--blink': verifyBlinkTier === 'red' }"
            >
              <label>Red max (ms)</label>
              <input
                v-model.number="latQos.red_max_ms"
                type="number"
                class="form-input"
                min="1"
                step="1"
                inputmode="numeric"
                required
              />
            </div>
          </div>
        </template>
        <template v-if="adapter === 'prometheus'">
          <h4 class="sub">QoS metric thresholds</h4>
          <div
            class="form-row qos-band qos-band--green"
            :class="{ 'qos-band--blink': verifyBlinkTier === 'green' }"
          >
            <div class="form-group"><label>Green op</label>
              <select v-model="pq.green_operator" class="form-input"><option v-for="o in PROM_OPERATORS" :key="'g'+o" :value="o">{{ o }}</option></select>
            </div>
            <div class="form-group"><label>Green value</label><input v-model.number="pq.green_threshold" type="number" step="any" class="form-input" /></div>
          </div>
          <div
            class="form-row qos-band qos-band--yellow"
            :class="{ 'qos-band--blink': verifyBlinkTier === 'yellow' }"
          >
            <div class="form-group"><label>Yellow op</label>
              <select v-model="pq.yellow_operator" class="form-input"><option v-for="o in PROM_OPERATORS" :key="'y'+o" :value="o">{{ o }}</option></select>
            </div>
            <div class="form-group"><label>Yellow value</label><input v-model.number="pq.yellow_threshold" type="number" step="any" class="form-input" /></div>
          </div>
          <div
            class="form-row qos-band qos-band--red"
            :class="{ 'qos-band--blink': verifyBlinkTier === 'red' }"
          >
            <div class="form-group hint-only">
              <span>Red band</span>
              <p class="hint">Values below the yellow threshold map to red.</p>
            </div>
          </div>
        </template>

        <TelemetryTestPanel
          ref="testRef"
          :adapter="adapter"
          :environments="environments"
          :clusters="clusters"
          :k8s-version="testPanelK8sVersion"
          :busy="testBusy"
          @verify="runTest"
        />

        <p v-if="formError" class="field-error">{{ formError }}</p>
        <p v-if="saveBlockedHint" class="hint verify-hint">{{ saveBlockedHint }}</p>

        <div class="modal-actions">
          <button type="button" class="btn" @click="$emit('close')">Cancel</button>
          <button
            type="submit"
            class="btn btn-save btn-pill"
            :class="canSave && !saving ? 'btn-save--ready' : 'btn-save--locked'"
            :disabled="saving || !canSave"
          >
            {{ saving ? 'Saving…' : 'Save' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import type {
  AdapterType,
  TelemetryFormInitial,
  ExecutionTarget,
  TelemetryK8sClusterRow,
  TelemetryServiceProviderRow,
} from '~/types/telemetry'
import {
  CLI_RUNNER_PRESETS,
  CLI_TEXT_OPERATORS,
  HTTP_METHODS,
  K8S_CHECK_TYPES,
  K8S_VERSIONS,
  PROM_OPERATORS,
} from '~/types/telemetry'

const props = defineProps<{
  title?: string
  initial?: TelemetryFormInitial | null
  environments: { id: string; name: string }[]
  clusters: TelemetryK8sClusterRow[]
  apiFetch: (path: string, opts?: RequestInit) => Promise<unknown>
}>()

const emit = defineEmits<{ close: []; saved: [] }>()

const title = computed(() => {
  if (props.title) return props.title
  if (!props.initial) return 'Create Telemetry'
  if (props.initial.id) return 'Edit Telemetry'
  return 'Duplicate Telemetry'
})

const name = ref('')
const displayName = ref('')
const nameTrimmed = computed(() => name.value.trim())
const adapter = ref<AdapterType>('http')
/** After verify: which QoS tier matched (for band highlight). */
const verifyBlinkTier = ref<'' | 'green' | 'yellow' | 'red'>('')
let verifyBlinkTimer: ReturnType<typeof setTimeout> | null = null
/** Server-backed name+type availability when name has ≥3 characters. */
const nameAvail = ref<'idle' | 'loading' | 'available' | 'taken' | 'error'>('idle')
let nameCheckTimer: ReturnType<typeof setTimeout> | null = null
let nameCheckSeq = 0

const http = reactive({
  service_provider_id: '',
  path: '',
  method: 'GET',
  body: '',
  expected_status: 200,
})
const prom = reactive({
  service_provider_id: '',
  query: '',
  threshold: 0,
  operator: 'gte',
})
const k8s = reactive({
  cluster_id: '',
  k8s_version: '1.30',
  check_type: 'api_health',
  namespace: '',
  resource_name: '',
  label_selector: '',
})
const cliFormMode = ref<'metric' | 'legacy'>('metric')
const cli = reactive({
  command: 'curl',
  args: [] as string[],
  cli_shell: '',
  container_prep: '',
  runner: 'alpine' as (typeof CLI_RUNNER_PRESETS)[number],
  parse_json: false,
  execution_target: 'backend' as ExecutionTarget,
  cluster_id: '',
  k8s_version: '1.30',
})
/** Metric shell QoS (Prometheus-style number or text bands). */
const cm = reactive({
  value_kind: 'number' as 'number' | 'text',
  green_operator: 'gte',
  green_threshold: 0,
  yellow_operator: 'gte',
  yellow_threshold: 0,
  green_text: '',
  yellow_text: '',
})
const cq = reactive({ green_command: '', yellow_command: '', red_command: '' })
const cqGreenArgs = ref('')
const cqYellowArgs = ref('')
const cqRedArgs = ref('')
const latQos = reactive({ green_max_ms: 1500, yellow_max_ms: 2000, red_max_ms: 3000 })
const pq = reactive({
  green_operator: 'gte',
  green_threshold: 0,
  yellow_operator: 'gte',
  yellow_threshold: 0,
})

const nameFieldClass = computed(() => {
  if (nameTrimmed.value.length < 3) return ''
  if (nameAvail.value === 'available') return 'form-input--name-ok'
  if (nameAvail.value === 'taken') return 'form-input--name-bad'
  if (nameAvail.value === 'error') return 'form-input--name-warn'
  return ''
})

async function runNameAvailabilityCheck() {
  const t = nameTrimmed.value
  if (t.length < 3) {
    nameAvail.value = 'idle'
    return
  }
  const seq = ++nameCheckSeq
  nameAvail.value = 'loading'
  try {
    const params = new URLSearchParams({ name: t })
    const ex = props.initial?.id?.trim()
    if (ex) params.set('exclude_id', ex)
    const res = (await props.apiFetch(`/api/admin/telemetry/check-name?${params.toString()}`)) as {
      available?: boolean
    }
    if (seq !== nameCheckSeq) return
    nameAvail.value = res.available === true ? 'available' : 'taken'
  } catch {
    if (seq !== nameCheckSeq) return
    nameAvail.value = 'error'
  }
}

function scheduleNameAvailabilityCheck() {
  if (nameCheckTimer) clearTimeout(nameCheckTimer)
  const t = nameTrimmed.value
  if (t.length < 3) {
    nameAvail.value = 'idle'
    return
  }
  nameAvail.value = 'loading'
  nameCheckTimer = setTimeout(() => {
    nameCheckTimer = null
    void runNameAvailabilityCheck()
  }, 320)
}

watch(name, scheduleNameAvailabilityCheck)

watch(
  () => cm.value_kind,
  (vk, prev) => {
    if (vk === prev) return
    if (vk === 'text') {
      cm.green_operator = 'eq'
      cm.yellow_operator = 'eq'
    } else {
      cm.green_operator = 'gte'
      cm.yellow_operator = 'gte'
    }
  },
)

watch(adapter, (a) => {
  if (a !== 'cli') return
  if (props.initial?.adapter === 'cli') return
  cliFormMode.value = 'metric'
})

function switchCliToMetric() {
  cliFormMode.value = 'metric'
  Object.assign(cq, { green_command: '', yellow_command: '', red_command: '' })
  cqGreenArgs.value = cqYellowArgs.value = cqRedArgs.value = ''
  lastSuccessVerifyFingerprint.value = null
}

const serviceProviders = ref<TelemetryServiceProviderRow[]>([])

async function loadServiceProviders() {
  try {
    serviceProviders.value =
      (await props.apiFetch('/api/admin/service-providers')) as TelemetryServiceProviderRow[]
  } catch {
    serviceProviders.value = []
  }
}

onMounted(() => {
  void loadServiceProviders()
})

const tcpProviders = computed(() =>
  serviceProviders.value.filter((p) => p.provider_type === 'tcp'),
)
const promProviders = computed(() =>
  serviceProviders.value.filter((p) => p.provider_type === 'prometheus'),
)

const selectedPromProvider = computed(() =>
  promProviders.value.find((p) => p.id === prom.service_provider_id),
)

function clusterMinor(ver: string): string {
  const v = (ver || '').replace(/^v/i, '')
  return v.split('.').slice(0, 2).join('.')
}

function clusterUsable(c: TelemetryK8sClusterRow): boolean {
  if (c.status !== 'connected') return false
  if (c.has_api_credential === false) return false
  return true
}

const k8sFilteredClusters = computed(() => {
  const want = (k8s.k8s_version || '').trim()
  return props.clusters.filter((c) => {
    if (!clusterUsable(c)) return false
    if (want && clusterMinor(c.k8s_version || '') !== want) return false
    return true
  })
})

const cliFilteredClusters = computed(() => {
  const want = (cli.k8s_version || '').trim()
  return props.clusters.filter((c) => {
    if (!clusterUsable(c)) return false
    if (want && clusterMinor(c.k8s_version || '') !== want) return false
    return true
  })
})

function clusterLabel(c: TelemetryK8sClusterRow): string {
  const n = (c.name || '').trim()
  if (n) return `${n} (${c.id.slice(0, 8)}…)`
  return c.id
}

const testPanelK8sVersion = computed(() => {
  if (adapter.value === 'kubernetes') return k8s.k8s_version
  if (adapter.value === 'cli' && cli.execution_target === 'k8s_cluster') return cli.k8s_version
  return ''
})

watch(
  () => k8s.k8s_version,
  () => {
    if (k8s.cluster_id && !k8sFilteredClusters.value.some((c) => c.id === k8s.cluster_id)) {
      k8s.cluster_id = ''
    }
  },
)

watch(
  () => [cli.k8s_version, cli.execution_target] as const,
  () => {
    if (cli.execution_target !== 'k8s_cluster') return
    if (cli.cluster_id && !cliFilteredClusters.value.some((c) => c.id === cli.cluster_id)) {
      cli.cluster_id = ''
    }
  },
)

onUnmounted(() => {
  if (nameCheckTimer) clearTimeout(nameCheckTimer)
  if (verifyBlinkTimer) clearTimeout(verifyBlinkTimer)
})

const saving = ref(false)
const testBusy = ref(false)
const formError = ref('')
const testRef = ref<{ setResult: (r: Record<string, unknown> | null) => void } | null>(null)
/** Fingerprint of the last successful full Verify (not “Test connectivity”). Save allowed only when it matches current probe inputs. */
const lastSuccessVerifyFingerprint = ref<string | null>(null)

function splitArgs(s: string) {
  return s.trim() ? s.trim().split(/\s+/).filter(Boolean) : []
}

function isPositiveIntMs(n: unknown): n is number {
  return typeof n === 'number' && Number.isFinite(n) && Number.isInteger(n) && n > 0
}

function loadInitial() {
  lastSuccessVerifyFingerprint.value = null
  verifyBlinkTier.value = ''
  if (verifyBlinkTimer) {
    clearTimeout(verifyBlinkTimer)
    verifyBlinkTimer = null
  }
  if (nameCheckTimer) {
    clearTimeout(nameCheckTimer)
    nameCheckTimer = null
  }
  nameCheckSeq++
  nameAvail.value = 'idle'
  const row = props.initial
  if (!row) {
    name.value = ''
    displayName.value = ''
    adapter.value = 'http'
    Object.assign(http, {
      service_provider_id: '',
      path: '',
      method: 'GET',
      body: '',
      expected_status: 200,
    })
    Object.assign(prom, { service_provider_id: '', query: '', threshold: 0, operator: 'gte' })
    Object.assign(k8s, {
      cluster_id: '',
      k8s_version: '1.30',
      check_type: 'api_health',
      namespace: '',
      resource_name: '',
      label_selector: '',
    })
    cliFormMode.value = 'metric'
    Object.assign(cli, {
      command: 'curl',
      args: [],
      cli_shell: '',
      container_prep: '',
      runner: 'alpine',
      parse_json: false,
      execution_target: 'backend',
      cluster_id: '',
      k8s_version: '1.30',
    })
    Object.assign(cm, {
      value_kind: 'number',
      green_operator: 'gte',
      green_threshold: 0,
      yellow_operator: 'gte',
      yellow_threshold: 0,
      green_text: '',
      yellow_text: '',
    })
    Object.assign(cq, { green_command: '', yellow_command: '', red_command: '' })
    cqGreenArgs.value = cqYellowArgs.value = cqRedArgs.value = ''
    Object.assign(latQos, { green_max_ms: 1500, yellow_max_ms: 2000, red_max_ms: 3000 })
    Object.assign(pq, { green_operator: 'gte', green_threshold: 0, yellow_operator: 'gte', yellow_threshold: 0 })
    return
  }
  name.value = row.name
  displayName.value = row.display_name?.trim() || ''
  adapter.value = row.adapter
  const c = row.config_json || {}
  const q = row.qos_thresholds || {}
  if (row.adapter === 'http') {
    Object.assign(http, {
      service_provider_id: String(c.service_provider_id || ''),
      path: String(c.path || ''),
      method: String(c.method || 'GET'),
      body: String(c.body || ''),
      expected_status: Number(c.expected_status ?? 200),
    })
  }
  if (row.adapter === 'prometheus') {
    Object.assign(prom, {
      service_provider_id: String(c.service_provider_id || ''),
      query: String(c.query || ''),
      threshold: Number(c.threshold ?? 0),
      operator: String(c.operator || 'gte'),
    })
  }
  if (row.adapter === 'kubernetes') {
    Object.assign(k8s, {
      cluster_id: String(c.cluster_id || ''),
      k8s_version: String(c.k8s_version || '1.30'),
      check_type: String(c.check_type || 'api_health'),
      namespace: String(c.namespace || ''),
      resource_name: String(c.resource_name || ''),
      label_selector: String(c.label_selector || ''),
    })
  }
  if (row.adapter === 'cli') {
    const rawRunner = String(c.runner || 'alpine').trim()
    const runnerVal = CLI_RUNNER_PRESETS.includes(rawRunner as (typeof CLI_RUNNER_PRESETS)[number])
      ? (rawRunner as (typeof CLI_RUNNER_PRESETS)[number])
      : 'alpine'
    const hasLegacyQos =
      String(q.green_command || '').trim() !== '' &&
      String(q.yellow_command || '').trim() !== '' &&
      String(q.red_command || '').trim() !== '' &&
      String(q.value_kind || '').trim() === ''
    cliFormMode.value = hasLegacyQos ? 'legacy' : 'metric'
    Object.assign(cli, {
      command: String(c.command || 'curl'),
      parse_json: Boolean(c.parse_json),
      execution_target: String(c.execution_target || 'backend') as ExecutionTarget,
      cluster_id: String(c.cluster_id || ''),
      k8s_version: String(c.k8s_version || '1.30'),
      container_prep: typeof c.container_prep === 'string' ? c.container_prep : '',
      runner: runnerVal,
      cli_shell: typeof c.cli_shell === 'string' ? c.cli_shell : '',
    })
    cli.args = Array.isArray(c.args) ? (c.args as string[]) : []
    if (cliFormMode.value === 'metric') {
      const vk = String(q.value_kind || 'number').toLowerCase()
      cm.value_kind = vk === 'text' ? 'text' : 'number'
      cm.green_operator = String(q.green_operator ?? (vk === 'text' ? 'eq' : 'gte'))
      cm.green_threshold = Number(q.green_threshold ?? 0)
      cm.yellow_operator = String(q.yellow_operator ?? (vk === 'text' ? 'eq' : 'gte'))
      cm.yellow_threshold = Number(q.yellow_threshold ?? 0)
      cm.green_text = String(q.green_text ?? '')
      cm.yellow_text = String(q.yellow_text ?? '')
    }
  }
  if (row.adapter === 'http' || row.adapter === 'kubernetes') {
    const g = Number(q.green_max_ms ?? 200)
    const y = Number(q.yellow_max_ms ?? 500)
    latQos.green_max_ms = g
    latQos.yellow_max_ms = y
    let r = Number(q.red_max_ms)
    if (!Number.isFinite(r) || !Number.isInteger(r) || r <= y) {
      r = y + 500
    }
    latQos.red_max_ms = r
  }
  if (row.adapter === 'prometheus') {
    pq.green_operator = String(q.green_operator ?? 'gte')
    pq.green_threshold = Number(q.green_threshold ?? 0)
    pq.yellow_operator = String(q.yellow_operator ?? 'gte')
    pq.yellow_threshold = Number(q.yellow_threshold ?? 0)
  }
  if (row.adapter === 'cli') {
    if (cliFormMode.value === 'legacy') {
      cq.green_command = String(q.green_command || '')
      cq.yellow_command = String(q.yellow_command || '')
      cq.red_command = String(q.red_command || '')
      cqGreenArgs.value = Array.isArray(q.green_args) ? (q.green_args as string[]).join(' ') : ''
      cqYellowArgs.value = Array.isArray(q.yellow_args) ? (q.yellow_args as string[]).join(' ') : ''
      cqRedArgs.value = Array.isArray(q.red_args) ? (q.red_args as string[]).join(' ') : ''
    } else {
      Object.assign(cq, { green_command: '', yellow_command: '', red_command: '' })
      cqGreenArgs.value = cqYellowArgs.value = cqRedArgs.value = ''
    }
  }
}

watch(() => props.initial, loadInitial, { immediate: true })

function buildConfig(): Record<string, unknown> {
  switch (adapter.value) {
    case 'http':
      return {
        service_provider_id: http.service_provider_id.trim(),
        path: http.path.trim(),
        method: http.method,
        body: http.body || undefined,
        expected_status: http.expected_status || 200,
      }
    case 'prometheus':
      return {
        service_provider_id: prom.service_provider_id.trim(),
        query: prom.query,
        threshold: prom.threshold,
        operator: prom.operator || 'gte',
      }
    case 'kubernetes':
      return {
        cluster_id: k8s.cluster_id.trim(),
        k8s_version: k8s.k8s_version,
        check_type: k8s.check_type,
        namespace: k8s.namespace.trim() || undefined,
        resource_name: k8s.resource_name.trim() || undefined,
        label_selector: k8s.label_selector.trim() || undefined,
      }
    case 'cli': {
      const prep = cli.container_prep.trim()
      if (cliFormMode.value === 'metric') {
        const base: Record<string, unknown> = {
          cli_shell: cli.cli_shell.trim(),
          parse_json: cli.parse_json,
          execution_target: cli.execution_target,
          runner: cli.runner,
        }
        if (prep) base.container_prep = prep
        if (cli.execution_target === 'k8s_cluster') {
          base.cluster_id = cli.cluster_id.trim()
          base.k8s_version = cli.k8s_version
        }
        return base
      }
      const base: Record<string, unknown> = {
        command: cq.green_command.trim(),
        args: splitArgs(cqGreenArgs.value),
        parse_json: cli.parse_json,
        execution_target: cli.execution_target,
        runner: cli.runner,
      }
      if (prep) base.container_prep = prep
      if (cli.execution_target === 'k8s_cluster') {
        base.cluster_id = cli.cluster_id.trim()
        base.k8s_version = cli.k8s_version
      }
      return base
    }
    default:
      return {}
  }
}

function buildQos(): Record<string, unknown> {
  if (adapter.value === 'http' || adapter.value === 'kubernetes') {
    return {
      green_max_ms: Math.trunc(latQos.green_max_ms),
      yellow_max_ms: Math.trunc(latQos.yellow_max_ms),
      red_max_ms: Math.trunc(latQos.red_max_ms),
    }
  }
  if (adapter.value === 'prometheus') {
    return {
      green_operator: pq.green_operator,
      green_threshold: pq.green_threshold,
      yellow_operator: pq.yellow_operator,
      yellow_threshold: pq.yellow_threshold,
    }
  }
  if (adapter.value === 'cli') {
    if (cliFormMode.value === 'metric') {
      if (cm.value_kind === 'number') {
        return {
          value_kind: 'number',
          green_operator: cm.green_operator,
          green_threshold: cm.green_threshold,
          yellow_operator: cm.yellow_operator,
          yellow_threshold: cm.yellow_threshold,
        }
      }
      return {
        value_kind: 'text',
        green_operator: cm.green_operator,
        green_text: cm.green_text,
        yellow_operator: cm.yellow_operator,
        yellow_text: cm.yellow_text,
      }
    }
    return {
      green_command: cq.green_command,
      green_args: splitArgs(cqGreenArgs.value),
      yellow_command: cq.yellow_command,
      yellow_args: splitArgs(cqYellowArgs.value),
      red_command: cq.red_command,
      red_args: splitArgs(cqRedArgs.value),
    }
  }
  return {}
}

function executionTarget(): ExecutionTarget {
  if (adapter.value === 'kubernetes') return 'k8s_cluster'
  if (adapter.value === 'http' || adapter.value === 'prometheus') return 'backend'
  return cli.execution_target
}

function validateLocal(): string {
  const trimmedName = name.value.trim()
  if (!trimmedName) return 'Name is required'
  if (trimmedName.length >= 3) {
    if (nameAvail.value === 'loading') return 'Checking whether this name is available…'
    if (nameAvail.value === 'taken') return 'This name is already in use'
    if (nameAvail.value === 'error') return 'Could not verify name availability; try again'
  }
  if (adapter.value === 'http') {
    if (!http.service_provider_id.trim()) return 'Select an HTTP source (TCP service provider)'
    if (!http.path.trim()) return 'Path is required'
  }
  if (adapter.value === 'prometheus') {
    if (!prom.service_provider_id.trim()) return 'Select a Prometheus service provider'
    if (!prom.query.trim()) return 'PromQL required'
  }
  if (adapter.value === 'kubernetes') {
    const allowed = K8S_VERSIONS as readonly string[]
    if (!allowed.includes(k8s.k8s_version)) return 'Pick a supported K8s version'
    if (!k8s.cluster_id.trim()) return 'Select a cluster'
    if (k8s.check_type === 'deployment_ready' && !k8s.resource_name.trim()) return 'Deployment name required'
  }
  if (adapter.value === 'cli') {
    if (cli.execution_target === 'k8s_cluster') {
      const allowed = K8S_VERSIONS as readonly string[]
      if (!allowed.includes(cli.k8s_version)) return 'Pick a supported K8s version for CLI'
      if (!cli.cluster_id.trim()) return 'Select a cluster for k8s CLI'
    }
    if (cliFormMode.value === 'metric') {
      if (!cli.cli_shell.trim()) return 'Probe shell (cli_shell) is required for metric CLI QoS'
      if (cm.value_kind === 'number') {
        if (!cm.green_operator || !cm.yellow_operator) {
          return 'Numeric CLI QoS requires green and yellow operators'
        }
      } else {
        if (!cm.green_operator || !cm.yellow_operator) {
          return 'Text CLI QoS requires green and yellow operators'
        }
      }
    } else if (!cq.green_command.trim() || !cq.yellow_command.trim() || !cq.red_command.trim()) {
      return 'Legacy CLI QoS: all three commands required (or switch to metric shell QoS)'
    }
  }
  if (adapter.value === 'http' || adapter.value === 'kubernetes') {
    if (!isPositiveIntMs(latQos.green_max_ms) || !isPositiveIntMs(latQos.yellow_max_ms) || !isPositiveIntMs(latQos.red_max_ms)) {
      return 'QoS latency: green, yellow, and red must be positive whole numbers (ms)'
    }
    const g = latQos.green_max_ms
    const y = latQos.yellow_max_ms
    const r = latQos.red_max_ms
    if (g === y || y === r || g === r) {
      return 'QoS latency: green, yellow, and red max must be three different values (ms)'
    }
    if (g >= y || y >= r) {
      return 'QoS latency: must be green max < yellow max < red max (ms)'
    }
  }
  if (adapter.value === 'prometheus') {
    if (!pq.green_operator || !pq.yellow_operator) return 'Prometheus QoS requires green and yellow operators'
  }
  return ''
}

function buildProbeTestBody(payload: Record<string, unknown>): Record<string, unknown> {
  const body: Record<string, unknown> = {
    adapter: adapter.value,
    config_json: buildConfig(),
    qos_thresholds: buildQos(),
    execution_target: executionTarget(),
  }
  if (payload.environment_id) body.environment_id = payload.environment_id
  if (payload.test_connect_only) body.test_connect_only = true
  return body
}

/** Stable key for “full” verify (excludes test_connect_only). */
function fullVerifyFingerprint(payload: Record<string, unknown>): string {
  const b = buildProbeTestBody({ ...payload, test_connect_only: false })
  delete b.test_connect_only
  return JSON.stringify(b)
}

function currentVerifyPayload(): Record<string, unknown> {
  return { test_connect_only: false }
}

const canSave = computed(() => {
  if (validateLocal() !== '') return false
  const okFp = lastSuccessVerifyFingerprint.value
  if (okFp == null) return false
  return okFp === fullVerifyFingerprint(currentVerifyPayload())
})

const saveBlockedHint = computed(() => {
  if (canSave.value || validateLocal() !== '') return ''
  if (lastSuccessVerifyFingerprint.value == null) return 'Run Verify successfully before saving.'
  return 'Configuration changed since the last successful verify — run Verify again before saving.'
})

async function runTest(payload: Record<string, unknown>) {
  formError.value = ''
  testRef.value?.setResult(null)
  if (!Boolean(payload.test_connect_only)) {
    const ve = validateLocal()
    if (ve) {
      formError.value = ve
      testRef.value?.setResult({
        success: false,
        operational_ok: false,
        error: ve,
      })
      return
    }
  }
  testBusy.value = true
  try {
    const body = buildProbeTestBody(payload)
    const res = (await props.apiFetch('/api/admin/probes/test', {
      method: 'POST',
      body: JSON.stringify(body),
    })) as Record<string, unknown>
    testRef.value?.setResult(res)
    const connectOnly = Boolean(payload.test_connect_only)
    if (!connectOnly) {
      const opOk = res.operational_ok === true || (res.operational_ok === undefined && res.success === true)
      if (opOk) {
        lastSuccessVerifyFingerprint.value = fullVerifyFingerprint(payload)
      } else {
        lastSuccessVerifyFingerprint.value = null
      }
      const tier = typeof res.qos_level === 'string' ? res.qos_level.toLowerCase() : ''
      if (verifyBlinkTimer) clearTimeout(verifyBlinkTimer)
      verifyBlinkTier.value = tier === 'green' || tier === 'yellow' || tier === 'red' ? tier : ''
      if (verifyBlinkTier.value) {
        verifyBlinkTimer = setTimeout(() => {
          verifyBlinkTier.value = ''
          verifyBlinkTimer = null
        }, 1400)
      }
    }
  } catch (e: unknown) {
    testRef.value?.setResult({ success: false, error: e instanceof Error ? e.message : 'Test failed' })
    if (!payload.test_connect_only) lastSuccessVerifyFingerprint.value = null
  } finally {
    testBusy.value = false
  }
}

async function submit() {
  formError.value = ''
  const ve = validateLocal()
  if (ve) {
    formError.value = ve
    return
  }
  if (!canSave.value) {
    formError.value = saveBlockedHint.value || 'Run Verify successfully before saving.'
    return
  }
  saving.value = true
  try {
    const payload: Record<string, unknown> = {
      name: name.value.trim(),
      display_name: displayName.value.trim(),
      adapter: adapter.value,
      config_json: buildConfig(),
      qos_thresholds: buildQos(),
      execution_target: executionTarget(),
      timeout_ms: adapter.value === 'cli' ? 60000 : 30000,
      retries: 1,
    }
    const existingId = props.initial?.id?.trim()
    if (existingId) {
      await props.apiFetch(`/api/admin/telemetry/${existingId}`, {
        method: 'PUT',
        body: JSON.stringify(payload),
      })
    } else {
      await props.apiFetch('/api/admin/telemetry', { method: 'POST', body: JSON.stringify(payload) })
    }
    emit('saved')
    emit('close')
  } catch (e: unknown) {
    const apiErr = (e as { data?: { error?: string } })?.data?.error
    formError.value = apiErr || (e instanceof Error ? e.message : 'Save failed')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.form-row { display: flex; gap: 16px; flex-wrap: wrap; }
.form-row .form-group { flex: 1; min-width: 140px; }
.form-row--triple .form-group { flex: 1 1 0; min-width: 7rem; }
.form-group--name .name-field-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.form-group--name .name-field-row .form-input { flex: 1; min-width: 12rem; }
.name-status { font-size: 0.8rem; font-weight: 600; white-space: nowrap; }
.name-status--loading { color: var(--color-text-secondary); }
.name-status--ok { color: #15803d; }
.name-status--bad { color: var(--color-red, #b91c1c); }
.name-status--err { color: #b45309; }
.name-avail-hint { margin: 0; }
.form-input--name-ok { border-color: #15803d; }
.form-input--name-bad { border-color: var(--color-red, #b91c1c); }
.form-input--name-warn { border-color: #d97706; }
.btn-save { font-weight: 600; }
.btn-save--ready {
  background: #15803d;
  color: #fff;
  border-color: #166534;
  cursor: pointer;
}
.btn-save--ready:hover:not(:disabled) {
  background: #166534;
  border-color: #14532d;
}
.btn-save--locked {
  background: #e5e7eb;
  color: #6b7280;
  border-color: #d1d5db;
  cursor: not-allowed;
}
.btn-save--locked:disabled {
  opacity: 1;
}
.field-error { color: #b91c1c; font-size: 0.85rem; margin: 0; }
.hint-legacy {
  border-left: 3px solid #d97706;
  padding-left: 10px;
  margin-bottom: 6px;
}
.hint { font-size: 0.8rem; color: var(--color-text-secondary); margin: 0; }
.sub { font-size: 0.9rem; margin: 8px 0 4px; font-weight: 600; }
.radio-col { display: flex; flex-direction: column; gap: 6px; font-size: 0.9rem; }
.chk { font-size: 0.85rem; margin-top: 6px; }
.verify-hint { margin-top: 4px; }
.qos-band {
  padding: 8px 10px;
  border-radius: 8px;
  margin-bottom: 8px;
  border: 1px solid transparent;
  transition: box-shadow 0.15s ease, border-color 0.15s ease;
}
.qos-band--green { background: rgba(34, 197, 94, 0.12); border-color: rgba(34, 197, 94, 0.35); }
.qos-band--yellow { background: rgba(234, 179, 8, 0.15); border-color: rgba(234, 179, 8, 0.4); }
.qos-band--red { background: rgba(239, 68, 68, 0.12); border-color: rgba(239, 68, 68, 0.35); }
.qos-band--blink {
  animation: qos-blink-pulse 0.45s ease-in-out 2;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.55);
}
@keyframes qos-blink-pulse {
  0%, 100% { filter: brightness(1); }
  50% { filter: brightness(1.12); }
}
.hint-only { padding: 4px 0; }
</style>
