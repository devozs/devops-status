package admin

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/providerverify"
	"github.com/devops-status/be/internal/secrets"
)

func writeVerifyOK(w http.ResponseWriter, latencyMs int) {
	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"latency_ms": latencyMs,
	})
}

func writeVerifyFail(w http.ResponseWriter, msg string) {
	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":    false,
		"error": msg,
	})
}

// VerifyReachability checks connectivity for any supported service provider type.
func (h *ServiceProvidersHandler) VerifyReachability(w http.ResponseWriter, r *http.Request) {
	var body verifyServiceProviderBody
	if err := handler.DecodeJSON(r, &body); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	pt := normalizeProviderType(body.ProviderType)
	if !isAllowedProviderType(pt) {
		handler.WriteError(w, http.StatusBadRequest, "unsupported provider_type")
		return
	}

	switch pt {
	case "tcp":
		latencyMs, err := providerverify.VerifyTCP(r.Context(), strings.TrimSpace(body.Host), body.Port)
		if err != nil {
			writeVerifyFail(w, err.Error())
			return
		}
		writeVerifyOK(w, latencyMs)
	case "prometheus":
		h.verifyPrometheusReachability(w, r, body)
	case "grafana":
		h.verifyGrafanaReachability(w, r, body)
	case "elasticsearch":
		h.verifyElasticsearchReachability(w, r, body)
	case "jenkins":
		h.verifyJenkinsReachability(w, r, body)
	case "artifactory":
		h.verifyArtifactoryReachability(w, r, body)
	case "dns":
		h.verifyDNSReachability(w, r, body)
	}
}

func (h *ServiceProvidersHandler) verifyPrometheusReachability(w http.ResponseWriter, r *http.Request, body verifyServiceProviderBody) {
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
			writeVerifyFail(w, "basic auth requires username and password (enter them or verify after saving credentials)")
			return
		}
	case "bearer":
		cfg.BearerToken = creds.BearerToken
		if cfg.BearerToken == "" {
			writeVerifyFail(w, "bearer auth requires a token")
			return
		}
	}

	latencyMs, err := providerverify.VerifyPrometheus(r.Context(), cfg)
	if err != nil {
		writeVerifyFail(w, err.Error())
		return
	}
	writeVerifyOK(w, latencyMs)
}

func (h *ServiceProvidersHandler) readMergedHTTPBasic(r *http.Request, body verifyServiceProviderBody, key string) secrets.HTTPBasicCredentials {
	var stored secrets.HTTPBasicCredentials
	if body.ServiceProviderID != "" && h.secret != nil {
		id, err := uuid.Parse(strings.TrimSpace(body.ServiceProviderID))
		if err == nil {
			stored, _ = h.secret.ReadServiceProviderHTTPBasic(r.Context(), id, key)
		}
	}
	return mergeHTTPBasicCreds(body.Credentials, stored)
}

func (h *ServiceProvidersHandler) verifyGrafanaReachability(w http.ResponseWriter, r *http.Request, body verifyServiceProviderBody) {
	baseURL := strings.TrimSpace(body.BaseURL)
	if baseURL == "" {
		handler.WriteError(w, http.StatusBadRequest, "base_url is required for grafana")
		return
	}
	authMethod := strings.ToLower(strings.TrimSpace(body.AuthMethod))
	if authMethod == "" {
		authMethod = "none"
	}
	creds := h.readMergedHTTPBasic(r, body, secrets.KeyGrafanaCreds)
	if authMethod == "basic" && (creds.Username == "" || creds.Password == "") {
		writeVerifyFail(w, "basic auth requires username and password (enter them or verify after saving credentials)")
		return
	}
	latencyMs, err := providerverify.VerifyGrafana(r.Context(), baseURL, authMethod, creds.Username, creds.Password)
	if err != nil {
		writeVerifyFail(w, err.Error())
		return
	}
	writeVerifyOK(w, latencyMs)
}

