package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anthropics/claude_code_bridge/internal/config"
	"github.com/anthropics/claude_code_bridge/internal/terminal"
)

// ResolvedSession holds the result of session resolution.
type ResolvedSession struct {
	SessionID  string
	ProjectKey string
	LogFile    string
	PaneID     string
	Source     string // "env", "registry_project", "registry_unfiltered", "session_file", "registry_pane", "fallback"
}

// SessionResolver resolves Claude sessions using a 6-stage fallback chain.
type SessionResolver struct {
	registry *PaneRegistry
	backend  terminal.Backend
}

// NewSessionResolver creates a new SessionResolver.
func NewSessionResolver(registry *PaneRegistry, backend terminal.Backend) *SessionResolver {
	return &SessionResolver{
		registry: registry,
		backend:  backend,
	}
}

// Resolve resolves the active Claude session for a work directory.
// It tries 6 stages in order:
//  1. Environment variables (CCB_SESSION_ID)
//  2. Registry by project ID
//  3. Registry unfiltered (any matching entry)
//  4. Session file in project directory
//  5. Registry by current pane ID
//  6. Best fallback (most recent session file)
func (r *SessionResolver) Resolve(workDir string) (*ResolvedSession, error) {
	// Stage 1: Environment variables
	if result := r.resolveFromEnv(); result != nil {
		return result, nil
	}

	projectID := config.ComputeCCBProjectID(workDir)

	// Stage 2: Registry by project ID
	if result := r.resolveFromRegistryByProject(projectID); result != nil {
		return result, nil
	}

	// Stage 3: Registry unfiltered
	if result := r.resolveFromRegistryUnfiltered(); result != nil {
		return result, nil
	}

	// Stage 4: Session file in project directory
	if result := r.resolveFromSessionFile(workDir); result != nil {
		return result, nil
	}

	// Stage 5: Registry by current pane ID
	if result := r.resolveFromRegistryByPane(); result != nil {
		return result, nil
	}

	// Stage 6: Best fallback
	return r.resolveBestFallback(workDir)
}

// resolveFromEnv checks environment variables for session info.
func (r *SessionResolver) resolveFromEnv() *ResolvedSession {
	sessionID := strings.TrimSpace(os.Getenv("CCB_SESSION_ID"))
	if sessionID == "" {
		return nil
	}

	if r.registry != nil {
		provider, entry := r.registry.GetBySessionID(sessionID)
		if entry != nil {
			return &ResolvedSession{
				SessionID:  sessionID,
				ProjectKey: provider,
				PaneID:     entry.PaneID,
				LogFile:    entry.SessionPath,
				Source:     "env",
			}
		}
	}

	return &ResolvedSession{
		SessionID: sessionID,
		Source:    "env",
	}
}

// resolveFromRegistryByProject looks up the registry by project ID.
func (r *SessionResolver) resolveFromRegistryByProject(projectID string) *ResolvedSession {
	if r.registry == nil {
		return nil
	}

	entry := r.registry.GetEntry("claude", projectID)
	if entry == nil || entry.PaneID == "" {
		return nil
	}

	// Verify pane is alive
	if r.backend != nil && !r.backend.IsAlive(entry.PaneID) {
		return nil
	}

	return &ResolvedSession{
		SessionID:  entry.SessionID,
		ProjectKey: projectID,
		PaneID:     entry.PaneID,
		LogFile:    entry.SessionPath,
		Source:     "registry_project",
	}
}

// resolveFromRegistryUnfiltered scans all Claude entries in the registry.
func (r *SessionResolver) resolveFromRegistryUnfiltered() *ResolvedSession {
	if r.registry == nil {
		return nil
	}

	entries := r.registry.GetByProvider("claude")
	if len(entries) == 0 {
		return nil
	}

	// Find the most recently updated entry that's alive
	var bestKey string
	var bestEntry *PaneEntry
	var bestTime int64

	for key, entry := range entries {
		if entry.PaneID == "" {
			continue
		}
		if r.backend != nil && !r.backend.IsAlive(entry.PaneID) {
			continue
		}
		if entry.UpdatedAt > bestTime {
			bestTime = entry.UpdatedAt
			bestKey = key
			bestEntry = entry
		}
	}

	if bestEntry == nil {
		return nil
	}

	return &ResolvedSession{
		SessionID:  bestEntry.SessionID,
		ProjectKey: bestKey,
		PaneID:     bestEntry.PaneID,
		LogFile:    bestEntry.SessionPath,
		Source:     "registry_unfiltered",
	}
}

