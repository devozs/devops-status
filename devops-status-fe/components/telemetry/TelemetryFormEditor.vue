<template>
  <div class="telemetry-editor-page">
    <div class="telemetry-editor-page__header">
      <button
        type="button"
        class="telemetry-editor-back"
        aria-label="Back to telemetry list"
        @click="requestCloseTelemetryModal"
      >
        <ArrowLeft :size="18" :stroke-width="2" />
        Back
      </button>
      <h1 class="page-title telemetry-editor-page__title">{{ title }}</h1>
    </div>
    <form class="modal-form telemetry-editor-form" @submit.prevent="submit">
        <div class="telemetry-editor-form__body">
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
          <AdminSegmentedRadioGroup
            v-model="adapter"
            name="telemetry-adapter"
            aria-label="Telemetry adapter"
            :grid-columns="adapterGridColumns"
            :options="adapterSegmentOptions"
          />
        </div>
        <div class="form-group">
          <label class="form-label">Probe timeout (ms)</label>
          <input
            v-model.number="telemetryTimeoutMs"
            type="number"
            class="form-input"
            min="1000"
            step="1000"
            :max="PROBE_TIMEOUT_MS_MAX"
            required
            aria-describedby="probe-timeout-hint"
          />
          <p id="probe-timeout-hint" class="hint">
            Maximum time for probe execution (Verify and scheduled runs). HLCTL/CLI may need several minutes.
          </p>
        </div>

        <!-- HTTP -->
        <template v-if="adapter === 'http'">
          <div class="form-group">
            <label class="form-label">Source</label>
            <AdminSelect
              v-model="http.service_provider_id"
              required
              aria-label="TCP service provider"
              placeholder="Select TCP service provider…"
              :options="tcpProviderSelectOptions"
            />
          </div>
          <div class="form-group">
            <label class="form-label">Path</label>
            <input v-model="http.path" class="form-input mono" placeholder="/health" required />
          </div>
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">Method</label>
              <AdminSelect v-model="http.method" aria-label="HTTP method" :options="httpMethodOptions" />
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
            <AdminSelect
              v-model="prom.service_provider_id"
              required
              aria-label="Prometheus service provider"
              placeholder="Select Prometheus service provider…"
              :options="promProviderSelectOptions"
            />
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
              <AdminSelect v-model="prom.operator" aria-label="Prometheus operator" :options="promOperatorOptions" />
            </div>
          </div>
        </template>

        <!-- Kubernetes -->
        <template v-else-if="adapter === 'kubernetes'">
          <div class="form-group">
            <label class="form-label">Probe style</label>
            <AdminRadioGroup
              v-model="k8sUiMode"
              name="k8s-probe-style"
              aria-label="Kubernetes probe style"
              direction="vertical"
              :options="k8sUiModeOptions"
            />
            <p class="hint">
              <strong>Shell on cluster</strong> runs your script in a short-lived Job (like CLI metric probes). <strong>Legacy API check</strong> uses HTTPS calls to the API server only.
            </p>
          </div>
          <div class="form-group">
            <label class="form-label">K8s version</label>
            <AdminSelect v-model="k8s.k8s_version" required aria-label="Kubernetes version" :options="k8sVersionOptions" />
          </div>
          <div class="form-group">
            <label class="form-label">Cluster</label>
            <AdminSelect
              v-model="k8s.cluster_id"
              aria-label="Kubernetes cluster"
              placeholder="Select cluster…"
              :options="k8sClusterSelectOptions"
            />
          </div>
          <template v-if="k8sUiMode === 'shell'">
            <div class="form-group">
              <label class="form-label">Runner image</label>
              <AdminSelect v-model="k8s.runner" aria-label="Kubernetes shell runner" :options="cliRunnerOptions" />
            </div>
            <div class="form-group">
              <div class="telemetry-shell-hint-label-row">
                <label class="form-label">Job environment (optional)</label>
                <ShellHintInsertControl
                  v-model="k8sProbeEnvLines"
                  v-model:hint-id="k8sJobEnvHintId"
                  :hints="shellHints.byKind.job_env"
                  aria-label="Insert Kubernetes job environment hint"
                />
              </div>
              <textarea
                v-model="k8sProbeEnvLines"
                class="form-input mono"
                rows="4"
                placeholder="REMOTE_KUBECONFIG_B64=…"
                aria-label="Kubernetes job environment variables"
                @input="k8sJobEnvHintId = ''"
              />
              <p class="hint">
                One <code class="mono">KEY=value</code> per line. Values are stored and visible to admins. For kubectl against
                another cluster, set e.g. <code class="mono">REMOTE_KUBECONFIG_B64</code> here and decode it in
                <strong>Container prep</strong> (see <code class="mono">devops-status-images/README.md</code>).
              </p>
            </div>
            <div class="form-group">
              <div class="telemetry-shell-hint-label-row">
                <label class="form-label">Job node selector (optional)</label>
                <ShellHintInsertControl
                  v-model="k8sJobNodeSelectorLines"
                  v-model:hint-id="k8sJobNodeSelectorHintId"
                  :hints="shellHints.byKind.job_node_selector"
                  aria-label="Insert Kubernetes job node selector hint"
                />
              </div>
              <textarea
                v-model="k8sJobNodeSelectorLines"
                class="form-input mono"
                rows="3"
                placeholder="e.g. kubernetes.io/arch=amd64"
                aria-label="Kubernetes job node selector"
                @input="k8sJobNodeSelectorHintId = ''"
              />
              <p class="hint">
                One <code class="mono">key=value</code> per line (Kubernetes <code class="mono">Pod</code> <code class="mono">nodeSelector</code>). Use to schedule the probe Job on matching nodes (e.g. GPU, OS, zone).
              </p>
            </div>
            <div class="form-group">
              <div class="telemetry-shell-hint-label-row">
                <label class="form-label">Container prep (shell)</label>
                <ShellHintInsertControl
                  v-model="k8s.container_prep"
                  v-model:hint-id="k8sContainerPrepHintId"
                  :hints="shellHints.byKind.container_prep"
                  aria-label="Insert Kubernetes container prep hint"
                />
              </div>
              <textarea
                v-model="k8s.container_prep"
                class="form-input mono"
                rows="3"
                placeholder="Optional: decode env vars into KUBECONFIG (see devops-status-images README)"
                @input="k8sContainerPrepHintId = ''"
              />
            </div>
            <div class="form-group">
              <div class="telemetry-shell-hint-label-row">
                <label class="form-label">Probe shell</label>
                <ShellHintInsertControl
                  v-model="k8s.cli_shell"
                  v-model:hint-id="k8sProbeShellHintId"
                  :hints="shellHints.byKind.probe_shell"
                  aria-label="Insert Kubernetes probe shell hint"
                />
              </div>
              <textarea
                v-model="k8s.cli_shell"
                class="form-input mono"
                rows="8"
                required
                placeholder="e.g. kubectl get pods -n default --no-headers | wc -l"
                @input="k8sProbeShellHintId = ''"
              />
              <p class="hint">
                Stdout is used for QoS: by default the first number in output, or JSON field <code class="mono">value</code> when parse JSON is on. For noisy logs, choose
                <strong>Last number</strong> below (same rules as CLI metric shell).
              </p>
            </div>
            <label class="chk"><input v-model="k8s.parse_json" type="checkbox" /> Parse JSON stdout (numeric: field <code class="mono">value</code>)</label>
            <h4 class="sub">QoS metric thresholds</h4>
            <div class="form-group">
              <label class="form-label">Value type</label>
              <AdminSelect v-model="cm.value_kind" aria-label="Kubernetes metric value type" :options="[...metricValueKindOptions]" />
            </div>
            <template v-if="cm.value_kind === 'number'">
              <div class="form-group">
                <label class="form-label">Numeric value from stdout</label>
                <AdminSelect
                  v-model="cm.numeric_value_pick"
                  aria-label="Kubernetes numeric value from stdout"
                  :options="[...metricNumericValuePickOptions]"
                />
                <p class="hint">
                  Use <strong>Last number</strong> when the probe prints logs and a final metric (e.g. duration) on the last line. JSON parse mode always uses field
                  <code class="mono">value</code>.
                </p>
              </div>
              <div
                class="form-row qos-band qos-band--green"
                :class="{ 'qos-band--blink': verifyBlinkTier === 'green' }"
              >
                <div class="form-group">
                  <label>Green op</label>
                  <AdminSelect v-model="cm.green_operator" aria-label="K8s green operator" :options="promOperatorOptions" />
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
                  <AdminSelect v-model="cm.yellow_operator" aria-label="K8s yellow operator" :options="promOperatorOptions" />
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
                  <AdminSelect v-model="cm.green_operator" aria-label="K8s green text operator" :options="cliTextOperatorOptions" />
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
                  <AdminSelect v-model="cm.yellow_operator" aria-label="K8s yellow text operator" :options="cliTextOperatorOptions" />
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
          <template v-else>
            <div class="form-group">
              <label class="form-label">Check type</label>
              <AdminSelect v-model="k8s.check_type" aria-label="Kubernetes check type" :options="k8sCheckTypeOptions" />
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
        </template>

        <!-- Liveness (Kubernetes API or service provider connectivity) -->
        <template v-else-if="adapter === 'liveness'">
          <div class="form-group">
            <label class="form-label">Source</label>
            <AdminRadioGroup
              v-model="livenessSource"
              name="liveness-source"
              aria-label="Liveness source"
              direction="vertical"
              :options="livenessSourceOptions"
            />
            <p class="hint">
              <strong>Kubernetes</strong> uses the same checks as Kubernetes telemetry (for environment links).
              <strong>Service provider</strong> runs a built-in connectivity probe for the selected provider type (for service links).
            </p>
          </div>
          <template v-if="livenessSource === 'kubernetes'">
            <div class="form-group">
              <label class="form-label">Probe style</label>
              <AdminRadioGroup
                v-model="k8sUiMode"
                name="liveness-k8s-probe-style"
                aria-label="Liveness Kubernetes probe style"
                direction="vertical"
                :options="k8sUiModeOptions"
              />
            </div>
            <div class="form-group">
              <label class="form-label">K8s version</label>
              <AdminSelect v-model="k8s.k8s_version" required aria-label="Liveness Kubernetes version" :options="k8sVersionOptions" />
            </div>
            <div class="form-group">
              <label class="form-label">Cluster</label>
              <AdminSelect
                v-model="k8s.cluster_id"
                aria-label="Liveness Kubernetes cluster"
                placeholder="Select cluster…"
                :options="k8sClusterSelectOptions"
              />
            </div>
            <template v-if="k8sUiMode === 'shell'">
              <div class="form-group">
                <label class="form-label">Runner image</label>
                <AdminSelect v-model="k8s.runner" aria-label="Liveness shell runner" :options="cliRunnerOptions" />
              </div>
              <div class="form-group">
                <div class="telemetry-shell-hint-label-row">
                  <label class="form-label">Job environment (optional)</label>
                  <ShellHintInsertControl
                    v-model="k8sProbeEnvLines"
                    v-model:hint-id="k8sJobEnvHintId"
                    :hints="shellHints.byKind.job_env"
                    aria-label="Insert liveness Kubernetes job environment hint"
                  />
                </div>
                <textarea
                  v-model="k8sProbeEnvLines"
                  class="form-input mono"
                  rows="4"
                  placeholder="REMOTE_KUBECONFIG_B64=…"
                  aria-label="Liveness Kubernetes job environment variables"
                  @input="k8sJobEnvHintId = ''"
                />
                <p class="hint">
                  Same as Kubernetes telemetry: <code class="mono">KEY=value</code> per line; admin-visible. Use with Container prep
                  to materialize kubeconfig for other clusters.
                </p>
              </div>
              <div class="form-group">
                <div class="telemetry-shell-hint-label-row">
                  <label class="form-label">Job node selector (optional)</label>
                  <ShellHintInsertControl
                    v-model="k8sJobNodeSelectorLines"
                    v-model:hint-id="k8sJobNodeSelectorHintId"
                    :hints="shellHints.byKind.job_node_selector"
                    aria-label="Insert liveness job node selector hint"
                  />
                </div>
                <textarea
                  v-model="k8sJobNodeSelectorLines"
                  class="form-input mono"
                  rows="3"
                  placeholder="e.g. kubernetes.io/arch=amd64"
                  aria-label="Liveness job node selector"
                  @input="k8sJobNodeSelectorHintId = ''"
                />
                <p class="hint">Same as Kubernetes shell Jobs: <code class="mono">key=value</code> per line for Pod <code class="mono">nodeSelector</code>.</p>
              </div>
              <div class="form-group">
                <div class="telemetry-shell-hint-label-row">
                  <label class="form-label">Container prep (shell)</label>
                  <ShellHintInsertControl
                    v-model="k8s.container_prep"
                    v-model:hint-id="k8sContainerPrepHintId"
                    :hints="shellHints.byKind.container_prep"
                    aria-label="Insert liveness container prep hint"
                  />
                </div>
                <textarea
                  v-model="k8s.container_prep"
                  class="form-input mono"
                  rows="3"
                  placeholder="Optional"
                  @input="k8sContainerPrepHintId = ''"
                />
              </div>
              <div class="form-group">
                <div class="telemetry-shell-hint-label-row">
                  <label class="form-label">Probe shell</label>
                  <ShellHintInsertControl
                    v-model="k8s.cli_shell"
                    v-model:hint-id="k8sProbeShellHintId"
                    :hints="shellHints.byKind.probe_shell"
                    aria-label="Insert liveness probe shell hint"
                  />
                </div>
                <textarea
                  v-model="k8s.cli_shell"
                  class="form-input mono"
                  rows="8"
                  required
                  placeholder="Shell to run on the cluster"
                  @input="k8sProbeShellHintId = ''"
                />
              </div>
              <label class="chk"><input v-model="k8s.parse_json" type="checkbox" /> Parse JSON stdout (numeric: field <code class="mono">value</code>)</label>
              <p class="hint">
                Shell mode requires <strong>QoS probe value</strong> below (maps to parsed numeric stdout). Latency bands still apply.
              </p>
            </template>
            <template v-else>
              <div class="form-group">
                <label class="form-label">Check type</label>
                <AdminSelect v-model="k8s.check_type" aria-label="Liveness check type" :options="k8sCheckTypeOptions" />
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
          </template>
          <template v-else>
            <div class="form-group">
              <label class="form-label">Service provider</label>
              <AdminSelect
                v-model="livenessSpId"
                required
                aria-label="Liveness service provider"
                placeholder="Select provider…"
                :options="livenessProviderSelectOptions"
              />
            </div>
            <p
              v-if="selectedLivenessProvider && selectedLivenessProvider.provider_type === 'prometheus' && selectedLivenessProvider.has_prometheus_credentials === false"
              class="hint"
            >
              This provider has no stored credentials — probes may fail until you add them under Service providers.
            </p>
            <template v-if="selectedLivenessProvider?.provider_type === 'artifactory'">
              <div class="form-group">
                <label class="form-label">Download test path</label>
                <input
                  v-model="livenessAfDownloadPath"
                  class="form-input mono"
                  autocomplete="off"
                  placeholder="e.g. libs-release-local/org/foo/artifact/1.0/file.jar"
                />
                <p class="hint">
                  Optional. Repository-relative path under your Artifactory base URL (same layout as in the UI after
                  <code class="inline-code">/artifactory/</code>). The probe pings first, then measures <strong>average download speed</strong> over a
                  capped read (default {{ (ARTIFACTORY_DOWNLOAD_DEFAULT_MAX_BYTES / (1024 * 1024)).toFixed(0) }} MiB). Enable
                  <strong>Also evaluate probe value</strong> below — thresholds are <strong>bytes per second</strong>.
                </p>
              </div>
              <div class="form-group">
                <label class="form-label">Max sample bytes (optional)</label>
                <input
                  v-model="livenessAfMaxBytesStr"
                  class="form-input mono"
                  type="text"
                  inputmode="numeric"
                  autocomplete="off"
                  :placeholder="String(ARTIFACTORY_DOWNLOAD_DEFAULT_MAX_BYTES)"
                />
                <p class="hint">Leave empty for default. Maximum {{ ARTIFACTORY_DOWNLOAD_ABS_MAX_BYTES.toLocaleString() }} (server limit).</p>
              </div>
            </template>
          </template>
        </template>

        <!-- CLI / HLCTL -->
        <template v-else-if="adapter === 'cli' || adapter === 'hlctl'">
          <template v-if="adapter === 'hlctl'">
            <div class="form-group">
              <label class="form-label">HLCTL service provider</label>
              <AdminSelect
                v-model="hlctlSpId"
                required
                aria-label="HLCTL service provider"
                placeholder="Select HLCTL provider…"
                :options="hlctlProviderSelectOptions"
              />
            </div>
            <p class="hint">
              Probes run in the <strong>management</strong> server. The server runs <code class="mono">hlctl kubeconfig</code> using credentials stored on the provider, then your shell or command (same patterns as CLI).
            </p>
          </template>
          <template v-if="adapter === 'cli'">
            <div class="form-group">
              <label class="form-label">Execution</label>
              <AdminRadioGroup
                v-model="cli.execution_target"
                name="cli-execution-target"
                aria-label="CLI execution target"
                direction="vertical"
                :options="cliExecutionTargetOptions"
              />
              <p class="hint">
                <strong>Backend</strong> runs the probe in a disposable Linux container via <code class="mono">docker run</code> (see server
                <code class="mono">CLI_BACKEND_EXECUTOR</code> / <code class="mono">CLI_DOCKER_NETWORK</code>), or as a Job on a
                <strong>designated cluster</strong> when <code class="mono">CLI_BACKEND_EXECUTOR=k8s</code>.
                <strong>Kubernetes Job</strong> uses the cluster you select below; both paths share the same shell + image presets.
              </p>
            </div>
            <div class="form-group">
              <label class="form-label">Runner image</label>
              <AdminSelect v-model="cli.runner" aria-label="CLI runner image" :options="cliRunnerOptions" />
              <p class="hint">
                Mapped on the server to <code class="mono">CLI_RUNNER_IMAGE_TOOLKIT</code>,
                <code class="mono">CLI_RUNNER_IMAGE_ALPINE</code>, or <code class="mono">CLI_RUNNER_IMAGE_UBUNTU_24</code>.
              </p>
            </div>
          </template>
          <div class="form-group">
            <div class="telemetry-shell-hint-label-row">
              <label class="form-label">Container environment (optional)</label>
              <ShellHintInsertControl
                v-model="cliProbeEnvLines"
                v-model:hint-id="cliJobEnvHintId"
                :hints="shellHints.byKind.job_env"
                aria-label="Insert CLI container environment hint"
              />
            </div>
            <textarea
              v-model="cliProbeEnvLines"
              class="form-input mono"
              rows="4"
              placeholder="KEY=value per line"
              aria-label="CLI container environment variables"
              @input="cliJobEnvHintId = ''"
            />
            <p v-if="adapter === 'cli'" class="hint">
              Passed to <code class="mono">docker run -e</code> or the Kubernetes Job; one <code class="mono">KEY=value</code> per line.
              Admin-visible.
            </p>
            <p v-else class="hint">
              Environment for the local shell after <code class="mono">hlctl kubeconfig</code>; one <code class="mono">KEY=value</code> per
              line. Admin-visible.
            </p>
          </div>
          <template v-if="adapter === 'cli'">
            <div class="form-group">
              <div class="telemetry-shell-hint-label-row">
                <label class="form-label">Job node selector (optional)</label>
                <ShellHintInsertControl
                  v-model="cliJobNodeSelectorLines"
                  v-model:hint-id="cliJobNodeSelectorHintId"
                  :hints="shellHints.byKind.job_node_selector"
                  aria-label="Insert CLI job node selector hint"
                />
              </div>
              <textarea
                v-model="cliJobNodeSelectorLines"
                class="form-input mono"
                rows="3"
                placeholder="e.g. topology.kubernetes.io/zone=us-east-1a"
                aria-label="CLI job node selector"
                @input="cliJobNodeSelectorHintId = ''"
              />
              <p class="hint">
                Applied when this probe runs as a <strong>Kubernetes Job</strong> (execution <strong>Kubernetes Job</strong>, or
                <strong>Backend</strong> with server <code class="mono">CLI_BACKEND_EXECUTOR=k8s</code>). Ignored for Docker backend.
                <code class="mono">key=value</code> per line.
              </p>
            </div>
          </template>
          <template v-if="adapter === 'cli' && cli.execution_target === 'k8s_cluster'">
            <div class="form-group">
              <label class="form-label">K8s version</label>
              <AdminSelect v-model="cli.k8s_version" required aria-label="CLI Kubernetes version" :options="k8sVersionOptions" />
            </div>
            <div class="form-group">
              <label class="form-label">Cluster</label>
              <AdminSelect
                v-model="cli.cluster_id"
                required
                aria-label="CLI Kubernetes cluster"
                placeholder="Select cluster…"
                :options="cliClusterSelectOptions"
              />
            </div>
          </template>
          <div class="form-group">
            <div class="telemetry-shell-hint-label-row">
              <label class="form-label">Container prep (shell)</label>
              <ShellHintInsertControl
                v-model="cli.container_prep"
                v-model:hint-id="cliContainerPrepHintId"
                :hints="shellHints.byKind.container_prep"
                aria-label="Insert CLI container prep hint"
              />
            </div>
            <textarea
              v-model="cli.container_prep"
              class="form-input mono"
              rows="4"
              placeholder="Optional. Runs before the probe in the same shell (e.g. apk add --no-cache curl)."
              @input="cliContainerPrepHintId = ''"
            />
            <p v-if="adapter === 'cli'" class="hint">
              Runs first in the runner container (subshell). Install tools here (e.g. <code class="mono">apk add</code> or <code class="mono">apt-get</code>). Prep <strong>stdout</strong> is sent to <strong>stderr</strong> so you still see it when verifying (combined log / stderr), but it is <strong>not</strong> used for QoS or parsed probe values — only the probe command or probe shell writes to stdout for that.
            </p>
            <p v-else class="hint">
              Runs first in the local shell on the management server (after <code class="mono">hlctl kubeconfig</code>). Prep <strong>stdout</strong> goes to <strong>stderr</strong> for debugging; only probe stdout is used for QoS.
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
              <div class="telemetry-shell-hint-label-row">
                <label class="form-label">Shell script</label>
                <ShellHintInsertControl
                  v-model="cli.cli_shell"
                  v-model:hint-id="cliProbeShellHintId"
                  :hints="shellHints.byKind.probe_shell"
                  :disabled="cliFormMode !== 'metric'"
                  aria-label="Insert CLI probe shell hint"
                />
              </div>
              <textarea
                v-model="cli.cli_shell"
                class="form-input mono"
                rows="10"
                placeholder="e.g. echo 42"
                :required="cliFormMode === 'metric'"
                @input="cliProbeShellHintId = ''"
              />
            </div>
            <label class="chk"><input v-model="cli.parse_json" type="checkbox" /> Parse JSON stdout (numeric mode: use number field <code class="mono">value</code>)</label>

            <h4 class="sub">QoS thresholds</h4>
            <div class="form-group">
              <label class="form-label">Value type</label>
              <AdminSelect
                v-model="cm.value_kind"
                aria-label="Metric value type"
                :options="[...metricValueKindOptions]"
              />
            </div>

            <template v-if="cm.value_kind === 'number'">
              <div class="form-group">
                <label class="form-label">Numeric value from stdout</label>
                <AdminSelect
                  v-model="cm.numeric_value_pick"
                  aria-label="CLI numeric value from stdout"
                  :options="[...metricNumericValuePickOptions]"
                />
                <p class="hint">
                  Use <strong>Last number</strong> when the probe prints logs and a final metric (e.g. duration) on the last line. JSON parse mode always uses field
                  <code class="mono">value</code>.
                </p>
              </div>
              <div
                class="form-row qos-band qos-band--green"
                :class="{ 'qos-band--blink': verifyBlinkTier === 'green' }"
              >
                <div class="form-group">
                  <label>Green op</label>
                  <AdminSelect v-model="cm.green_operator" aria-label="Green operator" :options="promOperatorOptions" />
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
                  <AdminSelect v-model="cm.yellow_operator" aria-label="Yellow operator" :options="promOperatorOptions" />
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
                  <AdminSelect v-model="cm.green_operator" aria-label="Green text operator" :options="cliTextOperatorOptions" />
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
                  <AdminSelect v-model="cm.yellow_operator" aria-label="Yellow text operator" :options="cliTextOperatorOptions" />
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
        <template v-if="adapter === 'http' || (adapter === 'kubernetes' && k8sUiMode === 'legacy') || adapter === 'liveness'">
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
        <template v-if="adapter === 'liveness'">
          <template v-if="livenessSource === 'kubernetes' && k8sUiMode === 'shell'">
            <h4 class="sub">QoS probe value (required for shell)</h4>
            <p class="hint">
              Required when using shell on the cluster: thresholds apply to the parsed number from probe stdout (same as Kubernetes metric telemetry).
            </p>
          </template>
          <template v-else>
            <h4 class="sub">QoS probe value (optional)</h4>
            <p class="hint">
              When enabled, each sample is scored as the worse of latency and value. Operators apply to the numeric probe result (1 = healthy for ping-only providers).
            </p>
            <p
              v-if="livenessSource === 'service_provider' && selectedLivenessProvider?.provider_type === 'artifactory' && livenessAfDownloadPath.trim()"
              class="hint"
            >
              With a download test path, <strong>probe value</strong> is average speed in <strong>bytes per second</strong> over the capped sample (ping latency still uses the bands above).
            </p>
            <label class="chk">
              <input v-model="livenessValueQosEnabled" type="checkbox" />
              Also evaluate probe value (RawValue)
            </label>
          </template>
          <template v-if="livenessShellValueFieldsVisible">
            <div
              class="form-row qos-band qos-band--green"
              :class="{ 'qos-band--blink': verifyBlinkTier === 'green' }"
            >
              <div class="form-group">
                <label>Green op</label>
                <AdminSelect v-model="livVal.green_operator" aria-label="Liveness value green operator" :options="promOperatorOptions" />
              </div>
              <div class="form-group">
                <label>Green value</label>
                <input v-model.number="livVal.green_threshold" type="number" step="any" class="form-input" />
              </div>
            </div>
            <div
              class="form-row qos-band qos-band--yellow"
              :class="{ 'qos-band--blink': verifyBlinkTier === 'yellow' }"
            >
              <div class="form-group">
                <label>Yellow op</label>
                <AdminSelect v-model="livVal.yellow_operator" aria-label="Liveness value yellow operator" :options="promOperatorOptions" />
              </div>
              <div class="form-group">
                <label>Yellow value</label>
                <input v-model.number="livVal.yellow_threshold" type="number" step="any" class="form-input" />
              </div>
            </div>
            <div
              class="form-row qos-band qos-band--red"
              :class="{ 'qos-band--blink': verifyBlinkTier === 'red' }"
            >
              <div class="form-group hint-only">
                <span>Red band</span>
                <p class="hint">Values that do not meet the yellow band map to red.</p>
              </div>
            </div>
          </template>
        </template>
        <template v-if="adapter === 'prometheus'">
          <h4 class="sub">QoS metric thresholds</h4>
          <div
            class="form-row qos-band qos-band--green"
            :class="{ 'qos-band--blink': verifyBlinkTier === 'green' }"
          >
            <div class="form-group"><label>Green op</label>
              <AdminSelect v-model="pq.green_operator" aria-label="QoS green operator" :options="promOperatorOptions" />
            </div>
            <div class="form-group"><label>Green value</label><input v-model.number="pq.green_threshold" type="number" step="any" class="form-input" /></div>
          </div>
          <div
            class="form-row qos-band qos-band--yellow"
            :class="{ 'qos-band--blink': verifyBlinkTier === 'yellow' }"
          >
            <div class="form-group"><label>Yellow op</label>
              <AdminSelect v-model="pq.yellow_operator" aria-label="QoS yellow operator" :options="promOperatorOptions" />
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
          @abort-verify="abortVerifyTest"
        />

        <p v-if="formError" class="field-error">{{ formError }}</p>
        <p v-if="saveBlockedHint" class="hint verify-hint">{{ saveBlockedHint }}</p>
        </div>

        <div class="modal-actions telemetry-editor-form__actions">
          <button type="button" class="btn btn-modal-cancel" @click="requestCloseTelemetryModal">Cancel</button>
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

  <AdminUnsavedConfirmDialog
    v-model="unsavedDialogOpen"
    :title="unsavedDialogTitle"
    :warning="unsavedDialogMessage"
    :confirm-label="unsavedConfirmLabel"
    :discard-confirm="confirmUnsavedDialog"
    @cancel="cancelUnsavedDialog"
  />