func (h *ServiceProvidersHandler) verifyElasticsearchReachability(w http.ResponseWriter, r *http.Request, body verifyServiceProviderBody) {
	clusterURL := strings.TrimSpace(body.URL)
	if clusterURL == "" {
		handler.WriteError(w, http.StatusBadRequest, "url is required for elasticsearch")
		return
	}
	authMethod := strings.ToLower(strings.TrimSpace(body.AuthMethod))
	if authMethod == "" {
		authMethod = "none"
	}
	creds := h.readMergedHTTPBasic(r, body, secrets.KeyElasticsearchCreds)
	if authMethod == "basic" && (creds.Username == "" || creds.Password == "") {
		writeVerifyFail(w, "basic auth requires username and password (enter them or verify after saving credentials)")
		return
	}
	user, pass := creds.Username, creds.Password
	if authMethod == "none" {
		user, pass = "", ""
	}
	latencyMs, err := providerverify.VerifyElasticsearch(r.Context(), clusterURL, user, pass)
	if err != nil {
		writeVerifyFail(w, err.Error())
		return
	}
	writeVerifyOK(w, latencyMs)
}

func (h *ServiceProvidersHandler) verifyJenkinsReachability(w http.ResponseWriter, r *http.Request, body verifyServiceProviderBody) {
	ju := strings.TrimSpace(body.URL)
	if ju == "" {
		handler.WriteError(w, http.StatusBadRequest, "url is required for jenkins")
		return
	}
	authMethod := strings.ToLower(strings.TrimSpace(body.AuthMethod))
	if authMethod == "" {
		authMethod = "none"
	}
	creds := h.readMergedHTTPBasic(r, body, secrets.KeyJenkinsCreds)
	if authMethod == "basic" && (creds.Username == "" || creds.Password == "") {
		writeVerifyFail(w, "basic auth requires username and password or API token (enter them or verify after saving credentials)")
		return
	}
	user, pass := creds.Username, creds.Password
	if authMethod == "none" {
		user, pass = "", ""
	}
	latencyMs, err := providerverify.VerifyJenkins(r.Context(), ju, authMethod, user, pass)
	if err != nil {
		writeVerifyFail(w, err.Error())
		return
	}
	writeVerifyOK(w, latencyMs)
}

func (h *ServiceProvidersHandler) verifyArtifactoryReachability(w http.ResponseWriter, r *http.Request, body verifyServiceProviderBody) {
	baseURL := strings.TrimSpace(body.BaseURL)
	if baseURL == "" {
		handler.WriteError(w, http.StatusBadRequest, "base_url is required for artifactory")
		return
	}
	authMethod := strings.ToLower(strings.TrimSpace(body.AuthMethod))
	if authMethod == "" {
		authMethod = "none"
	}
	creds := h.readMergedHTTPBasic(r, body, secrets.KeyArtifactoryCreds)
	if authMethod == "basic" && (creds.Username == "" || creds.Password == "") {
		writeVerifyFail(w, "basic auth requires username and password (enter them or verify after saving credentials)")
		return
	}
	latencyMs, err := providerverify.VerifyArtifactory(r.Context(), baseURL, authMethod, creds.Username, creds.Password)
	if err != nil {
		writeVerifyFail(w, err.Error())
		return
	}
	writeVerifyOK(w, latencyMs)
}

func (h *ServiceProvidersHandler) verifyDNSReachability(w http.ResponseWriter, r *http.Request, body verifyServiceProviderBody) {
	host := strings.TrimSpace(body.DNSHostname)
	if host == "" {
		handler.WriteError(w, http.StatusBadRequest, "dns_hostname is required for dns")
		return
	}
	rt := normalizeDNSRecordType(body.DNSRecordType)
	if !validateDNSRecordType(rt) {
		handler.WriteError(w, http.StatusBadRequest, "dns_record_type must be a, aaaa, cname, or txt")
		return
	}
	ns := strings.TrimSpace(body.DNSNameserver)
	latencyMs, err := providerverify.VerifyDNS(r.Context(), host, rt, ns)
	if err != nil {
		writeVerifyFail(w, err.Error())
		return
	}
	writeVerifyOK(w, latencyMs)
}
