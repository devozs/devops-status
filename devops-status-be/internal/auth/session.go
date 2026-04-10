package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

const (
	SessionName   = "devops-status-session"
	SessionUserID = "user_id"
	SessionRole   = "role"
)

type contextKey string

const userContextKey contextKey = "admin_user_id"
const roleContextKey contextKey = "admin_role"

type SessionManager struct {
	store sessions.Store
}

func NewSessionManager(secretKey string, secureCookie bool) *SessionManager {
	keyBytes := []byte(secretKey)
	if len(keyBytes) < 32 {
		keyBytes = securecookie.GenerateRandomKey(32)
	}
	cookieStore := sessions.NewCookieStore(keyBytes)
	cookieStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookie,
	}
	return &SessionManager{store: cookieStore}
}

func (sm *SessionManager) CreateSession(w http.ResponseWriter, r *http.Request, userID uuid.UUID, role string) error {
	session, _ := sm.store.Get(r, SessionName)
	session.Values[SessionUserID] = userID.String()
	session.Values[SessionRole] = role
	return session.Save(r, w)
}

func (sm *SessionManager) GetSession(r *http.Request) (uuid.UUID, string, error) {
	session, err := sm.store.Get(r, SessionName)
	if err != nil {
		return uuid.Nil, "", err
	}

	uidStr, ok := session.Values[SessionUserID].(string)
	if !ok {
		return uuid.Nil, "", nil
	}

	uid, err := uuid.Parse(uidStr)
	if err != nil {
		return uuid.Nil, "", nil
	}

	role, _ := session.Values[SessionRole].(string)
	return uid, role, nil
}

func (sm *SessionManager) DestroySession(w http.ResponseWriter, r *http.Request) error {
	session, _ := sm.store.Get(r, SessionName)
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

func WithUserContext(ctx context.Context, userID uuid.UUID, role string) context.Context {
	ctx = context.WithValue(ctx, userContextKey, userID)
	ctx = context.WithValue(ctx, roleContextKey, role)
	return ctx
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	uid, ok := ctx.Value(userContextKey).(uuid.UUID)
	return uid, ok
}

func RoleFromContext(ctx context.Context) string {
	role, _ := ctx.Value(roleContextKey).(string)
	return role
}
