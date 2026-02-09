package config

import (
	"os"
	"testing"
)

func TestEnvBool(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		defVal   bool
		expected bool
	}{
		{"empty uses default true", "", true, true},
		{"empty uses default false", "", false, false},
		{"true", "true", false, true},
		{"TRUE", "TRUE", false, true},
		{"1", "1", false, true},
		{"yes", "yes", false, true},
		{"on", "on", false, true},
		{"false", "false", true, false},
		{"FALSE", "FALSE", true, false},
		{"0", "0", true, false},
		{"no", "no", true, false},
		{"off", "off", true, false},
		{"invalid uses default", "maybe", true, true},
		{"whitespace trimmed", "  true  ", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "CCB_TEST_BOOL_" + tt.name
			if tt.envVal != "" {
				os.Setenv(key, tt.envVal)
				defer os.Unsetenv(key)
			} else {
				os.Unsetenv(key)
			}
			got := EnvBool(key, tt.defVal)
			if got != tt.expected {
				t.Errorf("EnvBool(%q, %v) = %v, want %v", tt.envVal, tt.defVal, got, tt.expected)
			}
		})
	}
}

func TestEnvInt(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		defVal   int
		expected int
	}{
		{"empty uses default", "", 42, 42},
		{"valid int", "123", 0, 123},
		{"negative int", "-5", 0, -5},
		{"invalid uses default", "abc", 99, 99},
		{"whitespace trimmed", "  456  ", 0, 456},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "CCB_TEST_INT_" + tt.name
			if tt.envVal != "" {
				os.Setenv(key, tt.envVal)
				defer os.Unsetenv(key)
			} else {
				os.Unsetenv(key)
			}
			got := EnvInt(key, tt.defVal)
			if got != tt.expected {
				t.Errorf("EnvInt(%q, %d) = %d, want %d", tt.envVal, tt.defVal, got, tt.expected)
			}
		})
	}
}

func TestEnvStr(t *testing.T) {
	key := "CCB_TEST_STR"

	os.Unsetenv(key)
	if got := EnvStr(key, "default"); got != "default" {
		t.Errorf("EnvStr empty = %q, want %q", got, "default")
	}

	os.Setenv(key, "  hello  ")
	defer os.Unsetenv(key)
	if got := EnvStr(key, "default"); got != "hello" {
		t.Errorf("EnvStr set = %q, want %q", got, "hello")
	}
}
