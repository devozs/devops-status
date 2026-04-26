package admin

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	adapt "github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/providerverify"
	"github.com/google/uuid"
)

const (
	maxTelemetryEnvKeys           = 32
	maxTelemetryEnvKeyLen         = 256
	maxTelemetryEnvValLen         = 262144 // 256 KiB (e.g. base64 kubeconfig)
	maxTelemetryNodeSelectorKeys  = 16
	maxTelemetryNodeSelectorKeyLen = 253
	maxTelemetryNodeSelectorValLen = 63
)

var (
	envVarNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	nsPattern         = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	k8sVersionAllow   = map[string]bool{
		"1.26": true, "1.27": true, "1.28": true, "1.29": true, "1.30": true, "1.31": true, "1.32": true,
	}
	cliAllowlist      = map[string]bool{"curl": true, "kubectl": true, "go": true, "echo": true}
	hlctlCommandAllowlist = map[string]bool{"curl": true, "kubectl": true, "go": true, "echo": true, "hlctl": true}
)

func allowedCLIRunnerPreset(rn string) bool {
	switch rn {
	case adapt.CLIRunnerProfileAlpine, adapt.CLIRunnerProfileUbuntu24, adapt.CLIRunnerProfileToolkit:
		return true
	default:
		return false
	}
}

// appendTelemetryProbeEnvErrors validates optional Job/container env (kubectl kubeconfig material, etc.).
func appendTelemetryProbeEnvErrors(add func(string), prefix string, env map[string]string) {
	if len(env) == 0 {
		return
	}
	if len(env) > maxTelemetryEnvKeys {
		add(prefix + fmt.Sprintf("env must have at most %d keys", maxTelemetryEnvKeys))
		return
	}
	for k, v := range env {
		kt := strings.TrimSpace(k)
		if kt != k || k == "" || len(k) > maxTelemetryEnvKeyLen {
			add(prefix + "env keys must be non-empty, without leading/trailing space, and within length limits")
			return
		}
		if !envVarNamePattern.MatchString(k) {
			add(prefix + "env keys must match [A-Za-z_][A-Za-z0-9_]*")
			return
		}
		if strings.ContainsAny(v, "\x00") || len(v) > maxTelemetryEnvValLen {
			add(prefix + fmt.Sprintf("env value for %q is invalid or too large", k))
			return
		}
	}
}

func nodeSelectorKeyCharOK(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '-', r == '_', r == '.', r == '/':
		return true
	default:
		return false
	}
}

func validTelemetryNodeSelectorKey(k string) bool {
	if len(k) < 1 || len(k) > maxTelemetryNodeSelectorKeyLen {
		return false
	}
	for _, r := range k {
		if r > 127 || !nodeSelectorKeyCharOK(r) {
			return false
		}
	}
	return true
}

func validTelemetryNodeSelectorValue(v string) bool {
	if len(v) > maxTelemetryNodeSelectorValLen {
		return false
	}
	return !strings.ContainsAny(v, "\x00\n\r")
}

// appendTelemetryNodeSelectorErrors validates optional Pod nodeSelector for probe Jobs.
func appendTelemetryNodeSelectorErrors(add func(string), prefix string, sel map[string]string) {
	if len(sel) == 0 {
		return
	}
	if len(sel) > maxTelemetryNodeSelectorKeys {
		add(prefix + fmt.Sprintf("node_selector must have at most %d keys", maxTelemetryNodeSelectorKeys))
		return
	}
	for k, v := range sel {
		kt := strings.TrimSpace(k)
		if kt != k || k == "" {
			add(prefix + "node_selector keys must be non-empty without leading/trailing space")
			return
		}
		if !validTelemetryNodeSelectorKey(k) {
			add(prefix + "node_selector key is invalid (use letters, digits, -, _, ., /; max 253 chars)")
			return
		}
		if !validTelemetryNodeSelectorValue(v) {
			add(prefix + fmt.Sprintf("node_selector value for %q is invalid (max %d chars, no newlines)", k, maxTelemetryNodeSelectorValLen))
			return
		}
	}
}