</template>

<script setup lang="ts">
import { ArrowLeft } from 'lucide-vue-next'
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import ShellHintInsertControl from '~/components/telemetry/ShellHintInsertControl.vue'
import { postProbeTestStream, ProbeStreamAbortedError } from '~/composables/useProbeTestStream'
import { useTelemetryShellHints } from '~/composables/useTelemetryShellHints'
import type {
  AdapterType,
  ExecutionTarget,
  LivenessConfigSource,
  TelemetryFormInitial,
  TelemetryK8sClusterRow,
  TelemetryServiceProviderRow,
} from '~/types/telemetry'
import {
  ARTIFACTORY_DOWNLOAD_ABS_MAX_BYTES,
  ARTIFACTORY_DOWNLOAD_DEFAULT_MAX_BYTES,
  CLI_RUNNER_PRESETS,
  CLI_TEXT_OPERATORS,
  HTTP_METHODS,
  K8S_CHECK_TYPES,
  K8S_VERSIONS,
  LIVENESS_SERVICE_PROVIDER_TYPES,
  PROM_OPERATORS,
} from '~/types/telemetry'
import { formatProbeEnvLines, parseProbeEnvLines } from '~/utils/probeEnvLines'
import { formatNodeSelectorLines, parseNodeSelectorLines } from '~/utils/nodeSelectorLines'