// resolveFromSessionFile reads the .claude-session file in the project directory.
func (r *SessionResolver) resolveFromSessionFile(workDir string) *ResolvedSession {
	sessionFile := config.FindProjectSessionFile(workDir, ".claude-session")
	if sessionFile == "" {
		return nil
	}

	content := config.ReadSessionFile(sessionFile)
	if content == "" {
		return nil
	}

	// Content could be a pane ID or session ID
	return &ResolvedSession{
		PaneID: content,
		Source: "session_file",
	}
}

// resolveFromRegistryByPane looks up the registry by the current pane ID.
func (r *SessionResolver) resolveFromRegistryByPane() *ResolvedSession {
	if r.registry == nil {
		return nil
	}

	// Get current pane from environment
	currentPane := strings.TrimSpace(os.Getenv("TMUX_PANE"))
	if currentPane == "" {
		currentPane = strings.TrimSpace(os.Getenv("WEZTERM_PANE"))
	}
	if currentPane == "" {
		return nil
	}

	provider, entry := r.registry.GetByClaudePane(currentPane)
	if entry == nil {
		return nil
	}

	return &ResolvedSession{
		SessionID:  entry.SessionID,
		ProjectKey: provider,
		PaneID:     entry.PaneID,
		LogFile:    entry.SessionPath,
		Source:     "registry_pane",
	}
}

// resolveBestFallback finds the best available session by scanning the filesystem.
func (r *SessionResolver) resolveBestFallback(workDir string) (*ResolvedSession, error) {
	info, err := ResolveClaudeSession(workDir)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}

	return &ResolvedSession{
		SessionID:  info.SessionID,
		ProjectKey: info.ProjectKey,
		LogFile:    info.LogFile,
		Source:     "fallback",
	}, nil
}

// --- Claude Session Info (filesystem-based resolution) ---

// ClaudeSessionInfo holds resolved Claude session information.
type ClaudeSessionInfo struct {
	SessionID  string
	ProjectKey string
	LogFile    string
}

// ResolveClaudeSession resolves the active Claude session for a work directory.
// It searches ~/.claude/projects/ for matching session files.
func ResolveClaudeSession(workDir string) (*ClaudeSessionInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	projectsDir := filepath.Join(home, ".claude", "projects")
	if _, err := os.Stat(projectsDir); err != nil {
		return nil, nil
	}

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	normWorkDir := strings.ToLower(strings.ReplaceAll(workDir, "\\", "/"))
	normWorkDir = strings.TrimRight(normWorkDir, "/")

	var candidates []ClaudeSessionInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectKey := entry.Name()
		projectDir := filepath.Join(projectsDir, projectKey)

		decodedKey := strings.ReplaceAll(projectKey, "-", "/")
		if !matchesWorkDirResolver(decodedKey, normWorkDir) {
			continue
		}

		sessionFiles, err := filepath.Glob(filepath.Join(projectDir, "*.jsonl"))
		if err != nil {
			continue
		}

		for _, sf := range sessionFiles {
			base := filepath.Base(sf)
			sessionID := strings.TrimSuffix(base, ".jsonl")

			// Skip sidechain sessions
			if strings.Contains(sessionID, "sidechain") {
				continue
			}

			candidates = append(candidates, ClaudeSessionInfo{
				SessionID:  sessionID,
				ProjectKey: projectKey,
				LogFile:    sf,
			})
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		iInfo, _ := os.Stat(candidates[i].LogFile)
		jInfo, _ := os.Stat(candidates[j].LogFile)
		if iInfo == nil || jInfo == nil {
			return false
		}
		return iInfo.ModTime().After(jInfo.ModTime())
	})

	return &candidates[0], nil
}

// matchesWorkDirResolver checks if a decoded project key matches a work directory.
func matchesWorkDirResolver(decodedKey string, normWorkDir string) bool {
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

// ReadClaudeSessionLog reads the last N lines from a Claude session JSONL file.
func ReadClaudeSessionLog(logFile string, maxLines int) ([]map[string]interface{}, error) {
	data, err := os.ReadFile(logFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	var entries []map[string]interface{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
