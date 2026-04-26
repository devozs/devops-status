package adapter

import (
	"strings"
	"testing"
)

func TestResolveCLIContainerImage(t *testing.T) {
	imgs := CLIRunnerImages{Alpine: "alpine:test", Ubuntu24: "ubuntu:test", Toolkit: "toolkit:test", Legacy: "legacy:x"}
	img, err := ResolveCLIContainerImage(CLIConfig{}, imgs)
	if err != nil || img != "legacy:x" {
		t.Fatalf("default legacy: %q %v", img, err)
	}
	img, err = ResolveCLIContainerImage(CLIConfig{Runner: CLIRunnerProfileAlpine}, imgs)
	if err != nil || img != "alpine:test" {
		t.Fatalf("alpine: %q %v", img, err)
	}
	img, err = ResolveCLIContainerImage(CLIConfig{Runner: CLIRunnerProfileUbuntu24}, imgs)
	if err != nil || img != "ubuntu:test" {
		t.Fatalf("ubuntu: %q %v", img, err)
	}
	img, err = ResolveCLIContainerImage(CLIConfig{Runner: CLIRunnerProfileToolkit}, imgs)
	if err != nil || img != "toolkit:test" {
		t.Fatalf("toolkit: %q %v", img, err)
	}
	_, err = ResolveCLIContainerImage(CLIConfig{Runner: "bogus"}, imgs)
	if err == nil {
		t.Fatal("expected error for bogus runner")
	}
	img, err = ResolveCLIContainerImage(CLIConfig{RunnerImage: "alpine:test"}, imgs)
	if err != nil || img != "alpine:test" {
		t.Fatalf("override: %q %v", img, err)
	}
	img, err = ResolveCLIContainerImage(CLIConfig{RunnerImage: "toolkit:test"}, imgs)
	if err != nil || img != "toolkit:test" {
		t.Fatalf("toolkit override: %q %v", img, err)
	}
	_, err = ResolveCLIContainerImage(CLIConfig{RunnerImage: "evil:latest"}, imgs)
	if err == nil {
		t.Fatal("expected error for disallowed override")
	}
}

func TestBuildCLIContainerShellScriptAlwaysUsesSh(t *testing.T) {
	s := BuildCLIContainerShellScript(CLIConfig{Command: "curl", Args: []string{"-s", "x"}})
	if !strings.Contains(s, "set -eu") || !strings.Contains(s, "exec") || !strings.Contains(s, "curl") {
		t.Fatalf("unexpected script: %q", s)
	}
}
