package comm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/claude_code_bridge/internal/protocol"
	"github.com/anthropics/claude_code_bridge/internal/terminal"
)

// GeminiCommunicator handles communication with Gemini CLI.
// Gemini stores chats in JSON files under ~/.gemini/tmp/<hash>/chats/
type GeminiCommunicator struct {
	BaseCommunicator
}

// NewGeminiCommunicator creates a new Gemini communicator.
func NewGeminiCommunicator(backend terminal.Backend) *GeminiCommunicator {
	return &GeminiCommunicator{
		BaseCommunicator: BaseCommunicator{
			ProviderName: "gemini",
			Backend:      backend,
			PollCfg:      DefaultPollConfig(),
		},
	}
}

func (c *GeminiCommunicator) Name() string { return "gemini" }

func (c *GeminiCommunicator) SendPrompt(ctx context.Context, paneID string, message string) error {
	return c.SendViaTerminal(paneID, message)
}

func (c *GeminiCommunicator) ReadReply(ctx context.Context, opts ReadOpts) (string, error) {
	if opts.LogPath == "" {
		return "", nil
	}
	return readGeminiChat(opts.LogPath, opts.ReqID)
}

func (c *GeminiCommunicator) WaitForReply(ctx context.Context, opts WaitOpts) (string, error) {
	cfg := c.PollCfg
	interval := cfg.InitialInterval
	if opts.PollMs > 0 {
		interval = time.Duration(opts.PollMs) * time.Millisecond
	}

	lastForceRead := time.Now()

	for {
		select {
		case <-ctx.Done():
			return "", &ErrTimeout{Provider: "gemini", ReqID: opts.ReqID}
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
				return "", &ErrPaneDead{Provider: "gemini", PaneID: opts.PaneID}
			}
		}

		time.Sleep(interval)
		interval = adaptiveSleep(interval, cfg)
	}
}

func (c *GeminiCommunicator) CaptureState(ctx context.Context, opts ReadOpts) (*CaptureState, error) {
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

func (c *GeminiCommunicator) HealthCheck(ctx context.Context, paneID string) error {
	if !c.IsAlive(paneID) {
		return &ErrPaneDead{Provider: "gemini", PaneID: paneID}
	}
	return nil
}

// GeminiMessage represents a message in a Gemini chat session.
type GeminiMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Parts   []string `json:"-"` // extracted from parts array
}

// readGeminiChat reads the latest chat from Gemini's session files.
func readGeminiChat(chatsDir string, reqID string) (string, error) {
	sessionFile, err := findLatestGeminiSession(chatsDir)
	if err != nil || sessionFile == "" {
		return "", err
	}

	messages, err := parseGeminiMessages(sessionFile)
	if err != nil {
		return "", nil // retry on parse error (in-place writes)
	}

	// Find the last model response after our request
	foundAnchor := false
	var replyParts []string

	for _, msg := range messages {
		if !foundAnchor {
			if strings.Contains(msg.Content, protocol.ReqIDPrefix+" "+reqID) {
				foundAnchor = true
			}
			continue
		}
		if msg.Role == "model" || msg.Role == "assistant" {
			replyParts = append(replyParts, msg.Content)
		}
	}

	return strings.Join(replyParts, "\n"), nil
}

// findLatestGeminiSession finds the most recently modified session JSON file.
func findLatestGeminiSession(chatsDir string) (string, error) {
	entries, err := os.ReadDir(chatsDir)
	if err != nil {
		return "", err
	}

	type fileEntry struct {
		path    string
		modTime time.Time
	}
	var files []fileEntry

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileEntry{
			path:    filepath.Join(chatsDir, e.Name()),
			modTime: info.ModTime(),
		})
	}

	if len(files) == 0 {
		return "", nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	return files[0].path, nil
}

// parseGeminiMessages parses a Gemini chat JSON file into messages.
func parseGeminiMessages(sessionFile string) ([]GeminiMessage, error) {
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return nil, err
	}

	// Gemini uses two possible formats:
	// 1. { "messages": [ { "role": "...", "content": "...", "parts": [...] } ] }
	// 2. Array of messages directly
	var chat struct {
		Messages []json.RawMessage `json:"messages"`
	}

	var rawMessages []json.RawMessage

	if err := json.Unmarshal(data, &chat); err == nil && len(chat.Messages) > 0 {
		rawMessages = chat.Messages
	} else {
		// Try as direct array
		if err := json.Unmarshal(data, &rawMessages); err != nil {
			return nil, err
		}
	}

	var messages []GeminiMessage
	for _, raw := range rawMessages {
		var msg struct {
			Role    string `json:"role"`
			Content string `json:"content"`
			Parts   []struct {
				Text string `json:"text"`
			} `json:"parts"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		content := msg.Content
		if content == "" && len(msg.Parts) > 0 {
			var parts []string
			for _, p := range msg.Parts {
				parts = append(parts, p.Text)
			}
			content = strings.Join(parts, "\n")
		}

		messages = append(messages, GeminiMessage{
			Role:    msg.Role,
			Content: content,
		})
	}

	return messages, nil
}

// GeminiProjectHash computes the Gemini project hash for a work directory.
// Gemini uses SHA256 of the normalized path.
func GeminiProjectHash(workDir string) string {
	norm := strings.ReplaceAll(filepath.Clean(workDir), "\\", "/")
	norm = strings.ToLower(norm)
	hash := sha256.Sum256([]byte(norm))
	return fmt.Sprintf("%x", hash)
}

// DiscoverGeminiChatsDir finds the chats directory for a work directory.
func DiscoverGeminiChatsDir(workDir string) (string, error) {
	root := strings.TrimSpace(os.Getenv("GEMINI_ROOT"))
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".gemini", "tmp")
	}

	// Try project hash first
	projHash := GeminiProjectHash(workDir)
	chatsDir := filepath.Join(root, projHash, "chats")
	if info, err := os.Stat(chatsDir); err == nil && info.IsDir() {
		return chatsDir, nil
	}

	// Fallback: scan all directories for the most recent chats/
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}

	type dirEntry struct {
		path    string
		modTime time.Time
	}
	var dirs []dirEntry

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		cd := filepath.Join(root, e.Name(), "chats")
		info, err := os.Stat(cd)
		if err != nil || !info.IsDir() {
			continue
		}
		dirs = append(dirs, dirEntry{path: cd, modTime: info.ModTime()})
	}

	if len(dirs) == 0 {
		return "", nil
	}

	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].modTime.After(dirs[j].modTime)
	})

	return dirs[0].path, nil
}
