package protocol

import (
	"strings"
	"testing"
)

func TestMakeReqID(t *testing.T) {
	id := MakeReqID()
	if id == "" {
		t.Fatal("MakeReqID returned empty string")
	}
	// Format: YYYYMMDD-HHMMSS-mmm-PID
	parts := strings.Split(id, "-")
	if len(parts) != 4 {
		t.Errorf("MakeReqID format wrong: %q has %d parts, want 4", id, len(parts))
	}
	if len(parts[0]) != 8 {
		t.Errorf("date part length = %d, want 8", len(parts[0]))
	}
	if len(parts[1]) != 6 {
		t.Errorf("time part length = %d, want 6", len(parts[1]))
	}
	if len(parts[2]) != 3 {
		t.Errorf("ms part length = %d, want 3", len(parts[2]))
	}

	// Two calls should produce different IDs (or at least not panic)
	id2 := MakeReqID()
	_ = id2
}

func TestWrapCodexPrompt(t *testing.T) {
	msg := "Hello world"
	reqID := "20260125-143000-123-12345"
	wrapped := WrapCodexPrompt(msg, reqID)

	if !strings.Contains(wrapped, ReqIDPrefix+" "+reqID) {
		t.Error("wrapped prompt missing REQ_ID marker")
	}
	if !strings.Contains(wrapped, msg) {
		t.Error("wrapped prompt missing original message")
	}
	if !strings.Contains(wrapped, DonePrefix+" "+reqID) {
		t.Error("wrapped prompt missing DONE marker")
	}
}

func TestIsDoneText(t *testing.T) {
	reqID := "20260125-143000-123-12345"

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"exact done line", "CCB_DONE: " + reqID, true},
		{"done with leading space", "  CCB_DONE: " + reqID, true},
		{"done with trailing blank", "CCB_DONE: " + reqID + "\n\n", true},
		{"no done", "Hello world", false},
		{"wrong req_id", "CCB_DONE: 99999999-999999-999-99999", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDoneText(tt.text, reqID)
			if got != tt.expected {
				t.Errorf("IsDoneText(%q) = %v, want %v", tt.text, got, tt.expected)
			}
		})
	}
}

func TestStripDoneText(t *testing.T) {
	reqID := "20260125-143000-123-12345"

	text := "Here is my reply.\n\nCCB_DONE: " + reqID + "\n"
	got := StripDoneText(text, reqID)
	if strings.Contains(got, "CCB_DONE") {
		t.Errorf("StripDoneText still contains CCB_DONE: %q", got)
	}
	if !strings.Contains(got, "Here is my reply.") {
		t.Errorf("StripDoneText lost the reply: %q", got)
	}
}

func TestStripTrailingMarkers(t *testing.T) {
	reqID := "20260125-143000-123-12345"
	text := "Reply content\nCCB_DONE: " + reqID + "\n\n"
	got := StripTrailingMarkers(text)
	if strings.Contains(got, "CCB_DONE") {
		t.Errorf("StripTrailingMarkers still contains CCB_DONE: %q", got)
	}
	if !strings.Contains(got, "Reply content") {
		t.Errorf("StripTrailingMarkers lost content: %q", got)
	}
}

func TestIsTrailingNoiseLine(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"", true},
		{"   ", true},
		{"CODEX_DONE", true},
		{"GEMINI_DONE: 20260125-143000-123-12345", true},
		{"CCB_DONE: 20260125-143000-123-12345", false}, // CCB_DONE is NOT noise
		{"Hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := isTrailingNoiseLine(tt.line)
			if got != tt.expected {
				t.Errorf("isTrailingNoiseLine(%q) = %v, want %v", tt.line, got, tt.expected)
			}
		})
	}
}

func TestDoneLineRE(t *testing.T) {
	reqID := "20260125-143000-123-12345"
	re := DoneLineRE(reqID)

	if !re.MatchString("CCB_DONE: " + reqID) {
		t.Error("DoneLineRE should match exact done line")
	}
	if !re.MatchString("  CCB_DONE:  " + reqID + "  ") {
		t.Error("DoneLineRE should match with whitespace")
	}
	if re.MatchString("CCB_DONE: wrong-id") {
		t.Error("DoneLineRE should not match wrong req_id")
	}
}
