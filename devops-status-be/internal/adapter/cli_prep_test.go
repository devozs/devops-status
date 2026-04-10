package adapter

import (
	"strings"
	"testing"
)

func TestShellSingleQuote(t *testing.T) {
	if got := shellSingleQuote(`a'b`); got != `'a'\''b'` {
		t.Fatalf("quote: %q", got)
	}
}

func TestCliScriptWithPrep(t *testing.T) {
	s := cliScriptWithPrep("apk add curl", "curl", []string{"-s", "http://x"})
	if !strings.Contains(s, "apk add curl") {
		t.Fatalf("missing prep: %q", s)
	}
	if !strings.Contains(s, ") 1>&2") {
		t.Fatalf("prep should redirect stdout to stderr: %q", s)
	}
	if !strings.Contains(s, "exec 'curl' '-s' 'http://x'") {
		t.Fatalf("bad exec: %q", s)
	}
}

func TestCliScriptWithArbitraryPrepRedirect(t *testing.T) {
	s := cliScriptWithArbitrary("echo prep", "echo probe")
	if !strings.Contains(s, "echo prep") || !strings.Contains(s, "echo probe") {
		t.Fatalf("expected prep and body: %q", s)
	}
	if !strings.Contains(s, ") 1>&2") {
		t.Fatalf("prep should redirect stdout to stderr: %q", s)
	}
}

func TestValidateCLIPrepLength(t *testing.T) {
	if ValidateCLIPrepLength("") != "" {
		t.Fatal("empty should be ok")
	}
	if ValidateCLIPrepLength("x") != "" {
		t.Fatal("short should be ok")
	}
	long := strings.Repeat("a", maxCLIPrepRunes+1)
	if ValidateCLIPrepLength(long) == "" {
		t.Fatal("expected too long")
	}
}
