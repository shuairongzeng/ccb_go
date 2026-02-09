package output

import (
	"os"
	"testing"
)

func TestNormalizeMessageParts(t *testing.T) {
	tests := []struct {
		parts    []string
		expected string
	}{
		{[]string{"hello", "world"}, "hello world"},
		{[]string{"  hello  ", "  world  "}, "hello     world"},
		{[]string{}, ""},
		{[]string{"single"}, "single"},
	}

	for _, tt := range tests {
		got := NormalizeMessageParts(tt.parts)
		if got != tt.expected {
			t.Errorf("NormalizeMessageParts(%v) = %q, want %q", tt.parts, got, tt.expected)
		}
	}
}

func TestDecodeStdinBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"empty", nil, ""},
		{"plain utf8", []byte("hello"), "hello"},
		{"utf8 bom", []byte{0xEF, 0xBB, 0xBF, 'h', 'e', 'l', 'l', 'o'}, "hello"},
		{"utf16le bom", []byte{0xFF, 0xFE, 'h', 0, 'i', 0}, "hi"},
		{"utf16be bom", []byte{0xFE, 0xFF, 0, 'h', 0, 'i'}, "hi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecodeStdinBytes(tt.input)
			if got != tt.expected {
				t.Errorf("DecodeStdinBytes = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAtomicWriteText(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/test.txt"

	err := AtomicWriteText(path, "hello world")
	if err != nil {
		t.Fatalf("AtomicWriteText failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("AtomicWriteText content = %q, want %q", string(data), "hello world")
	}
}
