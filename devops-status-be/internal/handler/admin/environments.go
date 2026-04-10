package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/model"
	"github.com/devops-status/be/internal/store"
)

type EnvironmentsHandler struct {
	store *store.Store
}

func NewEnvironmentsHandler(s *store.Store) *EnvironmentsHandler {
	return &EnvironmentsHandler{store: s}
}

func (h *EnvironmentsHandler) List(w http.ResponseWriter, r *http.Request) {
	envs, err := h.store.ListEnvironments(r.Context(), false)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list environments")
		return
	}
	handler.WriteJSON(w, http.StatusOK, envs)
}

// CheckSlugAvailable reports whether slug is free for create or update (optional exclude_id).
// Slugs shorter than 3 characters return available: true without hitting the DB.
func (h *EnvironmentsHandler) CheckSlugAvailable(w http.ResponseWriter, r *http.Request) {
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
	taken, err := h.store.EnvironmentSlugExists(r.Context(), slug, exclude)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to check slug")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]bool{"available": !taken})
}

type environmentDetailResponse struct {
	model.Environment
	TelemetryLinks []model.EnvironmentTelemetryLink `json:"telemetry_links"`
}

func (h *EnvironmentsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	env, err := h.store.GetEnvironmentByID(r.Context(), id)
	if err != nil {
		handler.WriteError(w, http.StatusNotFound, "environment not found")
		return
	}
	links, err := h.store.ListEnvironmentTelemetryLinks(r.Context(), &id)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load telemetry links")
		return
	}
	handler.WriteJSON(w, http.StatusOK, environmentDetailResponse{Environment: *env, TelemetryLinks: links})
}

// ListTelemetrySamples returns recent sample_results for one environment telemetry link (query: telemetry_id, optional probe_kind, limit).
func (h *EnvironmentsHandler) ListTelemetrySamples(w http.ResponseWriter, r *http.Request) {
	envID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid environment id")
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

	if _, err := h.store.GetEnvironmentByID(r.Context(), envID); err != nil {
		handler.WriteError(w, http.StatusNotFound, "environment not found")
		return
	}
	links, err := h.store.ListEnvironmentTelemetryLinks(r.Context(), &envID)
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

	samples, err := h.store.ListRecentSamplesForEnvironmentTelemetry(r.Context(), envID, telemetryID, probeKind, limit)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list samples")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]any{"samples": samples})
}

func (h *EnvironmentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input store.CreateEnvironmentInput
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Name == "" || input.Slug == "" {
		handler.WriteError(w, http.StatusBadRequest, "name and slug are required")
		return
	}

	env, err := h.store.CreateEnvironment(r.Context(), input)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to create environment")
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "create", "environment", &env.ID, input, r.RemoteAddr)

	handler.WriteJSON(w, http.StatusCreated, env)
}

func (h *EnvironmentsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	var input store.UpdateEnvironmentInput
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	env, err := h.store.UpdateEnvironment(r.Context(), id, input)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to update environment")
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "update", "environment", &env.ID, input, r.RemoteAddr)

	handler.WriteJSON(w, http.StatusOK, env)
}

func (h *EnvironmentsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	if err := h.store.DeleteEnvironment(r.Context(), id); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to delete environment")
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "delete", "environment", &id, nil, r.RemoteAddr)

	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *EnvironmentsHandler) CreateTelemetryLink(w http.ResponseWriter, r *http.Request) {
	envID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid environment id")
		return
	}
	if _, err := h.store.GetEnvironmentByID(r.Context(), envID); err != nil {
		handler.WriteError(w, http.StatusNotFound, "environment not found")
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
	if t.Adapter != "kubernetes" {
		handler.WriteError(w, http.StatusBadRequest, "environment links only support kubernetes telemetry")
		return
	}
	link, err := h.store.CreateEnvironmentTelemetryLink(r.Context(), envID, input)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to create link")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "create", "environment_telemetry_link", &link.ID, input, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusCreated, link)
}

func (h *EnvironmentsHandler) PatchTelemetryLink(w http.ResponseWriter, r *http.Request) {
	envID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid environment id")
		return
	}
	linkID, err := uuid.Parse(chi.URLParam(r, "linkId"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid link id")
		return
	}
	existing, err := h.store.GetEnvironmentTelemetryLinkByID(r.Context(), linkID)
	if err != nil || existing.EnvironmentID != envID {
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
	if t.Adapter != "kubernetes" {
		handler.WriteError(w, http.StatusBadRequest, "environment links only support kubernetes telemetry")
		return
	}
	link, err := h.store.UpdateEnvironmentTelemetryLink(r.Context(), linkID, input)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to update link")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "update", "environment_telemetry_link", &link.ID, input, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, link)
}

func (h *EnvironmentsHandler) DeleteTelemetryLink(w http.ResponseWriter, r *http.Request) {
	envID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid environment id")
		return
	}
	linkID, err := uuid.Parse(chi.URLParam(r, "linkId"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid link id")
		return
	}
	existing, err := h.store.GetEnvironmentTelemetryLinkByID(r.Context(), linkID)
	if err != nil || existing.EnvironmentID != envID {
		handler.WriteError(w, http.StatusNotFound, "link not found")
		return
	}
	if err := h.store.DeleteEnvironmentTelemetryLink(r.Context(), linkID); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to delete link")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "delete", "environment_telemetry_link", &linkID, nil, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}
