//go:build !windows

package terminal

import "os/exec"

func setSysProcAttrWindows(cmd *exec.Cmd) {
	// no-op on non-Windows
}
