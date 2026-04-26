package providerverify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerifyRancherPingNone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ping" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("pong"))
	}))
	defer srv.Close()

	ms, err := VerifyRancher(context.Background(), srv.URL, "none", "", false)
	if err != nil {
		t.Fatalf("VerifyRancher: %v", err)
	}
	if ms < 0 {
		t.Fatalf("latencyMs = %d", ms)
	}
}

func TestVerifyRancherPingEmptyBodyOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ping" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := VerifyRancher(context.Background(), srv.URL, "none", "", false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestVerifyRancherV3BasicAccessKeySecret(t *testing.T) {
	const combined = "token-abc:secret-part"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/" {
			http.NotFound(w, r)
			return
		}
		u, p, ok := r.BasicAuth()
		if !ok || u != "token-abc" || p != "secret-part" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"type":"collection"}`))
	}))
	defer srv.Close()

	ms, err := VerifyRancher(context.Background(), srv.URL, "bearer", combined, false)
	if err != nil {
		t.Fatalf("VerifyRancher: %v", err)
	}
	if ms < 0 {
		t.Fatalf("latencyMs = %d", ms)
	}
}

func TestVerifyRancherV3BearerFallbackNoColon(t *testing.T) {
	const tok = "opaque-single-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/" {
			http.NotFound(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != tok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"type":"collection"}`))
	}))
	defer srv.Close()

	ms, err := VerifyRancher(context.Background(), srv.URL, "bearer", tok, false)
	if err != nil {
		t.Fatalf("VerifyRancher: %v", err)
	}
	if ms < 0 {
		t.Fatalf("latencyMs = %d", ms)
	}
}

func TestVerifyRancherBearerMissingToken(t *testing.T) {
	_, err := VerifyRancher(context.Background(), "https://example.com", "bearer", "", false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyRancherInvalidAuthMethod(t *testing.T) {
	_, err := VerifyRancher(context.Background(), "https://example.com", "basic", "", false)
	if err == nil {
		t.Fatal("expected error")
	}
}
