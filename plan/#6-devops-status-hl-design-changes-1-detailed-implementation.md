---

name: HL Design Changes 1 (Resources, Entities, Screens)
overview: Introduce Service Provider; unify Operational and QoS into a single data source model with combined verify UX; fold probe bindings into Environment and Service with typed data-source links; remove standalone bindings. No backward compatibility—clean migrations only.
todos:

- id: db-service-providers
  content: "Migration 006: create service_providers (host, port, optional name/metadata); add services.service_provider_id FK; seed or document admin flow"
  status: pending
- id: db-unify-datasources
  content: "Migration 006: collapse data_sources to unified model—drop ds_type distinction (single CHECK or rename to unified), adjust unique index to name-only or unified type; migrate validation rules"
  status: pending
- id: db-env-cluster-link
  content: "Migration 006: add environments.k8s_cluster_id UUID FK nullable → k8s_clusters(id); enforce at app layer that k8s envs reference a cluster; keep k8s_clusters.environment_id if still needed for onboarding or reconcile to single direction"
  status: pending
- id: db-junction-tables
  content: "Migration 006: create environment_data_source_links and service_data_source_links (interval, window, failures_to_down, success_to_recover); drop service_probe_bindings and environment_probe_bindings"
  status: pending
- id: be-models-store
  content: "Update model.Service, model.Environment, new ServiceProvider; replace binding structs with link structs; rewrite store queries (list/create/update/delete links)"
  status: pending
- id: be-handlers-routes
  content: "Remove or narrow /api/admin/bindings routes; extend environments and services handlers for nested link CRUD; add service_providers CRUD"
  status: pending
- id: be-scheduler-unified-probe
  content: "Scheduler loads links instead of bindings; one execution per link—insert operational sample then derive QoS from same ProbeResult when thresholds present and operational success; adjust bindingKey to link id or (target, data_source_id)"
  status: pending
- id: be-probetest-verify
  content: "probetest: always run connectivity then QoS classification in one response; return operational ok + qos_level + latency per adapter"
  status: pending
- id: be-validation-ds
  content: "datasource_validation: require qos_thresholds shape for unified DS (all adapters); operational-only paths removed"
  status: pending
- id: fe-datasource-modal
  content: "DatasourceFormModal: remove ds_type toggle; single form with QoS threshold bands; Verify triggers blink/background on green_max/yellow_max/red_max regions per hit"
  status: pending
- id: fe-datasource-types
  content: "types/datasource.ts: drop DSType split or map to unified; align API contracts"
  status: pending
- id: fe-env-service-pages
  content: "Admin environments/services pages: edit cluster picker, embedded data source multi-select with per-row interval/window/failures/recover; service provider picker for services"
  status: pending
- id: fe-remove-bindings
  content: "Remove bindings.vue, sidebar link, dashboard bindings card; update useAdminDashboardData to surface link counts from env/service APIs"
  status: pending
- id: fe-data-sources-table
  content: "data-sources.vue table: drop ds_type column or show unified label; update duplicate-name logic (no longer per-type)"
  status: pending
- id: public-status-contract
  content: "Confirm public status API still resolves operational + qos from samples; adjust queries if probe_kind emission changes"
  status: pending

isProject: false

---

# HL Design Changes 1 — Detailed Implementation Plan

## Inputs

