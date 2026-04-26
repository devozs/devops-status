package admin

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/model"
	"github.com/devops-status/be/internal/secrets"
	"github.com/devops-status/be/internal/store"
)

type TelemetryHandler struct {
	store        *store.Store
	secret       *secrets.Store
	hlctlEnabled bool
}

func NewTelemetryHandler(s *store.Store, sec *secrets.Store, hlctlEnabled bool) *TelemetryHandler {
	return &TelemetryHandler{store: s, secret: sec, hlctlEnabled: hlctlEnabled}
}

type telemetryRequestBody struct {
	store.CreateTelemetryInput `json:",inline"`
}

// CheckNameAvailable reports whether name is free for create or update (optional exclude_id).
// Names shorter than 3 characters return available: true without hitting the DB.
func (h *TelemetryHandler) CheckNameAvailable(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	nm := strings.TrimSpace(q.Get("name"))
	if len(nm) < 3 {
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
	taken, err := h.store.TelemetryNameExists(r.Context(), nm, exclude)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to check name")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]bool{"available": !taken})
}

func (h *TelemetryHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.ListTelemetry(r.Context())
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list telemetry")
		return
	}
	for i := range items {
		h.decorateTelemetry(&items[i])
	}
	handler.WriteJSON(w, http.StatusOK, items)
}

func (h *TelemetryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := h.store.GetTelemetryByID(r.Context(), id)
	if err != nil {
		handler.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	h.decorateTelemetry(t)
	handler.WriteJSON(w, http.StatusOK, t)
}

func (h *TelemetryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body telemetryRequestBody
	if err := handler.DecodeJSON(r, &body); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	input := body.CreateTelemetryInput
	if input.Name == "" || input.Adapter == "" {
		handler.WriteError(w, http.StatusBadRequest, "name and adapter are required")
		return
	}
	if input.TimeoutMs <= 0 {
		input.TimeoutMs = DefaultTimeoutMs(input.Adapter)
	}
	input.ExecutionTarget = EffectiveExecutionTarget(input.Adapter, input.ExecutionTarget)
	if input.Adapter == "liveness" && LivenessTelemetrySource(input.ConfigJSON) == "kubernetes" {
		input.ExecutionTarget = "k8s_cluster"
	}
	normalizeTelemetryConfig(&input)

	cfgBytes, _ := json.Marshal(input.ConfigJSON)
	qosBytes, _ := json.Marshal(input.QosThresholds)
	if errs := ValidateTelemetryConfig(input.Adapter, cfgBytes, qosBytes, input.ExecutionTarget); len(errs) > 0 {
		handler.WriteError(w, http.StatusBadRequest, fmtValidationError(errs))
		return
	}
	if input.Adapter == "hlctl" && !h.hlctlEnabled {
		handler.WriteError(w, http.StatusBadRequest, "HLCTL telemetry requires kubectl and hlctl in the management image")
		return
	}
	if hintErrs := ValidateTelemetryShellHintRefs(r.Context(), h.store, input.Adapter, cfgBytes); len(hintErrs) > 0 {
		handler.WriteError(w, http.StatusBadRequest, fmtValidationError(hintErrs))
		return
	}

	taken, err := h.store.TelemetryNameExists(r.Context(), input.Name, nil)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to validate name")
		return
	}
	if taken {
		handler.WriteError(w, http.StatusConflict, "telemetry with this name already exists")
		return
	}

	t, err := h.store.CreateTelemetry(r.Context(), input)
	if err != nil {
		if isTelemetryNameUniqueViolation(err) {
			handler.WriteError(w, http.StatusConflict, "telemetry with this name already exists")
			return
		}
		slog.Error("create telemetry", "adapter", input.Adapter, "error", err)
		handler.WriteError(w, http.StatusInternalServerError, "failed to create")
		return
	}

	h.decorateTelemetry(t)
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "create", "telemetry", &t.ID, input, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusCreated, t)
}

