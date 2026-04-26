package liveness

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/providerverify"
)

type grafanaLivenessCfg struct {
	BaseURL    string `json:"base_url"`
	AuthMethod string `json:"auth_method"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	TimeoutMs  int    `json:"timeout_ms,omitempty"`
}

type esLivenessCfg struct {
	URL       string `json:"url"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

type jenkinsLivenessCfg struct {
	URL        string `json:"url"`
	AuthMethod string `json:"auth_method"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	TimeoutMs  int    `json:"timeout_ms,omitempty"`
}

type artifactoryLivenessCfg struct {
	BaseURL                string `json:"base_url"`
	AuthMethod             string `json:"auth_method"`
	Username               string `json:"username,omitempty"`
	Password               string `json:"password,omitempty"`
	TimeoutMs              int    `json:"timeout_ms,omitempty"`
	DownloadRepositoryPath string `json:"download_repository_path,omitempty"`
	DownloadMaxBytes       int    `json:"download_max_bytes,omitempty"`
}

type rancherLivenessCfg struct {
	BaseURL         string `json:"base_url"`
	AuthMethod      string `json:"auth_method"`
	BearerToken     string `json:"bearer_token,omitempty"`
	InsecureSkipTLS bool   `json:"insecure_skip_tls,omitempty"`
	TimeoutMs       int    `json:"timeout_ms,omitempty"`
}

func probeRancher(ctx context.Context, cfg json.RawMessage) (*adapter.ProbeResult, error) {
	var c rancherLivenessCfg
	if err := json.Unmarshal(cfg, &c); err != nil {
		return nil, fmt.Errorf("parse rancher liveness config: %w", err)
	}
	start := time.Now()
	ms, err := providerverify.VerifyRancher(ctx, c.BaseURL, c.AuthMethod, c.BearerToken, c.InsecureSkipTLS)
	return httpStyleResult(start, ms, err, map[string]any{"liveness_check": CheckRancher})
}

func probeGrafana(ctx context.Context, cfg json.RawMessage) (*adapter.ProbeResult, error) {
	var c grafanaLivenessCfg
	if err := json.Unmarshal(cfg, &c); err != nil {
		return nil, fmt.Errorf("parse grafana liveness config: %w", err)
	}
	am := strings.ToLower(strings.TrimSpace(c.AuthMethod))
	if am == "" {
		am = "none"
	}
	start := time.Now()
	ms, err := providerverify.VerifyGrafana(ctx, c.BaseURL, am, c.Username, c.Password)
	return httpStyleResult(start, ms, err, map[string]any{"liveness_check": CheckGrafana})
}

func probeElasticsearch(ctx context.Context, cfg json.RawMessage) (*adapter.ProbeResult, error) {
	var c esLivenessCfg
	if err := json.Unmarshal(cfg, &c); err != nil {
		return nil, fmt.Errorf("parse elasticsearch liveness config: %w", err)
	}
	start := time.Now()
	ms, err := providerverify.VerifyElasticsearch(ctx, c.URL, c.Username, c.Password)
	return httpStyleResult(start, ms, err, map[string]any{"liveness_check": CheckElasticsearch})
}

func probeJenkins(ctx context.Context, cfg json.RawMessage) (*adapter.ProbeResult, error) {
	var c jenkinsLivenessCfg
	if err := json.Unmarshal(cfg, &c); err != nil {
		return nil, fmt.Errorf("parse jenkins liveness config: %w", err)
	}
	am := strings.ToLower(strings.TrimSpace(c.AuthMethod))
	if am == "" {
		am = "none"
	}
	start := time.Now()
	ms, err := providerverify.VerifyJenkins(ctx, c.URL, am, c.Username, c.Password)
	return httpStyleResult(start, ms, err, map[string]any{"liveness_check": CheckJenkins})
}

func probeArtifactory(ctx context.Context, cfg json.RawMessage) (*adapter.ProbeResult, error) {
	var c artifactoryLivenessCfg
	if err := json.Unmarshal(cfg, &c); err != nil {
		return nil, fmt.Errorf("parse artifactory liveness config: %w", err)
	}
	am := strings.ToLower(strings.TrimSpace(c.AuthMethod))
	if am == "" {
		am = "none"
	}
	probeStart := time.Now()
	pingMs, err := providerverify.VerifyArtifactory(ctx, c.BaseURL, am, c.Username, c.Password)
	dlPath := strings.TrimSpace(c.DownloadRepositoryPath)
	if dlPath == "" {
		return httpStyleResult(probeStart, pingMs, err, map[string]any{"liveness_check": CheckArtifactory})
	}
	if err != nil {
		return httpStyleResult(probeStart, pingMs, err, map[string]any{"liveness_check": CheckArtifactory})
	}
	maxBytes := int64(c.DownloadMaxBytes)
	br, dlMs, avg, derr := providerverify.MeasureArtifactoryDownload(ctx, c.BaseURL, am, c.Username, c.Password, dlPath, maxBytes)
	meta := map[string]any{
		"liveness_check":       CheckArtifactory,
		"download_bytes":       br,
		"download_duration_ms": dlMs,
		"avg_bytes_per_sec":    avg,
	}
	if derr != nil {
		return &adapter.ProbeResult{
			LatencyMs: pingMs,
			ProbedAt:  probeStart,
			Success:   false,
			RawValue:  0,
			Error:     derr.Error(),
			Trace:     fmt.Sprintf("ping ok (%dms); download failed: %v", pingMs, derr),
			Metadata:  meta,
		}, nil
	}
	return &adapter.ProbeResult{
		LatencyMs: pingMs,
		ProbedAt:  probeStart,
		Success:   true,
		RawValue:  avg,
		Trace:     fmt.Sprintf("ping ok (%dms); sampled %d bytes in %dms (avg %.0f B/s)", pingMs, br, dlMs, avg),
		Metadata:  meta,
	}, nil
}

func httpStyleResult(start time.Time, latencyMs int, err error, meta map[string]any) (*adapter.ProbeResult, error) {
	if meta == nil {
		meta = map[string]any{}
	}
	res := &adapter.ProbeResult{
		LatencyMs: latencyMs,
		ProbedAt:  start,
		Metadata:  meta,
	}
	if err != nil {
		res.Success = false
		res.RawValue = 0
		res.Error = err.Error()
		res.Trace = err.Error()
		return res, nil
	}
	res.Success = true
	res.RawValue = 1
	res.Trace = "liveness ok"
	return res, nil
}
