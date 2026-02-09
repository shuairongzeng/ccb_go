package session

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/claude_code_bridge/internal/config"
)

// ProjectSession holds session state for a provider in a specific project.
type ProjectSession struct {
	Provider  string
	ProjectID string
	WorkDir   string
	PaneID    string
	SessionID string
	LogPath   string
}

// LoaderFunc is a function that loads a session for a provider.
type LoaderFunc func(workDir string) (*ProjectSession, error)

// --- Codex Session ---

// LoadCodexSession loads a Codex session from the work directory.
func LoadCodexSession(workDir string) (*ProjectSession, error) {
	sessionFile := config.FindProjectSessionFile(workDir, ".codex-session")
	if sessionFile == "" {
		return nil, nil
	}
	content := config.ReadSessionFile(sessionFile)
	if content == "" {
		return nil, nil
	}

	projectID := config.ComputeCCBProjectID(workDir)

	// Codex session file contains the pane ID
	return &ProjectSession{
		Provider:  "codex",
		ProjectID: projectID,
		WorkDir:   workDir,
		PaneID:    content,
		LogPath:   findCodexLogPath(workDir),
	}, nil
}

func findCodexLogPath(workDir string) string {
	// Check CODEX_SESSION_ROOT env
	root := strings.TrimSpace(os.Getenv("CODEX_SESSION_ROOT"))
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".codex", "sessions")
	}
	// Find the most recent session log
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	var latest string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		logFile := filepath.Join(root, e.Name(), "output.log")
		if _, err := os.Stat(logFile); err == nil {
			latest = logFile
		}
	}
	return latest
}

// --- Gemini Session ---

// LoadGeminiSession loads a Gemini session from the work directory.
func LoadGeminiSession(workDir string) (*ProjectSession, error) {
	sessionFile := config.FindProjectSessionFile(workDir, ".gemini-session")
	if sessionFile == "" {
		return nil, nil
	}
	content := config.ReadSessionFile(sessionFile)
	if content == "" {
		return nil, nil
	}

	projectID := config.ComputeCCBProjectID(workDir)

	return &ProjectSession{
		Provider:  "gemini",
		ProjectID: projectID,
		WorkDir:   workDir,
		PaneID:    content,
		LogPath:   findGeminiLogPath(workDir),
	}, nil
}

func findGeminiLogPath(workDir string) string {
	root := strings.TrimSpace(os.Getenv("GEMINI_ROOT"))
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".gemini", "tmp")
	}
	// Find session directory by project hash
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		chatsDir := filepath.Join(root, e.Name(), "chats")
		if _, err := os.Stat(chatsDir); err == nil {
			return chatsDir
		}
	}
	return ""
}

// --- OpenCode Session ---

// LoadOpenCodeSession loads an OpenCode session from the work directory.
func LoadOpenCodeSession(workDir string) (*ProjectSession, error) {
	sessionFile := config.FindProjectSessionFile(workDir, ".opencode-session")
	if sessionFile == "" {
		return nil, nil
	}
	content := config.ReadSessionFile(sessionFile)
	if content == "" {
		return nil, nil
	}

	projectID := config.ComputeCCBProjectID(workDir)

	return &ProjectSession{
		Provider:  "opencode",
		ProjectID: projectID,
		WorkDir:   workDir,
		PaneID:    content,
		LogPath:   findOpenCodeStoragePath(),
	}, nil
}

func findOpenCodeStoragePath() string {
	home, _ := os.UserHomeDir()
	storagePath := filepath.Join(home, ".local", "share", "opencode", "storage")
	if _, err := os.Stat(storagePath); err == nil {
		return storagePath
	}
	return ""
}

// --- Claude Session ---

// LoadClaudeSession loads a Claude session from the work directory.
func LoadClaudeSession(workDir string) (*ProjectSession, error) {
	sessionFile := config.FindProjectSessionFile(workDir, ".claude-session")
	if sessionFile == "" {
		return nil, nil
	}
	content := config.ReadSessionFile(sessionFile)
	if content == "" {
		return nil, nil
	}

	projectID := config.ComputeCCBProjectID(workDir)

	return &ProjectSession{
		Provider:  "claude",
		ProjectID: projectID,
		WorkDir:   workDir,
		PaneID:    content,
		LogPath:   findClaudeLogPath(workDir),
	}, nil
}

func findClaudeLogPath(workDir string) string {
	home, _ := os.UserHomeDir()
	projectsDir := filepath.Join(home, ".claude", "projects")
	if _, err := os.Stat(projectsDir); err != nil {
		return ""
	}
	// Claude uses a project key derived from the work directory
	return projectsDir
}

// --- Droid Session ---

// LoadDroidSession loads a Droid session from the work directory.
func LoadDroidSession(workDir string) (*ProjectSession, error) {
	sessionFile := config.FindProjectSessionFile(workDir, ".droid-session")
	if sessionFile == "" {
		return nil, nil
	}
	content := config.ReadSessionFile(sessionFile)
	if content == "" {
		return nil, nil
	}

	projectID := config.ComputeCCBProjectID(workDir)

	return &ProjectSession{
		Provider:  "droid",
		ProjectID: projectID,
		WorkDir:   workDir,
		PaneID:    content,
		LogPath:   findDroidLogPath(),
	}, nil
}

func findDroidLogPath() string {
	home, _ := os.UserHomeDir()
	sessionsDir := filepath.Join(home, ".factory", "sessions")
	if _, err := os.Stat(sessionsDir); err == nil {
		return sessionsDir
	}
	return ""
}

// --- Loader Registry ---

// AllLoaders returns session loaders for all providers.
var AllLoaders = map[string]LoaderFunc{
	"codex":    LoadCodexSession,
	"gemini":   LoadGeminiSession,
	"opencode": LoadOpenCodeSession,
	"claude":   LoadClaudeSession,
	"droid":    LoadDroidSession,
}
