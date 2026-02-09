package terminal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// WeztermBackend implements the Backend interface using WezTerm.
type WeztermBackend struct {
	socketPath string
}

// Name returns "wezterm".
func (w *WeztermBackend) Name() string { return "wezterm" }

// IsAvailable checks if WezTerm CLI is available.
func (w *WeztermBackend) IsAvailable() bool {
	// Check explicit env first
	if os.Getenv("WEZTERM_PANE") != "" {
		_, err := exec.LookPath("wezterm")
		return err == nil
	}

	// Even without WEZTERM_PANE, check if wezterm cli works
	// (e.g. launched from WezTerm but env not propagated)
	if _, err := exec.LookPath("wezterm"); err != nil {
		return false
	}

	// Try "wezterm cli list" to see if we're inside WezTerm
	cmd := exec.Command("wezterm", "cli", "list")
	setSysProcAttr(cmd)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// getSocketArgs returns the base arguments for wezterm cli commands.
func (w *WeztermBackend) getSocketArgs() []string {
	return []string{"cli"}
}

// discoverSocket finds the WezTerm Unix socket path with multi-path search and caching.
func (w *WeztermBackend) discoverSocket() string {
	if w.socketPath != "" {
		return w.socketPath
	}

	// Check WEZTERM_UNIX_SOCKET env
	sock := os.Getenv("WEZTERM_UNIX_SOCKET")
	if sock != "" {
		w.socketPath = sock
		return sock
	}

	// Try to find socket in common locations
	var searchDirs []string
	if runtime.GOOS == "windows" {
		localApp := os.Getenv("LOCALAPPDATA")
		if localApp != "" {
			searchDirs = append(searchDirs, filepath.Join(localApp, "wezterm"))
		}
		// Also check user profile
		userProfile := os.Getenv("USERPROFILE")
		if userProfile != "" {
			searchDirs = append(searchDirs, filepath.Join(userProfile, ".local", "share", "wezterm"))
		}
	} else {
		xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
		if xdgRuntime != "" {
			searchDirs = append(searchDirs, filepath.Join(xdgRuntime, "wezterm"))
		}
		searchDirs = append(searchDirs, filepath.Join(os.TempDir(), "wezterm"))
		// Also check home directory
		home, err := os.UserHomeDir()
		if err == nil {
			searchDirs = append(searchDirs, filepath.Join(home, ".local", "share", "wezterm"))
		}
	}

	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".sock") || strings.HasPrefix(e.Name(), "gui-sock-") {
				w.socketPath = filepath.Join(dir, e.Name())
				return w.socketPath
			}
		}
	}

	return ""
}

// SendKeys sends text to a WezTerm pane.
func (w *WeztermBackend) SendKeys(paneID string, text string) error {
	args := append(w.getSocketArgs(), "send-text")
	if paneID != "" {
		args = append(args, "--pane-id", paneID)
	}
	args = append(args, "--no-paste", text+"\r")
	cmd := exec.Command("wezterm", args...)
	setSysProcAttr(cmd)
	return cmd.Run()
}

// SendEnterWithRetry sends Enter to a pane with retries for reliability.
func (w *WeztermBackend) SendEnterWithRetry(paneID string, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		args := append(w.getSocketArgs(), "send-text")
		if paneID != "" {
			args = append(args, "--pane-id", paneID)
		}
		args = append(args, "--no-paste", "\r")
		cmd := exec.Command("wezterm", args...)
		setSysProcAttr(cmd)
		if err := cmd.Run(); err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("failed to send Enter to pane %s after %d retries", paneID, maxRetries)
}

// CapturePane captures the content of a WezTerm pane.
func (w *WeztermBackend) CapturePane(paneID string) (string, error) {
	return w.GetPaneContent(paneID, 0)
}

