package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/devops-status/be/internal/model"
	"github.com/devops-status/be/internal/providerverify"
	"github.com/devops-status/be/internal/secrets"
	"github.com/google/uuid"
)

// LivenessArtifactoryTelemetry is optional bandwidth-test settings in telemetry config_json (liveness + service_provider).
type LivenessArtifactoryTelemetry struct {
	DownloadRepositoryPath string `json:"download_repository_path"`
	DownloadMaxBytes       *int   `json:"download_max_bytes,omitempty"`
}

func (s *Store) resolveLivenessProbeConfig(ctx context.Context, sec *secrets.Store, raw json.RawMessage) (json.RawMessage, error) {
	var env struct {
		Source            string                        `json:"source"`
		Kubernetes        map[string]any                `json:"kubernetes"`
		ServiceProviderID string                        `json:"service_provider_id"`
		Artifactory       *LivenessArtifactoryTelemetry `json:"artifactory,omitempty"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("parse liveness config: %w", err)
	}
	src := strings.ToLower(strings.TrimSpace(env.Source))
	if env.Artifactory != nil && src == "kubernetes" {
		return nil, fmt.Errorf("liveness artifactory settings are only valid when source is service_provider")
	}
	switch src {
	case "kubernetes":
		if len(env.Kubernetes) == 0 {
			return nil, fmt.Errorf("liveness kubernetes block is required when source is kubernetes")
		}
		inner, err := json.Marshal(env.Kubernetes)
		if err != nil {
			return nil, err
		}
		return marshalLivenessWire("kubernetes", inner)
	case "service_provider":
		spIDStr := strings.TrimSpace(env.ServiceProviderID)
		if spIDStr == "" {
			return nil, fmt.Errorf("service_provider_id is required for liveness service_provider source")
		}
		spID, err := uuid.Parse(spIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid service_provider_id")
		}
		p, err := s.GetServiceProviderByID(ctx, spID)
		if err != nil {
			return nil, fmt.Errorf("service provider: %w", err)
		}
		if env.Artifactory != nil && strings.TrimSpace(env.Artifactory.DownloadRepositoryPath) != "" {
			if strings.ToLower(strings.TrimSpace(p.ProviderType)) != "artifactory" {
				return nil, fmt.Errorf("liveness artifactory.download_repository_path is only valid for an artifactory service provider")
			}
			if err := providerverify.ValidateArtifactoryDownloadRepositoryPath(env.Artifactory.DownloadRepositoryPath); err != nil {
				return nil, fmt.Errorf("liveness artifactory: %w", err)
			}
			if env.Artifactory.DownloadMaxBytes != nil {
				n := int64(*env.Artifactory.DownloadMaxBytes)
				if n < 1 || n > providerverify.MaxArtifactoryDownloadMaxBytes {
					return nil, fmt.Errorf("liveness artifactory.download_max_bytes must be between 1 and %d", providerverify.MaxArtifactoryDownloadMaxBytes)
				}
			}
		}
		return s.resolveLivenessFromServiceProvider(ctx, sec, p, env.Artifactory)
	default:
		return nil, fmt.Errorf("liveness source must be kubernetes or service_provider")
	}
}

func marshalLivenessWire(check string, config json.RawMessage) (json.RawMessage, error) {
	type wire struct {
		LivenessCheck string          `json:"liveness_check"`
		Config        json.RawMessage `json:"config"`
	}
	return json.Marshal(wire{LivenessCheck: check, Config: config})
}

func (s *Store) resolveLivenessFromServiceProvider(ctx context.Context, sec *secrets.Store, p *model.ServiceProvider, telAf *LivenessArtifactoryTelemetry) (json.RawMessage, error) {
	pt := strings.ToLower(strings.TrimSpace(p.ProviderType))
	switch pt {
	case "prometheus":
		return s.livenessWirePrometheus(ctx, sec, p)
	case "grafana":
		return s.livenessWireGrafana(ctx, sec, p)
	case "elasticsearch":
		return s.livenessWireElasticsearch(ctx, sec, p)
	case "jenkins":
		return s.livenessWireJenkins(ctx, sec, p)
	case "artifactory":
		return s.livenessWireArtifactory(ctx, sec, p, telAf)
	case "rancher":
		return s.livenessWireRancher(ctx, sec, p)
	default:
		return nil, fmt.Errorf("service provider type %q is not supported for liveness telemetry", p.ProviderType)
	}
}

func (s *Store) livenessWirePrometheus(ctx context.Context, sec *secrets.Store, p *model.ServiceProvider) (json.RawMessage, error) {
	endpoint, authMethod, insecureSkipTLS, err := prometheusEndpointFromProviderConfig(p.ConfigJSON)
	if err != nil {
		return nil, err
	}
	if endpoint == "" {
		return nil, fmt.Errorf("prometheus provider has no endpoint")
	}
	cfg := map[string]any{
		"endpoint":    endpoint,
		"auth_method": authMethod,
		"query":       "vector(1)",
		"qos_mode":    true,
		"timeout_ms":  15000,
		"operator":    "gte",
		"threshold":   0.0,
	}
	cfg["insecure_skip_tls"] = insecureSkipTLS
	if sec != nil {
		creds, err := sec.ReadServiceProviderPrometheusCredentials(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("prometheus provider credentials: %w", err)
		}
		if creds.BearerToken != "" {
			cfg["bearer_token"] = creds.BearerToken
		}
		if creds.Username != "" {
			cfg["username"] = creds.Username
		}
		if creds.Password != "" {
			cfg["password"] = creds.Password
		}
	}
	inner, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return marshalLivenessWire("prometheus", inner)
}

func rancherConnFromProviderJSON(configJSON any) (baseURL, authMethod string, insecureSkipTLS bool, err error) {
	b, err := json.Marshal(configJSON)
	if err != nil {
		return "", "", false, fmt.Errorf("provider config_json")
	}
	var m struct {
		BaseURL         string `json:"base_url"`
		AuthMethod      string `json:"auth_method"`
		InsecureSkipTLS bool   `json:"insecure_skip_tls"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", "", false, fmt.Errorf("provider config_json")
	}
	am := strings.ToLower(strings.TrimSpace(m.AuthMethod))
	if am == "" {
		am = "none"
	}
	return strings.TrimSpace(m.BaseURL), am, m.InsecureSkipTLS, nil
}

