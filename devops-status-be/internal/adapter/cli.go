package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type CLIAdapter struct {
	allowedCommands map[string]bool
}

type CLIConfig struct {
	Command    string            `json:"command"`
	Args       []string          `json:"args,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	TimeoutMs  int               `json:"timeout_ms,omitempty"`
	ParseJSON  bool              `json:"parse_json,omitempty"`
	SuccessExit int              `json:"success_exit,omitempty"`
}

type CLITimestampedStep struct {
	Step      string  `json:"step"`
	DurationMs float64 `json:"duration_ms"`
	Success   bool    `json:"success"`
}

type CLIStructuredOutput struct {
	Success    bool                 `json:"success"`
	Value      float64              `json:"value,omitempty"`
	Steps      []CLITimestampedStep `json:"steps,omitempty"`
	Message    string               `json:"message,omitempty"`
}

func NewCLIAdapter(allowedCommands []string) *CLIAdapter {
	allowed := make(map[string]bool, len(allowedCommands))
	for _, cmd := range allowedCommands {
		allowed[cmd] = true
	}
	return &CLIAdapter{allowedCommands: allowed}
}

func (a *CLIAdapter) Name() string { return "cli" }

func (a *CLIAdapter) Probe(ctx context.Context, configRaw json.RawMessage) (*ProbeResult, error) {
	var cfg CLIConfig
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		return nil, fmt.Errorf("parse cli config: %w", err)
	}

	baseBin := cfg.Command
	if idx := strings.Index(baseBin, "/"); idx >= 0 {
		baseBin = baseBin[idx+1:]
	}
	if !a.allowedCommands[baseBin] && !a.allowedCommands[cfg.Command] {
		return &ProbeResult{
			Success:  false,
			ProbedAt: time.Now(),
			Error:    fmt.Sprintf("command not in allowlist: %s", cfg.Command),
		}, nil
	}

	timeout := 60 * time.Second
	if cfg.TimeoutMs > 0 {
		timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, cfg.Command, cfg.Args...)

	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	latency := time.Since(start)

	result := &ProbeResult{
		LatencyMs: int(latency.Milliseconds()),
		ProbedAt:  start,
		Metadata: map[string]any{
			"command":   cfg.Command,
			"args":      cfg.Args,
			"exit_code": cmd.ProcessState.ExitCode(),
		},
	}

	expectedExit := cfg.SuccessExit
	exitCode := cmd.ProcessState.ExitCode()

	if err != nil && exitCode != expectedExit {
		result.Success = false
		result.Error = fmt.Sprintf("exit code %d: %s", exitCode, stderr.String())
		result.Trace = truncate(stdout.String(), 2000)
		return result, nil
	}

	if cfg.ParseJSON {
		var structured CLIStructuredOutput
		if jsonErr := json.Unmarshal(stdout.Bytes(), &structured); jsonErr == nil {
			result.Success = structured.Success
			result.RawValue = structured.Value
			result.Metadata["steps"] = structured.Steps
			result.Metadata["message"] = structured.Message

			var totalStepMs float64
			for _, step := range structured.Steps {
				totalStepMs += step.DurationMs
			}
			if totalStepMs > 0 {
				result.RawValue = totalStepMs
			}
		} else {
			result.Success = exitCode == expectedExit
			result.Error = fmt.Sprintf("json parse failed: %v", jsonErr)
		}
	} else {
		result.Success = exitCode == expectedExit
	}

	result.Trace = truncate(stdout.String(), 2000)
	return result, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