const props = defineProps<{
  title?: string
  initial?: TelemetryFormInitial | null
  environments: { id: string; name: string }[]
  clusters: TelemetryK8sClusterRow[]
  apiFetch: (path: string, opts?: RequestInit) => Promise<unknown>
}>()

const emit = defineEmits<{ saved: [] }>()

const shellHints = useTelemetryShellHints(props.apiFetch)
onMounted(() => {
  void shellHints.ensureAllLoaded()
})

const title = computed(() => {
  if (props.title) return props.title
  if (!props.initial) return 'Create Telemetry'
  if (props.initial.id) return 'Edit Telemetry'
  return 'Duplicate Telemetry'
})

const PROBE_TIMEOUT_MS_MIN = 1000
const PROBE_TIMEOUT_MS_MAX = 45 * 60 * 1000

function defaultTimeoutMsForAdapter(a: AdapterType): number {
  if (a === 'cli' || a === 'hlctl') return 600000
  return 30000
}

function clampProbeTimeoutMs(n: number): number {
  return Math.min(PROBE_TIMEOUT_MS_MAX, Math.max(PROBE_TIMEOUT_MS_MIN, Math.round(n)))
}

const name = ref('')
const displayName = ref('')
const nameTrimmed = computed(() => name.value.trim())
const adapter = ref<AdapterType>('http')
const telemetryTimeoutMs = ref(defaultTimeoutMsForAdapter('http'))
const capabilities = ref({ hlctl_enabled: false })

