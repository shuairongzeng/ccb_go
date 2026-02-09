package comm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/claude_code_bridge/internal/protocol"
	"github.com/anthropics/claude_code_bridge/internal/terminal"
)

// OpenCodeCommunicator handles communication with OpenCode.
// OpenCode stores data in ~/.local/share/opencode/storage/
// Layout: session/<projectID>/, message/<sessionID>/, part/<messageID>/
type OpenCodeCommunicator struct {
	BaseCommunicator
}

// NewOpenCodeCommunicator creates a new OpenCode communicator.
func NewOpenCodeCommunicator(backend terminal.Backend) *OpenCodeCommunicator {
	return &OpenCodeCommunicator{
		BaseCommunicator: BaseCommunicator{
			ProviderName: "opencode",
			Backend:      backend,
			PollCfg:      DefaultPollConfig(),
		},
	}
}

func (c *OpenCodeCommunicator) Name() string { return "opencode" }

func (c *OpenCodeCommunicator) SendPrompt(ctx context.Context, paneID string, message string) error {
	return c.SendViaTerminal(paneID, message)
}

func (c *OpenCodeCommunicator) ReadReply(ctx context.Context, opts ReadOpts) (string, error) {
	if opts.LogPath == "" {
		return "", nil
	}
	return readOpenCodeStorage(opts.LogPath, opts.ReqID)
}

func (c *OpenCodeCommunicator) WaitForReply(ctx context.Context, opts WaitOpts) (string, error) {
	cfg := c.PollCfg
	interval := cfg.InitialInterval
	if opts.PollMs > 0 {
		interval = time.Duration(opts.PollMs) * time.Millisecond
	}

	lastForceRead := time.Now()

	for {
		select {
		case <-ctx.Done():
			return "", &ErrTimeout{Provider: "opencode", ReqID: opts.ReqID}
		default:
		}

		reply, err := c.ReadReply(ctx, ReadOpts{
			LogPath: opts.LogPath,
			ReqID:   opts.ReqID,
		})
		if err == nil && reply != "" && protocol.IsDoneText(reply, opts.ReqID) {
			return protocol.StripDoneText(reply, opts.ReqID), nil
		}

		// Check pane alive periodically
		if opts.PaneID != "" && time.Since(lastForceRead) > cfg.ForceReadEvery {
			lastForceRead = time.Now()
			if !c.IsAlive(opts.PaneID) {
				return "", &ErrPaneDead{Provider: "opencode", PaneID: opts.PaneID}
			}
		}

		time.Sleep(interval)
		interval = adaptiveSleep(interval, cfg)
	}
}

func (c *OpenCodeCommunicator) CaptureState(ctx context.Context, opts ReadOpts) (*CaptureState, error) {
	state := &CaptureState{}
	if opts.LogPath == "" {
		return state, nil
	}

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

func (c *OpenCodeCommunicator) HealthCheck(ctx context.Context, paneID string) error {
	if !c.IsAlive(paneID) {
		return &ErrPaneDead{Provider: "opencode", PaneID: paneID}
	}
	return nil
}

// OpenCodeMessage represents a message from OpenCode storage.
type OpenCodeMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	SessionID string `json:"sessionID"`
	Error     string `json:"error"`
}

// readOpenCodeStorage reads the latest reply from OpenCode's storage directory.
func readOpenCodeStorage(storagePath string, reqID string) (string, error) {
	entries, err := os.ReadDir(storagePath)
	if err != nil {
		return "", err
	}

	type fileEntry struct {
		path    string
		modTime time.Time
	}
	var files []fileEntry

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Look for message files in session directories
		msgDir := filepath.Join(storagePath, e.Name())
		msgEntries, err := os.ReadDir(msgDir)
		if err != nil {
			continue
		}
		for _, me := range msgEntries {
			if me.IsDir() || !strings.HasSuffix(me.Name(), ".json") {
				continue
			}
			info, err := me.Info()
			if err != nil {
				continue
			}
			files = append(files, fileEntry{
				path:    filepath.Join(msgDir, me.Name()),
				modTime: info.ModTime(),
			})
		}
	}

	if len(files) == 0 {
		return "", nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	// Read the most recent files looking for our reply
	foundAnchor := false
	var replyParts []string

	// Scan in reverse chronological order, but we need forward order for anchor detection
	// So collect all recent messages and process in order
	var allMessages []OpenCodeMessage
	limit := 50
	if len(files) < limit {
		limit = len(files)
	}

	for _, f := range files[:limit] {
		data, err := os.ReadFile(f.path)
		if err != nil {
			continue
		}

		var msg OpenCodeMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		// Check for cancellation
		if msg.Error != "" && strings.Contains(msg.Error, "Aborted") {
			continue
		}

		allMessages = append(allMessages, msg)
	}

	// Reverse to get chronological order
	for i, j := 0, len(allMessages)-1; i < j; i, j = i+1, j-1 {
		allMessages[i], allMessages[j] = allMessages[j], allMessages[i]
	}

	for _, msg := range allMessages {
		if !foundAnchor {
			if strings.Contains(msg.Content, protocol.ReqIDPrefix+" "+reqID) {
				foundAnchor = true
			}
			continue
		}

		if msg.Role == "assistant" && msg.Content != "" {
			replyParts = append(replyParts, msg.Content)
		}
	}

	return strings.Join(replyParts, "\n"), nil
}

// DiscoverOpenCodeStorage finds the OpenCode storage directory.
func DiscoverOpenCodeStorage() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	storagePath := filepath.Join(home, ".local", "share", "opencode", "storage")
	if _, err := os.Stat(storagePath); err == nil {
		return storagePath, nil
	}

	// Windows fallback
	appData := os.Getenv("LOCALAPPDATA")
	if appData != "" {
		storagePath = filepath.Join(appData, "opencode", "storage")
		if _, err := os.Stat(storagePath); err == nil {
			return storagePath, nil
		}
	}

	return "", nil
}
