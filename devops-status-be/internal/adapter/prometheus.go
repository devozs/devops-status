package adapter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type PrometheusAdapter struct {
	client *http.Client
}

type PrometheusConfig struct {
	Endpoint    string  `json:"endpoint"`
	AuthMethod  string  `json:"auth_method,omitempty"` // none | basic | bearer
	Username    string  `json:"username,omitempty"`
	Password    string  `json:"password,omitempty"`
	BearerToken string  `json:"bearer_token,omitempty"`
	Query       string  `json:"query"`
	Threshold   float64 `json:"threshold,omitempty"`
	Operator    string  `json:"operator,omitempty"`
	TimeoutMs   int     `json:"timeout_ms,omitempty"`
	QoSMode     bool    `json:"qos_mode,omitempty"` // when true, Success if query returns scalar; thresholds applied in QoS engine
}

func NewPrometheusAdapter() *PrometheusAdapter {
	return &PrometheusAdapter{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *PrometheusAdapter) Name() string { return "prometheus" }

func (a *PrometheusAdapter) Probe(ctx context.Context, configRaw json.RawMessage) (*ProbeResult, error) {
	var cfg PrometheusConfig
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		return nil, fmt.Errorf("parse prometheus config: %w", err)
	}

	if cfg.Operator == "" {
		cfg.Operator = "gte"
	}

	timeout := 30 * time.Second
	if cfg.TimeoutMs > 0 {
		timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
	}
	client := &http.Client{Timeout: timeout}

	queryURL := fmt.Sprintf("%s/api/v1/query?query=%s", strings.TrimRight(cfg.Endpoint, "/"), url.QueryEscape(cfg.Query))
	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create prometheus request: %w", err)
	}
	applyPrometheusAuth(req, cfg)

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)

	result := &ProbeResult{
		LatencyMs: int(latency.Milliseconds()),
		ProbedAt:  start,
		Metadata:  map[string]any{"query": cfg.Query, "endpoint": cfg.Endpoint},
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Trace = fmt.Sprintf("Prometheus query failed: %v", err)
		return result, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("read response: %v", err)
		return result, nil
	}

	if resp.StatusCode != 200 {
		result.Success = false
		result.Error = fmt.Sprintf("prometheus returned %d", resp.StatusCode)
		result.Trace = string(body)
		return result, nil
	}

	var promResp promAPIResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("parse response: %v", err)
		return result, nil
	}

	if promResp.Status != "success" {
		result.Success = false
		result.Error = fmt.Sprintf("query status: %s", promResp.Status)
		return result, nil
	}

	value, ok := extractScalarValue(promResp.Data)
	if !ok {
		result.Success = false
		result.Error = "no scalar value in result"
		return result, nil
	}

	result.RawValue = value
	if cfg.QoSMode {
		result.Success = true
		result.Trace = fmt.Sprintf("Prometheus query=%s value=%.4f (qos mode)", cfg.Query, value)
	} else {
		result.Success = compareValue(value, cfg.Threshold, cfg.Operator)
		result.Trace = fmt.Sprintf("Prometheus query=%s value=%.4f threshold=%.4f op=%s", cfg.Query, value, cfg.Threshold, cfg.Operator)
	}

	return result, nil
}

type promAPIResponse struct {
	Status string         `json:"status"`
	Data   map[string]any `json:"data"`
}

func extractScalarValue(data map[string]any) (float64, bool) {
	resultType, _ := data["resultType"].(string)
	resultArr, ok := data["result"].([]any)
	if !ok || len(resultArr) == 0 {
		return 0, false
	}

	switch resultType {
	case "vector":
		first, ok := resultArr[0].(map[string]any)
		if !ok {
			return 0, false
		}
		valArr, ok := first["value"].([]any)
		if !ok || len(valArr) < 2 {
			return 0, false
		}
		strVal, ok := valArr[1].(string)
		if !ok {
			return 0, false
		}
		v, err := strconv.ParseFloat(strVal, 64)
		return v, err == nil

	case "scalar":
		if len(resultArr) >= 2 {
			strVal, ok := resultArr[1].(string)
			if !ok {
				return 0, false
			}
			v, err := strconv.ParseFloat(strVal, 64)
			return v, err == nil
		}
	}

	return 0, false
}

