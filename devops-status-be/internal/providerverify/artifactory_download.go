package providerverify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Defaults for liveness bandwidth sampling (capped read, not full file).
const (
	DefaultArtifactoryDownloadMaxBytes int64 = 5 * 1024 * 1024
	MaxArtifactoryDownloadMaxBytes     int64 = 50 * 1024 * 1024
)

const artifactoryDownloadTimeout = 90 * time.Second

var artifactoryRepoPathPattern = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)

// ValidateArtifactoryDownloadRepositoryPath checks telemetry path (repo-key/path/to/artifact).
func ValidateArtifactoryDownloadRepositoryPath(p string) error {
	p = strings.TrimSpace(p)
	if p == "" {
		return fmt.Errorf("download_repository_path is empty")
	}
	if strings.Contains(p, "..") {
		return fmt.Errorf("download_repository_path must not contain '..'")
	}
	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("download_repository_path must not start with '/'")
	}
	if !artifactoryRepoPathPattern.MatchString(p) {
		return fmt.Errorf("download_repository_path contains invalid characters (use repo/path segments: letters, digits, ._-/)")
	}
	return nil
}

// ClampArtifactoryDownloadMaxBytes returns a positive sample size within server limits.
func ClampArtifactoryDownloadMaxBytes(n int64) int64 {
	if n <= 0 {
		return DefaultArtifactoryDownloadMaxBytes
	}
	if n > MaxArtifactoryDownloadMaxBytes {
		return MaxArtifactoryDownloadMaxBytes
	}
	return n
}

// MeasureArtifactoryDownload streams up to maxBytes from baseURL/repositoryPath and returns average bytes/sec over the read.
func MeasureArtifactoryDownload(ctx context.Context, baseURL, authMethod, username, password string, repositoryPath string, maxBytes int64) (bytesRead int64, downloadMs int, avgBytesPerSec float64, err error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return 0, 0, 0, fmt.Errorf("base_url is required")
	}
	if u, err := url.Parse(base); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return 0, 0, 0, fmt.Errorf("base_url must be a valid http(s) URL")
	}
	if err := ValidateArtifactoryDownloadRepositoryPath(repositoryPath); err != nil {
		return 0, 0, 0, err
	}
	am := strings.ToLower(strings.TrimSpace(authMethod))
	if am == "" {
		am = "none"
	}
	if am != "none" && am != "basic" {
		return 0, 0, 0, fmt.Errorf("auth_method must be none or basic for artifactory")
	}
	if am == "basic" && (username == "" || password == "") {
		return 0, 0, 0, fmt.Errorf("basic auth requires username and password")
	}

	maxBytes = ClampArtifactoryDownloadMaxBytes(maxBytes)
	rel := strings.TrimLeft(strings.TrimSpace(repositoryPath), "/")
	reqURL := base + "/" + rel

	dlCtx, cancel := context.WithTimeout(ctx, artifactoryDownloadTimeout)
	defer cancel()

	client := &http.Client{Timeout: artifactoryDownloadTimeout}
	req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, 0, 0, err
	}
	if am == "basic" {
		req.SetBasicAuth(username, password)
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, int(time.Since(start).Milliseconds()), 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return 0, int(time.Since(start).Milliseconds()), 0, fmt.Errorf("artifactory download returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	limited := io.LimitReader(resp.Body, maxBytes)
	n, readErr := io.Copy(io.Discard, limited)
	bytesRead = n
	downloadMs = int(time.Since(start).Milliseconds())
	if downloadMs < 1 {
		downloadMs = 1
	}
	if bytesRead == 0 {
		return 0, downloadMs, 0, fmt.Errorf("artifactory download returned empty body")
	}
	secs := float64(downloadMs) / 1000.0
	if secs <= 0 {
		secs = 0.001
	}
	avgBytesPerSec = float64(bytesRead) / secs
	if readErr != nil && readErr != io.EOF {
		return bytesRead, downloadMs, avgBytesPerSec, readErr
	}
	return bytesRead, downloadMs, avgBytesPerSec, nil
}
