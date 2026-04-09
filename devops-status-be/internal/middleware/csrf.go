package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const csrfTokenHeader = "X-CSRF-Token"
const csrfCookieName = "csrf_token"

func CSRFProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			ensureCSRFCookie(w, r)
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(csrfCookieName)
		if err != nil {
			http.Error(w, `{"error":"csrf token missing"}`, http.StatusForbidden)
			return
		}

		headerToken := r.Header.Get(csrfTokenHeader)
		if headerToken == "" || headerToken != cookie.Value {
			http.Error(w, `{"error":"csrf token mismatch"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(csrfCookieName); err == nil {
		return
	}

	token := generateCSRFToken()
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // must be readable by JS
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // set true in production
	})
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
