package public

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/model"
	"github.com/devops-status/be/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type StatusHandler struct {
	store *store.Store
}

func NewStatusHandler(s *store.Store) *StatusHandler {
	return &StatusHandler{store: s}
}

type publicIncidentStub struct {
	ID                string     `json:"id"`
	Title             string     `json:"title"`
	Severity          string     `json:"severity"`
	Status            string     `json:"status"`
	StartedAt         time.Time  `json:"started_at"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`
	SourceTelemetryID *string    `json:"source_telemetry_id,omitempty"`
}

type serviceStatus struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Slug            string                 `json:"slug"`
	Description     string                 `json:"description"`
	Status          string                 `json:"status"`
	UptimePct       float64                `json:"uptime_pct"`
	Days            []store.DailyRollup    `json:"days"`
	RecentIncidents []publicIncidentStub   `json:"recent_incidents"`
}

type envStatus struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Slug            string                 `json:"slug"`
	Description     string                 `json:"description"`
	EnvType         string                 `json:"env_type"`
	Status          string                 `json:"status"`
	UptimePct       float64                `json:"uptime_pct"`
	Days            []store.DailyRollup    `json:"days"`
	RecentIncidents []publicIncidentStub   `json:"recent_incidents"`
}

type incidentTargetKey struct {
	typ string
	id  uuid.UUID
}

func recentStubsFor(m map[incidentTargetKey][]publicIncidentStub, typ string, id uuid.UUID) []publicIncidentStub {
	s := m[incidentTargetKey{typ: typ, id: id}]
	if s == nil {
		return []publicIncidentStub{}
	}
	return s
}

func incidentToPublicStub(inc model.Incident) publicIncidentStub {
	st := publicIncidentStub{
		ID:        inc.ID.String(),
		Title:     inc.Title,
		Severity:  inc.Severity,
		Status:    inc.Status,
		StartedAt: inc.StartedAt,
		ResolvedAt: inc.ResolvedAt,
	}
	if inc.SourceTelemetryID != nil {
		s := inc.SourceTelemetryID.String()
		st.SourceTelemetryID = &s
	}
	return st
}

func (h *StatusHandler) Summary(w http.ResponseWriter, r *http.Request) {
	services, err := h.store.ListServices(r.Context(), true)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load services")
		return
	}

	environments, err := h.store.ListEnvironments(r.Context(), true)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load environments")
		return
	}

	overallStatus := "operational"
	overallMessage := "All Systems Operational"

	since := time.Now().UTC().AddDate(0, 0, -90)
	svcIDs := make([]uuid.UUID, 0, len(services))
	for _, svc := range services {
		svcIDs = append(svcIDs, svc.ID)
	}
	envIDs := make([]uuid.UUID, 0, len(environments))
	for _, env := range environments {
		envIDs = append(envIDs, env.ID)
	}
	allIncidents, err := h.store.ListIncidentsForPublicTargets(r.Context(), svcIDs, envIDs, since)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load incidents")
		return
	}
	incByTarget := make(map[incidentTargetKey][]publicIncidentStub)
	for _, inc := range allIncidents {
		k := incidentTargetKey{typ: inc.TargetType, id: inc.TargetID}
		incByTarget[k] = append(incByTarget[k], incidentToPublicStub(inc))
	}

	var svcStatuses []serviceStatus
	for _, svc := range services {
		rollups, _ := h.store.GetDailyRollups(r.Context(), "service", svc.ID, 90, nil)
		status := "operational"
		uptimePct := 100.0

		if len(rollups) > 0 {
			total := 0.0
			for _, d := range rollups {
				total += d.AvailabilityPct
			}
			uptimePct = total / float64(len(rollups))
		}

		recent, _ := h.store.GetRecentSamples(r.Context(), "service", svc.ID, "operational", 3)
		if len(recent) > 0 && !recent[0].Success {
			status = "disruption"
			overallStatus = "disruption"
			overallMessage = "Disruption with some DevOps services"
		}

		svcStatuses = append(svcStatuses, serviceStatus{
			ID: svc.ID.String(), Name: svc.Name, Slug: svc.Slug,
			Description: svc.Description, Status: status, UptimePct: uptimePct, Days: rollups,
			RecentIncidents: recentStubsFor(incByTarget, "service", svc.ID),
		})
	}

	var envStatuses []envStatus
	for _, env := range environments {
		rollups, _ := h.store.GetDailyRollups(r.Context(), "environment", env.ID, 90, nil)
		status := "operational"
		uptimePct := 100.0

		if len(rollups) > 0 {
			total := 0.0
			for _, d := range rollups {
				total += d.AvailabilityPct
			}
			uptimePct = total / float64(len(rollups))
		}

		recent, _ := h.store.GetRecentSamples(r.Context(), "environment", env.ID, "operational", 3)
		if len(recent) > 0 && !recent[0].Success {
			status = "disruption"
			overallStatus = "disruption"
			overallMessage = "Disruption with some DevOps services"
		}

		envStatuses = append(envStatuses, envStatus{
			ID: env.ID.String(), Name: env.Name, Slug: env.Slug,
			Description: env.Description, EnvType: env.EnvType,
			Status: status, UptimePct: uptimePct, Days: rollups,
			RecentIncidents: recentStubsFor(incByTarget, "environment", env.ID),
		})
	}

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"status":       overallStatus,
		"message":      overallMessage,
		"services":     svcStatuses,
		"environments": envStatuses,
	})
}

