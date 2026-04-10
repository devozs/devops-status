package adapter

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type KubernetesAdapter struct{}

type KubernetesConfig struct {
	ClusterID        string `json:"cluster_id"`
	K8sVersion       string `json:"k8s_version,omitempty"`
	Endpoint         string `json:"endpoint"`
	Token            string `json:"token,omitempty"`
	InsecureSkipTLS  bool   `json:"insecure_skip_tls,omitempty"`
	CheckType        string `json:"check_type"`
	Namespace        string `json:"namespace,omitempty"`
	ResourceKind     string `json:"resource_kind,omitempty"`
	ResourceName     string `json:"resource_name,omitempty"`
	LabelSelector    string `json:"label_selector,omitempty"`
	FieldSelector    string `json:"field_selector,omitempty"`
	SuccessCondition string `json:"success_condition,omitempty"`
	TimeoutMs        int    `json:"timeout_ms,omitempty"`
	APIGroup         string `json:"api_group,omitempty"`
	APIVersion       string `json:"api_version,omitempty"`
}

func NewKubernetesAdapter() *KubernetesAdapter {
	return &KubernetesAdapter{}
}

func (a *KubernetesAdapter) Name() string { return "kubernetes" }

func (a *KubernetesAdapter) Probe(ctx context.Context, configRaw json.RawMessage) (*ProbeResult, error) {
	var cfg KubernetesConfig
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		return nil, fmt.Errorf("parse kubernetes config: %w", err)
	}

	start := time.Now()
	result := &ProbeResult{
		ProbedAt: start,
		Metadata: map[string]any{
			"cluster_id":    cfg.ClusterID,
			"check_type":    cfg.CheckType,
			"resource_kind": cfg.ResourceKind,
			"namespace":     cfg.Namespace,
		},
	}

	if cfg.Endpoint == "" {
		result.Success = false
		result.Error = "endpoint is required"
		result.LatencyMs = int(time.Since(start).Milliseconds())
		return result, nil
	}

	switch cfg.CheckType {
	case "api_health":
		return a.checkAPIHealth(ctx, cfg, result)
	case "deployment_ready":
		return a.checkDeploymentReady(ctx, cfg, result)
	case "node_status":
		return a.checkNodeStatus(ctx, cfg, result)
	case "pod_status":
		return a.checkPodStatus(ctx, cfg, result)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("unsupported check_type: %s", cfg.CheckType)
		result.LatencyMs = int(time.Since(start).Milliseconds())
		return result, nil
	}
}

func (a *KubernetesAdapter) k8sHTTPClient(cfg KubernetesConfig) *http.Client {
	timeout := 30 * time.Second
	if cfg.TimeoutMs > 0 {
		timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipTLS},
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}

func (a *KubernetesAdapter) k8sRequest(ctx context.Context, cfg KubernetesConfig, method, path string) ([]byte, int, error) {
	url := strings.TrimRight(cfg.Endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, 0, err
	}
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.k8sHTTPClient(cfg).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return body, resp.StatusCode, nil
}

func (a *KubernetesAdapter) checkAPIHealth(ctx context.Context, cfg KubernetesConfig, result *ProbeResult) (*ProbeResult, error) {
	_, status, err := a.k8sRequest(ctx, cfg, "GET", "/healthz")
	result.LatencyMs = int(time.Since(result.ProbedAt).Milliseconds())
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("api health request failed: %v", err)
		return result, nil
	}
	result.Success = status == 200
	result.Trace = fmt.Sprintf("GET /healthz -> %d", status)
	if !result.Success {
		result.Error = fmt.Sprintf("api returned status %d", status)
	}
	return result, nil
}