func applyPrometheusAuth(req *http.Request, cfg PrometheusConfig) {
	switch strings.ToLower(strings.TrimSpace(cfg.AuthMethod)) {
	case "basic":
		if cfg.Username != "" || cfg.Password != "" {
			u := cfg.Username + ":" + cfg.Password
			req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(u)))
		}
	case "bearer":
		if cfg.BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.BearerToken)
		}
	default:
		if cfg.BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.BearerToken)
		}
	}
}

// VerifyPrometheusIntegration runs a minimal instant query and requires HTTP 200 with Prometheus status=success.
// For auth_method basic or bearer, 401/403 are failures (wrong or missing credentials).
func VerifyPrometheusIntegration(ctx context.Context, cfg PrometheusConfig) (latencyMs int, err error) {
	timeout := 15 * time.Second
	if cfg.TimeoutMs > 0 {
		timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
	}
	client := &http.Client{Timeout: timeout}
	base := strings.TrimRight(cfg.Endpoint, "/")
	if base == "" {
		return 0, fmt.Errorf("endpoint is required")
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		return 0, fmt.Errorf("endpoint must start with http:// or https://")
	}

	q := "vector(1)"
	queryURL := fmt.Sprintf("%s/api/v1/query?query=%s", base, url.QueryEscape(q))
	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return 0, err
	}
	applyPrometheusAuth(req, cfg)

	authMode := strings.ToLower(strings.TrimSpace(cfg.AuthMethod))
	start := time.Now()
	resp, err := client.Do(req)
	latencyMs = int(time.Since(start).Milliseconds())
	if err != nil {
		return latencyMs, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return latencyMs, fmt.Errorf("read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusUnauthorized, http.StatusForbidden:
		if authMode == "none" || authMode == "" {
			return latencyMs, fmt.Errorf("prometheus returned %d (authentication may be required)", resp.StatusCode)
		}
		return latencyMs, fmt.Errorf("prometheus authentication failed (%d)", resp.StatusCode)
	default:
		return latencyMs, fmt.Errorf("prometheus returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var promResp promAPIResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return latencyMs, fmt.Errorf("parse prometheus JSON: %w", err)
	}
	if promResp.Status != "success" {
		return latencyMs, fmt.Errorf("prometheus query status: %s", promResp.Status)
	}
	if _, ok := extractScalarValue(promResp.Data); !ok {
		return latencyMs, fmt.Errorf("unexpected query result (expected scalar sample)")
	}
	return latencyMs, nil
}

// TestPrometheusConnectivity checks reachability (config endpoint or minimal query).
func TestPrometheusConnectivity(ctx context.Context, cfg PrometheusConfig) error {
	timeout := 15 * time.Second
	if cfg.TimeoutMs > 0 {
		timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
	}
	client := &http.Client{Timeout: timeout}
	base := strings.TrimRight(cfg.Endpoint, "/")
	try := []string{
		base + "/api/v1/status/config",
		base + "/api/v1/query?query=1",
	}
	var lastErr error
	for _, u := range try {
		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err != nil {
			lastErr = err
			continue
		}
		applyPrometheusAuth(req, cfg)
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
			return nil
		}
		lastErr = fmt.Errorf("status %d", resp.StatusCode)
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("unreachable")
}

// CompareValue compares a numeric probe value to a threshold using the operator name.
func CompareValue(value, threshold float64, operator string) bool {
	return compareValue(value, threshold, operator)
}

func compareValue(value, threshold float64, operator string) bool {
	switch operator {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	default:
		return value >= threshold
	}
}