func isCLICompareNumericOp(op string) bool {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "gt", "gte", "lt", "lte", "eq":
		return true
	default:
		return false
	}
}

func isCLICompareTextOp(op string) bool {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "eq", "neq":
		return true
	default:
		return false
	}
}

func qosThresholdsPresent(qosThresholds []byte) bool {
	return len(qosThresholds) > 0 && string(qosThresholds) != "{}" && string(qosThresholds) != "null"
}

// appendLatencyQoSThresholdErrors validates flat green_max_ms / yellow_max_ms / red_max_ms (HTTP and legacy Kubernetes).
func appendLatencyQoSThresholdErrors(add func(string), qosThresholds []byte) {
	var t struct {
		GreenMaxMs  int `json:"green_max_ms"`
		YellowMaxMs int `json:"yellow_max_ms"`
		RedMaxMs    int `json:"red_max_ms"`
	}
	if err := json.Unmarshal(qosThresholds, &t); err != nil {
		add("invalid qos_thresholds JSON")
		return
	}
	if t.GreenMaxMs <= 0 {
		add("qos green_max_ms must be > 0")
	}
	if t.YellowMaxMs <= t.GreenMaxMs {
		add("qos yellow_max_ms must be greater than green_max_ms")
	}
	if t.RedMaxMs <= 0 {
		t.RedMaxMs = t.YellowMaxMs + 1
	}
	if t.RedMaxMs <= t.YellowMaxMs {
		add("qos red_max_ms must be greater than yellow_max_ms")
	}
	if t.GreenMaxMs == t.YellowMaxMs || t.YellowMaxMs == t.RedMaxMs || t.GreenMaxMs == t.RedMaxMs {
		add("qos green, yellow, and red max (ms) must all be different")
	}
}

// appendMetricQoSValueKindErrors validates CLI-style metric qos (value_kind number|text). Returns true if qos used this shape (including errors).
func appendMetricQoSValueKindErrors(add func(string), qosThresholds []byte, subject string) bool {
	var qMeta struct {
		ValueKind string `json:"value_kind"`
	}
	if err := json.Unmarshal(qosThresholds, &qMeta); err != nil {
		add("invalid qos_thresholds JSON")
		return true
	}
	vk := strings.ToLower(strings.TrimSpace(qMeta.ValueKind))
	if vk == "" {
		return false
	}
	if vk != "number" && vk != "text" {
		add(subject + " metric qos value_kind must be number or text")
		return true
	}
	if vk == "number" {
		var t struct {
			NumericValuePick string  `json:"numeric_value_pick"`
			GreenOperator    string  `json:"green_operator"`
			GreenThreshold   float64 `json:"green_threshold"`
			YellowOperator   string  `json:"yellow_operator"`
			YellowThreshold  float64 `json:"yellow_threshold"`
		}
		if err := json.Unmarshal(qosThresholds, &t); err != nil {
			add("invalid qos_thresholds JSON")
		} else {
			if t.GreenOperator == "" || t.YellowOperator == "" {
				add(subject + " metric number qos requires green_operator and yellow_operator")
			} else if !isCLICompareNumericOp(t.GreenOperator) || !isCLICompareNumericOp(t.YellowOperator) {
				add(subject + " metric number qos: operators must be gt, gte, lt, lte, or eq")
			}
			if np := strings.ToLower(strings.TrimSpace(t.NumericValuePick)); np != "" && np != "first" && np != "last" {
				add(subject + " metric number qos numeric_value_pick must be first or last")
			}
		}
	}
	if vk == "text" {
		var t struct {
			GreenOperator  string `json:"green_operator"`
			YellowOperator string `json:"yellow_operator"`
		}
		if err := json.Unmarshal(qosThresholds, &t); err != nil {
			add("invalid qos_thresholds JSON")
		} else {
			if t.GreenOperator == "" || t.YellowOperator == "" {
				add(subject + " metric text qos requires green_operator and yellow_operator")
			} else if !isCLICompareTextOp(t.GreenOperator) || !isCLICompareTextOp(t.YellowOperator) {
				add(subject + " metric text qos: operators must be eq or neq")
			}
		}
	}
	return true
}

