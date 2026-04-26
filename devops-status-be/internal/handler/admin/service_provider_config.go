package admin

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/devops-status/be/internal/secrets"
	"github.com/devops-status/be/internal/store"
)

func isAllowedProviderType(pt string) bool {
	switch pt {
	case "tcp", "prometheus", "grafana", "elasticsearch", "jenkins", "artifactory", "dns", "rancher", "hlctl":
		return true
	default:
		return false
	}
}

// providerTypeUsesSecretStore is true when create/update may write to secrets for this type.
func providerTypeUsesSecretStore(pt string) bool {
	switch pt {
	case "prometheus", "grafana", "elasticsearch", "jenkins", "artifactory", "rancher", "hlctl":
		return true
	default:
		return false
	}
}

func secretKeyForHTTPBasicProvider(pt string) (string, bool) {
	switch pt {
	case "grafana":
		return secrets.KeyGrafanaCreds, true
	case "elasticsearch":
		return secrets.KeyElasticsearchCreds, true
	case "jenkins":
		return secrets.KeyJenkinsCreds, true
	case "artifactory":
		return secrets.KeyArtifactoryCreds, true
	case "hlctl":
		return secrets.KeyHlctlCreds, true
	default:
		return "", false
	}
}

func parsePrometheusConfigJSON(configJSON any) (endpoint, authMethod string, insecureSkipTLS bool, err error) {
	b, err := json.Marshal(configJSON)
	if err != nil {
		return "", "", false, errors.New("invalid config_json")
	}
	var m struct {
		Endpoint        string `json:"endpoint"`
		AuthMethod      string `json:"auth_method"`
		InsecureSkipTLS bool   `json:"insecure_skip_tls"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", "", false, errors.New("invalid config_json")
	}
	am := strings.ToLower(strings.TrimSpace(m.AuthMethod))
	if am == "" {
		am = "none"
	}
	return strings.TrimSpace(m.Endpoint), am, m.InsecureSkipTLS, nil
}

func parseBaseURLAuthConfig(configJSON any, field string) (baseURL, authMethod string, err error) {
	b, err := json.Marshal(configJSON)
	if err != nil {
		return "", "", errors.New("invalid config_json")
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return "", "", errors.New("invalid config_json")
	}
	var raw string
	switch field {
	case "base_url":
		raw, _ = m["base_url"].(string)
	case "url":
		raw, _ = m["url"].(string)
	default:
		return "", "", errors.New("invalid config field")
	}
	am, _ := m["auth_method"].(string)
	am = strings.ToLower(strings.TrimSpace(am))
	if am == "" {
		am = "none"
	}
	return strings.TrimSpace(raw), am, nil
}

func parseElasticsearchConfigJSON(configJSON any) (clusterURL, authMethod string, err error) {
	return parseBaseURLAuthConfig(configJSON, "url")
}

func parseJenkinsConfigJSON(configJSON any) (clusterURL, authMethod string, err error) {
	return parseBaseURLAuthConfig(configJSON, "url")
}

func parseGrafanaConfigJSON(configJSON any) (baseURL, authMethod string, err error) {
	return parseBaseURLAuthConfig(configJSON, "base_url")
}

func parseArtifactoryConfigJSON(configJSON any) (baseURL, authMethod string, err error) {
	return parseBaseURLAuthConfig(configJSON, "base_url")
}

func parseRancherConfigJSON(configJSON any) (baseURL, authMethod string, insecureSkipTLS bool, err error) {
	b, err := json.Marshal(configJSON)
	if err != nil {
		return "", "", false, errors.New("invalid config_json")
	}
	var m struct {
		BaseURL         string `json:"base_url"`
		AuthMethod      string `json:"auth_method"`
		InsecureSkipTLS bool   `json:"insecure_skip_tls"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", "", false, errors.New("invalid config_json")
	}
	am := strings.ToLower(strings.TrimSpace(m.AuthMethod))
	if am == "" {
		am = "none"
	}
	return strings.TrimSpace(m.BaseURL), am, m.InsecureSkipTLS, nil
}

