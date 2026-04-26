package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// HLCTLAdapter runs probes locally in the management process (hlctl kubeconfig + kubectl/hlctl shell).
type HLCTLAdapter struct {
	logMax int
}

// hlctlWire is config_json after server-side merge (credentials never persisted in DB).
type hlctlWire struct {
	CLIConfig
	HlctlUsername string `json:"_hlctl_username,omitempty"`
	HlctlPassword string `json:"_hlctl_password,omitempty"`
}

func NewHLCTLAdapter(logMax int) *HLCTLAdapter {
	if logMax <= 0 {
		logMax = DefaultCLILogMaxBytes
	}
	return &HLCTLAdapter{logMax: logMax}
}

func (a *HLCTLAdapter) Name() string { return "hlctl" }

func (a *HLCTLAdapter) Probe(ctx context.Context, configRaw json.RawMessage) (*ProbeResult, error) {
	var w hlctlWire
	if err := json.Unmarshal(configRaw, &w); err != nil {
		return nil, fmt.Errorf("parse hlctl config: %w", err)
	}
	user := strings.TrimSpace(w.HlctlUsername)
	pass := w.HlctlPassword
	if user == "" || pass == "" {
		return &ProbeResult{
			Success:  false,
			ProbedAt: time.Now(),
			Error:    "hlctl credentials missing after resolve (service provider username/password)",
		}, nil
	}
	cfg := w.CLIConfig
	if strings.TrimSpace(cfg.CLIShell) == "" {
		if bad := validateHLCTLCommand(cfg.Command); bad != nil {
			return bad, nil
		}
	}
	return RunHLCTLLocalProbe(ctx, cfg, user, pass, a.logMax)
}

var hlctlAllowedCommands = map[string]bool{"curl": true, "kubectl": true, "go": true, "echo": true, "hlctl": true}

func validateHLCTLCommand(command string) *ProbeResult {
	baseBin := command
	if idx := strings.Index(baseBin, "/"); idx >= 0 {
		baseBin = baseBin[idx+1:]
	}
	if !hlctlAllowedCommands[baseBin] && !hlctlAllowedCommands[command] {
		return &ProbeResult{
			Success:  false,
			ProbedAt: time.Now(),
			Error:    fmt.Sprintf("command not in allowlist: %s", command),
		}
	}
	return nil
}

// RunHLCTLLocalProbe runs hlctl kubeconfig then the same shell pipeline as CLI containers, using an isolated HOME.
func RunHLCTLLocalProbe(ctx context.Context, cfg CLIConfig, username, password string, logMax int) (*ProbeResult, error) {
	start := time.Now()
	homeDir, err := os.MkdirTemp("", "hlctl-home-*")
	if err != nil {
		return &ProbeResult{
			Success:  false,
			ProbedAt: start,
			Error:    fmt.Sprintf("temp home: %v", err),
		}, nil
	}
	defer func() { _ = os.RemoveAll(homeDir) }()

	timeout := 60 * time.Second
	if cfg.TimeoutMs > 0 {
		timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	script := buildHLCTLLocalScript(homeDir, username, password, cfg)
	cmd := exec.CommandContext(execCtx, "/bin/sh", "-c", script)
	cmd.Env = append(os.Environ(), "HOME="+homeDir)
	var stdout, stderr bytes.Buffer
	if sink := ProbeLogSinkFromContext(ctx); sink != nil {
		cmd.Stdout = io.MultiWriter(&stdout, &logChunkWriter{sink: sink, stream: "stdout"})
		cmd.Stderr = io.MultiWriter(&stderr, &logChunkWriter{sink: sink, stream: "stderr"})
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	runErr := cmd.Run()
	latency := time.Since(start)
	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	result := &ProbeResult{
		LatencyMs: int(latency.Milliseconds()),
		ProbedAt:  start,
		Metadata: map[string]any{
			"exit_code": exitCode,
		},
	}
	ApplyCLILogFields(result, cfg, "local", "management", stdout.String(), stderr.String(), logMax)

	if runErr != nil && exitCode < 0 {
		result.Success = false
		result.Error = fmt.Sprintf("probe exec failed: %v", runErr)
		return result, nil
	}

	expectedExit := cfg.SuccessExit
	if runErr != nil && exitCode != expectedExit {
		result.Success = false
		result.Error = fmt.Sprintf("exit code %d", exitCode)
		return result, nil
	}

	result.Success = true
	return result, nil
}

// buildHLCTLLocalScript sets HOME, runs hlctl kubeconfig, then the CLI shell body (metric or allowlisted exec).
func buildHLCTLLocalScript(homeDir, username, password string, cfg CLIConfig) string {
	var b strings.Builder
	b.WriteString("export HOME=")
	b.WriteString(shellSingleQuote(homeDir))
	b.WriteString("\nmkdir -p \"$HOME/.kube\"\n")
	for k, v := range cfg.Env {
		if strings.TrimSpace(k) == "" {
			continue
		}
		b.WriteString("export ")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(shellSingleQuote(v))
		b.WriteString("\n")
	}
	b.WriteString("set -eu\n")
	b.WriteString("hlctl kubeconfig --username=")
	b.WriteString(shellSingleQuote(username))
	b.WriteString(" --password=")
	b.WriteString(shellSingleQuote(password))
	b.WriteString("\n")
	b.WriteString(BuildCLIContainerShellForRun(cfg))
	return b.String()
}

// HLCTLVerifyKubeconfig runs only hlctl kubeconfig (for service provider verify). Returns combined log and whether exit was 0.
func HLCTLVerifyKubeconfig(ctx context.Context, homeDir, username, password string, logMax int) (latencyMs int, combinedLog string, success bool) {
	start := time.Now()
	script := strings.Builder{}
	script.WriteString("export HOME=")
	script.WriteString(shellSingleQuote(homeDir))
	script.WriteString("\nmkdir -p \"$HOME/.kube\"\nset -eu\n")
	script.WriteString("hlctl kubeconfig --username=")
	script.WriteString(shellSingleQuote(username))
	script.WriteString(" --password=")
	script.WriteString(shellSingleQuote(password))
	script.WriteString("\n")

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", script.String())
	cmd.Env = append(os.Environ(), "HOME="+homeDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	latencyMs = int(time.Since(start).Milliseconds())
	combined := CapCLILog(out.String(), logMax)
	if err != nil {
		if combined == "" {
			combined = err.Error()
		} else {
			combined = combined + "\n" + err.Error()
		}
		return latencyMs, combined, false
	}
	return latencyMs, combined, true
}
