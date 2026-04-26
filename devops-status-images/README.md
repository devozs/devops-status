# DevOps Status telemetry runner images

## `Dockerfile` (telemetry toolkit)

Ubuntu-based image with `curl`, `wget`, `apt`, `jq`, `kubectl`, and `hlctl` (installed from Artifactory via `hlctl/download-latest.sh`). Used when telemetry selects the **Telemetry toolkit** runner preset (mapped server-side to `CLI_RUNNER_IMAGE_TOOLKIT`).

### Multi-cluster `kubectl`

The probe Job runs **on** the cluster you pick in the UI; credentials for **that** cluster are injected by the backend for API access. To run `kubectl` against **other** clusters, provide a kubeconfig inside the Job:

1. **Environment variables** (recommended): add a variable such as `REMOTE_KUBECONFIG_B64` (base64-encoded kubeconfig) in the probe’s **Job environment** in the telemetry form, then in **Container prep** run:

   ```sh
   mkdir -p /root/.kube
   echo "$REMOTE_KUBECONFIG_B64" | base64 -d > /root/.kube/config
   export KUBECONFIG=/root/.kube/config
   ```

2. **Multiple files**: set `KUBECONFIG` to a colon-separated list, e.g. `KUBECONFIG=/tmp/a:/tmp/b`, then use `kubectl --context=...`.

Values in `env` are stored in the database and visible to admins via the API; prefer short-lived credentials where possible.

**Verify (admin UI)** uses the same cluster credentials and RBAC as scheduled probes; live log streaming does not change what the API is allowed to run on the cluster.

Long-running HLCTL/CLI probes (several minutes) are bounded by the telemetry **timeout** (`timeout_ms` on the telemetry row; Verify injects it into the probe request). If the browser still drops the stream with a network error, raise **ingress** / **proxy** read timeouts for the management API (e.g. nginx `proxy_read_timeout`) above your worst-case probe duration.

### Build locally

From repo root:

```bash
docker build -f devops-status/devops-status-images/Dockerfile -t devops-status-telemetry-toolkit:local devops-status
```