func (h *TelemetryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body telemetryRequestBody
	if err := handler.DecodeJSON(r, &body); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	input := body.CreateTelemetryInput
	if input.Name == "" || input.Adapter == "" {
		handler.WriteError(w, http.StatusBadRequest, "name and adapter are required")
		return
	}
	if input.TimeoutMs <= 0 {
		input.TimeoutMs = DefaultTimeoutMs(input.Adapter)
	}
	input.ExecutionTarget = EffectiveExecutionTarget(input.Adapter, input.ExecutionTarget)
	if input.Adapter == "liveness" && LivenessTelemetrySource(input.ConfigJSON) == "kubernetes" {
		input.ExecutionTarget = "k8s_cluster"
	}
	normalizeTelemetryConfig(&input)

	cfgBytes, _ := json.Marshal(input.ConfigJSON)
	qosBytes, _ := json.Marshal(input.QosThresholds)
	if errs := ValidateTelemetryConfig(input.Adapter, cfgBytes, qosBytes, input.ExecutionTarget); len(errs) > 0 {
		handler.WriteError(w, http.StatusBadRequest, fmtValidationError(errs))
		return
	}
	if input.Adapter == "hlctl" && !h.hlctlEnabled {
		handler.WriteError(w, http.StatusBadRequest, "HLCTL telemetry requires kubectl and hlctl in the management image")
		return
	}
	if hintErrs := ValidateTelemetryShellHintRefs(r.Context(), h.store, input.Adapter, cfgBytes); len(hintErrs) > 0 {
		handler.WriteError(w, http.StatusBadRequest, fmtValidationError(hintErrs))
		return
	}

	taken, err := h.store.TelemetryNameExists(r.Context(), input.Name, &id)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to validate name")
		return
	}
	if taken {
		handler.WriteError(w, http.StatusConflict, "telemetry with this name already exists")
		return
	}

	t, err := h.store.UpdateTelemetry(r.Context(), id, input)
	if err != nil {
		if isTelemetryNameUniqueViolation(err) {
			handler.WriteError(w, http.StatusConflict, "telemetry with this name already exists")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to update")
		return
	}

	h.decorateTelemetry(t)
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "update", "telemetry", &t.ID, input, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, t)
}

func (h *TelemetryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if h.secret != nil {
		_ = h.secret.DeleteAllForOwner(r.Context(), secrets.OwnerTelemetry, id)
	}
	if err := h.store.DeleteTelemetry(r.Context(), id); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to delete")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "delete", "telemetry", &id, nil, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *TelemetryHandler) decorateTelemetry(t *model.Telemetry) {
	stripSensitiveTelemetryConfig(t)
}

func stripSensitiveTelemetryConfig(t *model.Telemetry) {
	m, ok := t.ConfigJSON.(map[string]any)
	if !ok {
		return
	}
	delete(m, "username")
	delete(m, "password")
	delete(m, "bearer_token")
	delete(m, "_hlctl_username")
	delete(m, "_hlctl_password")
}

func normalizeTelemetryConfig(input *store.CreateTelemetryInput) {
	if input.Adapter != "prometheus" {
		return
	}
	m, ok := input.ConfigJSON.(map[string]any)
	if !ok {
		return
	}
	q, _ := json.Marshal(input.QosThresholds)
	if len(q) > 0 && string(q) != "{}" && string(q) != "null" {
		m["qos_mode"] = true
	}
	input.ConfigJSON = m
}

func isTelemetryNameUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	c := pgErr.ConstraintName
	msg := pgErr.Message
	return strings.Contains(c, "name_uidx") || strings.Contains(msg, "telemetry_name_uidx") ||
		strings.Contains(c, "name_lower_uidx") || strings.Contains(msg, "telemetry_name_lower_uidx") ||
		strings.Contains(c, "name_ds_type") || strings.Contains(msg, "name_ds_type")
}
