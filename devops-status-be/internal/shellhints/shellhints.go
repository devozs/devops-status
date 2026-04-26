// Package shellhints parses and validates telemetry shell hint bodies (aligned with devops-status-fe probeEnvLines / nodeSelectorLines).
package shellhints

import (
	"fmt"
	"regexp"
	"strings"
)

const MaxHintBodyBytes = 262144 // 256 KiB

const (
	KindJobEnv           = "job_env"
	KindJobNodeSelector  = "job_node_selector"
	KindContainerPrep    = "container_prep"
	KindProbeShell       = "probe_shell"
)

var kinds = map[string]bool{
	KindJobEnv:          true,
	KindJobNodeSelector: true,
	KindContainerPrep:   true,
	KindProbeShell:      true,
}

func ValidKind(k string) bool { return kinds[k] }

var envVarNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ParseProbeEnvLines parses KEY=value lines (# comments, blank lines skipped). Values are not trimmed (matches TS).
func ParseProbeEnvLines(text string) (map[string]string, error) {
	env := make(map[string]string)
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			return nil, fmt.Errorf("invalid env line %d: expected KEY=value", i+1)
		}
		key := strings.TrimSpace(line[:eq])
		val := line[eq+1:]
		if !envVarNamePattern.MatchString(key) {
			return nil, fmt.Errorf("invalid env key on line %d: %s", i+1, key)
		}
		env[key] = val
	}
	return env, nil
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

func validNodeSelectorKey(k string) bool {
	if len(k) < 1 || len(k) > 253 {
		return false
	}
	for _, r := range k {
		if r > 127 || !nodeSelectorKeyCharOK(r) {
			return false
		}
	}
	return true
}

func validNodeSelectorValue(v string) bool {
	if len(v) > 63 {
		return false
	}
	return !strings.ContainsAny(v, "\x00\n\r")
}

// ParseNodeSelectorLines parses key=value lines (# comments skipped). Keys and values trimmed (matches TS).
func ParseNodeSelectorLines(text string) (map[string]string, error) {
	sel := make(map[string]string)
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			return nil, fmt.Errorf("invalid node selector line %d: expected key=value", i+1)
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if !validNodeSelectorKey(key) {
			return nil, fmt.Errorf("invalid node selector key on line %d: %s", i+1, key)
		}
		if !validNodeSelectorValue(val) {
			return nil, fmt.Errorf("invalid node selector value on line %d (max 63 chars, no newlines)", i+1)
		}
		sel[key] = val
	}
	return sel, nil
}

// MapsEqual compares two string maps for equality.
func MapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// ValidateHintBody returns an error if body is invalid for kind (size, parse rules).
func ValidateHintBody(kind, body string) error {
	if strings.Contains(body, "\x00") {
		return fmt.Errorf("body must not contain NUL")
	}
	if len(body) > MaxHintBodyBytes {
		return fmt.Errorf("body exceeds max size (%d bytes)", MaxHintBodyBytes)
	}
	switch kind {
	case KindJobEnv:
		if _, err := ParseProbeEnvLines(body); err != nil {
			return err
		}
	case KindJobNodeSelector:
		if _, err := ParseNodeSelectorLines(body); err != nil {
			return err
		}
	case KindContainerPrep, KindProbeShell:
		return nil
	default:
		return fmt.Errorf("unknown hint kind %q", kind)
	}
	return nil
}
