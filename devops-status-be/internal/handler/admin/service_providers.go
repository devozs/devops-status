package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/model"
	"github.com/devops-status/be/internal/secrets"
	"github.com/devops-status/be/internal/store"
)

const maxServiceProviderImageURLLen = 2000

type ServiceProvidersHandler struct {
	store  *store.Store
	secret *secrets.Store
}

func NewServiceProvidersHandler(s *store.Store, sec *secrets.Store) *ServiceProvidersHandler {
	return &ServiceProvidersHandler{store: s, secret: sec}
}

type serviceProviderResponse struct {
	ID                       uuid.UUID `json:"id"`
	Name                     string    `json:"name"`
	Host                     string    `json:"host"`
	Port                     int       `json:"port"`
	ImageURL                 string    `json:"image_url"`
	ProviderType             string    `json:"provider_type"`
	ConfigJSON               any       `json:"config_json"`
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
	if h.secret != nil && p.ProviderType == "prometheus" {
		out.HasPrometheusCredentials, _ = h.secret.HasServiceProviderPrometheusCredentials(ctx, p.ID)
	}
	return out
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
	AuthMethod        string            `json:"auth_method"`
	Credentials       map[string]string `json:"credentials"`
	ServiceProviderID string            `json:"service_provider_id,omitempty"`
}

// VerifyReachability verifies TCP host:port or Prometheus API integration.
func (h *ServiceProvidersHandler) VerifyReachability(w http.ResponseWriter, r *http.Request) {
	var body verifyServiceProviderBody
	if err := handler.DecodeJSON(r, &body); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	pt := normalizeProviderType(body.ProviderType)
	switch pt {
	case "tcp":
		h.verifyTCP(w, r, body)
	case "prometheus":
		h.verifyPrometheus(w, r, body)
	default:
		handler.WriteError(w, http.StatusBadRequest, "provider_type must be tcp or prometheus")
	}
}

func (h *ServiceProvidersHandler) verifyTCP(w http.ResponseWriter, r *http.Request, body verifyServiceProviderBody) {
	host := strings.TrimSpace(body.Host)
	if host == "" || body.Port < 1 || body.Port > 65535 {
		handler.WriteError(w, http.StatusBadRequest, "host and valid port (1–65535) are required for tcp")
		return
	}
	addr := net.JoinHostPort(host, strconv.Itoa(body.Port))
	start := time.Now()
	d := net.Dialer{Timeout: 5 * time.Second}
	c, err := d.DialContext(r.Context(), "tcp", addr)
	if err != nil {
		handler.WriteJSON(w, http.StatusOK, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}
	_ = c.Close()
	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"latency_ms": time.Since(start).Milliseconds(),
	})
}

func (h *ServiceProvidersHandler) verifyPrometheus(w http.ResponseWriter, r *http.Request, body verifyServiceProviderBody) {
	endpoint := strings.TrimSpace(body.Endpoint)
	if endpoint == "" {
		handler.WriteError(w, http.StatusBadRequest, "endpoint is required for prometheus")
		return
	}
	if u, err := url.Parse(endpoint); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		handler.WriteError(w, http.StatusBadRequest, "endpoint must be a valid http(s) URL")
		return
	}
	authMethod := strings.ToLower(strings.TrimSpace(body.AuthMethod))
	if authMethod == "" {
		authMethod = "none"
	}
	if authMethod != "none" && authMethod != "basic" && authMethod != "bearer" {
		handler.WriteError(w, http.StatusBadRequest, "auth_method must be none, basic, or bearer")
		return
	}

	var stored secrets.PrometheusCredentials
	if body.ServiceProviderID != "" && h.secret != nil {
		id, err := uuid.Parse(strings.TrimSpace(body.ServiceProviderID))
		if err == nil {
			stored, _ = h.secret.ReadServiceProviderPrometheusCredentials(r.Context(), id)
		}
	}
	creds := mergePrometheusCreds(body.Credentials, stored)

	cfg := adapter.PrometheusConfig{
		Endpoint:   endpoint,
		AuthMethod: authMethod,
		TimeoutMs:  15000,
	}
	switch authMethod {
	case "basic":
		cfg.Username = creds.Username
		cfg.Password = creds.Password
		if cfg.Username == "" || cfg.Password == "" {
			handler.WriteJSON(w, http.StatusOK, map[string]any{
				"ok":    false,
				"error": "basic auth requires username and password (enter them or verify after saving credentials)",
			})
			return
		}
	case "bearer":
		cfg.BearerToken = creds.BearerToken
		if cfg.BearerToken == "" {
			handler.WriteJSON(w, http.StatusOK, map[string]any{
				"ok":    false,
				"error": "bearer auth requires a token",
			})
			return
		}
	}

	latencyMs, err := adapter.VerifyPrometheusIntegration(r.Context(), cfg)
	if err != nil {
		handler.WriteJSON(w, http.StatusOK, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"latency_ms": latencyMs,
	})
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

