package admin

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/config"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/model"
	"github.com/devops-status/be/internal/secrets"
	"github.com/devops-status/be/internal/store"
)

const maxServiceProviderImageURLLen = 2000

type ServiceProvidersHandler struct {
	store        *store.Store
	secret       *secrets.Store
	hlctlEnabled bool
	cliLogMax    int
}

func NewServiceProvidersHandler(s *store.Store, sec *secrets.Store, hlctlEnabled bool, cfg *config.Config) *ServiceProvidersHandler {
	logMax := cfg.CLILogMaxBytes
	if logMax <= 0 {
		logMax = adapter.DefaultCLILogMaxBytes
	}
	return &ServiceProvidersHandler{store: s, secret: sec, hlctlEnabled: hlctlEnabled, cliLogMax: logMax}
}

type serviceProviderResponse struct {
	ID                       uuid.UUID `json:"id"`
	Name                     string    `json:"name"`
	Host                     string    `json:"host"`
	Port                     int       `json:"port"`
	ImageURL                 string    `json:"image_url"`
	ProviderType             string    `json:"provider_type"`
	ConfigJSON               any       `json:"config_json"`
	HasStoredCredentials     bool      `json:"has_stored_credentials,omitempty"`
	HasPrometheusCredentials bool      `json:"has_prometheus_credentials,omitempty"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

func (h *ServiceProvidersHandler) responseFrom(ctx context.Context, p *model.ServiceProvider) serviceProviderResponse {
	out := serviceProviderResponse{
		ID:           p.ID,
		Name:         p.Name,
		Host:         p.Host,
		Port:         p.Port,
		ImageURL:     p.ImageURL,
		ProviderType: p.ProviderType,
		ConfigJSON:   p.ConfigJSON,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
	out.HasStoredCredentials = h.hasStoredCredentialsForProvider(ctx, p)
	if p.ProviderType == "prometheus" {
		out.HasPrometheusCredentials = out.HasStoredCredentials
	}
	return out
}

func (h *ServiceProvidersHandler) hasStoredCredentialsForProvider(ctx context.Context, p *model.ServiceProvider) bool {
	if h.secret == nil {
		return false
	}
	switch p.ProviderType {
	case "prometheus":
		ok, _ := h.secret.HasServiceProviderPrometheusCredentials(ctx, p.ID)
		return ok
	case "grafana":
		ok, _ := h.secret.HasServiceProviderSecretKey(ctx, p.ID, secrets.KeyGrafanaCreds)
		return ok
	case "elasticsearch":
		ok, _ := h.secret.HasServiceProviderSecretKey(ctx, p.ID, secrets.KeyElasticsearchCreds)
		return ok
	case "jenkins":
		ok, _ := h.secret.HasServiceProviderSecretKey(ctx, p.ID, secrets.KeyJenkinsCreds)
		return ok
	case "artifactory":
		ok, _ := h.secret.HasServiceProviderSecretKey(ctx, p.ID, secrets.KeyArtifactoryCreds)
		return ok
	case "rancher":
		ok, _ := h.secret.HasServiceProviderSecretKey(ctx, p.ID, secrets.KeyRancherCreds)
		return ok
	case "hlctl":
		ok, _ := h.secret.HasServiceProviderSecretKey(ctx, p.ID, secrets.KeyHlctlCreds)
		return ok
	default:
		return false
	}
}

// CheckNameAvailable reports whether a provider name is free (case-insensitive). Names shorter than 3 chars skip DB.
func (h *ServiceProvidersHandler) CheckNameAvailable(w http.ResponseWriter, r *http.Request) {
	nm := strings.TrimSpace(r.URL.Query().Get("name"))
	if len(nm) < 3 {
		handler.WriteJSON(w, http.StatusOK, map[string]any{"available": true, "skipped": true})
		return
	}
	var exclude *uuid.UUID
	if ex := strings.TrimSpace(r.URL.Query().Get("exclude_id")); ex != "" {
		id, err := uuid.Parse(ex)
		if err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid exclude_id")
			return
		}
		exclude = &id
	}
	taken, err := h.store.ServiceProviderNameExists(r.Context(), nm, exclude)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to check name")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]bool{"available": !taken})
}

type verifyServiceProviderBody struct {
	ProviderType      string            `json:"provider_type"`
	Host              string            `json:"host"`
	Port              int               `json:"port"`
	Endpoint          string            `json:"endpoint"`
	BaseURL           string            `json:"base_url"`
	URL               string            `json:"url"`
	AuthMethod        string            `json:"auth_method"`
	DNSHostname       string            `json:"dns_hostname"`
	DNSRecordType     string            `json:"dns_record_type"`
	DNSNameserver     string            `json:"dns_nameserver"`
	InsecureSkipTLS   bool              `json:"insecure_skip_tls"` // prometheus: private CA / self-signed (no omitempty — must decode explicit false/true)
	Credentials       map[string]string `json:"credentials"`
	ServiceProviderID string            `json:"service_provider_id,omitempty"`
}

func mergePrometheusCreds(body map[string]string, stored secrets.PrometheusCredentials) secrets.PrometheusCredentials {
	out := stored
	if body == nil {
		return out
	}
	if v := strings.TrimSpace(body["username"]); v != "" {
		out.Username = v
	}
	if v := strings.TrimSpace(body["password"]); v != "" {
		out.Password = v
	}
	if v := strings.TrimSpace(body["bearer_token"]); v != "" {
		out.BearerToken = v
	}
	return out
}

func mergeRancherCreds(body map[string]string, stored secrets.RancherCredentials) secrets.RancherCredentials {
	out := stored
	if body == nil {
		return out
	}
	if v := strings.TrimSpace(body["bearer_token"]); v != "" {
		out.BearerToken = v
	}
	return out
}

func (h *ServiceProvidersHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListServiceProviders(r.Context())
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list service providers")
		return
	}
	out := make([]serviceProviderResponse, 0, len(list))
	for i := range list {
		out = append(out, h.responseFrom(r.Context(), &list[i]))
	}
	handler.WriteJSON(w, http.StatusOK, out)
}

func (h *ServiceProvidersHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := h.store.GetServiceProviderByID(r.Context(), id)
	if err != nil {
		handler.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	handler.WriteJSON(w, http.StatusOK, h.responseFrom(r.Context(), p))
}

type serviceProviderRequestBody struct {
	store.CreateServiceProviderInput `json:",inline"`
	Credentials                      map[string]string `json:"credentials"`
}

func normalizeServiceProviderPayload(name, host, imageURL string) (string, string, string) {
	name = strings.TrimSpace(name)
	host = strings.TrimSpace(host)
	imageURL = strings.TrimSpace(imageURL)
	if len(imageURL) > maxServiceProviderImageURLLen {
		imageURL = imageURL[:maxServiceProviderImageURLLen]
	}
	return name, host, imageURL
}

func normalizeProviderType(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "tcp"
	}
	return s
}

func (h *ServiceProvidersHandler) persistServiceProviderPrometheusCreds(ctx context.Context, providerID uuid.UUID, authMethod string, merged secrets.PrometheusCredentials) error {
	if h.secret == nil {
		return errors.New("secrets store not configured")
	}
	am := strings.ToLower(strings.TrimSpace(authMethod))
	if am == "none" {
		_ = h.secret.Delete(ctx, secrets.OwnerServiceProvider, providerID, secrets.KeyPrometheusCreds)
		return nil
	}
	if am == "basic" {
		if merged.Username == "" || merged.Password == "" {
			return errors.New("prometheus basic auth requires username and password")
		}
	}
	if am == "bearer" {
		if merged.BearerToken == "" {
			return errors.New("prometheus bearer auth requires a token")
		}
	}
	return h.secret.WriteServiceProviderPrometheusCredentials(ctx, providerID, merged)
}

func (h *ServiceProvidersHandler) persistServiceProviderRancherCreds(ctx context.Context, providerID uuid.UUID, authMethod string, merged secrets.RancherCredentials) error {
	if h.secret == nil {
		return errors.New("secrets store not configured")
	}
	am := strings.ToLower(strings.TrimSpace(authMethod))
	if am == "none" {
		_ = h.secret.Delete(ctx, secrets.OwnerServiceProvider, providerID, secrets.KeyRancherCreds)
		return nil
	}
	if merged.BearerToken == "" {
		return errors.New("rancher bearer auth requires an API token")
	}
	return h.secret.WriteServiceProviderRancherCredentials(ctx, providerID, merged)
}

func (h *ServiceProvidersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body serviceProviderRequestBody
	if err := handler.DecodeJSON(r, &body); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	input := body.CreateServiceProviderInput
	input.Name, input.Host, input.ImageURL = normalizeServiceProviderPayload(input.Name, input.Host, input.ImageURL)
	input.ProviderType = normalizeProviderType(input.ProviderType)
	if input.Name == "" {
		handler.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(input.Name) < 2 {
		handler.WriteError(w, http.StatusBadRequest, "name must be at least 2 characters")
		return
	}
	if err := validateServiceProviderInput(input.ProviderType, input.Host, input.Port, input.ConfigJSON); err != nil {
		handler.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if input.ProviderType == "hlctl" && !h.hlctlEnabled {
		handler.WriteError(w, http.StatusBadRequest, "HLCTL service provider requires kubectl and hlctl in the management image")
		return
	}
	if providerTypeUsesSecretStore(input.ProviderType) && h.secret == nil {
		handler.WriteError(w, http.StatusInternalServerError, "secrets store not configured for this provider type")
		return
	}
	normalizeServiceProviderConfigForSave(input.ProviderType, &input)
	taken, err := h.store.ServiceProviderNameExists(r.Context(), input.Name, nil)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to validate name")
		return
	}
	if taken {
		handler.WriteError(w, http.StatusConflict, "a service provider with this name already exists")
		return
	}
	p, err := h.store.CreateServiceProvider(r.Context(), input)
	if err != nil {
		if isServiceProviderNameUniqueViolation(err) {
			handler.WriteError(w, http.StatusConflict, "a service provider with this name already exists")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to create")
		return
	}
	if err := h.persistProviderCredentialsAfterSave(r.Context(), p, body, false); err != nil {
		_ = h.store.DeleteServiceProvider(r.Context(), p.ID)
		handler.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	audit := auditServiceProviderPayload(input)
	_ = h.store.CreateAuditLog(r.Context(), &userID, "create", "service_provider", &p.ID, audit, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusCreated, h.responseFrom(r.Context(), p))
}

func (h *ServiceProvidersHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body serviceProviderRequestBody
	if err := handler.DecodeJSON(r, &body); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	input := body.CreateServiceProviderInput
	input.Name, input.Host, input.ImageURL = normalizeServiceProviderPayload(input.Name, input.Host, input.ImageURL)
	input.ProviderType = normalizeProviderType(input.ProviderType)
	if input.Name == "" {
		handler.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(input.Name) < 2 {
		handler.WriteError(w, http.StatusBadRequest, "name must be at least 2 characters")
		return
	}
	if err := validateServiceProviderInput(input.ProviderType, input.Host, input.Port, input.ConfigJSON); err != nil {
		handler.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if input.ProviderType == "hlctl" && !h.hlctlEnabled {
		handler.WriteError(w, http.StatusBadRequest, "HLCTL service provider requires kubectl and hlctl in the management image")
		return
	}
	if providerTypeUsesSecretStore(input.ProviderType) && h.secret == nil {
		handler.WriteError(w, http.StatusInternalServerError, "secrets store not configured for this provider type")
		return
	}
	normalizeServiceProviderConfigForSave(input.ProviderType, &input)
	taken, err := h.store.ServiceProviderNameExists(r.Context(), input.Name, &id)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to validate name")
		return
	}
	if taken {
		handler.WriteError(w, http.StatusConflict, "a service provider with this name already exists")
		return
	}
	prev, _ := h.store.GetServiceProviderByID(r.Context(), id)
	if h.secret != nil && prev != nil && prev.ProviderType != input.ProviderType {
		_ = h.secret.DeleteAllForOwner(r.Context(), secrets.OwnerServiceProvider, id)
	}
	p, err := h.store.UpdateServiceProvider(r.Context(), id, store.UpdateServiceProviderInput{
		Name:         input.Name,
		Host:         input.Host,
		Port:         input.Port,
		ImageURL:     input.ImageURL,
		ProviderType: input.ProviderType,
		ConfigJSON:   input.ConfigJSON,
	})
	if err != nil {
		if isServiceProviderNameUniqueViolation(err) {
			handler.WriteError(w, http.StatusConflict, "a service provider with this name already exists")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to update")
		return
	}
	if h.secret != nil {
		if err := h.persistProviderCredentialsAfterSave(r.Context(), p, body, true); err != nil {
			handler.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	audit := auditServiceProviderPayload(input)
	_ = h.store.CreateAuditLog(r.Context(), &userID, "update", "service_provider", &p.ID, audit, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, h.responseFrom(r.Context(), p))
}

func auditServiceProviderPayload(input store.CreateServiceProviderInput) any {
	return map[string]any{
		"name":          input.Name,
		"host":          input.Host,
		"port":          input.Port,
		"image_url":     input.ImageURL,
		"provider_type": input.ProviderType,
		"config_json":   input.ConfigJSON,
	}
}

func (h *ServiceProvidersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if h.secret != nil {
		_ = h.secret.DeleteAllForOwner(r.Context(), secrets.OwnerServiceProvider, id)
	}
	if err := h.store.DeleteServiceProvider(r.Context(), id); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to delete")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "delete", "service_provider", &id, nil, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func isServiceProviderNameUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	c := pgErr.ConstraintName
	msg := pgErr.Message
	return strings.Contains(c, "service_providers_name_lower_uidx") ||
		strings.Contains(msg, "service_providers_name_lower_uidx")
}
