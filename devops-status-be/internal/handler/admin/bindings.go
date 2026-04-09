package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/store"
)

type BindingsHandler struct {
	store *store.Store
}

func NewBindingsHandler(s *store.Store) *BindingsHandler {
	return &BindingsHandler{store: s}
}

func (h *BindingsHandler) ListServiceBindings(w http.ResponseWriter, r *http.Request) {
	var svcID *uuid.UUID
	if idStr := r.URL.Query().Get("service_id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid service_id")
			return
		}
		svcID = &id
	}
	bindings, err := h.store.ListServiceProbeBindings(r.Context(), svcID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list bindings")
		return
	}
	handler.WriteJSON(w, http.StatusOK, bindings)
}

func (h *BindingsHandler) CreateServiceBinding(w http.ResponseWriter, r *http.Request) {
	svcID, err := uuid.Parse(chi.URLParam(r, "serviceId"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}
	var input store.CreateBindingInput
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	b, err := h.store.CreateServiceProbeBinding(r.Context(), svcID, input)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to create binding")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "create", "service_probe_binding", &b.ID, input, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusCreated, b)
}

func (h *BindingsHandler) DeleteServiceBinding(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteServiceProbeBinding(r.Context(), id); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to delete")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "delete", "service_probe_binding", &id, nil, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *BindingsHandler) ListEnvironmentBindings(w http.ResponseWriter, r *http.Request) {
	var envID *uuid.UUID
	if idStr := r.URL.Query().Get("environment_id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid environment_id")
			return
		}
		envID = &id
	}
	bindings, err := h.store.ListEnvironmentProbeBindings(r.Context(), envID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list bindings")
		return
	}
	handler.WriteJSON(w, http.StatusOK, bindings)
}

func (h *BindingsHandler) CreateEnvironmentBinding(w http.ResponseWriter, r *http.Request) {
	envID, err := uuid.Parse(chi.URLParam(r, "envId"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid environment id")
		return
	}
	var input store.CreateBindingInput
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	b, err := h.store.CreateEnvironmentProbeBinding(r.Context(), envID, input)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to create binding")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "create", "environment_probe_binding", &b.ID, input, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusCreated, b)
}

func (h *BindingsHandler) DeleteEnvironmentBinding(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteEnvironmentProbeBinding(r.Context(), id); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to delete")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "delete", "environment_probe_binding", &id, nil, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}
