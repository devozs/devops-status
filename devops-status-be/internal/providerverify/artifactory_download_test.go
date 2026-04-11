package providerverify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidateArtifactoryDownloadRepositoryPath(t *testing.T) {
	if err := ValidateArtifactoryDownloadRepositoryPath("libs-local/foo/bar.jar"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateArtifactoryDownloadRepositoryPath(""); err == nil {
		t.Fatal("expected error on empty")
	}
	if err := ValidateArtifactoryDownloadRepositoryPath("a/../b"); err == nil {
		t.Fatal("expected error on ..")
	}
	if err := ValidateArtifactoryDownloadRepositoryPath("/repo/x"); err == nil {
		t.Fatal("expected error on leading slash")
	}
	if err := ValidateArtifactoryDownloadRepositoryPath("repo x"); err == nil {
		t.Fatal("expected error on space")
	}
}

func TestClampArtifactoryDownloadMaxBytes(t *testing.T) {
	if got := ClampArtifactoryDownloadMaxBytes(0); got != DefaultArtifactoryDownloadMaxBytes {
		t.Fatalf("default: got %d", got)
	}
	if got := ClampArtifactoryDownloadMaxBytes(MaxArtifactoryDownloadMaxBytes + 1); got != MaxArtifactoryDownloadMaxBytes {
		t.Fatalf("ceiling: got %d", got)
	}
}

func TestMeasureArtifactoryDownload(t *testing.T) {
	const bodySize = 8000
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generic-local/test.bin" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, io.LimitReader(strings.NewReader(strings.Repeat("x", bodySize*2)), bodySize))
	}))
	defer srv.Close()

	ctx := context.Background()
	br, dlMs, bps, err := MeasureArtifactoryDownload(ctx, srv.URL, "none", "", "", "generic-local/test.bin", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if br != 5000 {
		t.Fatalf("bytes read: got %d want 5000", br)
	}
	if dlMs < 1 {
		t.Fatalf("duration: %d", dlMs)
	}
	if bps <= 0 {
		t.Fatalf("avg bps: %f", bps)
	}
}

func TestMeasureArtifactoryDownload_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	_, _, _, err := MeasureArtifactoryDownload(context.Background(), srv.URL, "none", "", "", "empty/path", 1000)
	if err == nil || !strings.Contains(err.Error(), "empty body") {
		t.Fatalf("expected empty body error, got %v", err)
	}
}

func TestMeasureArtifactoryDownload_StatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()
	_, _, _, err := MeasureArtifactoryDownload(context.Background(), srv.URL, "none", "", "", "missing", 1000)
	if err == nil {
		t.Fatal("expected error")
	}
}

// SlowReader delays reads so average B/s is bounded and predictable.
type slowReader struct {
	n   int
	pos int
}

func (s *slowReader) Read(p []byte) (int, error) {
	if s.pos >= s.n {
		return 0, io.EOF
	}
	time.Sleep(20 * time.Millisecond)
	chunk := 400
	if chunk > len(p) {
		chunk = len(p)
	}
	remain := s.n - s.pos
	if chunk > remain {
		chunk = remain
	}
	for i := range chunk {
		p[i] = 'z'
	}
	s.pos += chunk
	return chunk, nil
}

func TestMeasureArtifactoryDownload_AverageSpeed(t *testing.T) {
	const total = 2000
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, &slowReader{n: total})
	}))
	defer srv.Close()

	br, dlMs, bps, err := MeasureArtifactoryDownload(context.Background(), srv.URL, "none", "", "", "slow/blob", int64(total))
	if err != nil {
		t.Fatal(err)
	}
	if br != total {
		t.Fatalf("bytes %d want %d", br, total)
	}
	expected := float64(total) / (float64(dlMs) / 1000.0)
	if bps < expected*0.5 || bps > expected*2.0 {
		t.Fatalf("bps %f expected ~%f (dlMs=%d)", bps, expected, dlMs)
	}
}
