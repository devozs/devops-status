package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/store"
)

type DataSourcesHandler struct {
	store *store.Store
}

func NewDataSourcesHandler(s *store.Store) *DataSourcesHandler {
	return &DataSourcesHandler{store: s}
}

func (h *DataSourcesHandler) List(w http.ResponseWriter, r *http.Request) {
	sources, err := h.store.ListDataSources(r.Context())
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list data sources")
		return
	}
	handler.WriteJSON(w, http.StatusOK, sources)
}

func (h *DataSourcesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	ds, err := h.store.GetDataSourceByID(r.Context(), id)
	if err != nil {
		handler.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	handler.WriteJSON(w, http.StatusOK, ds)
}

func (h *DataSourcesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input store.CreateDataSourceInput
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if input.Name == "" || input.DSType == "" || input.Adapter == "" {
		handler.WriteError(w, http.StatusBadRequest, "name, ds_type, and adapter are required")
		return
	}
	ds, err := h.store.CreateDataSource(r.Context(), input)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to create")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "create", "data_source", &ds.ID, input, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusCreated, ds)
}

func (h *DataSourcesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var input store.CreateDataSourceInput
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	ds, err := h.store.UpdateDataSource(r.Context(), id, input)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to update")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "update", "data_source", &ds.ID, input, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, ds)
}

func (h *DataSourcesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteDataSource(r.Context(), id); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to delete")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "delete", "data_source", &id, nil, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}
