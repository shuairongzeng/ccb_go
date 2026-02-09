package launcher

import (
	"testing"
)

func TestParseProviders(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "comma separated",
			args:     []string{"codex,claude"},
			expected: []string{"codex", "claude"},
		},
		{
			name:     "space separated",
			args:     []string{"codex", "claude"},
			expected: []string{"codex", "claude"},
		},
		{
			name:     "mixed",
			args:     []string{"codex,gemini", "claude"},
			expected: []string{"codex", "gemini", "claude"},
		},
		{
			name:     "duplicates removed",
			args:     []string{"codex,codex,claude"},
			expected: []string{"codex", "claude"},
		},
		{
			name:     "case insensitive",
			args:     []string{"Codex,CLAUDE"},
			expected: []string{"codex", "claude"},
		},
		{
			name:     "all providers",
			args:     []string{"codex,gemini,opencode,claude,droid"},
			expected: []string{"codex", "gemini", "opencode", "claude", "droid"},
		},
		{
			name:     "empty",
			args:     []string{},
			expected: nil,
		},
		{
			name:     "whitespace trimmed",
			args:     []string{" codex , claude "},
			expected: []string{"codex", "claude"},
		},
		{
			name:     "single provider",
			args:     []string{"codex"},
			expected: []string{"codex"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseProviders(tt.args)
			if len(got) != len(tt.expected) {
				t.Fatalf("ParseProviders(%v) = %v, want %v", tt.args, got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Fatalf("ParseProviders(%v)[%d] = %q, want %q", tt.args, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestBuildStartCommand(t *testing.T) {
	tests := []struct {
		provider string
		auto     bool
		resume   bool
		contains []string // substrings the command must contain
	}{
		{
			provider: "codex",
			auto:     false,
			contains: []string{"codex"},
		},
		{
			provider: "codex",
			auto:     true,
			contains: []string{"codex", "approval_policy", "never", "danger-full-access"},
		},
		{
			provider: "codex",
			resume:   true,
			contains: []string{"codex", "resume", "--last"},
		},
		{
			provider: "gemini",
			auto:     false,
			contains: []string{"gemini"},
		},
		{
			provider: "gemini",
			auto:     true,
			contains: []string{"gemini", "--yolo"},
		},
		{
			provider: "gemini",
			resume:   true,
			contains: []string{"gemini", "--resume", "latest"},
		},
		{
			provider: "claude",
			auto:     false,
			contains: []string{"claude"},
		},
		{
			provider: "claude",
			auto:     true,
			contains: []string{"claude", "--dangerously-skip-permissions"},
		},
		{
			provider: "claude",
			resume:   true,
			contains: []string{"claude", "--continue"},
		},
		{
			provider: "opencode",
			auto:     false,
			contains: []string{"opencode"},
		},
		{
			provider: "opencode",
			resume:   true,
			contains: []string{"opencode", "--continue"},
		},
		{
			provider: "droid",
			resume:   true,
			contains: []string{"droid", "-r"},
		},
	}

	for _, tt := range tests {
		name := tt.provider
		if tt.auto {
			name += "_auto"
		}
		if tt.resume {
			name += "_resume"
		}
		t.Run(name, func(t *testing.T) {
			cmd, err := BuildStartCommand(tt.provider, tt.auto, tt.resume)
			if err != nil {
				t.Fatalf("BuildStartCommand(%q, auto=%v, resume=%v) error: %v", tt.provider, tt.auto, tt.resume, err)
			}
			for _, sub := range tt.contains {
				if !containsStr(cmd, sub) {
					t.Errorf("BuildStartCommand(%q, auto=%v, resume=%v) = %q, missing %q", tt.provider, tt.auto, tt.resume, cmd, sub)
				}
			}
		})
	}
}

func TestBuildStartCommandUnknown(t *testing.T) {
	_, err := BuildStartCommand("unknown_provider", false, false)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestIsValidProvider(t *testing.T) {
	valid := []string{"codex", "gemini", "opencode", "claude", "droid"}
	for _, p := range valid {
		if !isValidProvider(p) {
			t.Errorf("isValidProvider(%q) = false, want true", p)
		}
	}

	invalid := []string{"unknown", "chatgpt", "copilot", ""}
	for _, p := range invalid {
		if isValidProvider(p) {
			t.Errorf("isValidProvider(%q) = true, want false", p)
		}
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