func urlAuthFromProviderJSON(configJSON any, field string) (urlStr, authMethod string, err error) {
	b, err := json.Marshal(configJSON)
	if err != nil {
		return "", "", fmt.Errorf("provider config_json")
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return "", "", fmt.Errorf("provider config_json")
	}
	var raw string
	switch field {
	case "base_url":
		raw, _ = m["base_url"].(string)
	case "url":
		raw, _ = m["url"].(string)
	default:
		return "", "", fmt.Errorf("invalid field")
	}
	am, _ := m["auth_method"].(string)
	am = strings.ToLower(strings.TrimSpace(am))
	if am == "" {
		am = "none"
	}
	return strings.TrimSpace(raw), am, nil
}

func (s *Store) livenessWireGrafana(ctx context.Context, sec *secrets.Store, p *model.ServiceProvider) (json.RawMessage, error) {
	baseURL, authMethod, err := urlAuthFromProviderJSON(p.ConfigJSON, "base_url")
	if err != nil || baseURL == "" {
		return nil, fmt.Errorf("grafana provider base_url is required")
	}
	cfg := map[string]any{
		"base_url":    baseURL,
		"auth_method": authMethod,
		"timeout_ms":  15000,
	}
	if sec != nil && authMethod == "basic" {
		c, err := sec.ReadServiceProviderHTTPBasic(ctx, p.ID, secrets.KeyGrafanaCreds)
		if err != nil {
			return nil, fmt.Errorf("grafana credentials: %w", err)
		}
		cfg["username"] = c.Username
		cfg["password"] = c.Password
	}
	inner, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return marshalLivenessWire("grafana", inner)
}

