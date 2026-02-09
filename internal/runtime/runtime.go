package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RunDir returns the CCB runtime directory for state/log files.
func RunDir() string {
	override := strings.TrimSpace(os.Getenv("CCB_RUN_DIR"))
	if override != "" {
		if strings.HasPrefix(override, "~") {
			home, err := os.UserHomeDir()
			if err == nil {
				override = home + override[1:]
			}
		}
		return override
	}

	if runtime.GOOS == "windows" {
		base := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
		if base == "" {
			base = strings.TrimSpace(os.Getenv("APPDATA"))
		}
		if base != "" {
			return filepath.Join(base, "ccb")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "AppData", "Local", "ccb")
	}

	xdgCache := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME"))
	if xdgCache != "" {
		return filepath.Join(xdgCache, "ccb")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "ccb")
}

// StateFilePath returns the path for a state file (JSON).
func StateFilePath(name string) string {
	if strings.HasSuffix(name, ".json") {
		return filepath.Join(RunDir(), name)
	}
	return filepath.Join(RunDir(), name+".json")
}

// LogPath returns the path for a log file.
func LogPath(name string) string {
	if strings.HasSuffix(name, ".log") {
		return filepath.Join(RunDir(), name)
	}
	return filepath.Join(RunDir(), name+".log")
}

var (
	lastLogShrinkCheck   = make(map[string]time.Time)
	lastLogShrinkCheckMu sync.Mutex
)

func envInt(name string, defaultVal int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return v
}

// maybeShrinkLog truncates a log file to its last N bytes when it exceeds CCB_LOG_MAX_BYTES.
func maybeShrinkLog(path string) {
	maxBytes := envInt("CCB_LOG_MAX_BYTES", 2*1024*1024) // 2 MiB default
	if maxBytes <= 0 {
		return
	}

	intervalS := envInt("CCB_LOG_SHRINK_CHECK_INTERVAL_S", 10)

	lastLogShrinkCheckMu.Lock()
	last, ok := lastLogShrinkCheck[path]
	now := time.Now()
	if ok && intervalS > 0 && now.Sub(last).Seconds() < float64(intervalS) {
		lastLogShrinkCheckMu.Unlock()
		return
	}
	lastLogShrinkCheck[path] = now
	lastLogShrinkCheckMu.Unlock()

	info, err := os.Stat(path)
	if err != nil {
		return
	}
	size := info.Size()
	if size <= int64(maxBytes) {
		return
	}

	// Read the tail
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	_, err = f.Seek(-int64(maxBytes), 2) // SEEK_END
	if err != nil {
		return
	}
	tail := make([]byte, maxBytes)
	n, err := f.Read(tail)
	if err != nil {
		return
	}
	tail = tail[:n]
	f.Close()

	// Write tail to temp file and replace
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)
	tmpFile := path + ".shrink.tmp"
	if err := os.WriteFile(tmpFile, tail, 0644); err != nil {
		return
	}
	os.Rename(tmpFile, path)
	// Clean up on failure
	os.Remove(tmpFile)
}

// WriteLog appends a message to a log file, with automatic log rotation.
func WriteLog(path string, msg string) {
	defer func() { recover() }()

	maybeShrinkLog(path)

	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	line := strings.TrimRight(msg, "\n") + "\n"
	f.WriteString(line)
}

// RandomToken generates a random 32-character hex token.
func RandomToken() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based token
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// NormalizeConnectHost normalizes a bind host to a connect host.
func NormalizeConnectHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" {
		return "127.0.0.1"
	}
	if host == "::" || host == "[::]" {
		return "::1"
	}
	return host
}

// EnsureRunDir creates the runtime directory if it doesn't exist.
func EnsureRunDir() error {
	return os.MkdirAll(RunDir(), 0755)
}
