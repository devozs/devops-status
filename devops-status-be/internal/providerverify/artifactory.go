package providerverify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const artifactoryVerifyTimeout = 15 * time.Second

// VerifyArtifactory GETs /api/system/ping relative to base_url (include path prefix if needed, e.g. .../artifactory).
func VerifyArtifactory(ctx context.Context, baseURL, authMethod, username, password string) (latencyMs int, err error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return 0, fmt.Errorf("base_url is required")
	}
	if u, err := url.Parse(base); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return 0, fmt.Errorf("base_url must be a valid http(s) URL")
	}
	am := strings.ToLower(strings.TrimSpace(authMethod))
	if am == "" {
		am = "none"
	}
	if am != "none" && am != "basic" {
		return 0, fmt.Errorf("auth_method must be none or basic for artifactory")
	}
	if am == "basic" && (username == "" || password == "") {
		return 0, fmt.Errorf("basic auth requires username and password")
	}

	client := &http.Client{Timeout: artifactoryVerifyTimeout}
	reqURL := base + "/api/system/ping"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, err
	}
	if am == "basic" {
		req.SetBasicAuth(username, password)
	}

	start := time.Now()
	resp, err := client.Do(req)
	latencyMs = int(time.Since(start).Milliseconds())
	if err != nil {
		return latencyMs, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	text := strings.TrimSpace(string(body))

	switch resp.StatusCode {
	case http.StatusOK:
		if text != "" && !strings.EqualFold(text, "OK") {
			// Some editions return JSON; accept 200
		}
		return latencyMs, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		if am == "none" {
			return latencyMs, fmt.Errorf("artifactory returned %d (authentication may be required)", resp.StatusCode)
		}
		return latencyMs, fmt.Errorf("artifactory authentication failed (%d)", resp.StatusCode)
	default:
		return latencyMs, fmt.Errorf("artifactory returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}
