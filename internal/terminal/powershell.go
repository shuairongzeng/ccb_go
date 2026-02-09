package terminal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// PowerShellBackend implements the Backend interface using PowerShell on Windows.
type PowerShellBackend struct {
	windowRegistry *WindowRegistry
}

// Name returns "powershell".
func (p *PowerShellBackend) Name() string { return "powershell" }

// IsAvailable checks if PowerShell is available (Windows only).
func (p *PowerShellBackend) IsAvailable() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	env := strings.ToLower(strings.TrimSpace(os.Getenv("CCB_BACKEND_ENV")))
	if env != "" && env != "windows" {
		return false
	}
	_, err := exec.LookPath("powershell.exe")
	if err != nil {
		_, err = exec.LookPath("pwsh.exe")
	}
	return err == nil
}

// getPSExe returns the PowerShell executable path.
func (p *PowerShellBackend) getPSExe() string {
	if path, err := exec.LookPath("pwsh.exe"); err == nil {
		return path
	}
	if path, err := exec.LookPath("powershell.exe"); err == nil {
		return path
	}
	return "powershell.exe"
}

// runPS runs a PowerShell command and returns stdout.
func (p *PowerShellBackend) runPS(script string) (string, error) {
	cmd := exec.Command(p.getPSExe(), "-NoProfile", "-NonInteractive", "-Command", script)
	setSysProcAttr(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("powershell error: %w", err)
	}
	return string(out), nil
}

