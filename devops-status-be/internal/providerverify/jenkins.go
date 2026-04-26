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

const jenkinsVerifyTimeout = 15 * time.Second

// jenkinsPingPath is the Jenkins metrics plugin liveness endpoint (often plain text or JSON).
// See: /metrics/currentUser/ping (light) vs /metrics/currentUser/metrics?pretty=true (full metrics).
const jenkinsPingPath = "/metrics/currentUser/ping"

// jenkinsRootURL returns scheme://host[:port] only so users may paste a full metrics URL
// without producing a broken double path when we append jenkinsPingPath.
func jenkinsRootURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("url is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("url must be a valid http(s) URL")
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

// VerifyJenkins GETs /metrics/currentUser/ping on the Jenkins root URL.
// authMethod is "none" or "basic" (password is typically an API token).
func VerifyJenkins(ctx context.Context, rootURL, authMethod, username, password string) (latencyMs int, err error) {
	u, err := jenkinsRootURL(rootURL)
	if err != nil {
		return 0, err
	}
	am := strings.ToLower(strings.TrimSpace(authMethod))
	if am == "" {
		am = "none"
	}
	if am != "none" && am != "basic" {
		return 0, fmt.Errorf("auth_method must be none or basic for jenkins")
	}
	user := strings.TrimSpace(username)
	if am == "basic" && (user == "" || password == "") {
		return 0, fmt.Errorf("basic auth requires username and password or API token")
	}

	reqURL := u + jenkinsPingPath
	client := &http.Client{Timeout: jenkinsVerifyTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, err
	}
	if am == "basic" {
		req.SetBasicAuth(user, password)
	}

	start := time.Now()
	resp, err := client.Do(req)
	latencyMs = int(time.Since(start).Milliseconds())
	if err != nil {
		return latencyMs, fmt.Errorf("jenkins: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	switch resp.StatusCode {
	case http.StatusOK:
		s := strings.TrimSpace(string(body))
		if len(s) > 0 && s[0] == '<' {
			return latencyMs, fmt.Errorf("jenkins: received HTML instead of metrics response (check URL is the Jenkins root and Basic auth uses username + API token)")
		}
		return latencyMs, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		if am == "none" {
			return latencyMs, fmt.Errorf("jenkins returned %d (use Basic auth with username and API token)", resp.StatusCode)
		}
		return latencyMs, fmt.Errorf("jenkins authentication failed (%d)", resp.StatusCode)
	case http.StatusNotFound:
		return latencyMs, fmt.Errorf("jenkins: %s not found (is the metrics plugin installed?)", jenkinsPingPath)
	default:
		return latencyMs, fmt.Errorf("jenkins returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}
