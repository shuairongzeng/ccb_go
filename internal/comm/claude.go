package comm

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/claude_code_bridge/internal/protocol"
	"github.com/anthropics/claude_code_bridge/internal/terminal"
)

// ClaudeCommunicator handles communication with Claude Code.
// Claude stores session logs as JSONL in ~/.claude/projects/<key>/<session>.jsonl
type ClaudeCommunicator struct {
	BaseCommunicator
}

// NewClaudeCommunicator creates a new Claude communicator.
func NewClaudeCommunicator(backend terminal.Backend) *ClaudeCommunicator {
	return &ClaudeCommunicator{
		BaseCommunicator: BaseCommunicator{
			ProviderName: "claude",
			Backend:      backend,
			PollCfg:      DefaultPollConfig(),
		},
	}
}

func (c *ClaudeCommunicator) Name() string { return "claude" }

func (c *ClaudeCommunicator) SendPrompt(ctx context.Context, paneID string, message string) error {
	return c.SendViaTerminal(paneID, message)
}

func (c *ClaudeCommunicator) ReadReply(ctx context.Context, opts ReadOpts) (string, error) {
	if opts.LogPath == "" {
		return "", nil
	}

	entries, err := readClaudeLog(opts.LogPath, opts.ReqID)
	if err != nil {
		return "", err
	}

	if len(entries) == 0 {
		return "", nil
	}

	// Find anchor and extract reply
	foundAnchor := false
	var replyParts []string

	for _, entry := range entries {
		entryType, _ := entry["type"].(string)

		// Check for anchor in human messages
		if entryType == "human" || entryType == "user" {
			content := extractClaudeEntryContent(entry)
			if strings.Contains(content, protocol.ReqIDPrefix+" "+opts.ReqID) {
				foundAnchor = true
				replyParts = nil // reset in case of duplicate anchors
				continue
			}
		}

		if !foundAnchor {
			continue
		}

		// Collect assistant messages after anchor
		if entryType == "assistant" {
			content := extractClaudeEntryContent(entry)
			if content != "" {
				replyParts = append(replyParts, content)
			}
		}
	}

	return strings.Join(replyParts, "\n"), nil
}

func (c *ClaudeCommunicator) WaitForReply(ctx context.Context, opts WaitOpts) (string, error) {
	cfg := c.PollCfg
	interval := cfg.InitialInterval
	if opts.PollMs > 0 {
		interval = time.Duration(opts.PollMs) * time.Millisecond
	}

	lastForceRead := time.Now()

	for {
		select {
		case <-ctx.Done():
			return "", &ErrTimeout{Provider: "claude", ReqID: opts.ReqID}
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
				return "", &ErrPaneDead{Provider: "claude", PaneID: opts.PaneID}
			}
		}

		time.Sleep(interval)
		interval = adaptiveSleep(interval, cfg)
	}
}

func (c *ClaudeCommunicator) CaptureState(ctx context.Context, opts ReadOpts) (*CaptureState, error) {
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

func (c *ClaudeCommunicator) HealthCheck(ctx context.Context, paneID string) error {
	if !c.IsAlive(paneID) {
		return &ErrPaneDead{Provider: "claude", PaneID: paneID}
	}
	return nil
}

// ClaudeEntry represents a parsed entry from a Claude JSONL log.
type ClaudeEntry = map[string]interface{}

// ContentBlock represents a content block in a Claude message.
type ContentBlock struct {
	Type string `json:"type"` // "text", "tool_use", "tool_result"
	Text string `json:"text"`
}

// readClaudeLog reads entries from a Claude JSONL log file or directory.
func readClaudeLog(logPath string, reqID string) ([]ClaudeEntry, error) {
	info, err := os.Stat(logPath)
	if err != nil {
		return nil, err
	}

	var logFile string
	if info.IsDir() {
		logFile = findMostRecentJSONL(logPath)
		if logFile == "" {
			return nil, nil
		}
	} else {
		logFile = logPath
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var entries []ClaudeEntry

	// Read from the end to find relevant entries (limit to last 200 for performance)
	startIdx := 0
	if len(lines) > 200 {
		startIdx = len(lines) - 200
	}

	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var entry ClaudeEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// findMostRecentJSONL finds the most recently modified .jsonl file in a directory tree.
func findMostRecentJSONL(dir string) string {
	var files []struct {
		path    string
		modTime time.Time
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".jsonl") {
			files = append(files, struct {
				path    string
				modTime time.Time
			}{path, info.ModTime()})
		}
		return nil
	})

	if len(files) == 0 {
		return ""
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	return files[0].path
}

// extractClaudeEntryContent extracts text content from a Claude log entry.
// Handles both string content and []ContentBlock arrays.
func extractClaudeEntryContent(entry ClaudeEntry) string {
	// Try message.content first
	if msg, ok := entry["message"].(map[string]interface{}); ok {
		if content, ok := msg["content"]; ok {
			return extractClaudeContent(content)
		}
	}

	// Try direct content field
	if content, ok := entry["content"]; ok {
		return extractClaudeContent(content)
	}

	return ""
}

// extractClaudeContent extracts text content from Claude's message content field.
// Content can be a string or an array of content blocks.
func extractClaudeContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return stripANSI(v)
	case []interface{}:
		var parts []string
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["type"].(string); ok && t == "text" {
					if text, ok := m["text"].(string); ok {
						parts = append(parts, stripANSI(text))
					}
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

// ansiRE matches ANSI escape sequences.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

// ClaudeProjectKey computes the project key for a work directory.
// Claude uses URL-encoded paths as project keys.
func ClaudeProjectKey(workDir string) string {
	abs, err := filepath.Abs(workDir)
	if err != nil {
		abs = workDir
	}
	// Normalize to forward slashes
	norm := strings.ReplaceAll(abs, "\\", "/")
	// URL-encode the path, replacing / with -
	encoded := url.PathEscape(norm)
	encoded = strings.ReplaceAll(encoded, "%2F", "-")
	encoded = strings.ReplaceAll(encoded, "/", "-")
	return encoded
}

// DiscoverClaudeProjectDir finds the Claude projects directory for a work directory.
func DiscoverClaudeProjectDir(workDir string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	projectsDir := filepath.Join(home, ".claude", "projects")
	if _, err := os.Stat(projectsDir); err != nil {
		return "", nil
	}

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return "", err
	}

	normWorkDir := strings.ToLower(strings.ReplaceAll(workDir, "\\", "/"))
	normWorkDir = strings.TrimRight(normWorkDir, "/")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectKey := entry.Name()
		decodedKey := strings.ReplaceAll(projectKey, "-", "/")
		if matchesWorkDir(decodedKey, normWorkDir) {
			return filepath.Join(projectsDir, projectKey), nil
		}
	}

	return "", nil
}

// matchesWorkDir checks if a decoded project key matches a work directory.
func matchesWorkDir(decodedKey string, normWorkDir string) bool {
	decodedKey = strings.ToLower(strings.ReplaceAll(decodedKey, "\\", "/"))
	decodedKey = strings.TrimRight(decodedKey, "/")

	if decodedKey == normWorkDir {
		return true
	}
	if strings.HasSuffix(decodedKey, normWorkDir) {
		return true
	}
	if strings.HasSuffix(normWorkDir, decodedKey) {
		return true
	}
	return false
}
