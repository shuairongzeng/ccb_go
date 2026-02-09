//go:build !windows

package client

import "os/exec"

func setDetachedProcessPlatform(cmd *exec.Cmd) {
	// On Unix, no special flags needed for detached processes.
	// The process is already detached via cmd.Start() + Process.Release().
}
