package adapter

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	cliK8sNamespace        = "devops-status-system"
	cliK8sServiceAccount   = "devops-status-agent" // same as onboarding Job; ClusterRole includes nodes list
)

// RunCLIAsK8sJob runs CLIConfig as a short-lived batch Job on the cluster (REST, no client-go).
// Pod logs are a single combined stream; stderr is left empty and cli_log_combined is set.
func RunCLIAsK8sJob(ctx context.Context, endpoint, token string, insecureSkipTLS bool, runnerImage string, cfg CLIConfig, logMax int) (*ProbeResult, error) {
	start := time.Now()
	if endpoint == "" || token == "" {
		return &ProbeResult{
			Success:   false,
			ProbedAt:  start,
			LatencyMs: int(time.Since(start).Milliseconds()),
			Error:     "cluster endpoint and token required for k8s CLI execution",
		}, nil
	}
	if sink := ProbeLogSinkFromContext(ctx); sink != nil {
		return runCLIAsK8sJobStreaming(ctx, start, endpoint, token, insecureSkipTLS, runnerImage, cfg, logMax, sink)
	}

	jobName := fmt.Sprintf("devops-cli-%s", strings.ReplaceAll(uuid.NewString(), "-", "")[:12])
	deadline := int64(120)
	if cfg.TimeoutMs > 0 {
		deadline = int64((cfg.TimeoutMs + 999) / 1000)
		if deadline < 30 {
			deadline = 30
		}
	}
	if deadline > 3600 {
		deadline = 3600
	}

	script := BuildCLIContainerShellForRun(cfg)
	cmd := []any{"/bin/sh", "-c", script}

	env := make([]map[string]string, 0)
	for k, v := range cfg.Env {
		env = append(env, map[string]string{"name": k, "value": v})
	}

	podSpec := map[string]any{
		"serviceAccountName": cliK8sServiceAccount,
		"restartPolicy":      "Never",
		"containers": []any{
			map[string]any{
				"name":    "runner",
				"image":   runnerImage,
				"command": cmd,
				"env":     env,
			},
		},
	}
	if len(cfg.NodeSelector) > 0 {
		podSpec["nodeSelector"] = cfg.NodeSelector
	}

	jobObj := map[string]any{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]any{
			"name":      jobName,
			"namespace": cliK8sNamespace,
			"labels": map[string]string{
				"app.kubernetes.io/name": "devops-status-cli",
			},
		},
		"spec": map[string]any{
			"ttlSecondsAfterFinished": 120,
			"backoffLimit":            0,
			"activeDeadlineSeconds":   deadline,
			"template": map[string]any{
				"spec": podSpec,
			},
		},
	}
	body, err := json.Marshal(jobObj)
	if err != nil {
		return nil, err
	}

	client := k8sHTTPClient(insecureSkipTLS, time.Duration(deadline)*time.Second+30*time.Second)
	path := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs", cliK8sNamespace)
	_, code, err := k8sAPI(ctx, client, endpoint, token, http.MethodPost, path, body)
	if err != nil {
		return probeFail(start, fmt.Sprintf("create job: %v", err)), nil
	}
	if code != 201 && code != 200 {
		return probeFail(start, fmt.Sprintf("create job returned %d", code)), nil
	}

	waitCtx, cancel := context.WithTimeout(ctx, time.Duration(deadline)*time.Second+45*time.Second)
	defer cancel()

	var exitCode int
	var logOut string
