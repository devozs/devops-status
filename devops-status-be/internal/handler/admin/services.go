package admin

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/model"
	"github.com/devops-status/be/internal/store"
)

type ServicesHandler struct {
	store *store.Store
}

func NewServicesHandler(s *store.Store) *ServicesHandler {
	return &ServicesHandler{store: s}
}

func (h *ServicesHandler) List(w http.ResponseWriter, r *http.Request) {
	services, err := h.store.ListServices(r.Context(), false)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list services")
		return
	}
	handler.WriteJSON(w, http.StatusOK, services)
}

// CheckSlugAvailable reports whether slug is free for create or update (optional exclude_id).
// Slugs shorter than 3 characters return available: true without hitting the DB.
func (h *ServicesHandler) CheckSlugAvailable(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	slug := strings.TrimSpace(q.Get("slug"))
	if len(slug) < 3 {
		handler.WriteJSON(w, http.StatusOK, map[string]any{"available": true, "skipped": true})
		return
	}
	var exclude *uuid.UUID
	if ex := strings.TrimSpace(q.Get("exclude_id")); ex != "" {
		id, err := uuid.Parse(ex)
		if err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid exclude_id")
			return
		}
		exclude = &id
	}
	taken, err := h.store.ServiceSlugExists(r.Context(), slug, exclude)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to check slug")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]bool{"available": !taken})
}

type serviceDetailResponse struct {
	model.Service
	TelemetryLinks []model.ServiceTelemetryLink `json:"telemetry_links"`
}

func (h *ServicesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	svc, err := h.store.GetServiceByID(r.Context(), id)
	if err != nil {
		handler.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	links, err := h.store.ListServiceTelemetryLinks(r.Context(), &id)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load telemetry links")
		return
	}
	handler.WriteJSON(w, http.StatusOK, serviceDetailResponse{Service: *svc, TelemetryLinks: links})
}

// Topology returns linked infra and telemetries for the resource map UI.
func (h *ServicesHandler) Topology(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}
	topo, err := h.store.BuildResourceTopologyForService(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			handler.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to load topology")
		return
	}
	handler.WriteJSON(w, http.StatusOK, topo)
}

// ListTelemetrySamples returns recent sample_results for one service telemetry link (query: telemetry_id, optional probe_kind, limit).
func (h *ServicesHandler) ListTelemetrySamples(w http.ResponseWriter, r *http.Request) {
	svcID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}
	telStr := strings.TrimSpace(r.URL.Query().Get("telemetry_id"))
	if telStr == "" {
		handler.WriteError(w, http.StatusBadRequest, "telemetry_id is required")
		return
	}
	telemetryID, err := uuid.Parse(telStr)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid telemetry_id")
		return
	}
	probeKind := strings.TrimSpace(r.URL.Query().Get("probe_kind"))
	if probeKind == "" {
		probeKind = "operational"
	}
	if probeKind != "operational" && probeKind != "qos" {
		handler.WriteError(w, http.StatusBadRequest, "probe_kind must be operational or qos")
		return
	}
	limit := 50
	if ls := strings.TrimSpace(r.URL.Query().Get("limit")); ls != "" {
		n, err := strconv.Atoi(ls)
		if err != nil || n < 1 {
			handler.WriteError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = n
	}

	if _, err := h.store.GetServiceByID(r.Context(), svcID); err != nil {
		handler.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	links, err := h.store.ListServiceTelemetryLinks(r.Context(), &svcID)
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

	samples, err := h.store.ListRecentSamplesForServiceTelemetry(r.Context(), svcID, telemetryID, probeKind, limit)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list samples")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]any{"samples": samples})
}

func (h *ServicesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input store.CreateServiceInput
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Name == "" || input.Slug == "" {
		handler.WriteError(w, http.StatusBadRequest, "name and slug are required")
		return
	}

	svc, err := h.store.CreateService(r.Context(), input)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to create service")
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "create", "service", &svc.ID, input, r.RemoteAddr)

	handler.WriteJSON(w, http.StatusCreated, svc)
}

