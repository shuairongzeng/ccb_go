package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Exit codes
const (
	ExitOK      = 0
	ExitError   = 1
	ExitNoReply = 2
)

// AtomicWriteText writes content to a file atomically via temp file + rename.
func AtomicWriteText(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	base := filepath.Base(path)
	tmpFile := filepath.Join(dir, "."+base+".tmp")

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		os.Remove(tmpFile)
		return err
	}

	if err := os.Rename(tmpFile, path); err != nil {
		os.Remove(tmpFile)
		return err
	}

	return nil
}

// NormalizeMessageParts joins message parts with spaces and trims.
func NormalizeMessageParts(parts []string) string {
	return strings.TrimSpace(strings.Join(parts, " "))
}

// DecodeStdinBytes decodes raw bytes robustly, handling BOMs and encoding overrides.
// In Go, strings are already UTF-8, so this is simpler than the Python version.
func DecodeStdinBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// BOM detection
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		// UTF-8 BOM
		return string(data[3:])
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		// UTF-16 LE BOM
		return decodeUTF16LE(data[2:])
	}
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		// UTF-16 BE BOM
		return decodeUTF16BE(data[2:])
	}

	// Check for forced encoding override
	forced := strings.TrimSpace(os.Getenv("CCB_STDIN_ENCODING"))
	if forced != "" {
		// In Go, we primarily handle UTF-8. For other encodings, best-effort.
		return string(data)
	}

	// Default: treat as UTF-8 (Go's native encoding)
	return string(data)
}

// decodeUTF16LE decodes UTF-16 Little Endian bytes to a Go string.
func decodeUTF16LE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		r := rune(data[i]) | rune(data[i+1])<<8
		runes = append(runes, r)
	}
	return string(runes)
}

// decodeUTF16BE decodes UTF-16 Big Endian bytes to a Go string.
func decodeUTF16BE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		r := rune(data[i])<<8 | rune(data[i+1])
		runes = append(runes, r)
	}
	return string(runes)
}

// Errorf prints a formatted error message to stderr.
func Errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// Infof prints a formatted info message to stdout.
func Infof(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}
