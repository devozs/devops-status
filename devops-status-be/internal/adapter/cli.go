package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// K8sCLIRunner runs CLIConfig on a Kubernetes cluster (optional).
type K8sCLIRunner func(ctx context.Context, endpoint, token string, insecure bool, image string, cfg CLIConfig) (*ProbeResult, error)

// BackendCLIRunner runs CLI locally via docker or a designated cluster (server-injected).
type BackendCLIRunner func(ctx context.Context, image string, cfg CLIConfig) (*ProbeResult, error)

type CLIAdapter struct {
	allowedCommands map[string]bool
	runnerImages    CLIRunnerImages
	logMax          int
	backendRun      BackendCLIRunner
	k8sRun          K8sCLIRunner
}

type CLIConfig struct {
	Command         string            `json:"command"`
	Args            []string          `json:"args,omitempty"`
	Env             map[string]string `json:"env,omitempty"`
	TimeoutMs       int               `json:"timeout_ms,omitempty"`
	ParseJSON       bool              `json:"parse_json,omitempty"`
	SuccessExit     int               `json:"success_exit,omitempty"`
	ContainerPrep   string            `json:"container_prep,omitempty"`
	Runner          string            `json:"runner,omitempty"`       // alpine | ubuntu_24
	RunnerImage     string            `json:"runner_image,omitempty"` // optional; must match server allowlist
	ExecutionTarget string            `json:"execution_target,omitempty"` // backend | k8s_cluster
	ClusterID       string            `json:"cluster_id,omitempty"`
	K8sVersion      string            `json:"k8s_version,omitempty"`
	Endpoint        string            `json:"endpoint,omitempty"`
	Token           string            `json:"token,omitempty"`
	InsecureSkipTLS bool              `json:"insecure_skip_tls,omitempty"`
	// CLIShell is arbitrary shell (metric QoS mode). When set, allowlist is skipped and this runs after container_prep.
	CLIShell string `json:"cli_shell,omitempty"`
}

type CLITimestampedStep struct {
	Step       string  `json:"step"`
	DurationMs float64 `json:"duration_ms"`
	Success    bool    `json:"success"`
}

type CLIStructuredOutput struct {
	Success    bool                 `json:"success"`
	Value      float64              `json:"value,omitempty"`
	Steps      []CLITimestampedStep `json:"steps,omitempty"`
	Message    string               `json:"message,omitempty"`
}

// DefaultCLILogMaxBytes is used when logMax is zero.
const DefaultCLILogMaxBytes = 32768

func NewCLIAdapter(allowedCommands []string, imgs CLIRunnerImages, logMax int, backend BackendCLIRunner, k8s K8sCLIRunner) *CLIAdapter {
	if imgs.Legacy == "" {
		imgs.Legacy = "alpine:3.20"
	}
	if imgs.Alpine == "" {
		imgs.Alpine = imgs.Legacy
	}
	if imgs.Ubuntu24 == "" {
		imgs.Ubuntu24 = "ubuntu:24.04"
	}
	if logMax <= 0 {
		logMax = DefaultCLILogMaxBytes
	}
	allowed := make(map[string]bool, len(allowedCommands))
	for _, cmd := range allowedCommands {
		allowed[cmd] = true
	}
	return &CLIAdapter{allowedCommands: allowed, runnerImages: imgs, logMax: logMax, backendRun: backend, k8sRun: k8s}
}

func (a *CLIAdapter) Name() string { return "cli" }

func (a *CLIAdapter) Probe(ctx context.Context, configRaw json.RawMessage) (*ProbeResult, error) {
	var cfg CLIConfig
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		return nil, fmt.Errorf("parse cli config: %w", err)
	}

	image, err := ResolveCLIContainerImage(cfg, a.runnerImages)
	if err != nil {
		return &ProbeResult{
			Success:  false,
			ProbedAt: time.Now(),
			Error:    err.Error(),
		}, nil
	}

	arbitrary := strings.TrimSpace(cfg.CLIShell) != ""

	if cfg.ExecutionTarget == "k8s_cluster" {
		if !arbitrary {
			if bad := a.validateCommand(cfg.Command); bad != nil {
				return bad, nil
			}
		}
		if a.k8sRun == nil {
			return &ProbeResult{
				Success:  false,
				ProbedAt: time.Now(),
				Error:    "k8s CLI runner not configured",
			}, nil
		}
		if cfg.Endpoint == "" || cfg.Token == "" {
			return &ProbeResult{
				Success:  false,
				ProbedAt: time.Now(),
				Error:    "cluster endpoint and token required after credential merge for k8s CLI",
			}, nil
		}
		return a.k8sRun(ctx, cfg.Endpoint, cfg.Token, cfg.InsecureSkipTLS, image, cfg)
	}

	if !arbitrary {
		if bad := a.validateCommand(cfg.Command); bad != nil {
			return bad, nil
		}
	}
	if a.backendRun == nil {
		return &ProbeResult{
			Success:  false,
			ProbedAt: time.Now(),
			Error:    "CLI backend runner not configured (set CLI_BACKEND_EXECUTOR and dependencies)",
		}, nil
	}
	return a.backendRun(ctx, image, cfg)
}

func (a *CLIAdapter) validateCommand(command string) *ProbeResult {
	baseBin := command
	if idx := strings.Index(baseBin, "/"); idx >= 0 {
		baseBin = baseBin[idx+1:]
	}
	if !a.allowedCommands[baseBin] && !a.allowedCommands[command] {
		return &ProbeResult{
			Success:  false,
			ProbedAt: time.Now(),
			Error:    fmt.Sprintf("command not in allowlist: %s", command),
		}
	}
	return nil
}
