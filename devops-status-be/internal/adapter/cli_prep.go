package adapter

import (
	"strings"
	"unicode/utf8"
)

const maxCLIPrepRunes = 16384
const maxCLIShellRunes = 32768

// shellSingleQuote returns s as a single-quoted POSIX shell word (safe for sh -c).
func shellSingleQuote(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `'\''`) + `'`
}

// cliScriptWithPrep builds "set -eu\n<prep>\nexec <cmd> <args...>" for /bin/sh -c.
// Prep stdout is redirected to stderr so probe stdout is only the allowlisted command (QoS / parsing).
func cliScriptWithPrep(prep, command string, args []string) string {
	p := strings.TrimSpace(prep)
	var b strings.Builder
	b.WriteString("set -eu\n")
	if p != "" {
		b.WriteString("(\nset -eu\n")
		b.WriteString(p)
		b.WriteString("\n) 1>&2\n")
	}
	b.WriteString("exec ")
	b.WriteString(shellSingleQuote(command))
	for _, a := range args {
		b.WriteByte(' ')
		b.WriteString(shellSingleQuote(a))
	}
	return b.String()
}

// cliScriptWithArbitrary runs set -eu, optional prep, then user shell body (no allowlisted exec).
// Prep stdout is redirected to stderr so probe stdout is only cli_shell (metric QoS / JSON / exit semantics).
func cliScriptWithArbitrary(prep, body string) string {
	p := strings.TrimSpace(prep)
	body = strings.TrimSpace(body)
	b := strings.Builder{}
	b.WriteString("set -eu\n")
	if p != "" {
		b.WriteString("(\nset -eu\n")
		b.WriteString(p)
		b.WriteString("\n) 1>&2\n")
	}
	b.WriteString(body)
	b.WriteByte('\n')
	return b.String()
}

// ValidateCLIShellLength returns an error message if cli_shell is too large or invalid UTF-8.
func ValidateCLIShellLength(shell string) string {
	if shell == "" {
		return ""
	}
	if !utf8.ValidString(shell) {
		return "cli_shell must be valid UTF-8"
	}
	if len([]rune(shell)) > maxCLIShellRunes {
		return "cli_shell exceeds maximum length"
	}
	return ""
}

// ValidateCLIPrepLength returns an error message if prep is too large or invalid UTF-8.
func ValidateCLIPrepLength(prep string) string {
	if prep == "" {
		return ""
	}
	if !utf8.ValidString(prep) {
		return "container_prep must be valid UTF-8"
	}
	if len([]rune(prep)) > maxCLIPrepRunes {
		return "container_prep exceeds maximum length"
	}
	return ""
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