async function loadCapabilities() {
  try {
    capabilities.value = (await props.apiFetch('/api/admin/capabilities')) as { hlctl_enabled: boolean }
  } catch {
    capabilities.value = { hlctl_enabled: false }
  }
}

const adapterSegmentOptions = computed(() => {
  const all: { value: AdapterType; label: string }[] = [
    { value: 'http', label: 'HTTP' },
    { value: 'prometheus', label: 'Prometheus' },
    { value: 'kubernetes', label: 'Kubernetes' },
    { value: 'liveness', label: 'Liveness' },
    { value: 'cli', label: 'CLI' },
    { value: 'hlctl', label: 'HLCTL' },
  ]
  let opts = capabilities.value.hlctl_enabled ? all : all.filter((o) => o.value !== 'hlctl')
  if (props.initial?.adapter === 'hlctl' && !opts.some((o) => o.value === 'hlctl')) {
    opts = [...opts, { value: 'hlctl', label: 'HLCTL' }]
  }
  return opts
})

/** Match provider grid layout: 8 visible → 4 cols; 9 → 3 cols; else one row by count. */
const adapterGridColumns = computed(() => {
  const n = adapterSegmentOptions.value.length
  if (n >= 9) return 3
  if (n === 8) return 4
  return n
})

const hlctlSpId = ref('')

const livenessSpTypes = new Set<string>(LIVENESS_SERVICE_PROVIDER_TYPES)
const livenessSourceOptions = [
  { value: 'kubernetes' as const, label: 'Kubernetes' },
  { value: 'service_provider' as const, label: 'Service provider' },
] as const satisfies readonly { value: LivenessConfigSource; label: string }[]

const livenessSource = ref<LivenessConfigSource>('kubernetes')
const livenessSpId = ref('')
/** Artifactory liveness: repo-relative path for bandwidth sample. */
const livenessAfDownloadPath = ref('')
const livenessAfMaxBytesStr = ref('')
const livenessValueQosEnabled = ref(false)
const livVal = reactive({
  green_operator: 'gte',
  green_threshold: 1,
  yellow_operator: 'gte',
  yellow_threshold: 0.5,
})

const k8sUiMode = ref<'shell' | 'legacy'>('shell')
const k8sUiModeOptions = [
  { value: 'shell' as const, label: 'Shell on cluster (metric QoS)' },
  { value: 'legacy' as const, label: 'Legacy API check' },
] as const satisfies readonly { value: 'shell' | 'legacy'; label: string }[]

const livenessArtifactoryBwPath = computed(
  () =>
    adapter.value === 'liveness' &&
    livenessSource.value === 'service_provider' &&
    selectedLivenessProvider.value?.provider_type === 'artifactory' &&
    livenessAfDownloadPath.value.trim() !== '',
)

const livenessShellValueFieldsVisible = computed(
  () =>
    adapter.value === 'liveness' &&
    (livenessValueQosEnabled.value ||
      (livenessSource.value === 'kubernetes' && k8sUiMode.value === 'shell') ||
      livenessArtifactoryBwPath.value),
)

const cliExecutionTargetOptions = [
  { value: 'backend', label: 'Backend' },
  { value: 'k8s_cluster', label: 'Kubernetes Job' },
] as const satisfies readonly { value: ExecutionTarget; label: string }[]
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
  cli_shell: '',
  container_prep: '',
  runner: 'toolkit' as (typeof CLI_RUNNER_PRESETS)[number],
  parse_json: false,
})
const cliFormMode = ref<'metric' | 'legacy'>('metric')
const cli = reactive({
  command: 'curl',
  args: [] as string[],
  cli_shell: '',
  container_prep: '',
  runner: 'toolkit' as (typeof CLI_RUNNER_PRESETS)[number],
  parse_json: false,
  execution_target: 'backend' as ExecutionTarget,
  cluster_id: '',
  k8s_version: '1.30',
})
/** KEY=value lines for Kubernetes / Liveness shell Jobs (shared). */
const k8sProbeEnvLines = ref('')
/** KEY=value lines for CLI docker / Job. */
const cliProbeEnvLines = ref('')
/** key=value lines for Pod nodeSelector (K8s / Liveness shell Jobs). */
const k8sJobNodeSelectorLines = ref('')
/** key=value lines for CLI Job nodeSelector. */
const cliJobNodeSelectorLines = ref('')

/** Optional telemetry_shell_hints links (Kubernetes / Liveness K8s shell share these). */
const k8sJobEnvHintId = ref('')
const k8sJobNodeSelectorHintId = ref('')
const k8sContainerPrepHintId = ref('')
const k8sProbeShellHintId = ref('')
/** Optional hint links for CLI adapter. */
const cliJobEnvHintId = ref('')
const cliJobNodeSelectorHintId = ref('')
const cliContainerPrepHintId = ref('')
const cliProbeShellHintId = ref('')

function attachShellHintIds(
  target: Record<string, unknown>,
  ids: {
    job_env_hint_id: string
    job_node_selector_hint_id: string
    container_prep_hint_id: string
    probe_shell_hint_id: string
  },
) {
  if (ids.job_env_hint_id.trim()) target.job_env_hint_id = ids.job_env_hint_id.trim()
  if (ids.job_node_selector_hint_id.trim()) {
    target.job_node_selector_hint_id = ids.job_node_selector_hint_id.trim()
  }
  if (ids.container_prep_hint_id.trim()) target.container_prep_hint_id = ids.container_prep_hint_id.trim()
  if (ids.probe_shell_hint_id.trim()) target.probe_shell_hint_id = ids.probe_shell_hint_id.trim()
}

function probeEnvPayload(lines: string): Record<string, string> | undefined {
  const t = lines.trim()
  if (!t) return undefined
  const p = parseProbeEnvLines(lines)
  if (!p.ok || Object.keys(p.env).length === 0) return undefined
  return p.env
}

