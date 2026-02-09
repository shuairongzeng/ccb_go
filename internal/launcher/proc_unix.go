//go:build !windows

package launcher

import "os/exec"

func setSysProcAttrLauncher(cmd *exec.Cmd) {
	// No special flags needed on Unix
}
