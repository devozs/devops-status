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

// DockerRunOpts configures docker run for CLI probes.
type DockerRunOpts struct {
	Network string // bridge, host, none — passed to --network=
}

// RunCLIAsDocker runs CLIConfig inside a disposable container via `docker run --rm`.
func RunCLIAsDocker(ctx context.Context, image string, cfg CLIConfig, opts DockerRunOpts, logMax int) (*ProbeResult, error) {
	start := time.Now()
	if image == "" {
		return &ProbeResult{
			Success:   false,
			ProbedAt:  start,
			LatencyMs: int(time.Since(start).Milliseconds()),
			Error:     "CLI runner image is empty",
		}, nil
	}

	timeout := 60 * time.Second
	if cfg.TimeoutMs > 0 {
		timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	script := BuildCLIContainerShellForRun(cfg)
	args := []string{"run", "--rm"}
	net := strings.TrimSpace(opts.Network)
	if net != "" && net != "default" {
		args = append(args, "--network", net)
	}
	for k, v := range cfg.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, image, "/bin/sh", "-c", script)

	cmd := exec.CommandContext(execCtx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
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
	ApplyCLILogFields(result, cfg, "docker", image, stdout.String(), stderr.String(), logMax)

	if err != nil && exitCode < 0 {
		result.Success = false
		result.Error = fmt.Sprintf("docker run failed: %v (stderr: %s)", err, strings.TrimSpace(stderr.String()))
		return result, nil
	}

	expectedExit := cfg.SuccessExit
	if err != nil && exitCode != expectedExit {
		result.Success = false
		result.Error = fmt.Sprintf("exit code %d: %s", exitCode, strings.TrimSpace(stderr.String()))
		// Trace already has stdout/stderr via ApplyCLILogFields
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

	if !result.Success && result.Error == "" && exitCode != expectedExit {
		result.Error = fmt.Sprintf("exit code %d", exitCode)
	}
	return result, nil
}
