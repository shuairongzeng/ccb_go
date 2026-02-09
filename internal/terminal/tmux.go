package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TmuxBackend implements the Backend interface using tmux.
type TmuxBackend struct {
	socketName string // optional: custom tmux socket name
}

// Name returns "tmux".
func (t *TmuxBackend) Name() string { return "tmux" }

// IsAvailable checks if tmux is available and we're inside a tmux session.
func (t *TmuxBackend) IsAvailable() bool {
	if os.Getenv("TMUX") == "" {
		return false
	}
	_, err := exec.LookPath("tmux")
	return err == nil
}

// socketArgs returns extra args for custom socket if configured.
func (t *TmuxBackend) socketArgs() []string {
	sock := t.socketName
	if sock == "" {
		sock = strings.TrimSpace(os.Getenv("CCB_TMUX_SOCKET"))
	}
	if sock != "" {
		return []string{"-L", sock}
	}
	return nil
}

// runCmd runs a tmux command with optional socket args.
func (t *TmuxBackend) runCmd(args ...string) error {
	fullArgs := append(t.socketArgs(), args...)
	cmd := exec.Command("tmux", fullArgs...)
	setSysProcAttr(cmd)
	return cmd.Run()
}

// runCmdOutput runs a tmux command and returns stdout.
func (t *TmuxBackend) runCmdOutput(args ...string) (string, error) {
	fullArgs := append(t.socketArgs(), args...)
	cmd := exec.Command("tmux", fullArgs...)
	setSysProcAttr(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// SendKeys sends text to a tmux pane via send-keys.
func (t *TmuxBackend) SendKeys(paneID string, text string) error {
	// Use bracketed paste for multiline text to avoid interpretation issues
	if strings.Contains(text, "\n") {
		return t.sendBracketedPaste(paneID, text)
	}
	return t.runCmd("send-keys", "-t", paneID, text, "Enter")
}

// sendBracketedPaste sends text using tmux's load-buffer + paste-buffer for reliability.
func (t *TmuxBackend) sendBracketedPaste(paneID string, text string) error {
	// Write to a temp file, load into tmux buffer, then paste
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("ccb-tmux-%d.txt", os.Getpid()))
	if err := os.WriteFile(tmpFile, []byte(text), 0600); err != nil {
		// Fallback to direct send-keys
		return t.runCmd("send-keys", "-t", paneID, text, "Enter")
	}
	defer os.Remove(tmpFile)

	if err := t.runCmd("load-buffer", tmpFile); err != nil {
		return t.runCmd("send-keys", "-t", paneID, text, "Enter")
	}

	if err := t.runCmd("paste-buffer", "-t", paneID, "-d"); err != nil {
		return err
	}

	// Send Enter after paste
	return t.runCmd("send-keys", "-t", paneID, "Enter")
}

// CapturePane captures the content of a tmux pane.
func (t *TmuxBackend) CapturePane(paneID string) (string, error) {
	return t.runCmdOutput("capture-pane", "-t", paneID, "-p", "-S", "-")
}

// SplitWindow splits a tmux window and runs a command in the new pane.
func (t *TmuxBackend) SplitWindow(target string, cmd string) (string, error) {
	args := []string{"split-window", "-t", target, "-h", "-P", "-F", "#{pane_id}"}
	if cmd != "" {
		args = append(args, cmd)
	}
	output, err := t.runCmdOutput(args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// ListPanes returns all tmux panes.
func (t *TmuxBackend) ListPanes() ([]PaneInfo, error) {
	format := "#{pane_id}\t#{pane_title}\t#{pane_current_command}\t#{pane_active}\t#{pane_width}\t#{pane_height}"
	output, err := t.runCmdOutput("list-panes", "-a", "-F", format)
	if err != nil {
		return nil, err
	}

	var panes []PaneInfo
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 6)
		if len(parts) < 6 {
			continue
		}
		width := 0
		height := 0
		fmt.Sscanf(parts[4], "%d", &width)
		fmt.Sscanf(parts[5], "%d", &height)
		panes = append(panes, PaneInfo{
			ID:      parts[0],
			Title:   parts[1],
			Command: parts[2],
			Active:  parts[3] == "1",
			Width:   width,
			Height:  height,
		})
	}
	return panes, nil
}

// KillPane kills a tmux pane.
func (t *TmuxBackend) KillPane(paneID string) error {
	return t.runCmd("kill-pane", "-t", paneID)
}

// HasSession checks if a tmux session/pane exists.
func (t *TmuxBackend) HasSession(sessionID string) bool {
	err := t.runCmd("has-session", "-t", sessionID)
	return err == nil
}

// IsAlive checks if a tmux pane is still alive.
func (t *TmuxBackend) IsAlive(paneID string) bool {
	// list-panes with target filter; if it fails, pane is dead
	output, err := t.runCmdOutput("display-message", "-t", paneID, "-p", "#{pane_id}")
	if err != nil {
		return false
	}
	return strings.TrimSpace(output) != ""
}

// SetPaneTitle sets the title of a tmux pane using select-pane -T.
func (t *TmuxBackend) SetPaneTitle(paneID string, title string) error {
	return t.runCmd("select-pane", "-t", paneID, "-T", title)
}

// GetPaneTitle returns the title of a tmux pane.
func (t *TmuxBackend) GetPaneTitle(paneID string) (string, error) {
	output, err := t.runCmdOutput("display-message", "-t", paneID, "-p", "#{pane_title}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// WaitReady waits for a tmux pane to become ready.
func (t *TmuxBackend) WaitReady(paneID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		if t.IsAlive(paneID) {
			return nil
		}
		time.Sleep(interval)
		if interval < 500*time.Millisecond {
			interval = time.Duration(float64(interval) * 1.5)
		}
	}
	return &ErrWaitTimeout{PaneID: paneID, Timeout: timeout}
}

// PipePane starts logging a pane's output to a file.
func (t *TmuxBackend) PipePane(paneID string, logFile string) error {
	dir := filepath.Dir(logFile)
	os.MkdirAll(dir, 0755)
	return t.runCmd("pipe-pane", "-t", paneID, fmt.Sprintf("cat >> %s", logFile))
}

// StopPipePane stops logging a pane's output.
func (t *TmuxBackend) StopPipePane(paneID string) error {
	return t.runCmd("pipe-pane", "-t", paneID)
}

// RespawnPane respawns a dead pane with a new command.
func (t *TmuxBackend) RespawnPane(paneID string, cmd string) error {
	args := []string{"respawn-pane", "-t", paneID, "-k"}
	if cmd != "" {
		args = append(args, cmd)
	}
	return t.runCmd(args...)
}

// CreateAutoLayout creates panes for multiple providers in an automatic layout.
// Returns a map of provider name to pane ID.
func (t *TmuxBackend) CreateAutoLayout(providers []string) (map[string]string, error) {
	result := make(map[string]string)
	if len(providers) == 0 {
		return result, nil
	}

	// Get current pane as the base
	output, err := t.runCmdOutput("display-message", "-p", "#{pane_id}")
	if err != nil {
		return nil, err
	}
	basePaneID := strings.TrimSpace(output)

	for i, provider := range providers {
		if i == 0 {
			// Use the current pane for the first provider
			result[provider] = basePaneID
			continue
		}

		// Split from the base pane
		newPaneID, err := t.SplitWindow(basePaneID, "")
		if err != nil {
			continue
		}
		result[provider] = newPaneID

		// Set title for identification
		t.SetPaneTitle(newPaneID, fmt.Sprintf("ccb-%s", provider))
	}

	return result, nil
}

// PaneLogManager manages pane log files with rotation and cleanup.
type PaneLogManager struct {
	LogDir   string
	MaxFiles int           // default 50
	MaxSize  int64         // default 5MB
	TTL      time.Duration // default 7 days
}

// NewPaneLogManager creates a new PaneLogManager.
func NewPaneLogManager(logDir string) *PaneLogManager {
	return &PaneLogManager{
		LogDir:   logDir,
		MaxFiles: 50,
		MaxSize:  5 * 1024 * 1024,
		TTL:      7 * 24 * time.Hour,
	}
}

// LogPathFor returns the log file path for a pane.
func (m *PaneLogManager) LogPathFor(paneID string) string {
	safe := strings.ReplaceAll(paneID, "%", "pct")
	safe = strings.ReplaceAll(safe, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	return filepath.Join(m.LogDir, fmt.Sprintf("pane-%s.log", safe))
}

// Cleanup removes old and oversized log files.
func (m *PaneLogManager) Cleanup() error {
	os.MkdirAll(m.LogDir, 0755)

	entries, err := os.ReadDir(m.LogDir)
	if err != nil {
		return err
	}

	now := time.Now()
	removed := 0

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		path := filepath.Join(m.LogDir, e.Name())

		// Remove files older than TTL
		if now.Sub(info.ModTime()) > m.TTL {
			os.Remove(path)
			removed++
			continue
		}

		// Truncate oversized files
		if info.Size() > m.MaxSize {
			truncateLogFile(path, m.MaxSize)
		}
	}

	// If still too many files, remove oldest
	entries, _ = os.ReadDir(m.LogDir)
	if len(entries) > m.MaxFiles {
		type fileAge struct {
			path    string
			modTime time.Time
		}
		var files []fileAge
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
				continue
			}
			info, _ := e.Info()
			if info == nil {
				continue
			}
			files = append(files, fileAge{
				path:    filepath.Join(m.LogDir, e.Name()),
				modTime: info.ModTime(),
			})
		}
		// Sort oldest first
		for i := 0; i < len(files); i++ {
			for j := i + 1; j < len(files); j++ {
				if files[j].modTime.Before(files[i].modTime) {
					files[i], files[j] = files[j], files[i]
				}
			}
		}
		// Remove excess
		excess := len(files) - m.MaxFiles
		for i := 0; i < excess && i < len(files); i++ {
			os.Remove(files[i].path)
		}
	}

	return nil
}

// truncateLogFile keeps only the last maxSize bytes of a file.
func truncateLogFile(path string, maxSize int64) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.Size() <= maxSize {
		return
	}

	f.Seek(-maxSize, 2)
	tail := make([]byte, maxSize)
	n, err := f.Read(tail)
	if err != nil {
		return
	}
	f.Close()

	os.WriteFile(path, tail[:n], 0644)
}

// subprocessKwargs returns common exec.Cmd settings to suppress console windows on Windows.
func subprocessKwargs() *exec.Cmd {
	cmd := &exec.Cmd{}
	setSysProcAttr(cmd)
	return cmd
}

// runTmux runs a tmux command (package-level helper for backward compat).
func runTmux(args ...string) error {
	cmd := exec.Command("tmux", args...)
	setSysProcAttr(cmd)
	return cmd.Run()
}

// runTmuxOutput runs a tmux command and returns its stdout.
func runTmuxOutput(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	setSysProcAttr(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
