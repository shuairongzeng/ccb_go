// Package config provides configuration, environment variable parsing, and project ID utilities.
package config

import (
	"os"
	"strconv"
	"strings"
)

// EnvBool reads a boolean from an environment variable.
// Truthy: "1", "true", "yes", "on"
// Falsy:  "0", "false", "no", "off"
func EnvBool(name string, defaultVal bool) bool {
	raw := os.Getenv(name)
	if raw == "" {
		return defaultVal
	}
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "0", "false", "no", "off":
		return false
	case "1", "true", "yes", "on":
		return true
	}
	return defaultVal
}

// EnvInt reads an integer from an environment variable.
func EnvInt(name string, defaultVal int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return defaultVal
	}
	return v
}

// EnvStr reads a string from an environment variable, returning defaultVal if empty.
func EnvStr(name string, defaultVal string) string {
	raw := os.Getenv(name)
	if strings.TrimSpace(raw) == "" {
		return defaultVal
	}
	return strings.TrimSpace(raw)
}
