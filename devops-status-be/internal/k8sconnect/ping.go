package k8sconnect

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PingResult is returned after calling the Kubernetes API /version (same family as kubectl cluster-info / version).
type PingResult struct {
	OK         bool   `json:"ok"`
	K8sVersion string `json:"k8s_version,omitempty"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code,omitempty"`
}

// PingVersion GETs /version on the cluster API server using the stored bearer token.
func PingVersion(ctx context.Context, endpoint, bearerToken string, insecureSkipTLS bool) PingResult {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return PingResult{OK: false, Message: "cluster endpoint is empty"}
	}
	if bearerToken == "" {
		return PingResult{OK: false, Message: "no bearer token configured"}
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipTLS},
		},
	}

	url := endpoint + "/version"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return PingResult{OK: false, Message: fmt.Sprintf("build request: %v", err)}
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return PingResult{OK: false, Message: fmt.Sprintf("request failed: %v", err)}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != http.StatusOK {
		return PingResult{
			OK:         false,
			Message:    fmt.Sprintf("GET /version -> HTTP %d", resp.StatusCode),
			StatusCode: resp.StatusCode,
		}
	}

	var ver struct {
		GitVersion string `json:"gitVersion"`
	}
	if err := json.Unmarshal(body, &ver); err != nil {
		return PingResult{
			OK:         true,
			Message:    "connected (could not parse version JSON)",
			StatusCode: resp.StatusCode,
		}
	}
	msg := "GET /version OK"
	if ver.GitVersion != "" {
		msg = fmt.Sprintf("GET /version OK — server %s", ver.GitVersion)
	}
	return PingResult{
		OK:         true,
		K8sVersion: ver.GitVersion,
		Message:    msg,
		StatusCode: resp.StatusCode,
	}
}
