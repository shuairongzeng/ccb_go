package launcher

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/anthropics/claude_code_bridge/internal/config"
	"github.com/anthropics/claude_code_bridge/internal/session"
	"github.com/anthropics/claude_code_bridge/internal/terminal"
)

// AutoApproveSpec defines the auto-approve flags for each provider.
var AutoApproveSpec = map[string]ProviderAutoSpec{
	"codex": {
		CLIFlags:   []string{"-c", `trust_level="trusted"`, "-c", `approval_policy="never"`, "-c", `sandbox_mode="danger-full-access"`},
		ConfigFunc: ensureCodexAutoApproval,
	},
	"gemini": {
		CLIFlags: []string{"--yolo"},
	},
	"claude": {
		CLIFlags: []string{"--dangerously-skip-permissions"},
	},
	"opencode": {
		ConfigFunc: ensureOpenCodeAutoConfig,
	},
	"droid": {
		// Droid does not have a known auto-approve mechanism
	},
}

// ProviderAutoSpec holds auto-approve configuration for a provider.
type ProviderAutoSpec struct {
	CLIFlags   []string             // extra CLI flags added in auto mode
	ConfigFunc func() error         // optional: write config file for auto mode
}

// LaunchConfig holds the configuration for a multi-provider launch.
type LaunchConfig struct {
	Providers []string // provider names to launch
	Auto      bool     // auto-approve mode (-a)
	Resume    bool     // resume existing sessions
	WorkDir   string   // working directory
}

// LaunchResult holds the result of a provider launch.
type LaunchResult struct {
	Provider string
	PaneID   string
	Command  string
	Error    error
}

// ParseProviders splits comma/space-separated provider tokens and validates them.
func ParseProviders(args []string) []string {
	var raw []string
	for _, arg := range args {
		for _, part := range strings.Split(arg, ",") {
			p := strings.TrimSpace(strings.ToLower(part))
			if p != "" {
				raw = append(raw, p)
			}
		}
	}

	// Deduplicate and validate
	seen := make(map[string]bool)
	var result []string
	for _, p := range raw {
		if seen[p] {
			continue
		}
		if !isValidProvider(p) {
			fmt.Fprintf(os.Stderr, "warning: unknown provider %q, skipping\n", p)
			continue
		}
		seen[p] = true
		result = append(result, p)
	}
	return result
}

func isValidProvider(name string) bool {
	switch name {
	case "codex", "gemini", "opencode", "claude", "droid":
		return true
	}
	return false
}

// BuildStartCommand builds the CLI start command for a provider.
// If auto is true, injects auto-approve flags.
// If resume is true, injects resume/continue flags for the provider.
func BuildStartCommand(provider string, auto bool, resume bool) (string, error) {
	exe := providerExe(provider)
	if exe == "" {
		return "", fmt.Errorf("no CLI executable known for provider %q", provider)
	}

	// Apply auto-approve config files first
	if auto {
		spec, ok := AutoApproveSpec[provider]
		if ok && spec.ConfigFunc != nil {
			if err := spec.ConfigFunc(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: auto-config for %s failed: %v\n", provider, err)
			}
		}
	}

	// Build command
	parts := []string{exe}

	// Provider-specific command construction
	switch provider {
	case "codex":
		if resume {
			// Codex resume: codex resume --last [flags]
			parts = append(parts, "resume", "--last")
			parts = append(parts, "-c", "disable_paste_burst=true")
			fmt.Printf("  Resuming %s session...\n", provider)
		} else {
			parts = append(parts, "-c", "disable_paste_burst=true")
		}
	case "claude":
		if resume {
			parts = append(parts, "--continue")
			fmt.Printf("  Resuming %s session...\n", provider)
		}
	case "gemini":
		if resume {
			parts = append(parts, "--resume", "latest")
			fmt.Printf("  Resuming %s session...\n", provider)
		}
	case "opencode":
		if resume {
			parts = append(parts, "--continue")
			fmt.Printf("  Resuming %s session...\n", provider)
		}
	case "droid":
		if resume {
			parts = append(parts, "-r")
			fmt.Printf("  Resuming %s session...\n", provider)
		}
	}

	// Auto-approve CLI flags
	if auto {
		spec, ok := AutoApproveSpec[provider]
		if ok && len(spec.CLIFlags) > 0 {
			parts = append(parts, spec.CLIFlags...)
		}
	}

	return strings.Join(parts, " "), nil
}

