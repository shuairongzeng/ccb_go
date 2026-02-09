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

// DroidCommunicator handles communication with Droid.
// Droid stores sessions in ~/.factory/sessions/<slug>/<session-id>.jsonl
type DroidCommunicator struct {
	BaseCommunicator
}

// NewDroidCommunicator creates a new Droid communicator.
func NewDroidCommunicator(backend terminal.Backend) *DroidCommunicator {
	return &DroidCommunicator{
		BaseCommunicator: BaseCommunicator{
			ProviderName: "droid",
			Backend:      backend,
			PollCfg:      DefaultPollConfig(),
		},
	}
}

func (c *DroidCommunicator) Name() string { return "droid" }

func (c *DroidCommunicator) SendPrompt(ctx context.Context, paneID string, message string) error {
	return c.SendViaTerminal(paneID, message)
}

func (c *DroidCommunicator) ReadReply(ctx context.Context, opts ReadOpts) (string, error) {
	if opts.LogPath == "" {
		return "", nil
	}
	return readDroidSession(opts.LogPath, opts.ReqID)
}

func (c *DroidCommunicator) WaitForReply(ctx context.Context, opts WaitOpts) (string, error) {
	cfg := c.PollCfg
	interval := cfg.InitialInterval
	if opts.PollMs > 0 {
		interval = time.Duration(opts.PollMs) * time.Millisecond
	}

	lastForceRead := time.Now()

	for {
		select {
		case <-ctx.Done():
			return "", &ErrTimeout{Provider: "droid", ReqID: opts.ReqID}
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
				return "", &ErrPaneDead{Provider: "droid", PaneID: opts.PaneID}
			}
		}

		time.Sleep(interval)
		interval = adaptiveSleep(interval, cfg)
	}
}

func (c *DroidCommunicator) CaptureState(ctx context.Context, opts ReadOpts) (*CaptureState, error) {
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

func (c *DroidCommunicator) HealthCheck(ctx context.Context, paneID string) error {
	if !c.IsAlive(paneID) {
		return &ErrPaneDead{Provider: "droid", PaneID: paneID}
	}
	return nil
}

// DroidEvent represents an event from a Droid events.jsonl file.
type DroidEvent struct {
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content string `json:"content"`
	Text    string `json:"text"`
	CWD     string `json:"cwd"`
	ID      string `json:"id"`
}

// readDroidSession reads the latest reply from Droid's session directory.
func readDroidSession(sessionsDir string, reqID string) (string, error) {
	eventsFile, err := findLatestDroidEvents(sessionsDir)
	if err != nil || eventsFile == "" {
		return "", err
	}

	events, err := parseDroidEvents(eventsFile)
	if err != nil {
		return "", err
	}

	// Find the anchor and collect reply
	foundAnchor := false
	var replyParts []string

	for _, event := range events {
		content := event.Content
		if content == "" {
			content = event.Text
		}

		if !foundAnchor {
			if strings.Contains(content, protocol.ReqIDPrefix+" "+reqID) {
				foundAnchor = true
			}
			continue
		}

		if event.Role == "assistant" || event.Type == "assistant" || event.Type == "message" {
			if content != "" {
				replyParts = append(replyParts, content)
			}
		}
	}

	return strings.Join(replyParts, "\n"), nil
}

// findLatestDroidEvents finds the most recent events.jsonl file in the sessions directory.
func findLatestDroidEvents(sessionsDir string) (string, error) {
	entries, err := os.ReadDir(sessionsDir)
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
			// Check for direct .jsonl files (slug-based layout)
			if strings.HasSuffix(e.Name(), ".jsonl") {
				info, err := e.Info()
				if err != nil {
					continue
				}
				files = append(files, fileEntry{
					path:    filepath.Join(sessionsDir, e.Name()),
					modTime: info.ModTime(),
				})
			}
			continue
		}

		// Check for events.jsonl in subdirectory
		eventsFile := filepath.Join(sessionsDir, e.Name(), "events.jsonl")
		info, err := os.Stat(eventsFile)
		if err != nil {
			// Also check for direct .jsonl files in subdirectory
			subEntries, err := os.ReadDir(filepath.Join(sessionsDir, e.Name()))
			if err != nil {
				continue
			}
			for _, se := range subEntries {
				if !se.IsDir() && strings.HasSuffix(se.Name(), ".jsonl") {
					sInfo, err := se.Info()
					if err != nil {
						continue
					}
					files = append(files, fileEntry{
						path:    filepath.Join(sessionsDir, e.Name(), se.Name()),
						modTime: sInfo.ModTime(),
					})
				}
			}
			continue
		}
		files = append(files, fileEntry{
			path:    eventsFile,
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

// parseDroidEvents parses a Droid events JSONL file.
func parseDroidEvents(eventsFile string) ([]DroidEvent, error) {
	data, err := os.ReadFile(eventsFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var events []DroidEvent

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var event DroidEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// DiscoverDroidSessions finds the Droid sessions directory.
func DiscoverDroidSessions() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	sessionsDir := filepath.Join(home, ".factory", "sessions")
	if _, err := os.Stat(sessionsDir); err == nil {
		return sessionsDir, nil
	}

	return "", nil
}

// FindDroidSessionByWorkDir finds a Droid session matching the given work directory.
func FindDroidSessionByWorkDir(sessionsDir string, workDir string) (string, error) {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return "", err
	}

	normWorkDir := strings.ToLower(strings.ReplaceAll(workDir, "\\", "/"))
	normWorkDir = strings.TrimRight(normWorkDir, "/")

	type candidate struct {
		path    string
		modTime time.Time
	}
	var candidates []candidate

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		// Check for events.jsonl or any .jsonl file
		dirPath := filepath.Join(sessionsDir, e.Name())
		subEntries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		for _, se := range subEntries {
			if se.IsDir() || !strings.HasSuffix(se.Name(), ".jsonl") {
				continue
			}

			filePath := filepath.Join(dirPath, se.Name())

			// Read first line to check cwd
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			firstLine := strings.SplitN(string(data), "\n", 2)[0]
			var firstEvent DroidEvent
			if err := json.Unmarshal([]byte(firstLine), &firstEvent); err != nil {
				continue
			}

			eventCWD := strings.ToLower(strings.ReplaceAll(firstEvent.CWD, "\\", "/"))
			eventCWD = strings.TrimRight(eventCWD, "/")

			if eventCWD == normWorkDir || strings.HasSuffix(eventCWD, normWorkDir) || strings.HasSuffix(normWorkDir, eventCWD) {
				info, err := se.Info()
				if err != nil {
					continue
				}
				candidates = append(candidates, candidate{path: filePath, modTime: info.ModTime()})
			}
		}
	}

	if len(candidates) == 0 {
		return "", nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].modTime.After(candidates[j].modTime)
	})

	return candidates[0].path, nil
}