type telemetryBreakdownRow struct {
	TelemetryID            string              `json:"telemetry_id"`
	Name                   string              `json:"name"`
	DisplayName            string              `json:"display_name,omitempty"`
	Adapter                string              `json:"adapter"`
	Status                 string              `json:"status"`
	UptimePct              float64             `json:"uptime_pct"`
	Days                   []store.DailyRollup `json:"days"`
	HasPerTelemetrySamples bool                `json:"has_per_telemetry_samples"`
}

// EnvironmentTelemetryBreakdown returns per-linked-telemetry operational rollups for a public environment (404 if missing or not public).
func (h *StatusHandler) EnvironmentTelemetryBreakdown(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	env, err := h.store.GetEnvironmentBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			handler.WriteError(w, http.StatusNotFound, "environment not found")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to load environment")
		return
	}
	if !env.IsPublic {
		handler.WriteError(w, http.StatusNotFound, "environment not found")
		return
	}

	links, err := h.store.ListEnvironmentTelemetryLinks(r.Context(), &env.ID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load telemetry links")
		return
	}

	rows := make([]telemetryBreakdownRow, 0, len(links))
	for _, link := range links {
		tel, err := h.store.GetTelemetryByID(r.Context(), link.TelemetryID)
		if err != nil {
			handler.WriteError(w, http.StatusInternalServerError, "failed to load telemetry")
			return
		}
		recent, _ := h.store.ListRecentSamplesForEnvironmentTelemetry(r.Context(), env.ID, link.TelemetryID, "operational", 1)
		hasSamples := len(recent) > 0

		row := telemetryBreakdownRow{
			TelemetryID:            link.TelemetryID.String(),
			Name:                   tel.Name,
			Adapter:                tel.Adapter,
			HasPerTelemetrySamples: hasSamples,
			Status:                 "operational",
			UptimePct:              100,
		}
		if d := strings.TrimSpace(tel.DisplayName); d != "" {
			row.DisplayName = d
		}
		if hasSamples {
			rollups, _ := h.store.GetDailyRollupsForEnvironmentTelemetry(r.Context(), env.ID, link.TelemetryID, 90)
			row.Days = rollups
			if len(rollups) > 0 {
				var total float64
				for _, d := range rollups {
					total += d.AvailabilityPct
				}
				row.UptimePct = total / float64(len(rollups))
			}
			if len(recent) > 0 && !recent[0].Success {
				row.Status = "disruption"
			}
		}
		rows = append(rows, row)
	}

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"environment": map[string]string{
			"id":   env.ID.String(),
			"name": env.Name,
			"slug": env.Slug,
		},
		"telemetries": rows,
	})
}

// ServiceTelemetryBreakdown returns per-linked-telemetry operational rollups for a public service (404 if missing or not public).
func (h *StatusHandler) ServiceTelemetryBreakdown(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	svc, err := h.store.GetServiceBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			handler.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to load service")
		return
	}
	if !svc.IsPublic {
		handler.WriteError(w, http.StatusNotFound, "service not found")
		return
	}

	links, err := h.store.ListServiceTelemetryLinks(r.Context(), &svc.ID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load telemetry links")
		return
	}

	rows := make([]telemetryBreakdownRow, 0, len(links))
	for _, link := range links {
		tel, err := h.store.GetTelemetryByID(r.Context(), link.TelemetryID)
		if err != nil {
			handler.WriteError(w, http.StatusInternalServerError, "failed to load telemetry")
			return
		}
		recent, _ := h.store.ListRecentSamplesForServiceTelemetry(r.Context(), svc.ID, link.TelemetryID, "operational", 1)
		hasSamples := len(recent) > 0

		row := telemetryBreakdownRow{
			TelemetryID:            link.TelemetryID.String(),
			Name:                   tel.Name,
			Adapter:                tel.Adapter,
			HasPerTelemetrySamples: hasSamples,
			Status:                 "operational",
			UptimePct:              100,
		}
		if d := strings.TrimSpace(tel.DisplayName); d != "" {
			row.DisplayName = d
		}
		if hasSamples {
			rollups, _ := h.store.GetDailyRollupsForServiceTelemetry(r.Context(), svc.ID, link.TelemetryID, 90)
			row.Days = rollups
			if len(rollups) > 0 {
				var total float64
				for _, d := range rollups {
					total += d.AvailabilityPct
				}
				row.UptimePct = total / float64(len(rollups))
			}
			if len(recent) > 0 && !recent[0].Success {
				row.Status = "disruption"
			}
		}
		rows = append(rows, row)
	}

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"service": map[string]string{
			"id":   svc.ID.String(),
			"name": svc.Name,
			"slug": svc.Slug,
		},
		"telemetries": rows,
	})
}