// Launch launches multiple providers in terminal panes.
func Launch(cfg LaunchConfig) ([]LaunchResult, error) {
	if len(cfg.Providers) == 0 {
		// Load from config
		cwd := cfg.WorkDir
		if cwd == "" {
			cwd, _ = os.Getwd()
		}
		startCfg := config.LoadStartConfig(cwd)
		cfg.Providers = startCfg.GetProviders()
	}

	if len(cfg.Providers) == 0 {
		return nil, fmt.Errorf("no providers specified")
	}

	// Detect terminal backend
	backend, err := terminal.DetectBackend()
	if err != nil {
		// Fallback: just print the commands
		return launchFallback(cfg)
	}

	return launchWithBackend(cfg, backend)
}

// launchWithBackend launches providers using the detected terminal backend.
func launchWithBackend(cfg LaunchConfig, backend terminal.Backend) ([]LaunchResult, error) {
	var results []LaunchResult

	// Resolve current pane ID for split targets
	currentPaneID := resolveCurrentPaneID(backend)

	for i, provider := range cfg.Providers {
		cmd, err := BuildStartCommand(provider, cfg.Auto, cfg.Resume)
		if err != nil {
			results = append(results, LaunchResult{Provider: provider, Error: err})
			continue
		}

		var paneID string
		if i == 0 && len(cfg.Providers) == 1 {
			// Single provider: run in current pane directly
			fmt.Printf("Starting %s...\n", provider)
			if cfg.Auto {
				fmt.Printf("  [auto-approve mode enabled]\n")
			}
			execErr := execInCurrentPane(backend, currentPaneID, cmd)
			if execErr != nil {
				results = append(results, LaunchResult{Provider: provider, Command: cmd, Error: execErr})
				fmt.Printf("Failed to start %s: %v\n", provider, execErr)
				continue
			}
			paneID = currentPaneID
			fmt.Printf("Started %s in pane %s\n", provider, paneID)
		} else if i == 0 {
			// First of multiple providers: send command to current pane
			fmt.Printf("Starting %s in current pane...\n", provider)
			if cfg.Auto {
				fmt.Printf("  [auto-approve mode enabled]\n")
			}
			execErr := execInCurrentPane(backend, currentPaneID, cmd)
			if execErr != nil {
				results = append(results, LaunchResult{Provider: provider, Command: cmd, Error: execErr})
				fmt.Printf("Failed to start %s: %v\n", provider, execErr)
				continue
			}
			paneID = currentPaneID
			fmt.Printf("Started %s in pane %s\n", provider, paneID)
		} else {
			// Subsequent providers: split from current pane
			newID, splitErr := backend.SplitWindow(currentPaneID, cmd)
			if splitErr != nil {
				// Fallback: try spawning a new tab
				fmt.Printf("  split failed, trying new tab for %s...\n", provider)
				newID, splitErr = trySpawnWindow(backend, provider, cmd)
			}
			if splitErr != nil {
				results = append(results, LaunchResult{Provider: provider, Command: cmd, Error: splitErr})
				fmt.Printf("Failed to start %s: %v\n", provider, splitErr)
				continue
			}
			paneID = newID
			fmt.Printf("Started %s in pane %s\n", provider, paneID)
			if cfg.Auto {
				fmt.Printf("  [auto-approve mode enabled]\n")
			}

			// Set pane title for identification
			backend.SetPaneTitle(paneID, fmt.Sprintf("ccb-%s", provider))
		}

		results = append(results, LaunchResult{Provider: provider, PaneID: paneID, Command: cmd})

		// Register session so /cask, /gask etc. can find this pane
		registerSession(provider, paneID, cfg.WorkDir)
	}

	return results, nil
}

// execInCurrentPane sends a command to the current pane via SendKeys.
func execInCurrentPane(backend terminal.Backend, paneID string, cmd string) error {
	return backend.SendKeys(paneID, cmd)
}

