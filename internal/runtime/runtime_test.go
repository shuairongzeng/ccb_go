package runtime

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunDir(t *testing.T) {
	// Clear override
	os.Unsetenv("CCB_RUN_DIR")

	dir := RunDir()
	if dir == "" {
		t.Fatal("RunDir returned empty string")
	}

	if runtime.GOOS == "windows" {
		if !strings.Contains(dir, "ccb") {
			t.Errorf("RunDir on Windows = %q, want to contain 'ccb'", dir)
		}
	} else {
		if !strings.Contains(dir, ".cache/ccb") && !strings.Contains(dir, "ccb") {
			t.Errorf("RunDir on Unix = %q, want to contain '.cache/ccb' or 'ccb'", dir)
		}
	}
}

func TestRunDirOverride(t *testing.T) {
	os.Setenv("CCB_RUN_DIR", "/tmp/ccb-test")
	defer os.Unsetenv("CCB_RUN_DIR")

	dir := RunDir()
	if dir != "/tmp/ccb-test" {
		t.Errorf("RunDir with override = %q, want /tmp/ccb-test", dir)
	}
}

func TestStateFilePath(t *testing.T) {
	path := StateFilePath("askd")
	if !strings.HasSuffix(path, "askd.json") {
		t.Errorf("StateFilePath(askd) = %q, want suffix askd.json", path)
	}

	path2 := StateFilePath("askd.json")
	if !strings.HasSuffix(path2, "askd.json") {
		t.Errorf("StateFilePath(askd.json) = %q, want suffix askd.json", path2)
	}
}

func TestLogPath(t *testing.T) {
	path := LogPath("askd")
	if !strings.HasSuffix(path, "askd.log") {
		t.Errorf("LogPath(askd) = %q, want suffix askd.log", path)
	}
}

func TestRandomToken(t *testing.T) {
	token := RandomToken()
	if len(token) != 32 {
		t.Errorf("RandomToken length = %d, want 32", len(token))
	}

	// Two tokens should be different
	token2 := RandomToken()
	if token == token2 {
		t.Error("RandomToken returned same value twice")
	}
}

func TestNormalizeConnectHost(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "127.0.0.1"},
		{"0.0.0.0", "127.0.0.1"},
		{"::", "::1"},
		{"[::]", "::1"},
		{"192.168.1.1", "192.168.1.1"},
		{"  127.0.0.1  ", "127.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeConnectHost(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeConnectHost(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestWriteLog(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	WriteLog(logFile, "hello world")
	WriteLog(logFile, "second line")

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "hello world") {
		t.Error("log missing first message")
	}
	if !strings.Contains(content, "second line") {
		t.Error("log missing second message")
	}
}
