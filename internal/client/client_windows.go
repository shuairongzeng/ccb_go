//go:build windows

package client

import (
	"os/exec"
	"syscall"
)

const (
	createNoWindow   = 0x08000000
	detachedProcess  = 0x00000008
)

func setDetachedProcessPlatform(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= createNoWindow | detachedProcess
}
