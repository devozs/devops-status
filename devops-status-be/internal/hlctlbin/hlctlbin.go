// Package hlctlbin detects whether HLCTL features can run in the management process (kubectl + hlctl on PATH).
package hlctlbin

import (
	"os/exec"
)

// Enabled reports whether both kubectl and hlctl binaries are available for local execution.
func Enabled() bool {
	_, e1 := exec.LookPath("kubectl")
	_, e2 := exec.LookPath("hlctl")
	return e1 == nil && e2 == nil
}
