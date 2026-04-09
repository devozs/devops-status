package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type PrometheusAdapter struct {
	client *http.Client
}

type PrometheusConfig struct {
	Endpoint  string `json:"endpoint"`
	Query     string `json:"query"`
	Threshold float64 `json:"threshold,omitempty"`
	Operator  string `json:"operator,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
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

	queryURL := fmt.Sprintf("%s/api/v1/query?query=%s", cfg.Endpoint, url.QueryEscape(cfg.Query))
	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create prometheus request: %w", err)
	}

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
	result.Success = compareValue(value, cfg.Threshold, cfg.Operator)
	result.Trace = fmt.Sprintf("Prometheus query=%s value=%.4f threshold=%.4f op=%s", cfg.Query, value, cfg.Threshold, cfg.Operator)

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