// registerSession writes the pane ID to the session file and pane registry
// so that /cask, /gask etc. can find the provider's pane.
func registerSession(provider string, paneID string, workDir string) {
	if paneID == "" {
		return
	}

	// 1. Write session file: .ccb_config/.<provider>-session
	sessionFilename := fmt.Sprintf(".%s-session", provider)
	sessionDir, err := config.EnsureSessionDir(workDir)
	if err == nil {
		sessionFile := filepath.Join(sessionDir, sessionFilename)
		writeSessionFile(sessionFile, provider, paneID, workDir)
	}

	// 2. Write to pane registry
	registryPath := filepath.Join(ccbRunDir(), "pane-registry.json")
	registry := session.NewPaneRegistry(registryPath)
	projectID := config.ComputeCCBProjectID(workDir)
	registry.Upsert(provider, projectID, &session.PaneEntry{
		PaneID:  paneID,
		WorkDir: workDir,
	})
}

// writeSessionFile writes or updates a session file.
// If the file already exists as JSON (Python format), update the pane_id and active fields.
// Otherwise, write a new JSON session file compatible with the Python askd.
func writeSessionFile(sessionFile string, provider string, paneID string, workDir string) {
	// Try to read existing JSON session
	if data, err := os.ReadFile(sessionFile); err == nil {
		var existing map[string]interface{}
		if json.Unmarshal(data, &existing) == nil {
			// Update existing JSON session
			existing["pane_id"] = paneID
			existing["active"] = true
			if workDir != "" {
				existing["work_dir"] = workDir
			}
			out, _ := json.MarshalIndent(existing, "", "  ")
			os.WriteFile(sessionFile, out, 0600)
			return
		}
	}

	// Write new JSON session file (compatible with Python askd)
	normWorkDir := strings.ToLower(strings.ReplaceAll(workDir, "\\", "/"))
	sess := map[string]interface{}{
		"pane_id":        paneID,
		"terminal":       "wezterm",
		"work_dir":       workDir,
		"work_dir_norm":  normWorkDir,
		"active":         true,
		"ccb_project_id": config.ComputeCCBProjectID(workDir),
	}
	out, _ := json.MarshalIndent(sess, "", "  ")
	os.WriteFile(sessionFile, out, 0600)
}

// ccbRunDir returns the CCB runtime directory.
func ccbRunDir() string {
	// Inline to avoid circular import with runtime package
	if v := strings.TrimSpace(os.Getenv("CCB_RUN_DIR")); v != "" {
		return v
	}
	if goruntime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = os.Getenv("APPDATA")
		}
		if base != "" {
			return filepath.Join(base, "ccb")
		}
	}
	home, _ := os.UserHomeDir()
	xdg := os.Getenv("XDG_CACHE_HOME")
	if xdg != "" {
		return filepath.Join(xdg, "ccb")
	}
	return filepath.Join(home, ".cache", "ccb")
}

// resolveCurrentPaneID gets the current pane ID from the environment.
func resolveCurrentPaneID(backend terminal.Backend) string {
	// WezTerm: WEZTERM_PANE env var
	if pane := os.Getenv("WEZTERM_PANE"); pane != "" {
		return pane
	}
	// tmux: TMUX_PANE env var
	if pane := os.Getenv("TMUX_PANE"); pane != "" {
		return pane
	}
	// Fallback: try to get active pane from backend listing
	panes, err := backend.ListPanes()
	if err == nil {
		for _, p := range panes {
			if p.Active {
				return p.ID
			}
		}
		if len(panes) > 0 {
			return panes[0].ID
		}
	}
	// WezTerm default: omit pane-id and let wezterm pick current
	return ""
}

// trySpawnWindow tries to spawn a new window/tab as fallback when split fails.
func trySpawnWindow(backend terminal.Backend, provider string, cmd string) (string, error) {
	// For WezTerm, use "wezterm cli spawn" instead of split-pane
	if backend.Name() == "wezterm" {
		return weztermSpawn(cmd)
	}
	// For PowerShell, use SplitWindow which already creates a new window
	return backend.SplitWindow("", cmd)
}