poll:
	for {
		select {
		case <-waitCtx.Done():
			k8sDeleteJobBestEffort(client, endpoint, token, insecureSkipTLS, jobName)
			return probeFail(start, "job wait timeout"), nil
		default:
		}
		stPath := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs/%s/status", cliK8sNamespace, jobName)
		b, sc, err := k8sAPI(waitCtx, client, endpoint, token, http.MethodGet, stPath, nil)
		if err != nil || sc != 200 {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		var st struct {
			Status struct {
				Succeeded int `json:"succeeded"`
				Failed    int `json:"failed"`
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
			} `json:"status"`
		}
		_ = json.Unmarshal(b, &st)
		if st.Status.Succeeded > 0 || st.Status.Failed > 0 {
			logOut, exitCode = fetchJobPodLog(waitCtx, client, endpoint, token, insecureSkipTLS, jobName)
			break poll
		}
		for _, c := range st.Status.Conditions {
			if c.Type == "Failed" && c.Status == "True" {
				logOut, exitCode = fetchJobPodLog(waitCtx, client, endpoint, token, insecureSkipTLS, jobName)
				break poll
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	_ = k8sDeleteJob(context.Background(), client, endpoint, token, insecureSkipTLS, jobName)

	latency := int(time.Since(start).Milliseconds())
	result := &ProbeResult{
		LatencyMs: latency,
		ProbedAt:  start,
		Metadata: map[string]any{
			"exit_code": exitCode,
			"job":       jobName,
			"namespace": cliK8sNamespace,
			"execution": "k8s_cluster",
		},
	}
	// Kubernetes log API merges streams
	ApplyCLILogFields(result, cfg, "k8s_job", runnerImage, logOut, "", logMax)
	result.Metadata["cli_log_combined"] = true

	expectedExit := cfg.SuccessExit
	if cfg.ParseJSON {
		var structured CLIStructuredOutput
		if jsonErr := json.Unmarshal([]byte(logOut), &structured); jsonErr == nil {
			result.Success = structured.Success
			result.RawValue = structured.Value
			result.Metadata["steps"] = structured.Steps
			result.Metadata["message"] = structured.Message
		} else {
			result.Success = exitCode == expectedExit
			if !result.Success {
				result.Error = fmt.Sprintf("json parse failed: %v", jsonErr)
			}
		}
	} else {
		result.Success = exitCode == expectedExit
	}
	if !result.Success && result.Error == "" && exitCode != expectedExit {
		result.Error = fmt.Sprintf("exit code %d", exitCode)
	}
	return result, nil
}

// runCLIAsK8sJobStreaming streams pod logs with follow=true while the Job runs.
func runCLIAsK8sJobStreaming(ctx context.Context, start time.Time, endpoint, token string, insecureSkipTLS bool, runnerImage string, cfg CLIConfig, logMax int, sink ProbeLogSink) (*ProbeResult, error) {
	jobName := fmt.Sprintf("devops-cli-%s", strings.ReplaceAll(uuid.NewString(), "-", "")[:12])
	deadline := int64(120)
	if cfg.TimeoutMs > 0 {
		deadline = int64((cfg.TimeoutMs + 999) / 1000)
		if deadline < 30 {
			deadline = 30
		}
	}
	if deadline > 3600 {
		deadline = 3600
	}

	script := BuildCLIContainerShellForRun(cfg)
	cmd := []any{"/bin/sh", "-c", script}

	env := make([]map[string]string, 0)
	for k, v := range cfg.Env {
		env = append(env, map[string]string{"name": k, "value": v})
	}

	podSpec := map[string]any{
		"serviceAccountName": cliK8sServiceAccount,
		"restartPolicy":      "Never",
		"containers": []any{
			map[string]any{
				"name":    "runner",
				"image":   runnerImage,
				"command": cmd,
				"env":     env,
			},
		},
	}
	if len(cfg.NodeSelector) > 0 {
		podSpec["nodeSelector"] = cfg.NodeSelector
	}

	jobObj := map[string]any{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]any{
			"name":      jobName,
			"namespace": cliK8sNamespace,
			"labels": map[string]string{
				"app.kubernetes.io/name": "devops-status-cli",
			},
		},
		"spec": map[string]any{
			"ttlSecondsAfterFinished": 120,
			"backoffLimit":            0,
			"activeDeadlineSeconds":   deadline,
			"template": map[string]any{
				"spec": podSpec,
			},
		},
	}
	body, err := json.Marshal(jobObj)
	if err != nil {
		return nil, err
	}

	client := k8sHTTPClient(insecureSkipTLS, time.Duration(deadline)*time.Second+30*time.Second)
	path := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs", cliK8sNamespace)
	_, code, err := k8sAPI(ctx, client, endpoint, token, http.MethodPost, path, body)
	if err != nil {
		return probeFail(start, fmt.Sprintf("create job: %v", err)), nil
	}
	if code != 201 && code != 200 {
		return probeFail(start, fmt.Sprintf("create job returned %d", code)), nil
	}

	waitCtx, cancel := context.WithTimeout(ctx, time.Duration(deadline)*time.Second+45*time.Second)
	defer cancel()

	streamClient := k8sHTTPClientStreaming(insecureSkipTLS)
	followCtx, followCancel := context.WithCancel(waitCtx)
	defer followCancel()

	var logBuf strings.Builder
	var streamWG sync.WaitGroup
	podSeen := ""

poll:
	for {
		select {
		case <-waitCtx.Done():
			followCancel()
			streamWG.Wait()
			k8sDeleteJobBestEffort(client, endpoint, token, insecureSkipTLS, jobName)
			return probeFail(start, "job wait timeout"), nil
		default:
		}

		if podSeen == "" {
			if pod := firstPodNameForJob(waitCtx, client, endpoint, token, jobName); pod != "" {
				podSeen = pod
				streamWG.Add(1)
				go func(p string) {
					defer streamWG.Done()
					_ = streamPodLogFollow(followCtx, streamClient, endpoint, token, cliK8sNamespace, p, "runner", sink, &logBuf)
				}(pod)
			}
		}

		stPath := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs/%s/status", cliK8sNamespace, jobName)
		b, sc, err := k8sAPI(waitCtx, client, endpoint, token, http.MethodGet, stPath, nil)
		if err != nil || sc != 200 {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		var st struct {
			Status struct {
				Succeeded  int `json:"succeeded"`
				Failed     int `json:"failed"`
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
			} `json:"status"`
		}
		_ = json.Unmarshal(b, &st)
		if st.Status.Succeeded > 0 || st.Status.Failed > 0 {
			followCancel()
			streamWG.Wait()
			break poll
		}
		for _, c := range st.Status.Conditions {
			if c.Type == "Failed" && c.Status == "True" {
				followCancel()
				streamWG.Wait()
				break poll
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	fetchedLog, exitCode := fetchJobPodLog(waitCtx, client, endpoint, token, insecureSkipTLS, jobName)
	logOut := logBuf.String()
	if logOut == "" {
		logOut = fetchedLog
	}

	_ = k8sDeleteJob(context.Background(), client, endpoint, token, insecureSkipTLS, jobName)

	latency := int(time.Since(start).Milliseconds())
	result := &ProbeResult{
		LatencyMs: latency,
		ProbedAt:  start,
		Metadata: map[string]any{
			"exit_code": exitCode,
			"job":       jobName,
			"namespace": cliK8sNamespace,
			"execution": "k8s_cluster",
		},
	}
	ApplyCLILogFields(result, cfg, "k8s_job", runnerImage, logOut, "", logMax)
	result.Metadata["cli_log_combined"] = true

	expectedExit := cfg.SuccessExit
	if cfg.ParseJSON {
		var structured CLIStructuredOutput
		if jsonErr := json.Unmarshal([]byte(logOut), &structured); jsonErr == nil {
			result.Success = structured.Success
			result.RawValue = structured.Value
			result.Metadata["steps"] = structured.Steps
			result.Metadata["message"] = structured.Message
		} else {
			result.Success = exitCode == expectedExit
			if !result.Success {
				result.Error = fmt.Sprintf("json parse failed: %v", jsonErr)
			}
		}
	} else {
		result.Success = exitCode == expectedExit
	}
	if !result.Success && result.Error == "" && exitCode != expectedExit {
		result.Error = fmt.Sprintf("exit code %d", exitCode)
	}
	return result, nil
}

func k8sHTTPClientStreaming(insecure bool) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &http.Client{Transport: tr, Timeout: 0}
}

func firstPodNameForJob(ctx context.Context, client *http.Client, endpoint, token, jobName string) string {
	listPath := fmt.Sprintf("/api/v1/namespaces/%s/pods?labelSelector=job-name=%s", cliK8sNamespace, jobName)
	b, code, err := k8sAPI(ctx, client, endpoint, token, http.MethodGet, listPath, nil)
	if err != nil || code != 200 {
		return ""
	}
	var pl struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
		} `json:"items"`
	}
	_ = json.Unmarshal(b, &pl)
	if len(pl.Items) == 0 {
		return ""
	}
	return pl.Items[0].Metadata.Name
}

func streamPodLogFollow(ctx context.Context, client *http.Client, endpoint, token, namespace, pod, container string, sink ProbeLogSink, accumulate *strings.Builder) error {
	u := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/log?container=%s&follow=true", strings.TrimRight(endpoint, "/"), namespace, pod, container)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("log stream status %d", resp.StatusCode)
	}
	buf := make([]byte, 8192)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if accumulate != nil {
				accumulate.Write(buf[:n])
			}
			if sink != nil {
				sink.WriteLog("combined", buf[:n])
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func probeFail(start time.Time, msg string) *ProbeResult {
	return &ProbeResult{
		Success:   false,
		ProbedAt:  start,
		LatencyMs: int(time.Since(start).Milliseconds()),
		Error:     msg,
	}
}

func k8sHTTPClient(insecure bool, timeout time.Duration) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &http.Client{Timeout: timeout, Transport: tr}
}

func k8sAPI(ctx context.Context, client *http.Client, endpoint, token, method, path string, body []byte) ([]byte, int, error) {
	url := strings.TrimRight(endpoint, "/") + path
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return b, resp.StatusCode, nil
}

func k8sDeleteJob(ctx context.Context, client *http.Client, endpoint, token string, insecure bool, jobName string) error {
	if client == nil {
		client = k8sHTTPClient(insecure, 30*time.Second)
	}
	path := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs/%s?propagationPolicy=Background", cliK8sNamespace, jobName)
	_, _, err := k8sAPI(ctx, client, endpoint, token, http.MethodDelete, path, nil)
	return err
}

// k8sDeleteJobBestEffort deletes the CLI probe Job using a fresh context so the DELETE still runs when the
// caller context is already cancelled (client disconnect / Stop verify).
func k8sDeleteJobBestEffort(client *http.Client, endpoint, token string, insecure bool, jobName string) {
	delCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = k8sDeleteJob(delCtx, client, endpoint, token, insecure, jobName)
}

func fetchJobPodLog(ctx context.Context, client *http.Client, endpoint, token string, insecure bool, jobName string) (string, int) {
	listPath := fmt.Sprintf("/api/v1/namespaces/%s/pods?labelSelector=job-name=%s", cliK8sNamespace, jobName)
	b, code, err := k8sAPI(ctx, client, endpoint, token, http.MethodGet, listPath, nil)
	if err != nil || code != 200 {
		return "", -1
	}
	var pl struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Status struct {
				ContainerStatuses []struct {
					Name string `json:"name"`
					State struct {
						Terminated *struct {
							ExitCode int `json:"exitCode"`
						} `json:"terminated"`
					} `json:"state"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}
	_ = json.Unmarshal(b, &pl)
	if len(pl.Items) == 0 {
		return "", -1
	}
	pod := pl.Items[0].Metadata.Name
	exitCode := 0
	for _, cs := range pl.Items[0].Status.ContainerStatuses {
		if cs.State.Terminated != nil {
			exitCode = cs.State.Terminated.ExitCode
			break
		}
	}
	logPath := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log?container=runner", cliK8sNamespace, pod)
	lb, lc, err := k8sAPI(ctx, client, endpoint, token, http.MethodGet, logPath, nil)
	if err != nil || lc != 200 {
		return "", exitCode
	}
	return string(lb), exitCode
}