func (h *ServicesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	var input store.UpdateServiceInput
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	svc, err := h.store.UpdateService(r.Context(), id, input)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to update service")
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "update", "service", &svc.ID, input, r.RemoteAddr)

	handler.WriteJSON(w, http.StatusOK, svc)
}

func (h *ServicesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	if err := h.store.DeleteService(r.Context(), id); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to delete service")
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "delete", "service", &id, nil, r.RemoteAddr)

	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

var serviceScopedAdapters = []string{"http", "prometheus", "cli", "liveness"}

func (h *ServicesHandler) CreateTelemetryLink(w http.ResponseWriter, r *http.Request) {
	svcID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}
	if _, err := h.store.GetServiceByID(r.Context(), svcID); err != nil {
		handler.WriteError(w, http.StatusNotFound, "service not found")
		return
	}
	var input store.CreateTelemetryLinkInput
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if input.TelemetryID == uuid.Nil {
		handler.WriteError(w, http.StatusBadRequest, "telemetry_id is required")
		return
	}
	t, err := h.store.GetTelemetryByID(r.Context(), input.TelemetryID)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "telemetry not found")
		return
	}
	if !slices.Contains(serviceScopedAdapters, t.Adapter) {
		handler.WriteError(w, http.StatusBadRequest, "telemetry adapter must be http, prometheus, cli, or liveness for service links")
		return
	}
	if t.Adapter == "liveness" && LivenessTelemetrySource(t.ConfigJSON) != "service_provider" {
		handler.WriteError(w, http.StatusBadRequest, "liveness telemetry for services must use source service_provider")
		return
	}
	link, err := h.store.CreateServiceTelemetryLink(r.Context(), svcID, input)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to create link")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "create", "service_telemetry_link", &link.ID, input, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusCreated, link)
}

func (h *ServicesHandler) PatchTelemetryLink(w http.ResponseWriter, r *http.Request) {
	svcID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}
	linkID, err := uuid.Parse(chi.URLParam(r, "linkId"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid link id")
		return
	}
	existing, err := h.store.GetServiceTelemetryLinkByID(r.Context(), linkID)
	if err != nil || existing.ServiceID != svcID {
		handler.WriteError(w, http.StatusNotFound, "link not found")
		return
	}
	var input store.CreateTelemetryLinkInput
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if input.TelemetryID == uuid.Nil {
		handler.WriteError(w, http.StatusBadRequest, "telemetry_id is required")
		return
	}
	t, err := h.store.GetTelemetryByID(r.Context(), input.TelemetryID)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "telemetry not found")
		return
	}
	if !slices.Contains(serviceScopedAdapters, t.Adapter) {
		handler.WriteError(w, http.StatusBadRequest, "telemetry adapter must be http, prometheus, cli, or liveness for service links")
		return
	}
	if t.Adapter == "liveness" && LivenessTelemetrySource(t.ConfigJSON) != "service_provider" {
		handler.WriteError(w, http.StatusBadRequest, "liveness telemetry for services must use source service_provider")
		return
	}
	link, err := h.store.UpdateServiceTelemetryLink(r.Context(), linkID, input)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to update link")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "update", "service_telemetry_link", &link.ID, input, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, link)
}

func (h *ServicesHandler) DeleteTelemetryLink(w http.ResponseWriter, r *http.Request) {
	svcID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}
	linkID, err := uuid.Parse(chi.URLParam(r, "linkId"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid link id")
		return
	}
	existing, err := h.store.GetServiceTelemetryLinkByID(r.Context(), linkID)
	if err != nil || existing.ServiceID != svcID {
		handler.WriteError(w, http.StatusNotFound, "link not found")
		return
	}
	if err := h.store.DeleteServiceTelemetryLink(r.Context(), linkID); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to delete link")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "delete", "service_telemetry_link", &linkID, nil, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}
