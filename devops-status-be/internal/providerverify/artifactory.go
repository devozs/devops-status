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

// artifactoryBaseURLIsHostOnly is true when baseURL has no path (or only "/"), so we may retry under /artifactory.
func artifactoryBaseURLIsHostOnly(base string) bool {
	u, err := url.Parse(base)
	if err != nil {
		return false
	}
	return strings.Trim(u.Path, "/") == ""
}

func artifactoryPing(ctx context.Context, client *http.Client, reqURL, am, username, password string) (latencyMs int, statusCode int, bodyText string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, 0, "", err
	}
	if am == "basic" {
		req.SetBasicAuth(username, password)
	}
	start := time.Now()
	resp, err := client.Do(req)
	latencyMs = int(time.Since(start).Milliseconds())
	if err != nil {
		return latencyMs, 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return latencyMs, resp.StatusCode, strings.TrimSpace(string(body)), nil
}

// VerifyArtifactory GETs /api/system/ping relative to base_url.
// If that returns 404 and base_url is host-only (no path), retries once with .../artifactory/api/system/ping (common JFrog layout).
func VerifyArtifactory(ctx context.Context, baseURL, authMethod, username, password string) (latencyMs int, err error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return 0, fmt.Errorf("base_url is required")
	}
	parsed, err := url.Parse(base)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
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
	latencyMs, code, text, err := artifactoryPing(ctx, client, reqURL, am, username, password)
	if err != nil {
		return latencyMs, err
	}

	if code == http.StatusNotFound && artifactoryBaseURLIsHostOnly(base) {
		retryURL := base + "/artifactory/api/system/ping"
		latencyMs, code, text, err = artifactoryPing(ctx, client, retryURL, am, username, password)
		if err != nil {
			return latencyMs, err
		}
	}

	switch code {
	case http.StatusOK:
		if text != "" && !strings.EqualFold(text, "OK") {
			// Some editions return JSON; accept 200
		}
		return latencyMs, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		if am == "none" {
			return latencyMs, fmt.Errorf("artifactory returned %d (authentication may be required)", code)
		}
		return latencyMs, fmt.Errorf("artifactory authentication failed (%d)", code)
	case http.StatusNotFound:
		hint := ""
		if !strings.Contains(base, "/artifactory") {
			hint = " (try base URL ending with /artifactory if the instance uses that context path)"
		}
		return latencyMs, fmt.Errorf("artifactory returned %d: %s%s", code, text, hint)
	default:
		return latencyMs, fmt.Errorf("artifactory returned %d: %s", code, text)
	}
}
