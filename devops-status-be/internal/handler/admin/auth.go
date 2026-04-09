package admin

import (
	"net/http"

	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/store"
)

type AuthHandler struct {
	store   *store.Store
	session *auth.SessionManager
}

func NewAuthHandler(s *store.Store, sm *auth.SessionManager) *AuthHandler {
	return &AuthHandler{store: s, session: sm}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := handler.DecodeJSON(r, &req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		handler.WriteError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	user, err := h.store.GetAdminUserByUsername(r.Context(), req.Username)
	if err != nil {
		handler.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if !user.IsActive {
		handler.WriteError(w, http.StatusUnauthorized, "account is disabled")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		handler.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := h.session.CreateSession(w, r, user.ID, user.Role); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	_ = h.store.UpdateAdminUserLastLogin(r.Context(), user.ID)

	_ = h.store.CreateAuditLog(r.Context(), &user.ID, "login", "admin_user", &user.ID, nil, r.RemoteAddr)

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.session.DestroySession(w, r); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to destroy session")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		handler.WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.store.GetAdminUserByID(r.Context(), userID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}