// GetPaneContent gets the content of a WezTerm pane, optionally limited to N lines.
func (w *WeztermBackend) GetPaneContent(paneID string, lines int) (string, error) {
	args := append(w.getSocketArgs(), "get-text")
	if paneID != "" {
		args = append(args, "--pane-id", paneID)
	}
	cmd := exec.Command("wezterm", args...)
	setSysProcAttr(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	content := string(out)
	if lines > 0 {
		allLines := strings.Split(content, "\n")
		if len(allLines) > lines {
			allLines = allLines[len(allLines)-lines:]
		}
		content = strings.Join(allLines, "\n")
	}

	return content, nil
}

// SplitWindow splits a WezTerm pane.
func (w *WeztermBackend) SplitWindow(target string, cmdStr string) (string, error) {
	args := append(w.getSocketArgs(), "split-pane")
	if target != "" {
		args = append(args, "--pane-id", target)
	}
	args = append(args, "--right")
	if cmdStr != "" {
		args = append(args, "--")
		// Split command string into args for proper exec
		args = append(args, splitShellCommand(cmdStr)...)
	}
	cmd := exec.Command("wezterm", args...)
	setSysProcAttr(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("wezterm split-pane failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// splitShellCommand splits a command string into arguments, respecting quotes.
func splitShellCommand(cmd string) []string {
	var args []string
	var current strings.Builder
	inQuote := byte(0)

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch {
		case c == inQuote:
			inQuote = 0
			current.WriteByte(c)
		case inQuote != 0:
			current.WriteByte(c)
		case c == '"' || c == '\'':
			inQuote = c
			current.WriteByte(c)
		case c == ' ' || c == '\t':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// ListPanes returns all WezTerm panes.
func (w *WeztermBackend) ListPanes() ([]PaneInfo, error) {
	args := append(w.getSocketArgs(), "list", "--format", "json")
	cmd := exec.Command("wezterm", args...)
	setSysProcAttr(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var rawPanes []struct {
		PaneID   int    `json:"pane_id"`
		Title    string `json:"title"`
		CWD      string `json:"cwd"`
		IsActive bool   `json:"is_active"`
		Width    int    `json:"size_cols"`
		Height   int    `json:"size_rows"`
	}
	if err := json.Unmarshal(out, &rawPanes); err != nil {
		// Fallback to text parsing
		return w.listPanesText(string(out))
	}

	panes := make([]PaneInfo, 0, len(rawPanes))
	for _, rp := range rawPanes {
		panes = append(panes, PaneInfo{
			ID:      fmt.Sprintf("%d", rp.PaneID),
			Title:   rp.Title,
			Command: rp.CWD,
			Active:  rp.IsActive,
			Width:   rp.Width,
			Height:  rp.Height,
		})
	}
	return panes, nil
}

// listPanesText parses pane list from text output as fallback.
func (w *WeztermBackend) listPanesText(output string) ([]PaneInfo, error) {
	var panes []PaneInfo
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			panes = append(panes, PaneInfo{
				ID:    parts[0],
				Title: strings.Join(parts[1:], " "),
			})
		}
	}
	return panes, nil
}

// KillPane kills a WezTerm pane.
func (w *WeztermBackend) KillPane(paneID string) error {
	args := append(w.getSocketArgs(), "kill-pane", "--pane-id", paneID)
	cmd := exec.Command("wezterm", args...)
	setSysProcAttr(cmd)
	return cmd.Run()
}

// HasSession checks if a WezTerm pane exists.
func (w *WeztermBackend) HasSession(sessionID string) bool {
	return w.IsAlive(sessionID)
}

// IsAlive checks if a WezTerm pane is still alive.
func (w *WeztermBackend) IsAlive(paneID string) bool {
	panes, err := w.ListPanes()
	if err != nil {
		return false
	}
	for _, p := range panes {
		if p.ID == paneID {
			return true
		}
	}
	return false
}

// SetPaneTitle sets the title of a WezTerm pane.
func (w *WeztermBackend) SetPaneTitle(paneID string, title string) error {
	// WezTerm sets title via escape sequence
	escSeq := fmt.Sprintf("\x1b]0;%s\x07", title)
	args := append(w.getSocketArgs(), "send-text", "--pane-id", paneID, "--no-paste", escSeq)
	cmd := exec.Command("wezterm", args...)
	setSysProcAttr(cmd)
	return cmd.Run()
}

// GetPaneTitle returns the title of a WezTerm pane.
func (w *WeztermBackend) GetPaneTitle(paneID string) (string, error) {
	panes, err := w.ListPanes()
	if err != nil {
		return "", err
	}
	for _, p := range panes {
		if p.ID == paneID {
			return p.Title, nil
		}
	}
	return "", &ErrPaneNotFound{PaneID: paneID, Backend: "wezterm"}
}

// WaitReady waits for a WezTerm pane to become ready.
func (w *WeztermBackend) WaitReady(paneID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		if w.IsAlive(paneID) {
			return nil
		}
		time.Sleep(interval)
		if interval < 500*time.Millisecond {
			interval = time.Duration(float64(interval) * 1.5)
		}
	}
	return &ErrWaitTimeout{PaneID: paneID, Timeout: timeout}
}

// FindPaneByTitleMarker finds a pane whose title contains the given marker.
func (w *WeztermBackend) FindPaneByTitleMarker(titleMarker string) (string, error) {
	return FindPaneByTitle(w, titleMarker)
}