func (s *Store) livenessWireElasticsearch(ctx context.Context, sec *secrets.Store, p *model.ServiceProvider) (json.RawMessage, error) {
	u, authMethod, err := urlAuthFromProviderJSON(p.ConfigJSON, "url")
	if err != nil || u == "" {
		return nil, fmt.Errorf("elasticsearch provider url is required")
	}
	cfg := map[string]any{
		"url":        u,
		"timeout_ms": 15000,
	}
	if sec != nil && authMethod == "basic" {
		c, err := sec.ReadServiceProviderHTTPBasic(ctx, p.ID, secrets.KeyElasticsearchCreds)
		if err != nil {
			return nil, fmt.Errorf("elasticsearch credentials: %w", err)
		}
		cfg["username"] = c.Username
		cfg["password"] = c.Password
	}
	inner, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return marshalLivenessWire("elasticsearch", inner)
}

func (s *Store) livenessWireJenkins(ctx context.Context, sec *secrets.Store, p *model.ServiceProvider) (json.RawMessage, error) {
	u, authMethod, err := urlAuthFromProviderJSON(p.ConfigJSON, "url")
	if err != nil || u == "" {
		return nil, fmt.Errorf("jenkins provider url is required")
	}
	cfg := map[string]any{
		"url":         u,
		"auth_method": authMethod,
		"timeout_ms":  15000,
	}
	if sec != nil && authMethod == "basic" {
		c, err := sec.ReadServiceProviderHTTPBasic(ctx, p.ID, secrets.KeyJenkinsCreds)
		if err != nil {
			return nil, fmt.Errorf("jenkins credentials: %w", err)
		}
		cfg["username"] = c.Username
		cfg["password"] = c.Password
	}
	inner, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return marshalLivenessWire("jenkins", inner)
}

func (s *Store) livenessWireArtifactory(ctx context.Context, sec *secrets.Store, p *model.ServiceProvider, telAf *LivenessArtifactoryTelemetry) (json.RawMessage, error) {
	baseURL, authMethod, err := urlAuthFromProviderJSON(p.ConfigJSON, "base_url")
	if err != nil || baseURL == "" {
		return nil, fmt.Errorf("artifactory provider base_url is required")
	}
	cfg := map[string]any{
		"base_url":    baseURL,
		"auth_method": authMethod,
		"timeout_ms":  15000,
	}
	if sec != nil && authMethod == "basic" {
		c, err := sec.ReadServiceProviderHTTPBasic(ctx, p.ID, secrets.KeyArtifactoryCreds)
		if err != nil {
			return nil, fmt.Errorf("artifactory credentials: %w", err)
		}
		cfg["username"] = c.Username
		cfg["password"] = c.Password
	}
	if telAf != nil && strings.TrimSpace(telAf.DownloadRepositoryPath) != "" {
		cfg["download_repository_path"] = strings.TrimSpace(telAf.DownloadRepositoryPath)
		if telAf.DownloadMaxBytes != nil {
			cfg["download_max_bytes"] = *telAf.DownloadMaxBytes
		}
	}
	inner, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return marshalLivenessWire("artifactory", inner)
}

func (s *Store) livenessWireRancher(ctx context.Context, sec *secrets.Store, p *model.ServiceProvider) (json.RawMessage, error) {
	baseURL, authMethod, insecureSkipTLS, err := rancherConnFromProviderJSON(p.ConfigJSON)
	if err != nil || baseURL == "" {
		return nil, fmt.Errorf("rancher provider base_url is required")
	}
	cfg := map[string]any{
		"base_url":            baseURL,
		"auth_method":         authMethod,
		"insecure_skip_tls":   insecureSkipTLS,
		"timeout_ms":          15000,
	}
	if sec != nil && authMethod == "bearer" {
		c, err := sec.ReadServiceProviderRancherCredentials(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("rancher credentials: %w", err)
		}
		cfg["bearer_token"] = c.BearerToken
	}
	inner, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return marshalLivenessWire("rancher", inner)
}