// ValidateTelemetryConfig returns human-readable errors; empty slice means valid.
// Unified telemetry rows always require adapter-specific qos_thresholds.
func ValidateTelemetryConfig(adapter string, configJSON []byte, qosThresholds []byte, executionTarget string) []string {
	var errs []string
	add := func(msg string) { errs = append(errs, msg) }
	var livenessArtifactoryNeedsValueQoS bool

	switch adapter {
	case "http":
		var cfg struct {
			ServiceProviderID string `json:"service_provider_id"`
			Path              string `json:"path"`
		}
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return []string{"invalid HTTP config JSON"}
		}
		sp := strings.TrimSpace(cfg.ServiceProviderID)
		if sp == "" {
			add("HTTP service_provider_id is required")
		} else if _, err := uuid.Parse(sp); err != nil {
			add("HTTP service_provider_id must be a valid UUID")
		}
		if strings.TrimSpace(cfg.Path) == "" {
			add("HTTP path is required")
		}
	case "prometheus":
		var cfg struct {
			ServiceProviderID string `json:"service_provider_id"`
			Query             string `json:"query"`
		}
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return []string{"invalid Prometheus config JSON"}
		}
		sp := strings.TrimSpace(cfg.ServiceProviderID)
		if sp == "" {
			add("Prometheus service_provider_id is required")
		} else if _, err := uuid.Parse(sp); err != nil {
			add("Prometheus service_provider_id must be a valid UUID")
		}
		if strings.TrimSpace(cfg.Query) == "" {
			add("Prometheus query is required")
		}
	case "kubernetes":
		var cfg adapt.KubernetesConfig
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return []string{"invalid Kubernetes config JSON"}
		}
		if cfg.ClusterID == "" {
			add("Kubernetes cluster_id is required")
		}
		if cfg.K8sVersion == "" || !k8sVersionAllow[cfg.K8sVersion] {
			add("Kubernetes k8s_version must be one of 1.26–1.32")
		}
		shellTrim := strings.TrimSpace(cfg.CLIShell)
		if shellTrim != "" {
			if msg := adapt.ValidateCLIPrepLength(cfg.ContainerPrep); msg != "" {
				add(msg)
			}
			if msg := adapt.ValidateCLIShellLength(cfg.CLIShell); msg != "" {
				add(msg)
			}
			var qosPeek struct {
				ValueKind string `json:"value_kind"`
			}
			_ = json.Unmarshal(qosThresholds, &qosPeek)
			if strings.TrimSpace(qosPeek.ValueKind) == "" {
				add("kubernetes shell requires qos_thresholds.value_kind (number or text)")
			}
			rn := strings.TrimSpace(cfg.Runner)
			if rn != "" && !allowedCLIRunnerPreset(rn) {
				add("Kubernetes runner must be alpine, ubuntu_24, or toolkit (or omit for default)")
			}
			if ri := strings.TrimSpace(cfg.RunnerImage); ri != "" {
				if len(ri) > 512 || strings.ContainsAny(ri, "\n\r") {
					add("runner_image is invalid")
				}
			}
			appendTelemetryProbeEnvErrors(add, "", cfg.Env)
			appendTelemetryNodeSelectorErrors(add, "", cfg.NodeSelector)
		} else {
			if cfg.CheckType == "" {
				add("Kubernetes check_type is required when cli_shell is empty")
			}
			switch cfg.CheckType {
			case "api_health", "node_status", "pod_status", "deployment_ready":
			default:
				add("unsupported kubernetes check_type")
			}
			if cfg.Namespace != "" && !nsPattern.MatchString(cfg.Namespace) {
				add("invalid kubernetes namespace")
			}
			if cfg.CheckType == "deployment_ready" && cfg.ResourceName == "" {
				add("resource_name is required for deployment_ready")
			}
		}
	case "cli":
		var cfg adapt.CLIConfig
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return []string{"invalid CLI config JSON"}
		}
		if msg := adapt.ValidateCLIPrepLength(cfg.ContainerPrep); msg != "" {
			add(msg)
		}
		shellTrim := strings.TrimSpace(cfg.CLIShell)
		if msg := adapt.ValidateCLIShellLength(cfg.CLIShell); msg != "" {
			add(msg)
		}
		var qosPeek struct {
			ValueKind string `json:"value_kind"`
		}
		_ = json.Unmarshal(qosThresholds, &qosPeek)
		vkPeek := strings.TrimSpace(qosPeek.ValueKind)
		if shellTrim != "" && vkPeek == "" {
			add("cli_shell requires qos_thresholds.value_kind (number or text)")
		}
		if vkPeek != "" && shellTrim == "" {
			add("qos_thresholds.value_kind requires non-empty config_json.cli_shell")
		}
		rn := strings.TrimSpace(cfg.Runner)
		if rn != "" && !allowedCLIRunnerPreset(rn) {
			add("CLI runner must be alpine, ubuntu_24, or toolkit (or omit for default)")
		}
		if ri := strings.TrimSpace(cfg.RunnerImage); ri != "" {
			if len(ri) > 512 || strings.ContainsAny(ri, "\n\r") {
				add("runner_image is invalid")
			}
		}
		appendTelemetryProbeEnvErrors(add, "", cfg.Env)
		appendTelemetryNodeSelectorErrors(add, "", cfg.NodeSelector)
		cliMetric := adapt.HasCLIMetricQoS(configJSON, qosThresholds)
		cliTriple := adapt.HasCLIPQoSThresholds(qosThresholds)
		if cliMetric {
			if shellTrim == "" {
				add("cli_shell is required for CLI metric QoS")
			}
		} else if cliTriple {
			var qfull struct {
				GreenCommand, YellowCommand, RedCommand string
			}
			_ = json.Unmarshal(qosThresholds, &qfull)
			for _, c := range []string{qfull.GreenCommand, qfull.YellowCommand, qfull.RedCommand} {
				if c == "" {
					continue
				}
				base := c
				if i := strings.Index(base, "/"); i >= 0 {
					base = base[i+1:]
				}
				if !cliAllowlist[base] && !cliAllowlist[c] {
					add("each CLI QoS command must be curl, kubectl, go, or echo")
					break
				}
			}
		} else {
			if cfg.Command == "" {
				add("CLI command is required (unless using metric QoS with cli_shell and value_kind, or legacy triple-command qos)")
			} else {
				base := cfg.Command
				if i := strings.Index(base, "/"); i >= 0 {
					base = base[i+1:]
				}
				if !cliAllowlist[base] && !cliAllowlist[cfg.Command] {
					add("CLI command must be one of: curl, kubectl, go, echo")
				}
			}
		}
		et := cfg.ExecutionTarget
		if et == "" {
			et = executionTarget
		}
		if et == "k8s_cluster" {
			if cfg.ClusterID == "" {
				add("cluster_id is required when CLI runs on k8s_cluster")
			}
			if cfg.K8sVersion == "" || !k8sVersionAllow[cfg.K8sVersion] {
				add("CLI k8s_version must be one of 1.26–1.32 when running on k8s_cluster")
			}
		}
	case "hlctl":
		var cfg struct {
			adapt.CLIConfig
			ServiceProviderID string `json:"service_provider_id"`
		}
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return []string{"invalid HLCTL config JSON"}
		}
		if sp := strings.TrimSpace(cfg.ServiceProviderID); sp == "" {
			add("HLCTL service_provider_id is required")
		} else if _, err := uuid.Parse(sp); err != nil {
			add("HLCTL service_provider_id must be a valid UUID")
		}
		if strings.TrimSpace(cfg.Runner) != "" {
			add("HLCTL does not use runner (runs locally in management)")
		}
		if strings.TrimSpace(cfg.RunnerImage) != "" {
			add("HLCTL does not use runner_image")
		}
		if len(cfg.NodeSelector) > 0 {
			add("HLCTL does not use node_selector")
		}
		et := cfg.ExecutionTarget
		if et == "" {
			et = executionTarget
		}
		if et == "k8s_cluster" || strings.TrimSpace(cfg.ClusterID) != "" {
			add("HLCTL runs only on the management backend; omit k8s_cluster execution and cluster_id")
		}
		if msg := adapt.ValidateCLIPrepLength(cfg.ContainerPrep); msg != "" {
			add(msg)
		}
		shellTrim := strings.TrimSpace(cfg.CLIShell)
		if msg := adapt.ValidateCLIShellLength(cfg.CLIShell); msg != "" {
			add(msg)
		}
		var qosPeek struct {
			ValueKind string `json:"value_kind"`
		}
		_ = json.Unmarshal(qosThresholds, &qosPeek)
		vkPeek := strings.TrimSpace(qosPeek.ValueKind)
		if shellTrim != "" && vkPeek == "" {
			add("cli_shell requires qos_thresholds.value_kind (number or text)")
		}
		if vkPeek != "" && shellTrim == "" {
			add("qos_thresholds.value_kind requires non-empty config_json.cli_shell")
		}
		appendTelemetryProbeEnvErrors(add, "", cfg.Env)
		hlctlMetric := adapt.HasCLIMetricQoS(configJSON, qosThresholds)
		hlctlTriple := adapt.HasCLIPQoSThresholds(qosThresholds)
		if hlctlMetric {
			if shellTrim == "" {
				add("cli_shell is required for HLCTL metric QoS")
			}
		} else if hlctlTriple {
			var qfull struct {
				GreenCommand, YellowCommand, RedCommand string
			}
			_ = json.Unmarshal(qosThresholds, &qfull)
			for _, c := range []string{qfull.GreenCommand, qfull.YellowCommand, qfull.RedCommand} {
				if c == "" {
					continue
				}
				base := c
				if i := strings.Index(base, "/"); i >= 0 {
					base = base[i+1:]
				}
				if !hlctlCommandAllowlist[base] && !hlctlCommandAllowlist[c] {
					add("each HLCTL QoS command must be curl, kubectl, go, echo, or hlctl")
					break
				}
			}
		} else {
			if cfg.Command == "" {
				add("HLCTL command is required (unless using metric QoS with cli_shell and value_kind, or legacy triple-command qos)")
			} else {
				base := cfg.Command
				if i := strings.Index(base, "/"); i >= 0 {
					base = base[i+1:]
				}
				if !hlctlCommandAllowlist[base] && !hlctlCommandAllowlist[cfg.Command] {
					add("HLCTL command must be one of: curl, kubectl, go, echo, hlctl")
				}
			}
		}
	case "liveness":
		var cfg struct {
			Source            string          `json:"source"`
			Kubernetes        json.RawMessage `json:"kubernetes"`
			ServiceProviderID string          `json:"service_provider_id"`
			Artifactory       *struct {
				DownloadRepositoryPath string `json:"download_repository_path"`
				DownloadMaxBytes       *int   `json:"download_max_bytes,omitempty"`
			} `json:"artifactory,omitempty"`
		}
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			add("invalid liveness config JSON")
			break
		}
		src := strings.ToLower(strings.TrimSpace(cfg.Source))
		if cfg.Artifactory != nil && src == "kubernetes" {
			add("liveness artifactory settings are only valid when source is service_provider")
		}
		switch src {
		case "kubernetes":
			var k adapt.KubernetesConfig
			if err := json.Unmarshal(cfg.Kubernetes, &k); err != nil {
				add("invalid liveness kubernetes JSON")
				break
			}
			if k.ClusterID == "" {
				add("liveness kubernetes.cluster_id is required")
			}
			if k.K8sVersion == "" || !k8sVersionAllow[k.K8sVersion] {
				add("liveness kubernetes.k8s_version must be one of 1.26–1.32")
			}
			if strings.TrimSpace(k.CLIShell) != "" {
				if msg := adapt.ValidateCLIPrepLength(k.ContainerPrep); msg != "" {
					add("liveness kubernetes: " + msg)
				}
				if msg := adapt.ValidateCLIShellLength(k.CLIShell); msg != "" {
					add("liveness kubernetes: " + msg)
				}
				if _, ok := adapt.BuildSyntheticMetricQoSFromLivenessValue(qosThresholds); !ok {
					add("liveness kubernetes shell requires qos_thresholds.value with green/yellow operators for parsed numeric output")
				}
				rn := strings.TrimSpace(k.Runner)
				if rn != "" && !allowedCLIRunnerPreset(rn) {
					add("liveness kubernetes runner must be alpine, ubuntu_24, or toolkit (or omit for default)")
				}
				appendTelemetryProbeEnvErrors(add, "liveness kubernetes: ", k.Env)
				appendTelemetryNodeSelectorErrors(add, "liveness kubernetes: ", k.NodeSelector)
			} else {
				if k.CheckType == "" {
					add("liveness kubernetes.check_type is required when cli_shell is empty")
				}
				switch k.CheckType {
				case "api_health", "node_status", "pod_status", "deployment_ready":
				default:
					add("unsupported liveness kubernetes.check_type")
				}
				if k.Namespace != "" && !nsPattern.MatchString(k.Namespace) {
					add("invalid liveness kubernetes.namespace")
				}
				if k.CheckType == "deployment_ready" && k.ResourceName == "" {
					add("liveness kubernetes.resource_name is required for deployment_ready")
				}
			}
		case "service_provider":
			sp := strings.TrimSpace(cfg.ServiceProviderID)
			if sp == "" {
				add("liveness service_provider_id is required when source is service_provider")
			} else if _, err := uuid.Parse(sp); err != nil {
				add("liveness service_provider_id must be a valid UUID")
			}
			if cfg.Artifactory != nil {
				afPath := strings.TrimSpace(cfg.Artifactory.DownloadRepositoryPath)
				if cfg.Artifactory.DownloadMaxBytes != nil && afPath == "" {
					add("liveness artifactory.download_max_bytes requires download_repository_path")
				}
				if afPath != "" {
					if err := providerverify.ValidateArtifactoryDownloadRepositoryPath(afPath); err != nil {
						add("liveness artifactory: " + err.Error())
					}
					if cfg.Artifactory.DownloadMaxBytes != nil {
						n := int64(*cfg.Artifactory.DownloadMaxBytes)
						if n < 1 || n > providerverify.MaxArtifactoryDownloadMaxBytes {
							add(fmt.Sprintf("liveness artifactory.download_max_bytes must be between 1 and %d", providerverify.MaxArtifactoryDownloadMaxBytes))
						}
					}
					livenessArtifactoryNeedsValueQoS = true
				}
			}
		default:
			add("liveness source must be kubernetes or service_provider")
		}
	default:
		add("unknown adapter")
	}

	if !qosThresholdsPresent(qosThresholds) {
		add("qos_thresholds is required")
		return errs
	}

	switch adapter {
	case "http":
		appendLatencyQoSThresholdErrors(add, qosThresholds)
	case "kubernetes":
		if adapt.HasCLIMetricQoS(configJSON, qosThresholds) {
			if !appendMetricQoSValueKindErrors(add, qosThresholds, "Kubernetes") {
				add("kubernetes metric qos requires value_kind in qos_thresholds")
			}
		} else {
			appendLatencyQoSThresholdErrors(add, qosThresholds)
		}
	case "prometheus":
		var t struct {
			GreenOperator   string  `json:"green_operator"`
			GreenThreshold  float64 `json:"green_threshold"`
			YellowOperator  string  `json:"yellow_operator"`
			YellowThreshold float64 `json:"yellow_threshold"`
		}
		if err := json.Unmarshal(qosThresholds, &t); err != nil {
			add("invalid qos_thresholds JSON")
		} else {
			if t.GreenOperator == "" || t.YellowOperator == "" {
				add("prometheus qos requires green_operator and yellow_operator")
			}
		}
	case "liveness":
		var lat struct {
			Latency struct {
				GreenMaxMs  int `json:"green_max_ms"`
				YellowMaxMs int `json:"yellow_max_ms"`
				RedMaxMs    int `json:"red_max_ms"`
			} `json:"latency"`
			Value *struct {
				GreenOperator   string  `json:"green_operator"`
				GreenThreshold  float64 `json:"green_threshold"`
				YellowOperator  string  `json:"yellow_operator"`
				YellowThreshold float64 `json:"yellow_threshold"`
			} `json:"value"`
		}
		if err := json.Unmarshal(qosThresholds, &lat); err != nil {
			add("invalid liveness qos_thresholds JSON")
		} else {
			t := lat.Latency
			if t.GreenMaxMs <= 0 {
				add("liveness qos latency.green_max_ms must be > 0")
			}
			if t.YellowMaxMs <= t.GreenMaxMs {
				add("liveness qos latency.yellow_max_ms must be greater than green_max_ms")
			}
			if t.RedMaxMs <= 0 {
				t.RedMaxMs = t.YellowMaxMs + 1
			}
			if t.RedMaxMs <= t.YellowMaxMs {
				add("liveness qos latency.red_max_ms must be greater than yellow_max_ms")
			}
			if lat.Value != nil {
				if lat.Value.GreenOperator == "" || lat.Value.YellowOperator == "" {
					add("liveness qos value requires green_operator and yellow_operator when set")
				} else if !isCLICompareNumericOp(lat.Value.GreenOperator) || !isCLICompareNumericOp(lat.Value.YellowOperator) {
					add("liveness qos value operators must be gt, gte, lt, lte, or eq")
				}
			}
			if livenessArtifactoryNeedsValueQoS && lat.Value == nil {
				add("liveness artifactory download bandwidth requires qos_thresholds.value (thresholds are bytes per second)")
			}
		}
	case "cli":
		if appendMetricQoSValueKindErrors(add, qosThresholds, "CLI") {
			break
		}
		var t struct {
			GreenCommand  string   `json:"green_command"`
			GreenArgs     []string `json:"green_args"`
			YellowCommand string   `json:"yellow_command"`
			YellowArgs    []string `json:"yellow_args"`
			RedCommand    string   `json:"red_command"`
			RedArgs       []string `json:"red_args"`
		}
		if err := json.Unmarshal(qosThresholds, &t); err != nil {
			add("invalid qos_thresholds JSON")
		} else if t.GreenCommand == "" || t.YellowCommand == "" || t.RedCommand == "" {
			add("CLI qos requires green_command, yellow_command, and red_command (or metric qos with value_kind)")
		}
	case "hlctl":
		if appendMetricQoSValueKindErrors(add, qosThresholds, "HLCTL") {
			break
		}
		var t struct {
			GreenCommand  string   `json:"green_command"`
			GreenArgs     []string `json:"green_args"`
			YellowCommand string   `json:"yellow_command"`
			YellowArgs    []string `json:"yellow_args"`
			RedCommand    string   `json:"red_command"`
			RedArgs       []string `json:"red_args"`
		}
		if err := json.Unmarshal(qosThresholds, &t); err != nil {
			add("invalid qos_thresholds JSON")
		} else if t.GreenCommand == "" || t.YellowCommand == "" || t.RedCommand == "" {
			add("HLCTL qos requires green_command, yellow_command, and red_command (or metric qos with value_kind)")
		}
	}

	return errs
}

// DefaultTimeoutMs returns probe timeout when client sends 0.
func DefaultTimeoutMs(adapter string) int {
	switch adapter {
	case "cli", "hlctl":
		return 60000
	default:
		return 30000
	}
}

// EffectiveExecutionTarget normalizes execution_target per adapter rules.
func EffectiveExecutionTarget(adapter, requested string) string {
	switch adapter {
	case "kubernetes":
		return "k8s_cluster"
	case "hlctl":
		return "backend"
	case "http", "prometheus":
		return "backend"
	default:
		if requested == "k8s_cluster" {
			return "k8s_cluster"
		}
		return "backend"
	}
}

func validationErrorMessages(errs []string) string {
	if len(errs) == 0 {
		return ""
	}
	return strings.Join(errs, "; ")
}

func fmtValidationError(errs []string) string {
	return fmt.Sprintf("validation failed: %s", validationErrorMessages(errs))
}