func parseDNSConfigJSON(configJSON any) (hostname, recordType, nameserver string, err error) {
	b, err := json.Marshal(configJSON)
	if err != nil {
		return "", "", "", errors.New("invalid config_json")
	}
	var m struct {
		Hostname   string `json:"hostname"`
		RecordType string `json:"record_type"`
		Nameserver string `json:"nameserver"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", "", "", errors.New("invalid config_json")
	}
	rt := strings.ToLower(strings.TrimSpace(m.RecordType))
	if rt == "" {
		rt = "a"
	}
	return strings.TrimSpace(m.Hostname), rt, strings.TrimSpace(m.Nameserver), nil
}

func normalizeDNSRecordType(rt string) string {
	rt = strings.ToLower(strings.TrimSpace(rt))
	if rt == "" {
		return "a"
	}
	return rt
}

func validateDNSRecordType(rt string) bool {
	switch rt {
	case "a", "aaaa", "cname", "txt":
		return true
	default:
		return false
	}
}

func prometheusConfigMap(endpoint, authMethod string, insecureSkipTLS bool) map[string]any {
	return map[string]any{
		"endpoint":          strings.TrimSpace(endpoint),
		"auth_method":       strings.ToLower(strings.TrimSpace(authMethod)),
		"insecure_skip_tls": insecureSkipTLS,
	}
}

func baseURLAuthConfigMap(field, value, authMethod string) map[string]any {
	return map[string]any{
		field:         strings.TrimSpace(value),
		"auth_method": strings.ToLower(strings.TrimSpace(authMethod)),
	}
}

func rancherConfigMap(baseURL, authMethod string, insecureSkipTLS bool) map[string]any {
	return map[string]any{
		"base_url":            strings.TrimSpace(baseURL),
		"auth_method":         strings.ToLower(strings.TrimSpace(authMethod)),
		"insecure_skip_tls":   insecureSkipTLS,
	}
}

func dnsConfigMap(hostname, recordType, nameserver string) map[string]any {
	m := map[string]any{
		"hostname":    strings.TrimSpace(hostname),
		"record_type": normalizeDNSRecordType(recordType),
	}
	ns := strings.TrimSpace(nameserver)
	if ns != "" {
		m["nameserver"] = ns
	}
	return m
}

func normalizeServiceProviderConfigForSave(pt string, input *store.CreateServiceProviderInput) {
	switch pt {
	case "tcp":
		input.ConfigJSON = map[string]any{}
	case "prometheus":
		endpoint, authMethod, insecureSkipTLS, _ := parsePrometheusConfigJSON(input.ConfigJSON)
		input.ConfigJSON = prometheusConfigMap(endpoint, authMethod, insecureSkipTLS)
		input.Host, input.Port = "", 0
	case "grafana":
		u, am, _ := parseGrafanaConfigJSON(input.ConfigJSON)
		input.ConfigJSON = baseURLAuthConfigMap("base_url", u, am)
		input.Host, input.Port = "", 0
	case "elasticsearch":
		u, am, _ := parseElasticsearchConfigJSON(input.ConfigJSON)
		input.ConfigJSON = baseURLAuthConfigMap("url", u, am)
		input.Host, input.Port = "", 0
	case "jenkins":
		u, am, _ := parseJenkinsConfigJSON(input.ConfigJSON)
		input.ConfigJSON = baseURLAuthConfigMap("url", u, am)
		input.Host, input.Port = "", 0
	case "artifactory":
		u, am, _ := parseArtifactoryConfigJSON(input.ConfigJSON)
		input.ConfigJSON = baseURLAuthConfigMap("base_url", u, am)
		input.Host, input.Port = "", 0
	case "dns":
		h, rt, ns, _ := parseDNSConfigJSON(input.ConfigJSON)
		input.ConfigJSON = dnsConfigMap(h, rt, ns)
		input.Host, input.Port = "", 0
	case "rancher":
		u, am, skip, _ := parseRancherConfigJSON(input.ConfigJSON)
		input.ConfigJSON = rancherConfigMap(u, am, skip)
		input.Host, input.Port = "", 0
	case "hlctl":
		input.ConfigJSON = map[string]any{}
		input.Host, input.Port = "", 0
	}
}

func validateServiceProviderInput(pt, host string, port int, configJSON any) error {
	if !isAllowedProviderType(pt) {
		return errors.New("unsupported provider_type")
	}
	switch pt {
	case "tcp":
		if host == "" {
			return errors.New("host is required for tcp")
		}
		if port < 1 || port > 65535 {
			return errors.New("port must be between 1 and 65535 for tcp")
		}
	case "prometheus":
		endpoint, authMethod, _, err := parsePrometheusConfigJSON(configJSON)
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
	case "grafana":
		baseURL, authMethod, err := parseGrafanaConfigJSON(configJSON)
		if err != nil {
			return err
		}
		if baseURL == "" {
			return errors.New("grafana base_url is required in config_json")
		}
		if u, err := url.Parse(baseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return errors.New("grafana base_url must be a valid http(s) URL")
		}
		if authMethod != "none" && authMethod != "basic" {
			return errors.New("invalid auth_method in config_json")
		}
	case "elasticsearch":
		clusterURL, authMethod, err := parseElasticsearchConfigJSON(configJSON)
		if err != nil {
			return err
		}
		if clusterURL == "" {
			return errors.New("elasticsearch url is required in config_json")
		}
		if u, err := url.Parse(clusterURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return errors.New("elasticsearch url must be a valid http(s) URL")
		}
		if authMethod != "none" && authMethod != "basic" {
			return errors.New("invalid auth_method in config_json")
		}
	case "jenkins":
		ju, authMethod, err := parseJenkinsConfigJSON(configJSON)
		if err != nil {
			return err
		}
		if ju == "" {
			return errors.New("jenkins url is required in config_json")
		}
		if u, err := url.Parse(ju); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return errors.New("jenkins url must be a valid http(s) URL")
		}
		if authMethod != "none" && authMethod != "basic" {
			return errors.New("invalid auth_method in config_json")
		}
	case "artifactory":
		baseURL, authMethod, err := parseArtifactoryConfigJSON(configJSON)
		if err != nil {
			return err
		}
		if baseURL == "" {
			return errors.New("artifactory base_url is required in config_json")
		}
		if u, err := url.Parse(baseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return errors.New("artifactory base_url must be a valid http(s) URL")
		}
		if authMethod != "none" && authMethod != "basic" {
			return errors.New("invalid auth_method in config_json")
		}
	case "dns":
		hostname, recordType, _, err := parseDNSConfigJSON(configJSON)
		if err != nil {
			return err
		}
		if hostname == "" {
			return errors.New("dns hostname is required in config_json")
		}
		if !validateDNSRecordType(recordType) {
			return errors.New("dns record_type must be a, aaaa, cname, or txt")
		}
	case "rancher":
		baseURL, authMethod, _, err := parseRancherConfigJSON(configJSON)
		if err != nil {
			return err
		}
		if baseURL == "" {
			return errors.New("rancher base_url is required in config_json")
		}
		if u, err := url.Parse(baseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return errors.New("rancher base_url must be a valid http(s) URL")
		}
		if authMethod != "none" && authMethod != "bearer" {
			return errors.New("invalid auth_method in config_json")
		}
	case "hlctl":
		// Credentials stored in secrets; config_json is empty object.
	}
	return nil
}

func mergeHTTPBasicCreds(body map[string]string, stored secrets.HTTPBasicCredentials) secrets.HTTPBasicCredentials {
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
	return out
}
