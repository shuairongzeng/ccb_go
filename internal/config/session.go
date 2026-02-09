package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const CCBProjectConfigDirname = ".ccb_config"

// ProjectConfigDir returns the .ccb_config directory for a given work directory.
func ProjectConfigDir(workDir string) string {
	abs, err := filepath.Abs(workDir)
	if err != nil {
		abs = workDir
	}
	return filepath.Join(abs, CCBProjectConfigDirname)
}

// CheckSessionWritable checks if a session file is writable.
// Returns (writable, errorReason, fixSuggestion).
func CheckSessionWritable(sessionFile string) (bool, string, string) {
	parent := filepath.Dir(sessionFile)

	// 1. Check if parent directory exists
	info, err := os.Stat(parent)
	if err != nil || !info.IsDir() {
		return false, fmt.Sprintf("Directory not found: %s", parent), fmt.Sprintf("mkdir -p %s", parent)
	}

	// 2. Check if parent directory is writable (try creating a temp file)
	tmpFile := filepath.Join(parent, ".ccb_write_test")
	f, err := os.Create(tmpFile)
	if err != nil {
		return false, fmt.Sprintf("Directory not writable: %s", parent), fmt.Sprintf("chmod u+w %s", parent)
	}
	f.Close()
	os.Remove(tmpFile)

	// 3. If file doesn't exist, directory writable is enough
	finfo, err := os.Lstat(sessionFile)
	if os.IsNotExist(err) {
		return true, "", ""
	}
	if err != nil {
		return false, fmt.Sprintf("Cannot stat file: %s", err), ""
	}

	// 4. Check if it's a symlink
	if finfo.Mode()&os.ModeSymlink != 0 {
		target, _ := os.Readlink(sessionFile)
		return false, fmt.Sprintf("Is symlink pointing to %s", target), fmt.Sprintf("rm -f %s", sessionFile)
	}

	// 5. Check if it's a directory
	if finfo.IsDir() {
		return false, "Is directory, not file", fmt.Sprintf("rmdir %s or rm -rf %s", sessionFile, sessionFile)
	}

	// 6. Check if it's a regular file
	if !finfo.Mode().IsRegular() {
		return false, "Not a regular file", fmt.Sprintf("rm -f %s", sessionFile)
	}

	// 7. Check file ownership (POSIX only)
	if runtime.GOOS != "windows" {
		checkOwnership(sessionFile, finfo)
	}

	// 8. Check if file is writable
	f, err = os.OpenFile(sessionFile, os.O_WRONLY, 0)
	if err != nil {
		return false, fmt.Sprintf("File not writable (mode: %s)", finfo.Mode().String()), fmt.Sprintf("chmod u+w %s", sessionFile)
	}
	f.Close()

	return true, "", ""
}

// SafeWriteSession safely writes a session file with atomic rename.
// Returns (success, errorMessage).
func SafeWriteSession(sessionFile string, content string) (bool, string) {
	// Pre-check
	writable, reason, fix := CheckSessionWritable(sessionFile)
	if !writable {
		name := filepath.Base(sessionFile)
		return false, fmt.Sprintf("Cannot write %s: %s\nFix: %s", name, reason, fix)
	}

	// Ensure parent directory exists
	parent := filepath.Dir(sessionFile)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return false, fmt.Sprintf("Cannot create directory: %s", err)
	}

	// Atomic write via temp file + rename
	tmpFile := sessionFile + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		os.Remove(tmpFile)
		return false, fmt.Sprintf("Cannot write %s: %s\nTry: rm -f %s then retry", filepath.Base(sessionFile), err, sessionFile)
	}

	if err := os.Rename(tmpFile, sessionFile); err != nil {
		os.Remove(tmpFile)
		return false, fmt.Sprintf("Write failed: %s", err)
	}

	return true, ""
}

// FindProjectSessionFile finds a session file for the given work directory.
// Lookup is local-only (no upward traversal):
//  1. <workDir>/.ccb_config/<sessionFilename>
//  2. <workDir>/<sessionFilename> (legacy)
func FindProjectSessionFile(workDir string, sessionFilename string) string {
	abs, err := filepath.Abs(workDir)
	if err != nil {
		abs = workDir
	}

	candidate := filepath.Join(abs, CCBProjectConfigDirname, sessionFilename)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	legacy := filepath.Join(abs, sessionFilename)
	if _, err := os.Stat(legacy); err == nil {
		return legacy
	}

	return ""
}

// PrintSessionError outputs a session-related error to stderr.
func PrintSessionError(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

// checkOwnership is a no-op on Windows; on POSIX it would check UID.
// Implemented in session_posix.go with build tags.
func checkOwnership(path string, info os.FileInfo) (bool, string, string) {
	// Default: assume OK. POSIX-specific checks in session_posix.go
	return true, "", ""
}

// EnsureSessionDir ensures the .ccb_config directory exists for a work directory.
func EnsureSessionDir(workDir string) (string, error) {
	dir := ProjectConfigDir(workDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// ReadSessionFile reads the content of a session file, returning empty string on error.
func ReadSessionFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