// ServiceTopology returns infra + linked telemetries for a public service (404 if missing or not public).
func (h *StatusHandler) ServiceTopology(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	svc, err := h.store.GetServiceBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			handler.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to load service")
		return
	}
	if !svc.IsPublic {
		handler.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	topo, err := h.store.BuildResourceTopologyForService(r.Context(), svc.ID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load topology")
		return
	}
	handler.WriteJSON(w, http.StatusOK, topo)
}

// EnvironmentTopology returns infra + linked telemetries for a public environment (404 if missing or not public).
func (h *StatusHandler) EnvironmentTopology(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	env, err := h.store.GetEnvironmentBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			handler.WriteError(w, http.StatusNotFound, "environment not found")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to load environment")
		return
	}
	if !env.IsPublic {
		handler.WriteError(w, http.StatusNotFound, "environment not found")
		return
	}
	topo, err := h.store.BuildResourceTopologyForEnvironment(r.Context(), env.ID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load topology")
		return
	}
	handler.WriteJSON(w, http.StatusOK, topo)
}

// parsePublicTelemetrySamplesQuery reads telemetry_id, probe_kind, and limit from the request (same rules as admin telemetry-samples).
func parsePublicTelemetrySamplesQuery(r *http.Request) (telemetryID uuid.UUID, probeKind string, limit int, errMsg string) {
	telStr := strings.TrimSpace(r.URL.Query().Get("telemetry_id"))
	if telStr == "" {
		return uuid.Nil, "", 0, "telemetry_id is required"
	}
	tid, err := uuid.Parse(telStr)
	if err != nil {
		return uuid.Nil, "", 0, "invalid telemetry_id"
	}
	pk := strings.TrimSpace(r.URL.Query().Get("probe_kind"))
	if pk == "" {
		pk = "operational"
	}
	if pk != "operational" && pk != "qos" {
		return uuid.Nil, "", 0, "probe_kind must be operational or qos"
	}
	lim := 50
	if ls := strings.TrimSpace(r.URL.Query().Get("limit")); ls != "" {
		n, err := strconv.Atoi(ls)
		if err != nil || n < 1 {
			return uuid.Nil, "", 0, "invalid limit"
		}
		lim = n
	}
	return tid, pk, lim, ""
}

// ServiceTelemetrySamples returns recent sample_results for one linked telemetry on a public service (404 if missing or not public).
func (h *StatusHandler) ServiceTelemetrySamples(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	svc, err := h.store.GetServiceBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			handler.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to load service")
		return
	}
	if !svc.IsPublic {
		handler.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	telemetryID, probeKind, limit, errMsg := parsePublicTelemetrySamplesQuery(r)
	if errMsg != "" {
		handler.WriteError(w, http.StatusBadRequest, errMsg)
		return
	}
	links, err := h.store.ListServiceTelemetryLinks(r.Context(), &svc.ID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load telemetry links")
		return
	}
	linked := false
	for _, l := range links {
		if l.TelemetryID == telemetryID {
			linked = true
			break
		}
	}
	if !linked {
		handler.WriteError(w, http.StatusForbidden, "telemetry is not linked to this service")
		return
	}
	samples, err := h.store.ListRecentSamplesForServiceTelemetry(r.Context(), svc.ID, telemetryID, probeKind, limit)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list samples")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]any{"samples": samples})
}

// EnvironmentTelemetrySamples returns recent sample_results for one linked telemetry on a public environment (404 if missing or not public).
func (h *StatusHandler) EnvironmentTelemetrySamples(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	env, err := h.store.GetEnvironmentBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			handler.WriteError(w, http.StatusNotFound, "environment not found")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to load environment")
		return
	}
	if !env.IsPublic {
		handler.WriteError(w, http.StatusNotFound, "environment not found")
		return
	}
	telemetryID, probeKind, limit, errMsg := parsePublicTelemetrySamplesQuery(r)
	if errMsg != "" {
		handler.WriteError(w, http.StatusBadRequest, errMsg)
		return
	}
	links, err := h.store.ListEnvironmentTelemetryLinks(r.Context(), &env.ID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load telemetry links")
		return
	}
	linked := false
	for _, l := range links {
		if l.TelemetryID == telemetryID {
			linked = true
			break
		}
	}
	if !linked {
		handler.WriteError(w, http.StatusForbidden, "telemetry is not linked to this environment")
		return
	}
	samples, err := h.store.ListRecentSamplesForEnvironmentTelemetry(r.Context(), env.ID, telemetryID, probeKind, limit)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list samples")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]any{"samples": samples})
}

func (h *StatusHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	services, err := h.store.ListServices(r.Context(), true)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load services")
		return
	}
	handler.WriteJSON(w, http.StatusOK, services)
}

func (h *StatusHandler) ListEnvironments(w http.ResponseWriter, r *http.Request) {
	environments, err := h.store.ListEnvironments(r.Context(), true)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load environments")
		return
	}
	handler.WriteJSON(w, http.StatusOK, environments)
}
