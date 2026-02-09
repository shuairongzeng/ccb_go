//go:build windows

package terminal

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

func setSysProcAttrWindows(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}
