package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/handler"
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
	handler.WriteJSON(w, http.StatusOK, svc)
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
