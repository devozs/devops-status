package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/handler"
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
	handler.WriteJSON(w, http.StatusOK, env)
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

func (h *EnvironmentsHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	members, err := h.store.ListEnvironmentMembers(r.Context(), id)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list members")
		return
	}
	handler.WriteJSON(w, http.StatusOK, members)
}

type membershipRequest struct {
	ServiceID string `json:"service_id"`
}

func (h *EnvironmentsHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	envID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	var req membershipRequest
	if err := handler.DecodeJSON(r, &req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	svcID, err := uuid.Parse(req.ServiceID)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	if err := h.store.AddEnvironmentMember(r.Context(), envID, svcID); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to add member")
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "add_member", "environment", &envID, map[string]any{"service_id": svcID}, r.RemoteAddr)

	handler.WriteJSON(w, http.StatusCreated, map[string]string{"message": "member added"})
}

func (h *EnvironmentsHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	envID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	svcID, err := uuid.Parse(chi.URLParam(r, "serviceId"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	if err := h.store.RemoveEnvironmentMember(r.Context(), envID, svcID); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "remove_member", "environment", &envID, map[string]any{"service_id": svcID}, r.RemoteAddr)

	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "member removed"})
}
