package comm

import (
	"context"
	"time"

	"github.com/anthropics/claude_code_bridge/internal/terminal"
)

// Communicator defines the interface for provider communication.
type Communicator interface {
	// Name returns the provider name.
	Name() string

	// SendPrompt sends a prompt to the provider via the terminal backend.
	SendPrompt(ctx context.Context, paneID string, message string) error

	// ReadReply reads the latest reply from the provider.
	ReadReply(ctx context.Context, opts ReadOpts) (string, error)

	// WaitForReply waits for a complete reply (with CCB_DONE marker).
	WaitForReply(ctx context.Context, opts WaitOpts) (string, error)

	// CaptureState captures the current communication state for diagnostics.
	CaptureState(ctx context.Context, opts ReadOpts) (*CaptureState, error)

	// HealthCheck verifies the provider pane is alive and responsive.
	HealthCheck(ctx context.Context, paneID string) error
}

// ReadOpts holds options for reading a reply.
type ReadOpts struct {
	SessionID string
	LogPath   string
	ReqID     string
	MaxLines  int
}

// WaitOpts holds options for waiting for a reply.
type WaitOpts struct {
	SessionID string
	LogPath   string
	ReqID     string
	PaneID    string
	PollMs    int
}

// CaptureState holds the state of an in-progress reply capture.
type CaptureState struct {
	LastOffset   int64    // file offset at time of capture
	AnchorSeen   bool     // whether the CCB_REQ_ID anchor was found
	AnchorMs     int64    // milliseconds from send to anchor detection
	DoneSeen     bool     // whether the CCB_DONE marker was found
	DoneMs       int64    // milliseconds from send to done detection
	ReplyLines   []string // collected reply lines so far
	FallbackScan bool     // whether fallback scanning was used
}

// PollConfig controls the adaptive polling strategy for WaitForReply.
type PollConfig struct {
	InitialInterval time.Duration // starting poll interval (default 20ms)
	MaxInterval     time.Duration // maximum poll interval (default 500ms)
	ForceReadEvery  time.Duration // force a read even if no mtime change (default 2s)
	BackoffFactor   float64       // multiplier per poll cycle (default 1.5)
}

// DefaultPollConfig returns the default polling configuration.
func DefaultPollConfig() PollConfig {
	return PollConfig{
		InitialInterval: 20 * time.Millisecond,
		MaxInterval:     500 * time.Millisecond,
		ForceReadEvery:  2 * time.Second,
		BackoffFactor:   1.5,
	}
}

// BaseCommunicator provides shared functionality for all communicators.
type BaseCommunicator struct {
	ProviderName string
	Backend      terminal.Backend
	PollCfg      PollConfig
}

// SendViaTerminal sends text to a terminal pane.
func (b *BaseCommunicator) SendViaTerminal(paneID string, text string) error {
	if b.Backend == nil {
		return &ErrNoBackend{Provider: b.ProviderName}
	}
	return b.Backend.SendKeys(paneID, text)
}

// IsAlive checks if a pane is still alive via the backend.
func (b *BaseCommunicator) IsAlive(paneID string) bool {
	if b.Backend == nil {
		return false
	}
	return b.Backend.HasSession(paneID)
}

// adaptiveSleep computes the next poll interval using exponential backoff.
func adaptiveSleep(current time.Duration, cfg PollConfig) time.Duration {
	next := time.Duration(float64(current) * cfg.BackoffFactor)
	if next > cfg.MaxInterval {
		next = cfg.MaxInterval
	}
	return next
}

// ErrNoBackend is returned when no terminal backend is available.
type ErrNoBackend struct {
	Provider string
}

func (e *ErrNoBackend) Error() string {
	return "no terminal backend available for " + e.Provider
}

// ErrTimeout is returned when waiting for a reply times out.
type ErrTimeout struct {
	Provider string
	ReqID    string
}

func (e *ErrTimeout) Error() string {
	return "timeout waiting for reply from " + e.Provider + " (req_id: " + e.ReqID + ")"
}

// ErrPaneDead is returned when the provider pane is no longer alive.
type ErrPaneDead struct {
	Provider string
	PaneID   string
}

func (e *ErrPaneDead) Error() string {
	return "pane " + e.PaneID + " for " + e.Provider + " is no longer alive"
}

// ErrNoSession is returned when no session could be found.
type ErrNoSession struct {
	Provider string
}

func (e *ErrNoSession) Error() string {
	return "no session found for " + e.Provider
}
