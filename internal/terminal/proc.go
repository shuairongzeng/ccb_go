package terminal

import (
	"os/exec"
	"runtime"
)

// setSysProcAttr sets platform-specific process attributes.
// On Windows, this hides the console window for subprocess calls.
func setSysProcAttr(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		setSysProcAttrWindows(cmd)
	}
}
