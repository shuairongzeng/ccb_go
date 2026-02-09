package config

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var (
	winDriveRE  = regexp.MustCompile(`^[A-Za-z]:([/\\]|$)`)
	mntDriveRE  = regexp.MustCompile(`^/mnt/([A-Za-z])/(.*)$`)
	msysDriveRE = regexp.MustCompile(`^/([A-Za-z])/(.*)$`)
)

// NormalizeWorkDir normalizes a work directory path into a stable string for hashing.
// It handles Windows drive letters, WSL /mnt/ paths, MSYS paths, and forward/back slashes.
func NormalizeWorkDir(value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ""
	}

	// Expand "~"
	if strings.HasPrefix(raw, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			raw = home + raw[1:]
		}
	}

	// Absolutize when relative
	preview := strings.ReplaceAll(raw, "\\", "/")
	isAbs := strings.HasPrefix(preview, "/") ||
		strings.HasPrefix(preview, "//") ||
		strings.HasPrefix(raw, "\\\\") ||
		winDriveRE.MatchString(preview)

	if !isAbs {
		cwd, err := os.Getwd()
		if err == nil {
			raw = filepath.Join(cwd, raw)
		}
	}

	s := strings.ReplaceAll(raw, "\\", "/")

	// Map WSL /mnt/<drive>/... to <drive>:/...
	if m := mntDriveRE.FindStringSubmatch(s); m != nil {
		drive := strings.ToLower(m[1])
		rest := m[2]
		s = drive + ":/" + rest
	} else if m := msysDriveRE.FindStringSubmatch(s); m != nil {
		// Map MSYS /<drive>/... to <drive>:/...
		_, hasMSYS := os.LookupEnv("MSYSTEM")
		if hasMSYS || runtime.GOOS == "windows" {
			drive := strings.ToLower(m[1])
			rest := m[2]
			s = drive + ":/" + rest
		}
	}

	// Collapse redundant separators using filepath.Clean logic on forward slashes
	if strings.HasPrefix(s, "//") {
		prefix := "//"
		rest := cleanPosixPath(s[2:])
		rest = strings.TrimLeft(rest, "/")
		s = prefix + rest
	} else {
		s = cleanPosixPath(s)
	}

	// Normalize Windows drive letter casing
	if winDriveRE.MatchString(s) {
		s = strings.ToLower(s[:1]) + s[1:]
	}

	return s
}

// cleanPosixPath normalizes a POSIX-style path (forward slashes).
func cleanPosixPath(p string) string {
	// Use filepath.Clean but ensure forward slashes
	cleaned := filepath.Clean(p)
	return strings.ReplaceAll(cleaned, "\\", "/")
}

// findCCBConfigRoot finds a .ccb_config/ directory in the given directory (no ancestor traversal).
func findCCBConfigRoot(startDir string) string {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		cwd, _ := os.Getwd()
		abs = cwd
	}
	cfg := filepath.Join(abs, ".ccb_config")
	info, err := os.Stat(cfg)
	if err == nil && info.IsDir() {
		return abs
	}
	return ""
}

// ComputeCCBProjectID computes the SHA256-based project ID for routing.
func ComputeCCBProjectID(workDir string) string {
	abs, err := filepath.Abs(workDir)
	if err != nil {
		abs, _ = os.Getwd()
	}

	base := findCCBConfigRoot(abs)
	if base == "" {
		base = abs
	}

	norm := NormalizeWorkDir(base)
	if norm == "" {
		norm = NormalizeWorkDir(abs)
	}

	hash := sha256.Sum256([]byte(norm))
	return fmt.Sprintf("%x", hash)
}