- High-level source: [plan/#5-devops-status-hl-design-changes-1.md](./#5-devops-status-hl-design-changes-1.md)
- **Kubernetes clusters:** explicitly out of scope for schema changes (working as designed).
- **Backward compatibility:** none—delete old rows, drop old tables, no data migration scripts.

## Goals (from HL)

1. **Service Provider (new):** entity with **host** and **port** (and minimal identity fields as needed for admin UX).
2. **Data sources:** merge **Operational** and **QoS** into one resource (same URL/config); QoS is meaningful only when the check is operationally successful.
3. **Verify in Create/Edit modal:** test URL/connectivity as today, then evaluate **QoS latency** (or adapter-specific QoS) against user-defined thresholds; **visual feedback**—background colors for green / yellow / red max bands and a **brief blink** on the band that matches each verify result (per adapter semantics).
4. **Environment:** absorb **bindings**—environment references a **K8s cluster**; environment references **one or more** data sources that are **K8s-related** (`kubernetes` adapter, and any other types you classify as “environment/K8s” in product rules), each with **Interval (sec), Window, Failures to Down, Success to Recover**.
5. **Service:** absorb **bindings**—service references a **Service Provider**; service references **one or more** data sources that are **service-related** (e.g. `http`, `prometheus`, `cli` where product rules say “service scope”), each with the same four probe-tuning fields.
6. **Remove** standalone **Bindings** admin surface and `service_probe_bindings` / `environment_probe_bindings`.

---

## Current State (codebase anchors)

| Area | Location |
|------|----------|
| Binding tables | `devops-status-dev/db/migrations/001_initial_schema.up.sql` (`service_probe_bindings`, `environment_probe_bindings`) |
| Binding store | `devops-status-be/internal/store/bindings.go` |
| Binding handlers | `devops-status-be/internal/handler/admin/bindings.go`, routes under `/api/admin/bindings` in `devops-status-be/internal/server/routes.go` |
| Scheduler dispatch | `devops-status-be/internal/scheduler/scheduler.go` (`ListServiceProbeBindings` / `ListEnvironmentProbeBindings`, `ProbeKind` on `ProbeJob`) |
| Data source model | `devops-status-be/internal/model/models.go` (`DataSource`, `DSType` operational vs qos) |
| FE bindings UI | `devops-status-fe/pages/admin/bindings.vue`, `layouts/admin.vue`, `pages/admin/index.vue`, `composables/useAdminDashboardData.ts` |
| DS form / verify | `devops-status-fe/components/datasource/DatasourceFormModal.vue`, `DatasourceTestPanel.vue`, `devops-status-be/internal/handler/admin/probetest.go` |

---

## 1. Domain Model (target)

### 1.1 Service Provider

- Table `service_providers`: at least `id`, `host`, `port`, timestamps; optional `name` for admin listing.
- Table `services`: add `service_provider_id UUID REFERENCES service_providers(id) ON DELETE SET NULL` (or `RESTRICT` if every service must have a provider—product choice).

### 1.2 Unified Data Source

- **Remove** the `operational` vs `qos` **split** at the data-source row level: one row describes the probe configuration; **`qos_thresholds`** holds green/yellow/red rules (already JSONB in migration 004).
- **Uniqueness:** today `data_sources_name_ds_type_uidx` allows the same name for operational and qos. After unification, use **unique `name`** (or `name` + `adapter` if you need two adapters with the same label—prefer one name per DS).
- **Semantics:**  
  - **Operational outcome** = adapter `Success` / connectivity.  
  - **QoS level** = derived from the same probe result **when** operational success (latency, Prometheus scalar, CLI triple, etc.), matching existing logic in `internal/engine/qos_eval.go` and `probetest.go` (`computeQoSLevel`).

### 1.3 Environment

- **Primary K8s cluster:** add `environments.k8s_cluster_id` (nullable for non-K8s env types, or required when `env_type` implies K8s—enforce in handler validation).
- **Note:** `k8s_clusters.environment_id` already links clusters to an environment. Reconcile so there is a single clear direction (either cluster owns env membership only, or env points to “primary” cluster for probes). Recommendation: keep `k8s_clusters.environment_id` for onboarding/registry; set `environments.k8s_cluster_id` to the **default cluster for probes** and validate consistency (cluster’s `environment_id` = this environment).

### 1.4 Environment ↔ Data Source links (replaces environment bindings)

New table, for example `environment_data_source_links`:

| Column | Purpose |
|--------|---------|
| `id` | PK |
| `environment_id` | FK → environments |
| `data_source_id` | FK → data_sources |
| `sample_interval_sec`, `window_size`, `consecutive_failures_to_down`, `consecutive_success_to_recover` | Same as current binding columns |
| timestamps | created/updated |

**Constraint (application or DB):** linked `data_sources.adapter` must be allowed for environments (e.g. `kubernetes` only, unless product expands).

### 1.5 Service ↔ Data Source links (replaces service bindings)

New table `service_data_source_links` with the same tuning columns as above.

**Constraint:** linked adapter must be service-scoped (`http`, `prometheus`, `cli`, … per product rules).

### 1.6 Drop old bindings

- `DROP TABLE service_probe_bindings;`  
- `DROP TABLE environment_probe_bindings;`  
- After cutover, remove `probe_kind` from scheduling for **per-link** jobs (see §3).

---

## 2. Database Migration Strategy (006)

**Single new migration pair** (e.g. `006_hl_design_changes_1.up.sql` / `.down.sql`):

1. Truncate or delete dependent rows in development (samples, links, etc.) as needed—**no** legacy preservation.
2. Create `service_providers`; alter `services`.
3. Add `environments.k8s_cluster_id`.
4. Create `environment_data_source_links` and `service_data_source_links`.
5. Alter `data_sources`: remove `ds_type` column and check; update unique index to `UNIQUE(name)`; default or backfill not needed if DB is wiped.
6. Drop `service_probe_bindings` and `environment_probe_bindings`.

**Down migration** (optional): recreate empty old binding tables and old `data_sources.ds_type` only if you maintain reversible migrations; otherwise minimal down is acceptable for dev.

---

## 3. Backend — Scheduler and Samples

### 3.1 Job model

- Replace lists from `ListServiceProbeBindings` / `ListEnvironmentProbeBindings` with `ListServiceDataSourceLinks` / `ListEnvironmentDataSourceLinks`.
- **Dispatch key:** e.g. `(target_type, target_id, data_source_id)` or link `id`—must be unique per scheduled probe.
- **Probe kind:** Prefer **one physical probe per link** that:
  1. Runs the adapter once.
  2. Writes an **operational** sample (`probe_kind = 'operational'`) with `success` / latency / metadata.
  3. If `success` and `qos_thresholds` is non-empty, compute QoS and write a **second** sample with `probe_kind = 'qos'` **or** attach `qos_level` to metadata and let evaluator read a single row—**decision:** mirror current public API expectations (`GetRecentSamples(..., 'operational'|'qos')`). Easiest path: **two inserts** from one execution when thresholds exist; if probe fails operational, optionally insert qos as `red` or skip qos row—align with HL (“QoS relevant if operational”).

### 3.2 CLI QoS

- Today `scheduler.go` branches on `job.ProbeKind == "qos"` for CLI triple probe. After unification, branch on **adapter == cli** and **non-empty qos_thresholds** with CLI shape, independent of removed `ProbeKind` on the job.

### 3.3 Status evaluator

- `internal/engine/status.go` continues to evaluate windows per `probe_kind` if two rows are stored; or simplify to one code path if samples are merged—**implementation choice** documented in the same PR as scheduler changes.

---

## 4. Backend — API

### 4.1 Service providers

- `GET/POST /api/admin/service-providers`, `GET/PATCH/DELETE /api/admin/service-providers/{id}` (or nest under services—prefer top-level resource for reuse).

### 4.2 Services

- Extend create/update payloads with `service_provider_id` and optional `data_source_links: [{ data_source_id, sample_interval_sec, ... }]` for replace-semantics or granular sub-routes:
  - `POST /api/admin/services/{id}/data-source-links`
  - `PATCH/DELETE` per link id.

### 4.3 Environments

- Extend create/update with `k8s_cluster_id` and nested or sub-resource **environment data source links** (same shape as service links).

### 4.4 Bindings removal

- Remove `/api/admin/bindings/*` handlers and routes; delete `bindings.go` or gut it.
- Audit log entity types: add new entity strings for links and service providers.

### 4.5 Data sources API

- Remove `ds_type` from create/update where applicable; update `CheckNameAvailable` query params (name only).
- `datasource_validation.go`: unified rules—**always** validate `qos_thresholds` when product requires bands for verify UX; keep adapter-specific checks.

### 4.6 Probe test

- `probetest.go`: single verify response including:
  - `operational_ok` (boolean)
  - `qos_level` (when thresholds and successful probe)
  - `latency_ms` / adapter-specific metrics  
  So the FE can drive **green/yellow/red** blink without two round-trips.

---

## 5. Frontend

### 5.1 Data source modal

- Remove Operational vs QoS type selector; always show **QoS threshold** UI (latency bands with **green / yellow / red max** backgrounds).
- **Verify:** on response, apply CSS animation (e.g. short pulse) on the matching threshold row/section; ensure **per-adapter** behavior (HTTP/K8s latency vs Prometheus vs CLI triple) matches BE response fields.

### 5.2 Types and pages

- `types/datasource.ts`: remove `DSType` or reduce to a single literal.
- `pages/admin/data-sources.vue`: column and duplicate-name behavior without `ds_type`.
- **Environments** admin page: form sections for **cluster** and **linked K8s data sources** with the four numeric fields per link.
- **Services** admin page: **Service provider** (host/port) selector/create shortcut; **linked service data sources** with the same four fields.

### 5.3 Remove bindings surface

- Delete or redirect `pages/admin/bindings.vue`.
- Remove sidebar entry and dashboard bindings widget; replace with summary lines from env/service link counts if still useful.

---

## 6. Public Status and OpenAPI

- Verify `internal/handler/public/status.go` and any FE consumers still receive **operational** and **qos** aggregates per service/environment after sample write changes.
- Regenerate or hand-update OpenAPI if the repo tracks it.

---

## 7. Testing Checklist

- Create service provider → attach to service → add service data source link → scheduler runs and samples appear.
- Create environment with `k8s_cluster_id` → add kubernetes DS link → probes run with merged kubeconfig path unchanged.
- Data source modal: verify blink and colors for green/yellow/red across at least HTTP and one other adapter.
- Confirm bindings routes return 404 and no FE references remain (`grep` for `bindings`).

---

## 8. Ordering of Work (suggested)

1. Migration 006 + models + store (providers, links, unified DS column drop).
2. Handlers and routes (new resources, remove bindings).
3. Scheduler + evaluator + probetest alignment.
4. FE: types, data source modal UX, env/service forms, remove bindings pages.
5. Public status smoke test and dashboard composable update.

This sequence avoids a long window where the API and DB disagree.
