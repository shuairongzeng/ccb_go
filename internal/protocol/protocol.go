package protocol

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Protocol markers
const (
	ReqIDPrefix = "CCB_REQ_ID:"
	DonePrefix  = "CCB_DONE:"
)

var (
	// Matches any *_DONE tag line (e.g., "CODEX_DONE", "GEMINI_DONE: 20260125-143000-123-12345")
	genericDoneTagRE = regexp.MustCompile(`^\s*[A-Z][A-Z0-9_]*_DONE(?:\s*:\s*\d{8}-\d{6}-\d{3}-\d+)?\s*$`)
	// Matches specifically CCB_DONE lines
	ccbDonePrefixRE  = regexp.MustCompile(`^\s*CCB_DONE\s*:`)
	anyCCBDoneLineRE = regexp.MustCompile(`^\s*CCB_DONE:\s*\d{8}-\d{6}-\d{3}-\d+\s*$`)
)

// isGenericDoneTag checks if a line is a generic *_DONE tag but NOT a CCB_DONE line.
func isGenericDoneTag(line string) bool {
	return genericDoneTagRE.MatchString(line) && !ccbDonePrefixRE.MatchString(line)
}

// MakeReqID generates a unique request ID with datetime-PID format.
// Format: YYYYMMDD-HHMMSS-mmm-PID (e.g., 20260125-143000-123-12345)
func MakeReqID() string {
	now := time.Now()
	ms := now.Nanosecond() / 1_000_000
	return fmt.Sprintf("%s-%03d-%d", now.Format("20060102-150405"), ms, os.Getpid())
}

// DoneLineRE returns a compiled regex that matches the CCB_DONE line for a specific req_id.
func DoneLineRE(reqID string) *regexp.Regexp {
	escaped := regexp.QuoteMeta(reqID)
	return regexp.MustCompile(`^\s*CCB_DONE:\s*` + escaped + `\s*$`)
}

// isTrailingNoiseLine checks if a line is trailing noise (blank or generic *_DONE tag).
func isTrailingNoiseLine(line string) bool {
	if strings.TrimSpace(line) == "" {
		return true
	}
	return isGenericDoneTag(line)
}

// StripTrailingMarkers removes trailing protocol/harness marker lines.
// Used for display commands (e.g., cpend) where we want a clean view.
func StripTrailingMarkers(text string) string {
	lines := splitLines(text)
	for len(lines) > 0 {
		last := lines[len(lines)-1]
		if isTrailingNoiseLine(last) || anyCCBDoneLineRE.MatchString(last) {
			lines = lines[:len(lines)-1]
			continue
		}
		break
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n\r\t ")
}

// IsDoneText checks if text contains the CCB_DONE marker for the given req_id.
func IsDoneText(text string, reqID string) bool {
	re := DoneLineRE(reqID)
	lines := splitLines(text)
	for i := len(lines) - 1; i >= 0; i-- {
		if isTrailingNoiseLine(lines[i]) {
			continue
		}
		return re.MatchString(lines[i])
	}
	return false
}

// StripDoneText removes the CCB_DONE marker and trailing noise from text.
func StripDoneText(text string, reqID string) string {
	lines := splitLines(text)
	if len(lines) == 0 {
		return ""
	}

	// Strip trailing noise
	for len(lines) > 0 && isTrailingNoiseLine(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}

	// Strip the DONE line itself
	re := DoneLineRE(reqID)
	if len(lines) > 0 && re.MatchString(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}

	// Strip more trailing noise
	for len(lines) > 0 && isTrailingNoiseLine(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}

	return strings.TrimRight(strings.Join(lines, "\n"), "\n\r\t ")
}

// WrapCodexPrompt wraps a message with CCB protocol markers for Codex.
func WrapCodexPrompt(message string, reqID string) string {
	message = strings.TrimRight(message, "\n\r\t ")
	return fmt.Sprintf(
		"%s %s\n\n%s\n\nIMPORTANT:\n- Reply normally.\n- Reply normally, in English.\n- End your reply with this exact final line (verbatim, on its own line):\n%s %s\n",
		ReqIDPrefix, reqID,
		message,
		DonePrefix, reqID,
	)
}

// splitLines splits text into lines, stripping trailing \n from each.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	raw := strings.Split(text, "\n")
	lines := make([]string, len(raw))
	for i, l := range raw {
		lines[i] = strings.TrimRight(l, "\r\n")
	}
	return lines
}

// CaskdRequest represents a request to the unified ask daemon.
type CaskdRequest struct {
	ClientID   string  `json:"client_id"`
	WorkDir    string  `json:"work_dir"`
	TimeoutS   float64 `json:"timeout_s"`
	Quiet      bool    `json:"quiet"`
	Message    string  `json:"message"`
	OutputPath string  `json:"output_path,omitempty"`
	ReqID      string  `json:"req_id,omitempty"`
	Caller     string  `json:"caller,omitempty"`
}

// CaskdResult represents a result from the unified ask daemon.
type CaskdResult struct {
	ExitCode     int    `json:"exit_code"`
	Reply        string `json:"reply"`
	ReqID        string `json:"req_id"`
	SessionKey   string `json:"session_key"`
	LogPath      string `json:"log_path,omitempty"`
	AnchorSeen   bool   `json:"anchor_seen"`
	DoneSeen     bool   `json:"done_seen"`
	FallbackScan bool   `json:"fallback_scan"`
	AnchorMs     *int   `json:"anchor_ms,omitempty"`
	DoneMs       *int   `json:"done_ms,omitempty"`
}
