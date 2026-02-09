package protocol

import (
	"testing"
)

func TestProviderSpecs(t *testing.T) {
	// Verify all daemon specs exist
	specs := AllDaemonSpecs()
	if len(specs) != 5 {
		t.Errorf("AllDaemonSpecs count = %d, want 5", len(specs))
	}

	// Verify all client specs exist
	clientSpecs := AllClientSpecs()
	if len(clientSpecs) != 5 {
		t.Errorf("AllClientSpecs count = %d, want 5", len(clientSpecs))
	}

	// Verify DaemonSpecByKey
	caskd := DaemonSpecByKey("caskd")
	if caskd == nil {
		t.Fatal("DaemonSpecByKey(caskd) returned nil")
	}
	if caskd.ProtocolPrefix != "cask" {
		t.Errorf("caskd prefix = %q, want cask", caskd.ProtocolPrefix)
	}

	// Verify ClientSpecByPrefix
	cask := ClientSpecByPrefix("cask")
	if cask == nil {
		t.Fatal("ClientSpecByPrefix(cask) returned nil")
	}
	if cask.SessionFilename != ".codex-session" {
		t.Errorf("cask session filename = %q, want .codex-session", cask.SessionFilename)
	}

	// Verify unknown returns nil
	if DaemonSpecByKey("unknown") != nil {
		t.Error("DaemonSpecByKey(unknown) should return nil")
	}
	if ClientSpecByPrefix("unknown") != nil {
		t.Error("ClientSpecByPrefix(unknown) should return nil")
	}
}

func TestProviderNameMap(t *testing.T) {
	expected := map[string]string{
		"codex":    "cask",
		"gemini":   "gask",
		"opencode": "oask",
		"claude":   "lask",
		"droid":    "dask",
	}

	for name, prefix := range expected {
		got, ok := ProviderNameMap[name]
		if !ok {
			t.Errorf("ProviderNameMap missing %q", name)
			continue
		}
		if got != prefix {
			t.Errorf("ProviderNameMap[%q] = %q, want %q", name, got, prefix)
		}
	}
}

func TestProtoByName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"codex", "codex"},
		{"cask", "codex"},
		{"gemini", "gemini"},
		{"gask", "gemini"},
		{"opencode", "opencode"},
		{"claude", "claude"},
		{"droid", "droid"},
	}

	for _, tt := range tests {
		proto := ProtoByName(tt.name)
		if proto == nil {
			t.Errorf("ProtoByName(%q) returned nil", tt.name)
			continue
		}
		if proto.Name != tt.expected {
			t.Errorf("ProtoByName(%q).Name = %q, want %q", tt.name, proto.Name, tt.expected)
		}
	}

	if ProtoByName("unknown") != nil {
		t.Error("ProtoByName(unknown) should return nil")
	}
}
