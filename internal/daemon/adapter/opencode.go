package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/claude_code_bridge/internal/comm"
	"github.com/anthropics/claude_code_bridge/internal/protocol"
	"github.com/anthropics/claude_code_bridge/internal/session"
	"github.com/anthropics/claude_code_bridge/internal/terminal"
)

// OpenCodeAdapter implements the Adapter interface for OpenCode.
type OpenCodeAdapter struct {
	BaseAdapter
	Backend   terminal.Backend
	Comm      *comm.OpenCodeCommunicator
	lastReply string
}

func NewOpenCodeAdapter(backend terminal.Backend) *OpenCodeAdapter {
	return &OpenCodeAdapter{
		BaseAdapter: BaseAdapter{ProviderName: "opencode"},
		Backend:     backend,
		Comm:        comm.NewOpenCodeCommunicator(backend),
	}
}

func (a *OpenCodeAdapter) Send(ctx context.Context, req *ProviderRequest) (*ProviderResult, error) {
	startTime := time.Now()

	sess, err := session.LoadOpenCodeSession(req.WorkDir)
	if err != nil || sess == nil {
		return &ProviderResult{ExitCode: 1, ReqID: req.ReqID, Error: "opencode session not found"}, nil
	}

	reqID := req.ReqID
	if reqID == "" {
		reqID = protocol.MakeReqID()
	}

	wrapped := protocol.OpenCodeProto.WrapPrompt(req.Message, reqID)
	if err := a.Comm.SendPrompt(ctx, sess.PaneID, wrapped); err != nil {
		return &ProviderResult{ExitCode: 1, ReqID: reqID, Error: fmt.Sprintf("send failed: %v", err)}, nil
	}

	timeout := time.Duration(req.TimeoutS) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	reply, err := a.Comm.WaitForReply(ctx, comm.WaitOpts{
		LogPath: sess.LogPath, ReqID: reqID, PaneID: sess.PaneID, PollMs: 20,
	})

	result := &ProviderResult{ReqID: reqID, SessionKey: sess.ProjectID, LogPath: sess.LogPath}
	if err != nil {
		result.ExitCode = 2
		result.Error = err.Error()
		state, _ := a.Comm.CaptureState(ctx, comm.ReadOpts{LogPath: sess.LogPath, ReqID: reqID})
		if state != nil {
			result.AnchorSeen = state.AnchorSeen
			result.AnchorMs = state.AnchorMs
		}
		return result, nil
	}

	result.ExitCode = 0
	result.Reply = reply
	result.DoneSeen = true
	result.DoneMs = time.Since(startTime).Milliseconds()
	a.lastReply = reply
	return result, nil
}

func (a *OpenCodeAdapter) Ping(ctx context.Context, sessionID string) error {
	if a.Backend == nil {
		return fmt.Errorf("no terminal backend")
	}
	if sessionID != "" && !a.Backend.IsAlive(sessionID) {
		return fmt.Errorf("opencode pane %s not found", sessionID)
	}
	return nil
}

func (a *OpenCodeAdapter) Pend(ctx context.Context, sessionID string) (string, error) {
	if a.lastReply != "" {
		return a.lastReply, nil
	}
	return "", nil
}

func (a *OpenCodeAdapter) EnsurePane(ctx context.Context, workDir string) (string, error) {
	sess, err := session.LoadOpenCodeSession(workDir)
	if err != nil {
		return "", err
	}
	if sess != nil && sess.PaneID != "" {
		if a.Backend != nil && a.Backend.IsAlive(sess.PaneID) {
			return sess.PaneID, nil
		}
	}
	return "", fmt.Errorf("no opencode session configured")
}