// weztermSpawn spawns a new WezTerm pane using "wezterm cli spawn".
func weztermSpawn(cmd string) (string, error) {
	args := []string{"cli", "spawn", "--"}
	args = append(args, splitCommand(cmd)...)
	execCmd := exec.Command("wezterm", args...)
	setSysProcAttrLauncher(execCmd)
	out, err := execCmd.Output()
	if err != nil {
		// Try with combined stderr for better error message
		execCmd2 := exec.Command("wezterm", args...)
		setSysProcAttrLauncher(execCmd2)
		combined, err2 := execCmd2.CombinedOutput()
		if err2 != nil {
			return "", fmt.Errorf("%w: %s", err2, strings.TrimSpace(string(combined)))
		}
		return strings.TrimSpace(string(combined)), nil
	}
	return strings.TrimSpace(string(out)), nil
}

// splitCommand splits a command string into args, respecting quotes.
func splitCommand(cmd string) []string {
	// Simple split for common cases
	// For commands like: /path/to/exe -c "foo=bar" --flag
	var args []string
	var current strings.Builder
	inQuote := byte(0)

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch {
		case c == inQuote:
			inQuote = 0
			current.WriteByte(c)
		case inQuote != 0:
			current.WriteByte(c)
		case c == '"' || c == '\'':
			inQuote = c
			current.WriteByte(c)
		case c == ' ' || c == '\t':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// launchFallback prints commands when no terminal backend is available.
func launchFallback(cfg LaunchConfig) ([]LaunchResult, error) {
	fmt.Println("No terminal backend detected. Run these commands manually:")
	fmt.Println()

	var results []LaunchResult
	for _, provider := range cfg.Providers {
		cmd, err := BuildStartCommand(provider, cfg.Auto, cfg.Resume)
		if err != nil {
			results = append(results, LaunchResult{Provider: provider, Error: err})
			continue
		}
		fmt.Printf("  %s:  %s\n", provider, cmd)
		results = append(results, LaunchResult{Provider: provider, Command: cmd})
	}
	fmt.Println()
	return results, nil
}

// --- Provider executable detection ---

func providerExe(provider string) string {
	switch provider {
	case "codex":
		return findExe("codex")
	case "gemini":
		return findExe("gemini")
	case "opencode":
		return findExe("opencode")
	case "claude":
		return findExe("claude")
	case "droid":
		return findExe("droid")
	}
	return ""
}

func findExe(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	// Try common suffixes on Windows
	for _, suffix := range []string{".exe", ".cmd", ".bat"} {
		if path, err := exec.LookPath(name + suffix); err == nil {
			return path
		}
	}
	// Return bare name as fallback
	return name
}

// --- Auto-approve config writers ---

// ensureCodexAutoApproval writes auto-approve config for Codex CLI.
// Codex reads from ~/.codex/config.toml
func ensureCodexAutoApproval() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".codex")
	configFile := filepath.Join(configDir, "config.toml")

	// Check if config already has the right settings
	if data, err := os.ReadFile(configFile); err == nil {
		content := string(data)
		if strings.Contains(content, `approval_policy = "never"`) &&
			strings.Contains(content, `sandbox_mode = "danger-full-access"`) {
			return nil // Already configured
		}
	}

	os.MkdirAll(configDir, 0755)

	toml := `# CCB auto-approve configuration
trust_level = "trusted"
approval_policy = "never"
sandbox_mode = "danger-full-access"
`
	return os.WriteFile(configFile, []byte(toml), 0600)
}

// ensureOpenCodeAutoConfig writes auto-approve config for OpenCode.
func ensureOpenCodeAutoConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config", "opencode")
	configFile := filepath.Join(configDir, "config.json")

	// Check if config exists and already has auto-approve
	if data, err := os.ReadFile(configFile); err == nil {
		var cfg map[string]interface{}
		if json.Unmarshal(data, &cfg) == nil {
			if perms, ok := cfg["permissions"].(map[string]interface{}); ok {
				if auto, ok := perms["auto_approve"].(bool); ok && auto {
					return nil // Already configured
				}
			}
			// Merge auto_approve into existing config
			if cfg["permissions"] == nil {
				cfg["permissions"] = map[string]interface{}{}
			}
			if perms, ok := cfg["permissions"].(map[string]interface{}); ok {
				perms["auto_approve"] = true
			}
			out, _ := json.MarshalIndent(cfg, "", "  ")
			return os.WriteFile(configFile, out, 0600)
		}
	}

	os.MkdirAll(configDir, 0755)

	cfg := map[string]interface{}{
		"permissions": map[string]interface{}{
			"auto_approve": true,
		},
	}
	out, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(configFile, out, 0600)
}