// SendKeys sends text to a window using PowerShell SendKeys.
func (p *PowerShellBackend) SendKeys(paneID string, text string) error {
	// Use file-based messaging for reliability
	msgDir := filepath.Join(os.TempDir(), "ccb", "messages")
	os.MkdirAll(msgDir, 0755)

	msgFile := filepath.Join(msgDir, paneID+".msg")
	if err := os.WriteFile(msgFile, []byte(text), 0644); err != nil {
		return err
	}

	// Fallback: use SendKeys via COM automation
	escaped := strings.ReplaceAll(text, "'", "''")
	escaped = strings.ReplaceAll(escaped, "`", "``")
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$wshell = New-Object -ComObject WScript.Shell
$wshell.AppActivate('%s') | Out-Null
Start-Sleep -Milliseconds 200
[System.Windows.Forms.SendKeys]::SendWait('%s{ENTER}')
`, paneID, escaped)

	_, err := p.runPS(script)
	return err
}

// CapturePane captures content from a PowerShell window (limited support).
func (p *PowerShellBackend) CapturePane(paneID string) (string, error) {
	// Check for file-based response first
	msgDir := filepath.Join(os.TempDir(), "ccb", "messages")
	respFile := filepath.Join(msgDir, paneID+".resp")
	if data, err := os.ReadFile(respFile); err == nil {
		return string(data), nil
	}
	return "", fmt.Errorf("capture not supported for PowerShell backend pane %s", paneID)
}

// SplitWindow starts a new PowerShell window with a command.
func (p *PowerShellBackend) SplitWindow(target string, cmdStr string) (string, error) {
	paneID := fmt.Sprintf("ccb-%d", os.Getpid())
	script := fmt.Sprintf(`Start-Process %s -ArgumentList '-NoExit','-Command','%s'`, p.getPSExe(), strings.ReplaceAll(cmdStr, "'", "''"))
	_, err := p.runPS(script)
	if err != nil {
		return "", err
	}
	return paneID, nil
}

// CreateWindow creates a new window for a provider with the given command.
func (p *PowerShellBackend) CreateWindow(provider string, cmd string, cwd string) (string, error) {
	paneID := fmt.Sprintf("ccb-%s-%d", provider, os.Getpid())

	script := fmt.Sprintf(`
$proc = Start-Process %s -ArgumentList '-NoExit','-Command','cd "%s"; %s' -PassThru
$proc.Id
`, p.getPSExe(), strings.ReplaceAll(cwd, "'", "''"), strings.ReplaceAll(cmd, "'", "''"))

	out, err := p.runPS(script)
	if err != nil {
		return "", err
	}

	pid := strings.TrimSpace(out)
	if pid != "" {
		paneID = pid
	}

	// Register the window
	if p.windowRegistry != nil {
		p.windowRegistry.Set(provider, paneID, WindowInfo{
			PaneID:   paneID,
			Provider: provider,
			CWD:      cwd,
			Created:  time.Now(),
		})
	}

	return paneID, nil
}

// ListPanes returns running CCB-related processes as panes.
func (p *PowerShellBackend) ListPanes() ([]PaneInfo, error) {
	script := `Get-Process | Where-Object { $_.MainWindowTitle -like '*ccb*' -or $_.ProcessName -like '*codex*' -or $_.ProcessName -like '*gemini*' -or $_.ProcessName -like '*claude*' } | Select-Object Id,ProcessName,MainWindowTitle | ConvertTo-Json`
	out, err := p.runPS(script)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(out) == "" {
		return nil, nil
	}

	var rawPanes []struct {
		ID    int    `json:"Id"`
		Name  string `json:"ProcessName"`
		Title string `json:"MainWindowTitle"`
	}

	// Handle both single object and array
	if err := json.Unmarshal([]byte(out), &rawPanes); err != nil {
		var single struct {
			ID    int    `json:"Id"`
			Name  string `json:"ProcessName"`
			Title string `json:"MainWindowTitle"`
		}
		if err := json.Unmarshal([]byte(out), &single); err == nil {
			rawPanes = append(rawPanes, single)
		}
	}

	panes := make([]PaneInfo, 0, len(rawPanes))
	for _, rp := range rawPanes {
		panes = append(panes, PaneInfo{
			ID:      fmt.Sprintf("%d", rp.ID),
			Title:   rp.Title,
			Command: rp.Name,
		})
	}
	return panes, nil
}

// KillPane kills a process by PID.
func (p *PowerShellBackend) KillPane(paneID string) error {
	script := fmt.Sprintf(`Stop-Process -Id %s -Force -ErrorAction SilentlyContinue`, paneID)
	_, err := p.runPS(script)
	return err
}

// HasSession checks if a process with the given PID exists.
func (p *PowerShellBackend) HasSession(sessionID string) bool {
	return p.IsAlive(sessionID)
}

// IsAlive checks if a process with the given PID is still running.
func (p *PowerShellBackend) IsAlive(paneID string) bool {
	script := fmt.Sprintf(`Get-Process -Id %s -ErrorAction SilentlyContinue | Select-Object -First 1`, paneID)
	out, err := p.runPS(script)
	return err == nil && strings.TrimSpace(out) != ""
}

// SetPaneTitle sets the window title of a process.
func (p *PowerShellBackend) SetPaneTitle(paneID string, title string) error {
	// Send title escape sequence via file message
	msgDir := filepath.Join(os.TempDir(), "ccb", "messages")
	os.MkdirAll(msgDir, 0755)
	titleFile := filepath.Join(msgDir, paneID+".title")
	return os.WriteFile(titleFile, []byte(title), 0644)
}

// GetPaneTitle returns the window title of a process.
func (p *PowerShellBackend) GetPaneTitle(paneID string) (string, error) {
	script := fmt.Sprintf(`(Get-Process -Id %s -ErrorAction SilentlyContinue).MainWindowTitle`, paneID)
	out, err := p.runPS(script)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// WaitReady waits for a process to become ready.
func (p *PowerShellBackend) WaitReady(paneID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 200 * time.Millisecond

	for time.Now().Before(deadline) {
		if p.IsAlive(paneID) {
			return nil
		}
		time.Sleep(interval)
		if interval < time.Second {
			interval = time.Duration(float64(interval) * 1.5)
		}
	}
	return &ErrWaitTimeout{PaneID: paneID, Timeout: timeout}
}

// ActivateWindow brings a window to the foreground.
func (p *PowerShellBackend) ActivateWindow(paneID string) error {
	script := fmt.Sprintf(`
$wshell = New-Object -ComObject WScript.Shell
$wshell.AppActivate(%s) | Out-Null
`, paneID)
	_, err := p.runPS(script)
	return err
}

// CleanupDead removes dead processes from the window registry.
func (p *PowerShellBackend) CleanupDead() int {
	if p.windowRegistry == nil {
		return 0
	}

	removed := 0
	p.windowRegistry.mu.Lock()
	defer p.windowRegistry.mu.Unlock()

	for key, info := range p.windowRegistry.data {
		if !p.IsAlive(info.PaneID) {
			delete(p.windowRegistry.data, key)
			removed++
		}
	}

	if removed > 0 {
		p.windowRegistry.saveLocked()
	}

	return removed
}

// WindowInfo holds information about a tracked window.
type WindowInfo struct {
	PaneID   string    `json:"pane_id"`
	Provider string    `json:"provider"`
	CWD      string    `json:"cwd"`
	Created  time.Time `json:"created"`
}

// WindowRegistry tracks PowerShell windows for CCB providers.
type WindowRegistry struct {
	mu       sync.RWMutex
	filePath string
	data     map[string]WindowInfo // key: provider → WindowInfo
}

// NewWindowRegistry creates a new WindowRegistry.
func NewWindowRegistry(filePath string) *WindowRegistry {
	r := &WindowRegistry{
		filePath: filePath,
		data:     make(map[string]WindowInfo),
	}
	r.load()
	return r
}

// Get returns the window info for a provider.
func (r *WindowRegistry) Get(provider string) (WindowInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.data[provider]
	return info, ok
}

// Set registers a window for a provider.
func (r *WindowRegistry) Set(provider string, paneID string, info WindowInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[provider] = info
	r.saveLocked()
}

// Remove removes a window registration.
func (r *WindowRegistry) Remove(provider string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, provider)
	r.saveLocked()
}

// All returns all registered windows.
func (r *WindowRegistry) All() map[string]WindowInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]WindowInfo, len(r.data))
	for k, v := range r.data {
		result[k] = v
	}
	return result
}

// load reads the registry from disk.
func (r *WindowRegistry) load() {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return
	}
	json.Unmarshal(data, &r.data)
}

// saveLocked writes the registry to disk (caller must hold lock).
func (r *WindowRegistry) saveLocked() {
	dir := filepath.Dir(r.filePath)
	os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return
	}
	tmpFile := r.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return
	}
	os.Rename(tmpFile, r.filePath)
}