function probeNodeSelectorPayload(lines: string): Record<string, string> | undefined {
  const t = lines.trim()
  if (!t) return undefined
  const p = parseNodeSelectorLines(lines)
  if (!p.ok || Object.keys(p.node_selector).length === 0) return undefined
  return p.node_selector
}
/** Metric shell QoS (Prometheus-style number or text bands). */
const cm = reactive({
  value_kind: 'number' as 'number' | 'text',
  numeric_value_pick: 'first' as 'first' | 'last',
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

const {
  requestClose: requestModalClose,
  unsavedDialogOpen,
  unsavedDialogMessage,
  unsavedDialogTitle,
  unsavedConfirmLabel,
  confirmUnsavedDialog,
  cancelUnsavedDialog,
} = useModalUnsavedGuard()
const formSnapshotBaseline = ref('')

function captureBaselineSnapshot(): string {
  return JSON.stringify({
    name: name.value.trim(),
    displayName: displayName.value.trim(),
    adapter: adapter.value,
    telemetryTimeoutMs: telemetryTimeoutMs.value,
    livenessSource: livenessSource.value,
    livenessSpId: livenessSpId.value,
    livenessAfPath: livenessAfDownloadPath.value,
    livenessAfMax: livenessAfMaxBytesStr.value,
    livenessValQoSEn: livenessValueQosEnabled.value,
    livVal: { ...livVal },
    k8sUiMode: k8sUiMode.value,
    k8sProbeEnvLines: k8sProbeEnvLines.value,
    cliProbeEnvLines: cliProbeEnvLines.value,
    k8sJobNodeSelectorLines: k8sJobNodeSelectorLines.value,
    cliJobNodeSelectorLines: cliJobNodeSelectorLines.value,
    k8sJobEnvHintId: k8sJobEnvHintId.value,
    k8sJobNodeSelectorHintId: k8sJobNodeSelectorHintId.value,
    k8sContainerPrepHintId: k8sContainerPrepHintId.value,
    k8sProbeShellHintId: k8sProbeShellHintId.value,
    cliJobEnvHintId: cliJobEnvHintId.value,
    cliJobNodeSelectorHintId: cliJobNodeSelectorHintId.value,
    cliContainerPrepHintId: cliContainerPrepHintId.value,
    cliProbeShellHintId: cliProbeShellHintId.value,
    http: { ...http },
    prom: { ...prom },
    k8s: { ...k8s },
    cliFormMode: cliFormMode.value,
    hlctlSpId: hlctlSpId.value,
    cli: { ...cli, args: [...cli.args] },
    cm: { ...cm },
    cq: { ...cq },
    cqG: cqGreenArgs.value,
    cqY: cqYellowArgs.value,
    cqR: cqRedArgs.value,
    latQos: { ...latQos },
    pq: { ...pq },
  })
}

function isTelemetryFormDirty() {
  return captureBaselineSnapshot() !== formSnapshotBaseline.value
}

/** One-shot bypass so confirmed discard navigation is not blocked again by page route-leave guard. */
const allowNextUnsavedBypass = ref(false)

function consumeNextUnsavedBypass(): boolean {
  if (!allowNextUnsavedBypass.value) return false
  allowNextUnsavedBypass.value = false
  return true
}

function requestCloseTelemetryModal() {
  requestModalClose(
    () => {
      allowNextUnsavedBypass.value = true
      void navigateTo('/admin/telemetry')
    },
    {
      isDirty: () => isTelemetryFormDirty(),
      title: 'Discard telemetry changes?',
      message: 'You have unsaved changes to this telemetry. Discard them?',
      confirmLabel: 'Discard',
    },
  )
}

/** After `next(false)` in onBeforeRouteLeave when dirty; always shows discard dialog then runs `perform`. */
function promptDiscardThenNavigate(perform: () => void) {
  requestModalClose(() => {
    allowNextUnsavedBypass.value = true
    perform()
  }, {
    isDirty: () => true,
    title: 'Discard telemetry changes?',
    message: 'You have unsaved changes to this telemetry. Discard them?',
    confirmLabel: 'Discard',
  })
}

defineExpose({
  isTelemetryFormDirty,
  consumeNextUnsavedBypass,
  promptDiscardThenNavigate,
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
  void loadCapabilities()
})

const tcpProviders = computed(() =>
  serviceProviders.value.filter((p) => p.provider_type === 'tcp'),
)
const promProviders = computed(() =>
  serviceProviders.value.filter((p) => p.provider_type === 'prometheus'),
)
const livenessProviders = computed(() =>
  serviceProviders.value.filter((p) => livenessSpTypes.has(p.provider_type)),
)

const httpMethodOptions = computed(() => HTTP_METHODS.map((m) => ({ value: m, label: m })))
const promOperatorOptions = computed(() => PROM_OPERATORS.map((o) => ({ value: o, label: o })))
const cliTextOperatorOptions = computed(() => CLI_TEXT_OPERATORS.map((o) => ({ value: o, label: o })))
const k8sVersionOptions = computed(() => K8S_VERSIONS.map((v) => ({ value: v, label: v })))
const k8sCheckTypeOptions = computed(() => K8S_CHECK_TYPES.map((c) => ({ value: c, label: c })))
const cliRunnerOptions = computed(() =>
  CLI_RUNNER_PRESETS.map((p) => ({
    value: p,
    label:
      p === 'toolkit'
        ? 'Telemetry toolkit (kubectl, jq, apt, …)'
        : p === 'alpine'
          ? 'Alpine (stable 3.x)'
          : 'Ubuntu 24.04',
  })),
)
const metricValueKindOptions = [
  { value: 'number', label: 'Numeric' },
  { value: 'text', label: 'Text (trimmed stdout)' },
] as const

const metricNumericValuePickOptions = [
  { value: 'first', label: 'First number' },
  { value: 'last', label: 'Last number' },
] as const

const tcpProviderSelectOptions = computed(() =>
  tcpProviders.value.map((p) => ({ value: p.id, label: p.name })),
)
const promProviderSelectOptions = computed(() =>
  promProviders.value.map((p) => ({ value: p.id, label: p.name })),
)

const livenessProviderSelectOptions = computed(() =>
  livenessProviders.value.map((p) => ({ value: p.id, label: `${p.name} (${p.provider_type})` })),
)

const hlctlProviders = computed(() => serviceProviders.value.filter((p) => p.provider_type === 'hlctl'))
const hlctlProviderSelectOptions = computed(() =>
  hlctlProviders.value.map((p) => ({ value: p.id, label: p.name })),
)

const selectedPromProvider = computed(() =>
  promProviders.value.find((p) => p.id === prom.service_provider_id),
)

const selectedLivenessProvider = computed(() =>
  livenessProviders.value.find((p) => p.id === livenessSpId.value),
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

const k8sClusterSelectOptions = computed(() =>
  k8sFilteredClusters.value.map((c) => ({ value: c.id, label: clusterLabel(c) })),
)
const cliClusterSelectOptions = computed(() =>
  cliFilteredClusters.value.map((c) => ({ value: c.id, label: clusterLabel(c) })),
)

const testPanelK8sVersion = computed(() => {
  if (adapter.value === 'kubernetes') return k8s.k8s_version
  if (adapter.value === 'liveness' && livenessSource.value === 'kubernetes') return k8s.k8s_version
  if (adapter.value === 'cli' && cli.execution_target === 'k8s_cluster') return cli.k8s_version
  return ''
})

watch(
  () => k8s.k8s_version,
  () => {
    const k8sProbe =
      adapter.value === 'kubernetes' || (adapter.value === 'liveness' && livenessSource.value === 'kubernetes')
    if (!k8sProbe) return
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
const verifyAbortController = ref<AbortController | null>(null)

function abortVerifyTest() {
  verifyAbortController.value?.abort()
}
const formError = ref('')
/** Fingerprint of the last successful full Verify (not “Test connectivity”). Save allowed only when it matches current probe inputs. */
const lastSuccessVerifyFingerprint = ref<string | null>(null)
const testRef = ref<{ setResult: (r: Record<string, unknown> | null) => void } | null>(null)

function clearVerifyPanelState() {
  lastSuccessVerifyFingerprint.value = null
  verifyBlinkTier.value = ''
  if (verifyBlinkTimer) {
    clearTimeout(verifyBlinkTimer)
    verifyBlinkTimer = null
  }
  testRef.value?.setResult(null)
}

watch(livenessSpId, () => {
  const p = livenessProviders.value.find((x) => x.id === livenessSpId.value)
  if (p?.provider_type !== 'artifactory') {
    livenessAfDownloadPath.value = ''
    livenessAfMaxBytesStr.value = ''
  }
  if (adapter.value === 'liveness') clearVerifyPanelState()
})

watch([livenessAfDownloadPath, livenessAfMaxBytesStr], () => {
  if (adapter.value === 'liveness') clearVerifyPanelState()
})

watch(adapter, (a, prev) => {
  if (prev != null && prev !== a) {
    clearVerifyPanelState()
  }
  if ((a === 'cli' || a === 'hlctl') && props.initial?.adapter !== a) {
    cliFormMode.value = 'metric'
  }
  if (!props.initial?.id) {
    telemetryTimeoutMs.value = defaultTimeoutMsForAdapter(a)
  }
})

watch(hlctlSpId, () => {
  if (adapter.value === 'hlctl') clearVerifyPanelState()
})

watch(k8sUiMode, (m) => {
  if (m === 'legacy') {
    k8sJobEnvHintId.value = ''
    k8sJobNodeSelectorHintId.value = ''
    k8sContainerPrepHintId.value = ''
    k8sProbeShellHintId.value = ''
  }
})

watch(livenessSource, (s) => {
  if (s !== 'kubernetes') {
    k8sJobEnvHintId.value = ''
    k8sJobNodeSelectorHintId.value = ''
    k8sContainerPrepHintId.value = ''
    k8sProbeShellHintId.value = ''
  }
})

watch(cliFormMode, (m) => {
  if (m === 'legacy') {
    cliProbeShellHintId.value = ''
  }
})

watch([k8sProbeEnvLines, cliProbeEnvLines, k8sJobNodeSelectorLines, cliJobNodeSelectorLines], () => {
  clearVerifyPanelState()
})

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
    k8sUiMode.value = 'shell'
    Object.assign(k8s, {
      cluster_id: '',
      k8s_version: '1.30',
      check_type: 'api_health',
      namespace: '',
      resource_name: '',
      label_selector: '',
      cli_shell: '',
      container_prep: '',
      runner: 'toolkit',
      parse_json: false,
    })
    cliFormMode.value = 'metric'
    Object.assign(cli, {
      command: 'curl',
      args: [],
      cli_shell: '',
      container_prep: '',
      runner: 'toolkit',
      parse_json: false,
      execution_target: 'backend',
      cluster_id: '',
      k8s_version: '1.30',
    })
    Object.assign(cm, {
      value_kind: 'number',
      numeric_value_pick: 'first',
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
    livenessSource.value = 'kubernetes'
    livenessSpId.value = ''
    livenessAfDownloadPath.value = ''
    livenessAfMaxBytesStr.value = ''
    livenessValueQosEnabled.value = false
    Object.assign(livVal, {
      green_operator: 'gte',
      green_threshold: 1,
      yellow_operator: 'gte',
      yellow_threshold: 0.5,
    })
    k8sProbeEnvLines.value = ''
    cliProbeEnvLines.value = ''
    k8sJobNodeSelectorLines.value = ''
    cliJobNodeSelectorLines.value = ''
    k8sJobEnvHintId.value = ''
    k8sJobNodeSelectorHintId.value = ''
    k8sContainerPrepHintId.value = ''
    k8sProbeShellHintId.value = ''
    cliJobEnvHintId.value = ''
    cliJobNodeSelectorHintId.value = ''
    cliContainerPrepHintId.value = ''
    cliProbeShellHintId.value = ''
    hlctlSpId.value = ''
    telemetryTimeoutMs.value = defaultTimeoutMsForAdapter('http')
    return
  }
  name.value = row.name
  displayName.value = row.display_name?.trim() || ''
  adapter.value = row.adapter
  telemetryTimeoutMs.value =
    typeof row.timeout_ms === 'number' && row.timeout_ms > 0
      ? clampProbeTimeoutMs(row.timeout_ms)
      : defaultTimeoutMsForAdapter(row.adapter)
  k8sProbeEnvLines.value = ''
  cliProbeEnvLines.value = ''
  k8sJobNodeSelectorLines.value = ''
  cliJobNodeSelectorLines.value = ''
  k8sJobEnvHintId.value = ''
  k8sJobNodeSelectorHintId.value = ''
  k8sContainerPrepHintId.value = ''
  k8sProbeShellHintId.value = ''
  cliJobEnvHintId.value = ''
  cliJobNodeSelectorHintId.value = ''
  cliContainerPrepHintId.value = ''
  cliProbeShellHintId.value = ''
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
    const shell = String(c.cli_shell || '').trim() !== ''
    k8sUiMode.value = shell ? 'shell' : 'legacy'
    const rawKr = String(c.runner || 'toolkit').trim()
    const runnerVal = CLI_RUNNER_PRESETS.includes(rawKr as (typeof CLI_RUNNER_PRESETS)[number])
      ? (rawKr as (typeof CLI_RUNNER_PRESETS)[number])
      : 'toolkit'
    Object.assign(k8s, {
      cluster_id: String(c.cluster_id || ''),
      k8s_version: String(c.k8s_version || '1.30'),
      check_type: String(c.check_type || 'api_health'),
      namespace: String(c.namespace || ''),
      resource_name: String(c.resource_name || ''),
      label_selector: String(c.label_selector || ''),
      cli_shell: typeof c.cli_shell === 'string' ? c.cli_shell : '',
      container_prep: typeof c.container_prep === 'string' ? c.container_prep : '',
      runner: runnerVal,
      parse_json: Boolean(c.parse_json),
    })
    k8sProbeEnvLines.value = shell
      ? formatProbeEnvLines(
          typeof c.env === 'object' && c.env !== null && !Array.isArray(c.env)
            ? (c.env as Record<string, string>)
            : undefined,
        )
      : ''
    k8sJobNodeSelectorLines.value = shell
      ? formatNodeSelectorLines(
          typeof c.node_selector === 'object' && c.node_selector !== null && !Array.isArray(c.node_selector)
            ? (c.node_selector as Record<string, string>)
            : undefined,
        )
      : ''
    if (shell) {
      k8sJobEnvHintId.value = String(c.job_env_hint_id || '')
      k8sJobNodeSelectorHintId.value = String(c.job_node_selector_hint_id || '')
      k8sContainerPrepHintId.value = String(c.container_prep_hint_id || '')
      k8sProbeShellHintId.value = String(c.probe_shell_hint_id || '')
      const vk = String(q.value_kind || 'number').toLowerCase()
      cm.value_kind = vk === 'text' ? 'text' : 'number'
      cm.green_operator = String(q.green_operator ?? (vk === 'text' ? 'eq' : 'gte'))
      cm.green_threshold = Number(q.green_threshold ?? 0)
      cm.yellow_operator = String(q.yellow_operator ?? (vk === 'text' ? 'eq' : 'gte'))
      cm.yellow_threshold = Number(q.yellow_threshold ?? 0)
      cm.green_text = String(q.green_text ?? '')
      cm.yellow_text = String(q.yellow_text ?? '')
      {
        const pick = String((q as Record<string, unknown>).numeric_value_pick ?? 'first')
          .toLowerCase()
          .trim()
        cm.numeric_value_pick = pick === 'last' ? 'last' : 'first'
      }
    }
  }
  if (row.adapter === 'liveness') {
    const src = String(c.source || '').toLowerCase().trim()
    livenessSource.value = src === 'service_provider' ? 'service_provider' : 'kubernetes'
    if (livenessSource.value === 'kubernetes') {
      const inner =
        typeof c.kubernetes === 'object' && c.kubernetes !== null
          ? (c.kubernetes as Record<string, unknown>)
          : {}
      const shell = String(inner.cli_shell || '').trim() !== ''
      k8sUiMode.value = shell ? 'shell' : 'legacy'
      const rawKr = String(inner.runner || 'toolkit').trim()
      const runnerVal = CLI_RUNNER_PRESETS.includes(rawKr as (typeof CLI_RUNNER_PRESETS)[number])
        ? (rawKr as (typeof CLI_RUNNER_PRESETS)[number])
        : 'toolkit'
      Object.assign(k8s, {
        cluster_id: String(inner.cluster_id || ''),
        k8s_version: String(inner.k8s_version || '1.30'),
        check_type: String(inner.check_type || 'api_health'),
        namespace: String(inner.namespace || ''),
        resource_name: String(inner.resource_name || ''),
        label_selector: String(inner.label_selector || ''),
        cli_shell: typeof inner.cli_shell === 'string' ? inner.cli_shell : '',
        container_prep: typeof inner.container_prep === 'string' ? inner.container_prep : '',
        runner: runnerVal,
        parse_json: Boolean(inner.parse_json),
      })
      k8sProbeEnvLines.value = shell
        ? formatProbeEnvLines(
            typeof inner.env === 'object' && inner.env !== null && !Array.isArray(inner.env)
              ? (inner.env as Record<string, string>)
              : undefined,
          )
        : ''
      k8sJobNodeSelectorLines.value = shell
        ? formatNodeSelectorLines(
            typeof inner.node_selector === 'object' &&
              inner.node_selector !== null &&
              !Array.isArray(inner.node_selector)
              ? (inner.node_selector as Record<string, string>)
              : undefined,
          )
        : ''
      if (shell) {
        k8sJobEnvHintId.value = String(inner.job_env_hint_id || '')
        k8sJobNodeSelectorHintId.value = String(inner.job_node_selector_hint_id || '')
        k8sContainerPrepHintId.value = String(inner.container_prep_hint_id || '')
        k8sProbeShellHintId.value = String(inner.probe_shell_hint_id || '')
      }
      livenessSpId.value = ''
      livenessAfDownloadPath.value = ''
      livenessAfMaxBytesStr.value = ''
    } else {
      livenessSpId.value = String(c.service_provider_id || '')
      livenessAfDownloadPath.value = ''
      livenessAfMaxBytesStr.value = ''
      const af = (c as Record<string, unknown>).artifactory
      if (af && typeof af === 'object' && !Array.isArray(af)) {
        const A = af as Record<string, unknown>
        livenessAfDownloadPath.value = String(A.download_repository_path || '')
        const mx = A.download_max_bytes
        if (typeof mx === 'number' && Number.isFinite(mx)) {
          livenessAfMaxBytesStr.value = String(Math.trunc(mx))
        }
      }
    }
  }
  if (row.adapter === 'cli' || row.adapter === 'hlctl') {
    hlctlSpId.value = row.adapter === 'hlctl' ? String(c.service_provider_id || '') : ''
    const rawRunner = String(c.runner || 'toolkit').trim()
    const runnerVal = CLI_RUNNER_PRESETS.includes(rawRunner as (typeof CLI_RUNNER_PRESETS)[number])
      ? (rawRunner as (typeof CLI_RUNNER_PRESETS)[number])
      : 'toolkit'
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
    cliProbeEnvLines.value = formatProbeEnvLines(
      typeof c.env === 'object' && c.env !== null && !Array.isArray(c.env)
        ? (c.env as Record<string, string>)
        : undefined,
    )
    cliJobNodeSelectorLines.value = formatNodeSelectorLines(
      typeof c.node_selector === 'object' && c.node_selector !== null && !Array.isArray(c.node_selector)
        ? (c.node_selector as Record<string, string>)
        : undefined,
    )
    cliJobEnvHintId.value = String(c.job_env_hint_id || '')
    cliJobNodeSelectorHintId.value = String(c.job_node_selector_hint_id || '')
    cliContainerPrepHintId.value = String(c.container_prep_hint_id || '')
    cliProbeShellHintId.value = String(c.probe_shell_hint_id || '')
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
      {
        const pick = String((q as Record<string, unknown>).numeric_value_pick ?? 'first')
          .toLowerCase()
          .trim()
        cm.numeric_value_pick = pick === 'last' ? 'last' : 'first'
      }
    }
  }
  if (row.adapter === 'http' || (row.adapter === 'kubernetes' && k8sUiMode.value === 'legacy')) {
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
  if (row.adapter === 'liveness') {
    const lat = q.latency
    if (lat && typeof lat === 'object' && !Array.isArray(lat)) {
      const L = lat as Record<string, unknown>
      const g = Number(L.green_max_ms ?? 200)
      const y = Number(L.yellow_max_ms ?? 500)
      latQos.green_max_ms = g
      latQos.yellow_max_ms = y
      let r = Number(L.red_max_ms)
      if (!Number.isFinite(r) || !Number.isInteger(r) || r <= y) {
        r = y + 500
      }
      latQos.red_max_ms = r
    } else {
      Object.assign(latQos, { green_max_ms: 1500, yellow_max_ms: 2000, red_max_ms: 3000 })
    }
    const val = q.value
    livenessValueQosEnabled.value = false
    Object.assign(livVal, {
      green_operator: 'gte',
      green_threshold: 1,
      yellow_operator: 'gte',
      yellow_threshold: 0.5,
    })
    if (val && typeof val === 'object' && !Array.isArray(val)) {
      const V = val as Record<string, unknown>
      livenessValueQosEnabled.value = true
      livVal.green_operator = String(V.green_operator ?? 'gte')
      livVal.green_threshold = Number(V.green_threshold ?? 1)
      livVal.yellow_operator = String(V.yellow_operator ?? 'gte')
      livVal.yellow_threshold = Number(V.yellow_threshold ?? 0.5)
    }
    if (livenessSource.value === 'kubernetes' && k8sUiMode.value === 'shell') {
      livenessValueQosEnabled.value = true
    }
  }
  if (row.adapter === 'cli' || row.adapter === 'hlctl') {
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

watch(
  () => props.initial,
  () => {
    loadInitial()
    void nextTick(() => {
      formSnapshotBaseline.value = captureBaselineSnapshot()
    })
  },
  { immediate: true },
)

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
      if (k8sUiMode.value === 'shell') {
        const prep = k8s.container_prep.trim()
        const base: Record<string, unknown> = {
          cluster_id: k8s.cluster_id.trim(),
          k8s_version: k8s.k8s_version,
          cli_shell: k8s.cli_shell.trim(),
          parse_json: k8s.parse_json,
          runner: k8s.runner,
        }
        if (prep) base.container_prep = prep
        const envK = probeEnvPayload(k8sProbeEnvLines.value)
        if (envK) base.env = envK
        const nsK = probeNodeSelectorPayload(k8sJobNodeSelectorLines.value)
        if (nsK) base.node_selector = nsK
        attachShellHintIds(base, {
          job_env_hint_id: k8sJobEnvHintId.value,
          job_node_selector_hint_id: k8sJobNodeSelectorHintId.value,
          container_prep_hint_id: k8sContainerPrepHintId.value,
          probe_shell_hint_id: k8sProbeShellHintId.value,
        })
        return base
      }
      return {
        cluster_id: k8s.cluster_id.trim(),
        k8s_version: k8s.k8s_version,
        check_type: k8s.check_type,
        namespace: k8s.namespace.trim() || undefined,
        resource_name: k8s.resource_name.trim() || undefined,
        label_selector: k8s.label_selector.trim() || undefined,
      }
    case 'liveness':
      if (livenessSource.value === 'kubernetes') {
        if (k8sUiMode.value === 'shell') {
          const prep = k8s.container_prep.trim()
          const nest: Record<string, unknown> = {
            cluster_id: k8s.cluster_id.trim(),
            k8s_version: k8s.k8s_version,
            cli_shell: k8s.cli_shell.trim(),
            parse_json: k8s.parse_json,
            runner: k8s.runner,
          }
          if (prep) nest.container_prep = prep
          const envK = probeEnvPayload(k8sProbeEnvLines.value)
          if (envK) nest.env = envK
          const nsK = probeNodeSelectorPayload(k8sJobNodeSelectorLines.value)
          if (nsK) nest.node_selector = nsK
          attachShellHintIds(nest, {
            job_env_hint_id: k8sJobEnvHintId.value,
            job_node_selector_hint_id: k8sJobNodeSelectorHintId.value,
            container_prep_hint_id: k8sContainerPrepHintId.value,
            probe_shell_hint_id: k8sProbeShellHintId.value,
          })
          return { source: 'kubernetes', kubernetes: nest }
        }
        return {
          source: 'kubernetes',
          kubernetes: {
            cluster_id: k8s.cluster_id.trim(),
            k8s_version: k8s.k8s_version,
            check_type: k8s.check_type,
            namespace: k8s.namespace.trim() || undefined,
            resource_name: k8s.resource_name.trim() || undefined,
            label_selector: k8s.label_selector.trim() || undefined,
          },
        }
      }
      {
        const base: Record<string, unknown> = {
          source: 'service_provider',
          service_provider_id: livenessSpId.value.trim(),
        }
        if (selectedLivenessProvider.value?.provider_type === 'artifactory') {
          const path = livenessAfDownloadPath.value.trim()
          if (path) {
            const nest: Record<string, unknown> = { download_repository_path: path }
            const rawMax = livenessAfMaxBytesStr.value.trim()
            if (rawMax !== '') {
              const n = parseInt(rawMax, 10)
              if (Number.isFinite(n) && n > 0) nest.download_max_bytes = n
            }
            base.artifactory = nest
          }
        }
        return base
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
        const envC = probeEnvPayload(cliProbeEnvLines.value)
        if (envC) base.env = envC
        const nsC = probeNodeSelectorPayload(cliJobNodeSelectorLines.value)
        if (nsC) base.node_selector = nsC
        attachShellHintIds(base, {
          job_env_hint_id: cliJobEnvHintId.value,
          job_node_selector_hint_id: cliJobNodeSelectorHintId.value,
          container_prep_hint_id: cliContainerPrepHintId.value,
          probe_shell_hint_id: cliProbeShellHintId.value,
        })
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
      const envC = probeEnvPayload(cliProbeEnvLines.value)
      if (envC) base.env = envC
      const nsC = probeNodeSelectorPayload(cliJobNodeSelectorLines.value)
      if (nsC) base.node_selector = nsC
      attachShellHintIds(base, {
        job_env_hint_id: cliJobEnvHintId.value,
        job_node_selector_hint_id: cliJobNodeSelectorHintId.value,
        container_prep_hint_id: cliContainerPrepHintId.value,
        probe_shell_hint_id: cliProbeShellHintId.value,
      })
      return base
    }
    case 'hlctl': {
      const prep = cli.container_prep.trim()
      if (cliFormMode.value === 'metric') {
        const base: Record<string, unknown> = {
          service_provider_id: hlctlSpId.value.trim(),
          cli_shell: cli.cli_shell.trim(),
          parse_json: cli.parse_json,
        }
        if (prep) base.container_prep = prep
        const envC = probeEnvPayload(cliProbeEnvLines.value)
        if (envC) base.env = envC
        attachShellHintIds(base, {
          job_env_hint_id: cliJobEnvHintId.value,
          job_node_selector_hint_id: '',
          container_prep_hint_id: cliContainerPrepHintId.value,
          probe_shell_hint_id: cliProbeShellHintId.value,
        })
        return base
      }
      const base: Record<string, unknown> = {
        service_provider_id: hlctlSpId.value.trim(),
        command: cq.green_command.trim(),
        args: splitArgs(cqGreenArgs.value),
        parse_json: cli.parse_json,
      }
      if (prep) base.container_prep = prep
      const envC = probeEnvPayload(cliProbeEnvLines.value)
      if (envC) base.env = envC
      attachShellHintIds(base, {
        job_env_hint_id: cliJobEnvHintId.value,
        job_node_selector_hint_id: '',
        container_prep_hint_id: cliContainerPrepHintId.value,
        probe_shell_hint_id: cliProbeShellHintId.value,
      })
      return base
    }
    default:
      return {}
  }
}

function buildQos(): Record<string, unknown> {
  if (adapter.value === 'http' || (adapter.value === 'kubernetes' && k8sUiMode.value === 'legacy')) {
    return {
      green_max_ms: Math.trunc(latQos.green_max_ms),
      yellow_max_ms: Math.trunc(latQos.yellow_max_ms),
      red_max_ms: Math.trunc(latQos.red_max_ms),
    }
  }
  if (adapter.value === 'kubernetes' && k8sUiMode.value === 'shell') {
    if (cm.value_kind === 'number') {
      const out: Record<string, unknown> = {
        value_kind: 'number',
        green_operator: cm.green_operator,
        green_threshold: cm.green_threshold,
        yellow_operator: cm.yellow_operator,
        yellow_threshold: cm.yellow_threshold,
      }
      if (cm.numeric_value_pick === 'last') {
        out.numeric_value_pick = 'last'
      }
      return out
    }
    return {
      value_kind: 'text',
      green_operator: cm.green_operator,
      green_text: cm.green_text,
      yellow_operator: cm.yellow_operator,
      yellow_text: cm.yellow_text,
    }
  }
  if (adapter.value === 'liveness') {
    const latency = {
      green_max_ms: Math.trunc(latQos.green_max_ms),
      yellow_max_ms: Math.trunc(latQos.yellow_max_ms),
      red_max_ms: Math.trunc(latQos.red_max_ms),
    }
    const shellK = livenessSource.value === 'kubernetes' && k8sUiMode.value === 'shell'
    if (!livenessValueQosEnabled.value && !shellK) {
      return { latency }
    }
    return {
      latency,
      value: {
        green_operator: livVal.green_operator,
        green_threshold: livVal.green_threshold,
        yellow_operator: livVal.yellow_operator,
        yellow_threshold: livVal.yellow_threshold,
      },
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
  if (adapter.value === 'cli' || adapter.value === 'hlctl') {
    if (cliFormMode.value === 'metric') {
      if (cm.value_kind === 'number') {
        const out: Record<string, unknown> = {
          value_kind: 'number',
          green_operator: cm.green_operator,
          green_threshold: cm.green_threshold,
          yellow_operator: cm.yellow_operator,
          yellow_threshold: cm.yellow_threshold,
        }
        if (cm.numeric_value_pick === 'last') {
          out.numeric_value_pick = 'last'
        }
        return out
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
  if (adapter.value === 'liveness') {
    return livenessSource.value === 'kubernetes' ? 'k8s_cluster' : 'backend'
  }
  if (adapter.value === 'http' || adapter.value === 'prometheus') return 'backend'
  if (adapter.value === 'hlctl') return 'backend'
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
  {
    const n = Number(telemetryTimeoutMs.value)
    if (!Number.isFinite(n) || n < PROBE_TIMEOUT_MS_MIN || n > PROBE_TIMEOUT_MS_MAX) {
      return `Probe timeout must be between ${PROBE_TIMEOUT_MS_MIN} and ${PROBE_TIMEOUT_MS_MAX} ms`
    }
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
    if (k8sUiMode.value === 'shell') {
      if (!k8s.cli_shell.trim()) return 'Probe shell is required for Kubernetes shell mode'
      if (cm.value_kind === 'number') {
        if (!cm.green_operator || !cm.yellow_operator) {
          return 'Kubernetes metric QoS requires green and yellow operators'
        }
      } else if (!cm.green_operator || !cm.yellow_operator) {
        return 'Kubernetes text QoS requires green and yellow operators'
      }
      const ke = parseProbeEnvLines(k8sProbeEnvLines.value)
      if (!ke.ok) return ke.error
      if (Object.keys(ke.env).length > 32) return 'Job environment: at most 32 variables'
      const kn = parseNodeSelectorLines(k8sJobNodeSelectorLines.value)
      if (!kn.ok) return kn.error
      if (Object.keys(kn.node_selector).length > 16) return 'Job node selector: at most 16 keys'
    } else if (k8s.check_type === 'deployment_ready' && !k8s.resource_name.trim()) {
      return 'Deployment name required'
    }
  }
  if (adapter.value === 'liveness') {
    if (livenessSource.value === 'kubernetes') {
      const allowed = K8S_VERSIONS as readonly string[]
      if (!allowed.includes(k8s.k8s_version)) return 'Pick a supported K8s version'
      if (!k8s.cluster_id.trim()) return 'Select a cluster'
      if (k8sUiMode.value === 'shell') {
        if (!k8s.cli_shell.trim()) return 'Probe shell is required for liveness Kubernetes shell mode'
        const ke = parseProbeEnvLines(k8sProbeEnvLines.value)
        if (!ke.ok) return ke.error
        if (Object.keys(ke.env).length > 32) return 'Job environment: at most 32 variables'
        const kn = parseNodeSelectorLines(k8sJobNodeSelectorLines.value)
        if (!kn.ok) return kn.error
        if (Object.keys(kn.node_selector).length > 16) return 'Job node selector: at most 16 keys'
      } else if (k8s.check_type === 'deployment_ready' && !k8s.resource_name.trim()) {
        return 'Deployment name required'
      }
    } else {
      if (!livenessSpId.value.trim()) {
        return 'Select a service provider for liveness'
      }
      if (selectedLivenessProvider.value?.provider_type === 'artifactory') {
        const afPath = livenessAfDownloadPath.value.trim()
        const rawMax = livenessAfMaxBytesStr.value.trim()
        if (rawMax !== '' && !afPath) {
          return 'Artifactory: set a download test path before max sample bytes'
        }
        if (afPath) {
          if (afPath.includes('..')) return 'Artifactory path must not contain ..'
          if (afPath.startsWith('/')) return 'Artifactory path must not start with /'
          if (!/^[a-zA-Z0-9._/-]+$/.test(afPath)) {
            return 'Artifactory path contains invalid characters (use letters, digits, ._-/)'
          }
          if (!livenessValueQosEnabled.value) {
            return 'Artifactory download test requires “Also evaluate probe value” (thresholds are bytes per second)'
          }
          if (!livVal.green_operator || !livVal.yellow_operator) {
            return 'Liveness value QoS requires green and yellow operators'
          }
          if (rawMax !== '') {
            const n = parseInt(rawMax, 10)
            if (!Number.isFinite(n) || n < 1 || n > ARTIFACTORY_DOWNLOAD_ABS_MAX_BYTES) {
              return `Max sample bytes must be between 1 and ${ARTIFACTORY_DOWNLOAD_ABS_MAX_BYTES}`
            }
          }
        }
      }
    }
  }
  if (adapter.value === 'cli') {
    const ce = parseProbeEnvLines(cliProbeEnvLines.value)
    if (!ce.ok) return ce.error
    if (Object.keys(ce.env).length > 32) return 'CLI environment: at most 32 variables'
    const cn = parseNodeSelectorLines(cliJobNodeSelectorLines.value)
    if (!cn.ok) return cn.error
    if (Object.keys(cn.node_selector).length > 16) return 'CLI node selector: at most 16 keys'
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
  if (adapter.value === 'hlctl') {
    if (!hlctlSpId.value.trim()) return 'Select an HLCTL service provider'
    const ce = parseProbeEnvLines(cliProbeEnvLines.value)
    if (!ce.ok) return ce.error
    if (Object.keys(ce.env).length > 32) return 'HLCTL environment: at most 32 variables'
    if (cliFormMode.value === 'metric') {
      if (!cli.cli_shell.trim()) return 'Probe shell (cli_shell) is required for metric HLCTL QoS'
      if (cm.value_kind === 'number') {
        if (!cm.green_operator || !cm.yellow_operator) {
          return 'Numeric HLCTL QoS requires green and yellow operators'
        }
      } else {
        if (!cm.green_operator || !cm.yellow_operator) {
          return 'Text HLCTL QoS requires green and yellow operators'
        }
      }
    } else if (!cq.green_command.trim() || !cq.yellow_command.trim() || !cq.red_command.trim()) {
      return 'Legacy HLCTL QoS: all three commands required (or switch to metric shell QoS)'
    }
  }
  if (
    adapter.value === 'http' ||
    (adapter.value === 'kubernetes' && k8sUiMode.value === 'legacy') ||
    adapter.value === 'liveness'
  ) {
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
  if (
    adapter.value === 'liveness' &&
    (livenessValueQosEnabled.value || (livenessSource.value === 'kubernetes' && k8sUiMode.value === 'shell'))
  ) {
    if (!livVal.green_operator || !livVal.yellow_operator) {
      return 'Liveness value QoS requires green and yellow operators'
    }
  }
  return ''
}

/** Probe timeout for Verify (clamped to backend max, see probeTestContextTimeout). */
function probeTimeoutMsForVerify(): number {
  return clampProbeTimeoutMs(Number(telemetryTimeoutMs.value))
}

function buildProbeTestBody(payload: Record<string, unknown>): Record<string, unknown> {
  const cfgRaw = buildConfig()
  const cfg =
    typeof cfgRaw === 'object' && cfgRaw !== null && !Array.isArray(cfgRaw)
      ? { ...(cfgRaw as Record<string, unknown>) }
      : {}
  cfg.timeout_ms = probeTimeoutMsForVerify()
  const body: Record<string, unknown> = {
    adapter: adapter.value,
    config_json: cfg,
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
  verifyAbortController.value = new AbortController()
  const signal = verifyAbortController.value.signal
  try {
    const body = buildProbeTestBody(payload)
    testRef.value?.clearStream?.()
    const res = (await postProbeTestStream(
      JSON.stringify(body),
      {
        onHost: (meta) => testRef.value?.setStreamHost?.(meta),
        onLog: (stream, chunk) => testRef.value?.appendStreamLog?.(stream, chunk),
        onHeartbeat: () => testRef.value?.markStreamHeartbeat?.(),
      },
      { signal },
    )) as Record<string, unknown>
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
    if (e instanceof ProbeStreamAbortedError) {
      testRef.value?.setResult({
        success: false,
        operational_ok: false,
        error: 'Verify cancelled',
      })
      if (!payload.test_connect_only) lastSuccessVerifyFingerprint.value = null
      return
    }
    testRef.value?.setResult({ success: false, error: e instanceof Error ? e.message : 'Test failed' })
    if (!payload.test_connect_only) lastSuccessVerifyFingerprint.value = null
  } finally {
    verifyAbortController.value = null
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
      timeout_ms: clampProbeTimeoutMs(Number(telemetryTimeoutMs.value)),
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
    // So @saved navigation (e.g. to list) is not blocked by onBeforeRouteLeave dirty guard.
    formSnapshotBaseline.value = captureBaselineSnapshot()
    emit('saved')
  } catch (e: unknown) {
    const apiErr = (e as { data?: { error?: string } })?.data?.error
    formError.value = apiErr || (e instanceof Error ? e.message : 'Save failed')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.telemetry-editor-page {
  width: 100%;
  padding-bottom: 32px;
  /* Match admin main / body canvas so sticky footer does not read as a white card */
  background: var(--color-canvas, var(--admin-main-bg, #f9fafb));
}
.telemetry-editor-page__header {
  margin-bottom: 20px;
}
.telemetry-editor-back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 10px;
  padding: 6px 12px 6px 8px;
  border: none;
  border-radius: var(--radius-pill);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.telemetry-editor-back:hover {
  background: var(--color-bg-secondary);
  color: var(--color-text);
}
.telemetry-editor-page__title {
  margin-bottom: 0;
}
.telemetry-editor-form {
  gap: 0;
  background: transparent;
}
.telemetry-editor-form__body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.telemetry-editor-form__actions {
  position: sticky;
  bottom: 0;
  z-index: 2;
  flex-shrink: 0;
  margin-top: 16px;
  margin-bottom: 0;
  padding-top: 16px;
  padding-bottom: 8px;
  background-color: #f9fafb;
  background: var(--color-canvas, var(--admin-main-bg, #f9fafb));
  border-top: 1px solid var(--color-border);
  box-shadow: 0 -6px 18px rgba(15, 23, 42, 0.04);
}
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
  background: var(--color-primary);
  color: #fff;
  border-color: var(--color-primary);
  cursor: pointer;
}
.btn-save--ready:hover:not(:disabled) {
  background: var(--color-primary-hover);
  border-color: var(--color-primary-hover);
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
.telemetry-shell-hint-label-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
  margin-bottom: 0.35rem;
}
.telemetry-shell-hint-label-row .form-label {
  margin-bottom: 0;
}
</style>

<!-- Unscoped: win over global .modal-actions / form defaults so sticky bar matches admin canvas -->
<style>
.telemetry-editor-form__actions.modal-actions {
  background-color: #f9fafb !important;
  background-image: linear-gradient(
    to bottom,
    var(--color-canvas, #f9fafb),
    var(--color-canvas, #f9fafb)
  );
}
</style>
