package store

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/devops-status/be/internal/secrets"
	"github.com/google/uuid"
)

// ResolveTelemetryProbeConfig expands service_provider references into adapter-ready JSON (http url, prometheus endpoint + creds).
// For adapters other than http and prometheus, returns raw unchanged.
func (s *Store) ResolveTelemetryProbeConfig(ctx context.Context, sec *secrets.Store, adapter string, raw json.RawMessage) (json.RawMessage, error) {
	switch adapter {
	case "http":
		return s.resolveHTTPFromProvider(ctx, raw)
	case "prometheus":
		return s.resolvePrometheusFromProvider(ctx, sec, raw)
	default:
		return raw, nil
	}
}

func (s *Store) resolveHTTPFromProvider(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse http config: %w", err)
	}
	spIDStr, _ := cfg["service_provider_id"].(string)
	spIDStr = strings.TrimSpace(spIDStr)
	if spIDStr == "" {
		return nil, fmt.Errorf("service_provider_id is required for http telemetry")
	}
	spID, err := uuid.Parse(spIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid service_provider_id")
	}
	path, _ := cfg["path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("http path is required")
	}
	p, err := s.GetServiceProviderByID(ctx, spID)
	if err != nil {
		return nil, fmt.Errorf("service provider: %w", err)
	}
	if p.ProviderType != "tcp" {
		return nil, fmt.Errorf("service provider is not tcp type")
	}
	fullURL, err := joinTCPProviderURL(p.Host, p.Port, path)
	if err != nil {
		return nil, err
	}
	cfg["url"] = fullURL
	delete(cfg, "service_provider_id")
	delete(cfg, "path")
	out, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal http config: %w", err)
	}
	return out, nil
}

func joinTCPProviderURL(host string, port int, path string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("tcp provider host is empty")
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("tcp provider port is invalid")
	}
	scheme := "https"
	if port == 80 {
		scheme = "http"
	}
	hp := net.JoinHostPort(host, strconv.Itoa(port))
	base, err := url.Parse(scheme + "://" + hp + "/")
	if err != nil {
		return "", fmt.Errorf("build base url: %w", err)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	rel, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("parse path: %w", err)
	}
	return base.ResolveReference(rel).String(), nil
}

func (s *Store) resolvePrometheusFromProvider(ctx context.Context, sec *secrets.Store, raw json.RawMessage) (json.RawMessage, error) {
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse prometheus config: %w", err)
	}
	spIDStr, _ := cfg["service_provider_id"].(string)
	spIDStr = strings.TrimSpace(spIDStr)
	if spIDStr == "" {
		return nil, fmt.Errorf("service_provider_id is required for prometheus telemetry")
	}
	spID, err := uuid.Parse(spIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid service_provider_id")
	}
	p, err := s.GetServiceProviderByID(ctx, spID)
	if err != nil {
		return nil, fmt.Errorf("service provider: %w", err)
	}
	if p.ProviderType != "prometheus" {
		return nil, fmt.Errorf("service provider is not prometheus type")
	}
	endpoint, authMethod, err := prometheusEndpointFromProviderConfig(p.ConfigJSON)
	if err != nil {
		return nil, err
	}
	if endpoint == "" {
		return nil, fmt.Errorf("prometheus provider has no endpoint")
	}
	cfg["endpoint"] = endpoint
	cfg["auth_method"] = authMethod
	delete(cfg, "service_provider_id")

	if sec != nil {
		creds, err := sec.ReadServiceProviderPrometheusCredentials(ctx, spID)
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

	out, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal prometheus config: %w", err)
	}
	return out, nil
}

func prometheusEndpointFromProviderConfig(configJSON any) (endpoint, authMethod string, err error) {
	b, err := json.Marshal(configJSON)
	if err != nil {
		return "", "", fmt.Errorf("provider config_json")
	}
	var m struct {
		Endpoint   string `json:"endpoint"`
		AuthMethod string `json:"auth_method"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", "", fmt.Errorf("provider config_json")
	}
	am := strings.ToLower(strings.TrimSpace(m.AuthMethod))
	if am == "" {
		am = "none"
	}
	return strings.TrimSpace(m.Endpoint), am, nil
}
