package providerverify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const grafanaVerifyTimeout = 15 * time.Second

// VerifyGrafana GETs /api/health on the Grafana root URL.
func VerifyGrafana(ctx context.Context, baseURL, authMethod, username, password string) (latencyMs int, err error) {
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
		return 0, fmt.Errorf("auth_method must be none or basic for grafana")
	}
	if am == "basic" && (username == "" || password == "") {
		return 0, fmt.Errorf("basic auth requires username and password")
	}

	client := &http.Client{Timeout: grafanaVerifyTimeout}
	reqURL := base + "/api/health"
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
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	switch resp.StatusCode {
	case http.StatusOK:
		var h struct {
			Database string `json:"database"`
		}
		if json.Unmarshal(body, &h) == nil && h.Database != "" && h.Database != "ok" {
			// Non-fatal: some versions omit or differ; still 200
		}
		return latencyMs, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		if am == "none" {
			return latencyMs, fmt.Errorf("grafana returned %d (authentication may be required)", resp.StatusCode)
		}
		return latencyMs, fmt.Errorf("grafana authentication failed (%d)", resp.StatusCode)
	default:
		return latencyMs, fmt.Errorf("grafana returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}
