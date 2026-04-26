package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/shellhints"
	"github.com/devops-status/be/internal/store"
)

type TelemetryShellHintsHandler struct {
	store *store.Store
}

func NewTelemetryShellHintsHandler(s *store.Store) *TelemetryShellHintsHandler {
	return &TelemetryShellHintsHandler{store: s}
}

func (h *TelemetryShellHintsHandler) List(w http.ResponseWriter, r *http.Request) {
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	var kindPtr *string
	if kind != "" {
		if !shellhints.ValidKind(kind) {
			handler.WriteError(w, http.StatusBadRequest, "invalid kind")
			return
		}
		kindPtr = &kind
	}
	items, err := h.store.ListTelemetryShellHints(r.Context(), kindPtr)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list shell hints")
		return
	}
	handler.WriteJSON(w, http.StatusOK, items)
}

type shellHintCreateBody struct {
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	Description string `json:"description"`
	SortOrder   *int   `json:"sort_order"`
}

func (h *TelemetryShellHintsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body shellHintCreateBody
	if err := handler.DecodeJSON(r, &body); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	kind := strings.TrimSpace(body.Kind)
	title := strings.TrimSpace(body.Title)
	if kind == "" || title == "" {
		handler.WriteError(w, http.StatusBadRequest, "kind and title are required")
		return
	}
	if !shellhints.ValidKind(kind) {
		handler.WriteError(w, http.StatusBadRequest, "invalid kind")
		return
	}
	if errs := validateShellHintBody(kind, body.Body); len(errs) > 0 {
		handler.WriteError(w, http.StatusBadRequest, fmtValidationError(errs))
		return
	}
	item, err := h.store.CreateTelemetryShellHint(r.Context(), store.CreateTelemetryShellHintInput{
		Kind:        kind,
		Title:       title,
		Body:        body.Body,
		Description: body.Description,
		SortOrder:   body.SortOrder,
	})
	if err != nil {
		if isShellHintUniqueViolation(err) {
			handler.WriteError(w, http.StatusConflict, "a hint with this title already exists for this kind")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to create shell hint")
		return
	}
	handler.WriteJSON(w, http.StatusCreated, item)
}

type shellHintUpdateBody struct {
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	Description string `json:"description"`
	SortOrder   *int   `json:"sort_order"`
}

func (h *TelemetryShellHintsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	oldHint, err := h.store.GetTelemetryShellHintByID(r.Context(), id)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load shell hint")
		return
	}
	if oldHint == nil {
		handler.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	var body shellHintUpdateBody
	if err := handler.DecodeJSON(r, &body); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	kind := strings.TrimSpace(body.Kind)
	title := strings.TrimSpace(body.Title)
	if kind == "" || title == "" {
		handler.WriteError(w, http.StatusBadRequest, "kind and title are required")
		return
	}
	if !shellhints.ValidKind(kind) {
		handler.WriteError(w, http.StatusBadRequest, "invalid kind")
		return
	}
	if errs := validateShellHintBody(kind, body.Body); len(errs) > 0 {
		handler.WriteError(w, http.StatusBadRequest, fmtValidationError(errs))
		return
	}
	item, err := h.store.UpdateTelemetryShellHint(r.Context(), id, store.UpdateTelemetryShellHintInput{
		Kind:        kind,
		Title:       title,
		Body:        body.Body,
		Description: body.Description,
		SortOrder:   body.SortOrder,
	})
	if err != nil {
		if isShellHintUniqueViolation(err) {
			handler.WriteError(w, http.StatusConflict, "a hint with this title already exists for this kind")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to update shell hint")
		return
	}
	if item == nil {
		handler.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	bodyChanged := strings.TrimSpace(oldHint.Body) != strings.TrimSpace(body.Body)
	var syncUpdated int
	var syncWarns []string
	if bodyChanged {
		syncUpdated, syncWarns = SyncTelemetryAfterShellHintBodyChange(r.Context(), h.store, id, kind, body.Body)
	}
	raw, _ := json.Marshal(item)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	if bodyChanged {
		out["telemetry_rows_updated"] = syncUpdated
		if len(syncWarns) > 0 {
			out["telemetry_sync_errors"] = syncWarns
		}
	}
	handler.WriteJSON(w, http.StatusOK, out)
}

func (h *TelemetryShellHintsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteTelemetryShellHint(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			handler.WriteError(w, http.StatusNotFound, "not found")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to delete")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

type shellHintReorderBody struct {
	Kind       string   `json:"kind"`
	OrderedIDs []string `json:"ordered_ids"`
}

func (h *TelemetryShellHintsHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	var body shellHintReorderBody
	if err := handler.DecodeJSON(r, &body); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	kind := strings.TrimSpace(body.Kind)
	if !shellhints.ValidKind(kind) {
		handler.WriteError(w, http.StatusBadRequest, "invalid kind")
		return
	}
	if len(body.OrderedIDs) == 0 {
		handler.WriteError(w, http.StatusBadRequest, "ordered_ids is required")
		return
	}
	ids := make([]uuid.UUID, 0, len(body.OrderedIDs))
	for _, s := range body.OrderedIDs {
		id, err := uuid.Parse(strings.TrimSpace(s))
		if err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid uuid in ordered_ids")
			return
		}
		ids = append(ids, id)
	}
	if err := h.store.ReorderTelemetryShellHints(r.Context(), kind, ids); err != nil {
		handler.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func validateShellHintBody(kind, body string) []string {
	var errs []string
	add := func(msg string) { errs = append(errs, msg) }
	if err := shellhints.ValidateHintBody(kind, body); err != nil {
		add(err.Error())
		return errs
	}
	switch kind {
	case shellhints.KindJobEnv:
		m, err := shellhints.ParseProbeEnvLines(body)
		if err != nil {
			add(err.Error())
			return errs
		}
		appendTelemetryProbeEnvErrors(add, "", m)
	case shellhints.KindJobNodeSelector:
		m, err := shellhints.ParseNodeSelectorLines(body)
		if err != nil {
			add(err.Error())
			return errs
		}
		appendTelemetryNodeSelectorErrors(add, "", m)
	}
	return errs
}

func isShellHintUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	return strings.Contains(pgErr.ConstraintName, "telemetry_shell_hints") || strings.Contains(pgErr.Message, "telemetry_shell_hints")
}
