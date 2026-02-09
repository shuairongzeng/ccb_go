package terminal

import (
	"fmt"
	"time"
)

// PaneInfo holds information about a terminal pane.
type PaneInfo struct {
	ID      string
	Title   string
	Command string
	Active  bool
	Width   int
	Height  int
}

// Backend defines the interface for terminal backends (tmux, WezTerm, PowerShell).
type Backend interface {
	// Name returns the backend name.
	Name() string

	// SendKeys sends text to a terminal pane.
	SendKeys(paneID string, text string) error

	// CapturePane captures the current content of a terminal pane.
	CapturePane(paneID string) (string, error)

	// SplitWindow creates a new pane by splitting an existing one.
	// Returns the new pane ID.
	SplitWindow(target string, cmd string) (string, error)

	// ListPanes returns all available panes.
	ListPanes() ([]PaneInfo, error)

	// KillPane kills a terminal pane.
	KillPane(paneID string) error

	// HasSession checks if a session/pane exists.
	HasSession(sessionID string) bool

	// IsAlive checks if a specific pane is still alive and responsive.
	IsAlive(paneID string) bool

	// SetPaneTitle sets the title of a pane.
	SetPaneTitle(paneID string, title string) error

	// GetPaneTitle returns the title of a pane.
	GetPaneTitle(paneID string) (string, error)

	// WaitReady waits for a pane to become ready (responsive) within the timeout.
	WaitReady(paneID string, timeout time.Duration) error
}

// ErrBackendNotAvailable is returned when a terminal backend is not available.
type ErrBackendNotAvailable struct {
	Backend string
	Reason  string
}

func (e *ErrBackendNotAvailable) Error() string {
	return fmt.Sprintf("backend %s not available: %s", e.Backend, e.Reason)
}

// ErrPaneNotFound is returned when a pane cannot be found.
type ErrPaneNotFound struct {
	PaneID  string
	Backend string
}

func (e *ErrPaneNotFound) Error() string {
	return fmt.Sprintf("pane %s not found in %s", e.PaneID, e.Backend)
}

// ErrWaitTimeout is returned when WaitReady times out.
type ErrWaitTimeout struct {
	PaneID  string
	Timeout time.Duration
}

func (e *ErrWaitTimeout) Error() string {
	return fmt.Sprintf("pane %s not ready after %v", e.PaneID, e.Timeout)
}

// DetectBackend detects the available terminal backend.
// Priority: CCB_BACKEND env > tmux > wezterm > powershell (Windows only)
func DetectBackend() (Backend, error) {
	// Try tmux first
	tmux := &TmuxBackend{}
	if tmux.IsAvailable() {
		return tmux, nil
	}

	// Try WezTerm
	wez := &WeztermBackend{}
	if wez.IsAvailable() {
		return wez, nil
	}

	// Try PowerShell on Windows
	ps := &PowerShellBackend{}
	if ps.IsAvailable() {
		return ps, nil
	}

	return nil, &ErrBackendNotAvailable{
		Backend: "any",
		Reason:  "no terminal backend available (tmux, wezterm, or powershell required)",
	}
}

// FindPaneByTitle searches all panes for one whose title contains the marker string.
func FindPaneByTitle(b Backend, titleMarker string) (string, error) {
	panes, err := b.ListPanes()
	if err != nil {
		return "", err
	}
	for _, p := range panes {
		if containsIgnoreCase(p.Title, titleMarker) {
			return p.ID, nil
		}
	}
	return "", &ErrPaneNotFound{PaneID: titleMarker, Backend: b.Name()}
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	sLower := make([]byte, len(s))
	subLower := make([]byte, len(substr))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		sLower[i] = c
	}
	for i := range substr {
		c := substr[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		subLower[i] = c
	}
	return bytesContains(sLower, subLower)
}

func bytesContains(s, sub []byte) bool {
	if len(sub) == 0 {
		return true
	}
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		match := true
		for j := range sub {
			if s[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
