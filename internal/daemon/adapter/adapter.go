package adapter

import (
	"context"
)

// ProviderRequest represents a request to a provider adapter.
type ProviderRequest struct {
	ClientID   string  `json:"client_id"`
	WorkDir    string  `json:"work_dir"`
	Message    string  `json:"message"`
	ReqID      string  `json:"req_id"`
	TimeoutS   float64 `json:"timeout_s"`
	Quiet      bool    `json:"quiet"`
	OutputPath string  `json:"output_path,omitempty"`
	Caller     string  `json:"caller,omitempty"`
}

// ProviderResult represents a result from a provider adapter.
type ProviderResult struct {
	ExitCode     int    `json:"exit_code"`
	Reply        string `json:"reply"`
	ReqID        string `json:"req_id"`
	SessionKey   string `json:"session_key"`
	LogPath      string `json:"log_path,omitempty"`
	AnchorSeen   bool   `json:"anchor_seen"`
	DoneSeen     bool   `json:"done_seen"`
	FallbackScan bool   `json:"fallback_scan"`
	AnchorMs     int64  `json:"anchor_ms,omitempty"`
	DoneMs       int64  `json:"done_ms,omitempty"`
	Error        string `json:"error,omitempty"`
}

// QueuedTask wraps a request with a result channel.
type QueuedTask struct {
	Request  *ProviderRequest
	ResultCh chan *ProviderResult
	Ctx      context.Context
	Cancel   context.CancelFunc
}

// Adapter defines the interface for provider adapters.
type Adapter interface {
	// Name returns the provider name.
	Name() string

	// Send sends a request and waits for a reply.
	Send(ctx context.Context, req *ProviderRequest) (*ProviderResult, error)

	// Ping checks if the provider is reachable.
	Ping(ctx context.Context, sessionID string) error

	// Pend returns the latest reply from the provider.
	Pend(ctx context.Context, sessionID string) (string, error)

	// EnsurePane ensures a terminal pane exists for the provider.
	EnsurePane(ctx context.Context, workDir string) (string, error)

	// OnStart is called when the adapter is registered with the daemon.
	OnStart() error

	// OnStop is called when the daemon shuts down.
	OnStop() error
}

// BaseAdapter provides shared functionality for all adapters.
type BaseAdapter struct {
	ProviderName string
}

func (b *BaseAdapter) Name() string {
	return b.ProviderName
}

func (b *BaseAdapter) OnStart() error {
	return nil
}

func (b *BaseAdapter) OnStop() error {
	return nil
}
