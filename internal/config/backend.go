package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const ConfigFilename = "ccb.config"

var (
	DefaultProviders = []string{"codex", "gemini", "opencode", "claude"}
	allowedProviders = map[string]bool{
		"codex": true, "gemini": true, "opencode": true, "claude": true, "droid": true,
	}
)

// StartConfig holds parsed CCB start configuration.
type StartConfig struct {
	Data map[string]interface{}
	Path string // empty if no config file found
}

// GetProviders returns the configured provider list.
func (c *StartConfig) GetProviders() []string {
	if c.Data == nil {
		return DefaultProviders
	}
	raw, ok := c.Data["providers"]
	if !ok {
		return DefaultProviders
	}
	switch v := raw.(type) {
	case []string:
		if len(v) > 0 {
			return v
		}
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return DefaultProviders
}

// CmdEnabled returns whether the "cmd" mode is enabled.
func (c *StartConfig) CmdEnabled() bool {
	if c.Data == nil {
		return false
	}
	v, ok := c.Data["cmd"]
	if !ok {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// parseTokens extracts provider tokens from a raw config string.
func parseTokens(raw string) []string {
	if raw == "" {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		// Strip comments
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		lines = append(lines, line)
	}
	cleaned := strings.Join(lines, " ")
	// Remove JSON syntax characters
	re := regexp.MustCompile(`[\[\]{}"']`)
	cleaned = re.ReplaceAllString(cleaned, " ")
	// Split on commas and whitespace
	parts := regexp.MustCompile(`[,\s]+`).Split(cleaned, -1)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// normalizeProviders filters and deduplicates provider tokens.
func normalizeProviders(tokens []string) ([]string, bool) {
	var providers []string
	seen := make(map[string]bool)
	cmdEnabled := false

	for _, raw := range tokens {
		token := strings.ToLower(strings.TrimSpace(raw))
		if token == "" {
			continue
		}
		if token == "cmd" {
			cmdEnabled = true
			continue
		}
		if !allowedProviders[token] {
			continue
		}
		if seen[token] {
			continue
		}
		seen[token] = true
		providers = append(providers, token)
	}
	return providers, cmdEnabled
}

// readConfig reads and parses a config file.
func readConfig(path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return nil
	}

	// Try JSON parse
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		return parseConfigObj(obj)
	}

	// Fallback: parse as token list
	tokens := parseTokens(raw)
	providers, cmdEnabled := normalizeProviders(tokens)
	result := map[string]interface{}{"providers": providers}
	if cmdEnabled {
		result["cmd"] = true
	}
	return result
}

// parseConfigObj parses a JSON-decoded config object.
func parseConfigObj(obj interface{}) map[string]interface{} {
	switch v := obj.(type) {
	case map[string]interface{}:
		data := make(map[string]interface{})
		for k, val := range v {
			data[k] = val
		}
		rawProviders, ok := data["providers"]
		if !ok {
			return data
		}
		var tokens []string
		switch rp := rawProviders.(type) {
		case string:
			tokens = parseTokens(rp)
		case []interface{}:
			for _, p := range rp {
				if s, ok := p.(string); ok {
					tokens = append(tokens, s)
				}
			}
		}
		if len(tokens) > 0 {
			providers, cmdEnabled := normalizeProviders(tokens)
			data["providers"] = providers
			if cmdEnabled {
				if _, exists := data["cmd"]; !exists {
					data["cmd"] = true
				}
			}
		}
		return data

	case []interface{}:
		tokens := make([]string, 0, len(v))
		for _, p := range v {
			if s, ok := p.(string); ok {
				tokens = append(tokens, s)
			}
		}
		providers, cmdEnabled := normalizeProviders(tokens)
		data := map[string]interface{}{"providers": providers}
		if cmdEnabled {
			data["cmd"] = true
		}
		return data

	case string:
		tokens := parseTokens(v)
		providers, cmdEnabled := normalizeProviders(tokens)
		data := map[string]interface{}{"providers": providers}
		if cmdEnabled {
			data["cmd"] = true
		}
		return data
	}

	return nil
}

// configPaths returns the project and global config file paths.
func configPaths(workDir string) (string, string) {
	project := filepath.Join(workDir, ".ccb_config", ConfigFilename)
	home, _ := os.UserHomeDir()
	global := filepath.Join(home, ".ccb", ConfigFilename)
	return project, global
}

// LoadStartConfig loads the CCB start configuration.
func LoadStartConfig(workDir string) *StartConfig {
	project, global := configPaths(workDir)
	if _, err := os.Stat(project); err == nil {
		return &StartConfig{Data: readConfig(project), Path: project}
	}
	if _, err := os.Stat(global); err == nil {
		return &StartConfig{Data: readConfig(global), Path: global}
	}
	return &StartConfig{Data: nil, Path: ""}
}

// EnsureDefaultStartConfig ensures a default config file exists.
func EnsureDefaultStartConfig(workDir string) (string, bool) {
	project, _ := configPaths(workDir)
	if _, err := os.Stat(project); err == nil {
		return project, false
	}
	dir := filepath.Dir(project)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", false
	}
	payload := strings.Join(DefaultProviders, ",") + "\n"
	if err := os.WriteFile(project, []byte(payload), 0600); err != nil {
		return "", false
	}
	return project, true
}

// GetBackendEnv returns the backend environment ("wsl" or "windows").
func GetBackendEnv() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("CCB_BACKEND_ENV")))
	if v == "wsl" || v == "windows" {
		return v
	}

	// Check .ccb-config.json
	configPath := filepath.Join(".", ".ccb-config.json")
	data, err := os.ReadFile(configPath)
	if err == nil {
		var cfg map[string]interface{}
		if json.Unmarshal(data, &cfg) == nil {
			if be, ok := cfg["BackendEnv"].(string); ok {
				be = strings.ToLower(strings.TrimSpace(be))
				if be == "wsl" || be == "windows" {
					return be
				}
			}
		}
	}

	if runtime.GOOS == "windows" {
		return "windows"
	}
	return ""
}