func prometheusConfigMap(endpoint, authMethod string) map[string]any {
	return map[string]any{
		"endpoint":    strings.TrimSpace(endpoint),
		"auth_method": strings.ToLower(strings.TrimSpace(authMethod)),
	}
}

func validateServiceProviderInput(pt, host string, port int, configJSON any) error {
	switch pt {
	case "tcp":
		if host == "" {
			return errors.New("host is required for tcp")
		}
		if port < 1 || port > 65535 {
			return errors.New("port must be between 1 and 65535 for tcp")
		}
	case "prometheus":
		endpoint, authMethod, err := parsePrometheusConfigJSON(configJSON)
		if err != nil {
			return err
		}
		if endpoint == "" {
			return errors.New("prometheus endpoint is required in config_json")
		}
		if u, err := url.Parse(endpoint); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return errors.New("prometheus endpoint must be a valid http(s) URL")
		}
		if authMethod != "none" && authMethod != "basic" && authMethod != "bearer" {
			return errors.New("invalid auth_method in config_json")
		}
	default:
		return errors.New("provider_type must be tcp or prometheus")
	}
	return nil
}

func parsePrometheusConfigJSON(configJSON any) (endpoint, authMethod string, err error) {
	b, err := json.Marshal(configJSON)
	if err != nil {
		return "", "", errors.New("invalid config_json")
	}
	var m struct {
		Endpoint   string `json:"endpoint"`
		AuthMethod string `json:"auth_method"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", "", errors.New("invalid config_json")
	}
	am := strings.ToLower(strings.TrimSpace(m.AuthMethod))
	if am == "" {
		am = "none"
	}
	return strings.TrimSpace(m.Endpoint), am, nil
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
	if input.ProviderType == "tcp" {
		input.ConfigJSON = map[string]any{}
	}
	if input.ProviderType == "prometheus" {
		if h.secret == nil {
			handler.WriteError(w, http.StatusInternalServerError, "secrets store not configured for prometheus providers")
			return
		}
		input.Host = ""
		input.Port = 0
		endpoint, authMethod, _ := parsePrometheusConfigJSON(input.ConfigJSON)
		input.ConfigJSON = prometheusConfigMap(endpoint, authMethod)
	}
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
	if p.ProviderType == "prometheus" && h.secret != nil {
		_, authMethod, _ := parsePrometheusConfigJSON(p.ConfigJSON)
		if authMethod == "none" {
			_ = h.secret.Delete(r.Context(), secrets.OwnerServiceProvider, p.ID, secrets.KeyPrometheusCreds)
		} else {
			merged := mergePrometheusCreds(body.Credentials, secrets.PrometheusCredentials{})
			if err := h.persistServiceProviderPrometheusCreds(r.Context(), p.ID, authMethod, merged); err != nil {
				_ = h.store.DeleteServiceProvider(r.Context(), p.ID)
				handler.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
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
	if input.ProviderType == "tcp" {
		input.ConfigJSON = map[string]any{}
	}
	if input.ProviderType == "prometheus" {
		if h.secret == nil {
			handler.WriteError(w, http.StatusInternalServerError, "secrets store not configured for prometheus providers")
			return
		}
		input.Host = ""
		input.Port = 0
		endpoint, authMethod, _ := parsePrometheusConfigJSON(input.ConfigJSON)
		input.ConfigJSON = prometheusConfigMap(endpoint, authMethod)
	}
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
		if prev != nil && prev.ProviderType == "prometheus" && p.ProviderType != "prometheus" {
			_ = h.secret.DeleteAllForOwner(r.Context(), secrets.OwnerServiceProvider, id)
		}
		if p.ProviderType == "prometheus" {
			_, authMethod, _ := parsePrometheusConfigJSON(p.ConfigJSON)
			if authMethod == "none" {
				_ = h.secret.Delete(r.Context(), secrets.OwnerServiceProvider, p.ID, secrets.KeyPrometheusCreds)
			} else {
				existing, _ := h.secret.ReadServiceProviderPrometheusCredentials(r.Context(), p.ID)
				merged := mergePrometheusCreds(body.Credentials, existing)
				if err := h.persistServiceProviderPrometheusCreds(r.Context(), p.ID, authMethod, merged); err != nil {
					handler.WriteError(w, http.StatusBadRequest, err.Error())
					return
				}
			}
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
