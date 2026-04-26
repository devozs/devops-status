package providerverify

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const rancherVerifyTimeout = 15 * time.Second

func newRancherHTTPClient(insecureSkipTLS bool) *http.Client {
	timeout := rancherVerifyTimeout
	if insecureSkipTLS {
		return &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
					MinVersion:         tls.VersionTLS12,
				},
			},
		}
	}
	tr, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Client{Timeout: timeout}
	}
	return &http.Client{Timeout: timeout, Transport: tr.Clone()}
}

// applyRancherV3Auth sets auth for the Rancher v3 API.
// Official docs: v3 uses HTTP Basic with API keys (access key = username, secret key = password).
// The UI "Bearer Token" value is typically "accessKey:secretKey" — use that whole string here.
// If the value has no ':', we fall back to Authorization: Bearer <token> for compatibility.
func applyRancherV3Auth(req *http.Request, apiToken string) {
	t := strings.TrimSpace(apiToken)
	if i := strings.Index(t, ":"); i > 0 {
		user := t[:i]
		pass := t[i+1:]
		if user != "" {
			req.SetBasicAuth(user, pass)
			return
		}
	}
	req.Header.Set("Authorization", "Bearer "+t)
}

// VerifyRancher checks Rancher 2.x reachability.
// authMethod "none" uses GET /ping (HTTP 2xx counts as OK; avoids strict body checks for proxies).
// authMethod "bearer" uses GET /v3/ with API key auth (see applyRancherV3Auth).
func VerifyRancher(ctx context.Context, baseURL, authMethod, bearerToken string, insecureSkipTLS bool) (latencyMs int, err error) {
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
	if am != "none" && am != "bearer" {
		return 0, fmt.Errorf("auth_method must be none or bearer for rancher")
	}
	if am == "bearer" && strings.TrimSpace(bearerToken) == "" {
		return 0, fmt.Errorf("bearer auth requires an API token")
	}

	var reqURL string
	switch am {
	case "none":
		reqURL = base + "/ping"
	case "bearer":
		// Trailing slash avoids 301 → follow-up request that can drop Authorization (Basic/Bearer).
		reqURL = base + "/v3/"
	}

	client := newRancherHTTPClient(insecureSkipTLS)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, err
	}
	if am == "bearer" {
		applyRancherV3Auth(req, bearerToken)
	}

	start := time.Now()
	resp, err := client.Do(req)
	latencyMs = int(time.Since(start).Milliseconds())
	if err != nil {
		return latencyMs, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return latencyMs, nil
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		if am == "none" {
			return latencyMs, fmt.Errorf("rancher returned %d on /ping (ingress may block it; try API key auth)", resp.StatusCode)
		}
		return latencyMs, fmt.Errorf("rancher API authentication failed (%d): paste the full key from Rancher (Access Key:Secret Key); v3 uses HTTP Basic, not a separate username field", resp.StatusCode)
	default:
		return latencyMs, fmt.Errorf("rancher returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}
