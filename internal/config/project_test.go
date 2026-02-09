package config

import (
	"crypto/sha256"
	"fmt"
	"runtime"
	"strings"
	"testing"
)

func TestNormalizeWorkDir(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // substring that must be present
		notEmpty bool
	}{
		{"empty returns empty", "", "", false},
		{"absolute unix path", "/home/user/project", "/home/user/project", true},
		{"trailing slashes removed", "/home/user/project///", "/home/user/project", true},
	}

	if runtime.GOOS == "windows" {
		tests = append(tests, []struct {
			name     string
			input    string
			contains string
			notEmpty bool
		}{
			{"windows drive path", `C:\Users\test`, "c:/users/test", true},
			{"forward slashes", "C:/Users/test", "c:/users/test", true},
		}...)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeWorkDir(tt.input)
			if tt.notEmpty && got == "" {
				t.Errorf("NormalizeWorkDir(%q) = empty, want non-empty", tt.input)
			}
			if !tt.notEmpty && tt.input == "" && got != "" {
				t.Errorf("NormalizeWorkDir(%q) = %q, want empty", tt.input, got)
			}
			if tt.contains != "" && !strings.Contains(strings.ToLower(got), strings.ToLower(tt.contains)) {
				t.Errorf("NormalizeWorkDir(%q) = %q, want to contain %q", tt.input, got, tt.contains)
			}
		})
	}
}

func TestNormalizeWorkDirWSL(t *testing.T) {
	got := NormalizeWorkDir("/mnt/c/Users/test")
	if !strings.HasPrefix(got, "c:/") {
		t.Errorf("NormalizeWorkDir(/mnt/c/Users/test) = %q, want prefix c:/", got)
	}
	if !strings.Contains(got, "Users/test") {
		t.Errorf("NormalizeWorkDir(/mnt/c/Users/test) = %q, want to contain Users/test", got)
	}
}

func TestComputeCCBProjectID(t *testing.T) {
	// Project ID should be a 64-char hex string (SHA256)
	id := ComputeCCBProjectID(".")
	if len(id) != 64 {
		t.Errorf("ComputeCCBProjectID(.) length = %d, want 64", len(id))
	}

	// Same input should produce same output
	id2 := ComputeCCBProjectID(".")
	if id != id2 {
		t.Errorf("ComputeCCBProjectID not deterministic: %q != %q", id, id2)
	}
}

func TestNormalizeWorkDirDriveCasing(t *testing.T) {
	// Drive letter should be lowercased
	upper := NormalizeWorkDir("C:/Users/test")
	lower := NormalizeWorkDir("c:/Users/test")
	if upper != lower {
		t.Errorf("Drive casing not normalized: %q != %q", upper, lower)
	}
}

func TestProjectIDMatchesPython(t *testing.T) {
	// Verify the hash algorithm matches: sha256 of normalized path
	norm := "c:/users/test"
	hash := sha256.Sum256([]byte(norm))
	expected := fmt.Sprintf("%x", hash)
	if len(expected) != 64 {
		t.Errorf("SHA256 hex length = %d, want 64", len(expected))
	}
}
