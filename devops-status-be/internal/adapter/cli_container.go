package adapter

import (
	"fmt"
	"strings"
)

// Preset runner profiles for CLI container images (telemetry config_json.runner).
const (
	CLIRunnerProfileAlpine   = "alpine"
	CLIRunnerProfileUbuntu24 = "ubuntu_24"
)

// CLIRunnerImages maps preset + legacy default to concrete image references (from server env).
type CLIRunnerImages struct {
	Alpine   string
	Ubuntu24 string
	Legacy   string // CLI_RUNNER_IMAGE fallback when runner is empty
}

// ResolveCLIContainerImage picks the image for a probe from config and server defaults.
func ResolveCLIContainerImage(cfg CLIConfig, imgs CLIRunnerImages) (string, error) {
	override := strings.TrimSpace(cfg.RunnerImage)
	if override != "" {
		if !isAllowedRunnerImage(override, imgs) {
			return "", fmt.Errorf("runner_image is not an allowed image for this server")
		}
		return override, nil
	}
	p := strings.TrimSpace(cfg.Runner)
	if p == "" {
		if imgs.Legacy != "" {
			return imgs.Legacy, nil
		}
		if imgs.Alpine != "" {
			return imgs.Alpine, nil
		}
		return "", fmt.Errorf("no CLI runner image configured")
	}
	switch p {
	case CLIRunnerProfileAlpine:
		if imgs.Alpine == "" {
			return "", fmt.Errorf("alpine runner image not configured")
		}
		return imgs.Alpine, nil
	case CLIRunnerProfileUbuntu24:
		if imgs.Ubuntu24 == "" {
			return "", fmt.Errorf("ubuntu_24 runner image not configured")
		}
		return imgs.Ubuntu24, nil
	default:
		return "", fmt.Errorf("unsupported runner profile %q (use alpine or ubuntu_24)", p)
	}
}

func isAllowedRunnerImage(ref string, imgs CLIRunnerImages) bool {
	for _, c := range []string{imgs.Alpine, imgs.Ubuntu24, imgs.Legacy} {
		if c != "" && ref == c {
			return true
		}
	}
	return false
}

// BuildCLIContainerShellScript returns the single shell script run inside the container (set -eu, prep, exec).
func BuildCLIContainerShellScript(cfg CLIConfig) string {
	return cliScriptWithPrep(cfg.ContainerPrep, cfg.Command, cfg.Args)
}

// BuildCLIContainerShellForRun uses cli_shell (arbitrary) when set, else allowlisted exec script.
func BuildCLIContainerShellForRun(cfg CLIConfig) string {
	if strings.TrimSpace(cfg.CLIShell) != "" {
		return cliScriptWithArbitrary(cfg.ContainerPrep, cfg.CLIShell)
	}
	return BuildCLIContainerShellScript(cfg)
}

// CapCLILog truncates a log string to max bytes (suffix marker if cut).
func CapCLILog(s string, max int) string {
	if max <= 0 {
		return s
	}
	return truncate(s, max)
}

// ApplyCLILogFields sets Trace, Metadata stdout/stderr, runner type, and image (capped).
func ApplyCLILogFields(res *ProbeResult, cfg CLIConfig, runnerKind, image, stdout, stderr string, logMax int) {
	if res.Metadata == nil {
		res.Metadata = map[string]any{}
	}
	out := CapCLILog(stdout, logMax)
	errLog := CapCLILog(stderr, logMax)
	res.Metadata["cli_stdout"] = out
	res.Metadata["cli_stderr"] = errLog
	res.Metadata["cli_runner"] = runnerKind
	res.Metadata["cli_image"] = image
	if strings.TrimSpace(cfg.CLIShell) != "" {
		res.Metadata["cli_shell"] = true
	} else {
		res.Metadata["command"] = cfg.Command
		res.Metadata["args"] = cfg.Args
	}
	if strings.TrimSpace(cfg.ContainerPrep) != "" {
		res.Metadata["container_prep"] = true
	}
	// Human-friendly combined trace for UIs that only read Trace
	var b strings.Builder
	if out != "" {
		b.WriteString(out)
	}
	if errLog != "" {
		if b.Len() > 0 {
			b.WriteString("\n--- stderr ---\n")
		}
		b.WriteString(errLog)
	}
	res.Trace = CapCLILog(b.String(), logMax)
}
