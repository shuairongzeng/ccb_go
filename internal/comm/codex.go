package comm

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/claude_code_bridge/internal/protocol"
	"github.com/anthropics/claude_code_bridge/internal/terminal"
)

// CodexCommunicator handles communication with Codex CLI.
// Codex stores session logs in ~/.codex/sessions/<id>/output.log
type CodexCommunicator struct {
	BaseCommunicator
	logReader *LogReader
	revReader *ReverseReader
}

// NewCodexCommunicator creates a new Codex communicator.
func NewCodexCommunicator(backend terminal.Backend) *CodexCommunicator {
	return &CodexCommunicator{
		BaseCommunicator: BaseCommunicator{
			ProviderName: "codex",
			Backend:      backend,
			PollCfg:      DefaultPollConfig(),
		},
	}
}

func (c *CodexCommunicator) Name() string { return "codex" }

func (c *CodexCommunicator) SendPrompt(ctx context.Context, paneID string, message string) error {
	return c.SendViaTerminal(paneID, message)
}

func (c *CodexCommunicator) ReadReply(ctx context.Context, opts ReadOpts) (string, error) {
	if opts.LogPath == "" {
		return "", nil
	}

	// Use reverse reader for efficient tail scanning
	rr := NewReverseReader(opts.LogPath)
	lines, err := rr.ReadLastLines(500)
	if err != nil {
		return "", err
	}

	if len(lines) == 0 {
		return "", nil
	}

	// Find the anchor line (CCB_REQ_ID: <reqID>) searching backward
	anchorPrefix := protocol.ReqIDPrefix + " " + opts.ReqID
	anchorIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], anchorPrefix) {
			anchorIdx = i
			break
		}
	}

	if anchorIdx < 0 {
		return "", nil
	}

	// Collect everything after the anchor
	var replyLines []string
	for i := anchorIdx + 1; i < len(lines); i++ {
		replyLines = append(replyLines, lines[i])
	}

	return strings.Join(replyLines, "\n"), nil
}

func (c *CodexCommunicator) WaitForReply(ctx context.Context, opts WaitOpts) (string, error) {
	cfg := c.PollCfg
	interval := cfg.InitialInterval
	if opts.PollMs > 0 {
		interval = time.Duration(opts.PollMs) * time.Millisecond
	}

	lastForceRead := time.Now()
	startTime := time.Now()
	var anchorMs int64

	for {
		select {
		case <-ctx.Done():
			return "", &ErrTimeout{Provider: "codex", ReqID: opts.ReqID}
		default:
		}

		reply, err := c.ReadReply(ctx, ReadOpts{
			LogPath: opts.LogPath,
			ReqID:   opts.ReqID,
		})
		if err == nil && reply != "" {
			if anchorMs == 0 {
				anchorMs = time.Since(startTime).Milliseconds()
			}
			if protocol.IsDoneText(reply, opts.ReqID) {
				return protocol.StripDoneText(reply, opts.ReqID), nil
			}
		}

		// Check pane alive periodically
		if opts.PaneID != "" && time.Since(lastForceRead) > cfg.ForceReadEvery {
			lastForceRead = time.Now()
			if !c.IsAlive(opts.PaneID) {
				return "", &ErrPaneDead{Provider: "codex", PaneID: opts.PaneID}
			}
		}

		time.Sleep(interval)
		interval = adaptiveSleep(interval, cfg)
	}
}

func (c *CodexCommunicator) CaptureState(ctx context.Context, opts ReadOpts) (*CaptureState, error) {
	state := &CaptureState{}

	if opts.LogPath == "" {
		return state, nil
	}

	info, err := os.Stat(opts.LogPath)
	if err != nil {
		return state, err
	}
	state.LastOffset = info.Size()

	reply, err := c.ReadReply(ctx, opts)
	if err != nil {
		return state, err
	}

	if reply != "" {
		state.AnchorSeen = true
		state.ReplyLines = strings.Split(reply, "\n")
		if protocol.IsDoneText(reply, opts.ReqID) {
			state.DoneSeen = true
		}
	}

	return state, nil
}

func (c *CodexCommunicator) HealthCheck(ctx context.Context, paneID string) error {
	if !c.IsAlive(paneID) {
		return &ErrPaneDead{Provider: "codex", PaneID: paneID}
	}
	return nil
}

// DiscoverSession finds the most recent Codex session directory for a work directory.
func DiscoverCodexSession(workDir string) (string, error) {
	root := strings.TrimSpace(os.Getenv("CODEX_SESSION_ROOT"))
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".codex", "sessions")
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}

	type sessionEntry struct {
		path    string
		modTime time.Time
	}
	var sessions []sessionEntry

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		logFile := filepath.Join(root, e.Name(), "output.log")
		info, err := os.Stat(logFile)
		if err != nil {
			continue
		}
		sessions = append(sessions, sessionEntry{
			path:    filepath.Join(root, e.Name()),
			modTime: info.ModTime(),
		})
	}

	if len(sessions) == 0 {
		return "", nil
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].modTime.After(sessions[j].modTime)
	})

	return sessions[0].path, nil
}