func (a *KubernetesAdapter) checkDeploymentReady(ctx context.Context, cfg KubernetesConfig, result *ProbeResult) (*ProbeResult, error) {
	ns := cfg.Namespace
	if ns == "" {
		ns = "default"
	}

	apiPrefix := "/apis/apps/v1"
	if cfg.APIGroup != "" && cfg.APIVersion != "" {
		apiPrefix = fmt.Sprintf("/apis/%s/%s", cfg.APIGroup, cfg.APIVersion)
	}

	path := fmt.Sprintf("%s/namespaces/%s/deployments/%s", apiPrefix, ns, cfg.ResourceName)
	body, status, err := a.k8sRequest(ctx, cfg, "GET", path)
	result.LatencyMs = int(time.Since(result.ProbedAt).Milliseconds())
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("deployment request failed: %v", err)
		return result, nil
	}
	if status != 200 {
		result.Success = false
		result.Error = fmt.Sprintf("deployment GET returned %d", status)
		return result, nil
	}

	var dep struct {
		Status struct {
			Replicas            int `json:"replicas"`
			ReadyReplicas       int `json:"readyReplicas"`
			UpdatedReplicas     int `json:"updatedReplicas"`
			AvailableReplicas   int `json:"availableReplicas"`
			UnavailableReplicas int `json:"unavailableReplicas"`
		} `json:"status"`
	}
	if err := json.Unmarshal(body, &dep); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("parse deployment: %v", err)
		return result, nil
	}

	result.Success = dep.Status.ReadyReplicas > 0 && dep.Status.UnavailableReplicas == 0
	result.RawValue = float64(dep.Status.ReadyReplicas)
	result.Metadata["replicas"] = dep.Status.Replicas
	result.Metadata["ready_replicas"] = dep.Status.ReadyReplicas
	result.Metadata["unavailable_replicas"] = dep.Status.UnavailableReplicas
	result.Trace = fmt.Sprintf("deployment %s/%s: %d/%d ready, %d unavailable",
		ns, cfg.ResourceName, dep.Status.ReadyReplicas, dep.Status.Replicas, dep.Status.UnavailableReplicas)
	return result, nil
}

func (a *KubernetesAdapter) checkNodeStatus(ctx context.Context, cfg KubernetesConfig, result *ProbeResult) (*ProbeResult, error) {
	path := "/api/v1/nodes"
	if cfg.LabelSelector != "" {
		path += "?labelSelector=" + cfg.LabelSelector
	}
	body, status, err := a.k8sRequest(ctx, cfg, "GET", path)
	result.LatencyMs = int(time.Since(result.ProbedAt).Milliseconds())
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("node list request failed: %v", err)
		return result, nil
	}
	if status != 200 {
		result.Success = false
		result.Error = fmt.Sprintf("node list returned %d", status)
		return result, nil
	}

	var nodeList struct {
		Items []struct {
			Metadata struct{ Name string } `json:"metadata"`
			Status   struct {
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &nodeList); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("parse node list: %v", err)
		return result, nil
	}

	totalNodes := len(nodeList.Items)
	readyNodes := 0
	for _, node := range nodeList.Items {
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				readyNodes++
			}
		}
	}

	result.Success = readyNodes == totalNodes && totalNodes > 0
	result.RawValue = float64(readyNodes)
	result.Metadata["total_nodes"] = totalNodes
	result.Metadata["ready_nodes"] = readyNodes
	result.Trace = fmt.Sprintf("nodes: %d/%d ready", readyNodes, totalNodes)
	return result, nil
}

func (a *KubernetesAdapter) checkPodStatus(ctx context.Context, cfg KubernetesConfig, result *ProbeResult) (*ProbeResult, error) {
	ns := cfg.Namespace
	if ns == "" {
		ns = "default"
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods", ns)
	params := []string{}
	if cfg.LabelSelector != "" {
		params = append(params, "labelSelector="+cfg.LabelSelector)
	}
	if cfg.FieldSelector != "" {
		params = append(params, "fieldSelector="+cfg.FieldSelector)
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}

	body, status, err := a.k8sRequest(ctx, cfg, "GET", path)
	result.LatencyMs = int(time.Since(result.ProbedAt).Milliseconds())
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("pod list request failed: %v", err)
		return result, nil
	}
	if status != 200 {
		result.Success = false
		result.Error = fmt.Sprintf("pod list returned %d", status)
		return result, nil
	}

	var podList struct {
		Items []struct {
			Metadata struct{ Name string } `json:"metadata"`
			Status   struct {
				Phase string `json:"phase"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &podList); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("parse pod list: %v", err)
		return result, nil
	}

	totalPods := len(podList.Items)
	runningPods := 0
	for _, pod := range podList.Items {
		if pod.Status.Phase == "Running" || pod.Status.Phase == "Succeeded" {
			runningPods++
		}
	}

	result.Success = runningPods == totalPods && totalPods > 0
	result.RawValue = float64(runningPods)
	result.Metadata["total_pods"] = totalPods
	result.Metadata["running_pods"] = runningPods
	result.Trace = fmt.Sprintf("pods in %s: %d/%d running", ns, runningPods, totalPods)
	return result, nil
}
