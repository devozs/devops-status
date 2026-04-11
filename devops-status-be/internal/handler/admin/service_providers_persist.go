package admin

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/devops-status/be/internal/model"
	"github.com/devops-status/be/internal/secrets"
)

func (h *ServiceProvidersHandler) persistHTTPBasicProviderCreds(ctx context.Context, providerID uuid.UUID, pt string, configJSON any, creds map[string]string, mergeExisting bool) error {
	if h.secret == nil {
		return errors.New("secrets store not configured")
	}
	key, ok := secretKeyForHTTPBasicProvider(pt)
	if !ok {
		return nil
	}
	var authMethod string
	switch pt {
	case "grafana":
		_, authMethod, _ = parseGrafanaConfigJSON(configJSON)
	case "elasticsearch":
		_, authMethod, _ = parseElasticsearchConfigJSON(configJSON)
	case "jenkins":
		_, authMethod, _ = parseJenkinsConfigJSON(configJSON)
	case "artifactory":
		_, authMethod, _ = parseArtifactoryConfigJSON(configJSON)
	default:
		return nil
	}
	am := strings.ToLower(strings.TrimSpace(authMethod))
	if am == "" {
		am = "none"
	}
	if am == "none" {
		return h.secret.Delete(ctx, secrets.OwnerServiceProvider, providerID, key)
	}
	base := secrets.HTTPBasicCredentials{}
	if mergeExisting {
		base, _ = h.secret.ReadServiceProviderHTTPBasic(ctx, providerID, key)
	}
	merged := mergeHTTPBasicCreds(creds, base)
	if merged.Username == "" || merged.Password == "" {
		return errors.New("basic auth requires username and password")
	}
	return h.secret.WriteServiceProviderHTTPBasic(ctx, providerID, key, merged)
}

func (h *ServiceProvidersHandler) persistProviderCredentialsAfterSave(ctx context.Context, p *model.ServiceProvider, body serviceProviderRequestBody, mergeExisting bool) error {
	if h.secret == nil {
		return nil
	}
	switch p.ProviderType {
	case "prometheus":
		_, authMethod, err := parsePrometheusConfigJSON(p.ConfigJSON)
		if err != nil {
			return err
		}
		if authMethod == "none" {
			return h.secret.Delete(ctx, secrets.OwnerServiceProvider, p.ID, secrets.KeyPrometheusCreds)
		}
		base := secrets.PrometheusCredentials{}
		if mergeExisting {
			base, _ = h.secret.ReadServiceProviderPrometheusCredentials(ctx, p.ID)
		}
		merged := mergePrometheusCreds(body.Credentials, base)
		return h.persistServiceProviderPrometheusCreds(ctx, p.ID, authMethod, merged)
	case "grafana", "elasticsearch", "jenkins", "artifactory":
		return h.persistHTTPBasicProviderCreds(ctx, p.ID, p.ProviderType, p.ConfigJSON, body.Credentials, mergeExisting)
	default:
		return nil
	}
}
