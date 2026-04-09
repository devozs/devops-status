package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPAdapter struct {
	client *http.Client
}

type HTTPConfig struct {
	URL            string            `json:"url"`
	Method         string            `json:"method"`
	Headers        map[string]string `json:"headers,omitempty"`
	Body           string            `json:"body,omitempty"`
	ExpectedStatus int               `json:"expected_status,omitempty"`
	TimeoutMs      int               `json:"timeout_ms,omitempty"`
}

func NewHTTPAdapter() *HTTPAdapter {
	return &HTTPAdapter{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *HTTPAdapter) Name() string { return "http" }

func (a *HTTPAdapter) Probe(ctx context.Context, configRaw json.RawMessage) (*ProbeResult, error) {
	var cfg HTTPConfig
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		return nil, fmt.Errorf("parse http config: %w", err)
	}

	if cfg.Method == "" {
		cfg.Method = "GET"
	}
	if cfg.ExpectedStatus == 0 {
		cfg.ExpectedStatus = 200
	}

	timeout := 30 * time.Second
	if cfg.TimeoutMs > 0 {
		timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
	}

	client := &http.Client{Timeout: timeout}

	var bodyReader io.Reader
	if cfg.Body != "" {
		bodyReader = strings.NewReader(cfg.Body)
	}

	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)

	result := &ProbeResult{
		LatencyMs: int(latency.Milliseconds()),
		ProbedAt:  start,
		Metadata:  map[string]any{"method": cfg.Method, "url": cfg.URL},
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Trace = fmt.Sprintf("HTTP %s %s failed: %v", cfg.Method, cfg.URL, err)
		return result, nil
	}
	defer resp.Body.Close()

	result.RawValue = float64(resp.StatusCode)
	result.Success = resp.StatusCode == cfg.ExpectedStatus
	result.Metadata["status_code"] = resp.StatusCode
	result.Trace = fmt.Sprintf("HTTP %s %s -> %d (%dms)", cfg.Method, cfg.URL, resp.StatusCode, result.LatencyMs)

	if !result.Success {
		result.Error = fmt.Sprintf("expected status %d, got %d", cfg.ExpectedStatus, resp.StatusCode)
	}

	return result, nil
}
