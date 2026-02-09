package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPaneRegistryBasicCRUD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	r := NewPaneRegistry(path)

	// Set
	r.Set("codex", "proj1", "%10")
	if got := r.Get("codex", "proj1"); got != "%10" {
		t.Fatalf("expected %%10, got %q", got)
	}

	// Update
	r.Set("codex", "proj1", "%20")
	if got := r.Get("codex", "proj1"); got != "%20" {
		t.Fatalf("expected %%20, got %q", got)
	}

	// Remove
	r.Remove("codex", "proj1")
	if got := r.Get("codex", "proj1"); got != "" {
		t.Fatalf("expected empty after remove, got %q", got)
	}
}

func TestPaneRegistryUpsert(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	r := NewPaneRegistry(path)

	entry := &PaneEntry{
		PaneID:    "%15",
		SessionID: "sess-123",
		WorkDir:   "/home/user/project",
		UpdatedAt: time.Now().Unix(),
	}
	r.Upsert("gemini", "proj2", entry)

	got := r.GetEntry("gemini", "proj2")
	if got == nil {
		t.Fatal("expected entry, got nil")
	}
	if got.PaneID != "%15" {
		t.Fatalf("expected %%15, got %q", got.PaneID)
	}
	if got.SessionID != "sess-123" {
		t.Fatalf("expected sess-123, got %q", got.SessionID)
	}
}

func TestPaneRegistryGetBySessionID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	r := NewPaneRegistry(path)
	r.Upsert("claude", "proj1", &PaneEntry{
		PaneID:    "%5",
		SessionID: "session-abc",
	})

	provider, entry := r.GetBySessionID("session-abc")
	if provider != "claude" {
		t.Fatalf("expected provider 'claude', got %q", provider)
	}
	if entry == nil || entry.PaneID != "%5" {
		t.Fatal("expected entry with pane %5")
	}

	// Not found
	provider, entry = r.GetBySessionID("nonexistent")
	if provider != "" || entry != nil {
		t.Fatal("expected nil for nonexistent session")
	}
}

func TestPaneRegistryGetByClaudePane(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	r := NewPaneRegistry(path)
	r.Upsert("codex", "proj1", &PaneEntry{
		PaneID:     "%10",
		ClaudePane: "%3",
	})

	provider, entry := r.GetByClaudePane("%3")
	if provider != "codex" {
		t.Fatalf("expected provider 'codex', got %q", provider)
	}
	if entry == nil || entry.PaneID != "%10" {
		t.Fatal("expected entry with pane %10")
	}
}

func TestPaneRegistryGetByProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	r := NewPaneRegistry(path)
	r.Set("codex", "proj1", "%10")
	r.Set("codex", "proj2", "%11")
	r.Set("gemini", "proj1", "%20")

	entries := r.GetByProvider("codex")
	if len(entries) != 2 {
		t.Fatalf("expected 2 codex entries, got %d", len(entries))
	}

	entries = r.GetByProvider("gemini")
	if len(entries) != 1 {
		t.Fatalf("expected 1 gemini entry, got %d", len(entries))
	}
}

func TestPaneRegistryPruneStalePanes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	r := NewPaneRegistry(path)

	// Add an old entry
	r.Upsert("codex", "old-proj", &PaneEntry{
		PaneID:    "%1",
		UpdatedAt: time.Now().Add(-8 * 24 * time.Hour).Unix(), // 8 days ago
	})

	// Add a fresh entry
	r.Upsert("codex", "new-proj", &PaneEntry{
		PaneID:    "%2",
		UpdatedAt: time.Now().Unix(),
	})

	removed := r.PruneStalePanes(7 * 24 * time.Hour)
	if removed != 1 {
		t.Fatalf("expected 1 removed, got %d", removed)
	}

	if r.Get("codex", "old-proj") != "" {
		t.Fatal("old entry should be removed")
	}
	if r.Get("codex", "new-proj") != "%2" {
		t.Fatal("new entry should still exist")
	}
}

func TestPaneRegistryPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	// Write
	r1 := NewPaneRegistry(path)
	r1.Set("codex", "proj1", "%10")
	r1.Upsert("gemini", "proj2", &PaneEntry{
		PaneID:    "%20",
		SessionID: "sess-xyz",
	})

	// Read back
	r2 := NewPaneRegistry(path)
	if got := r2.Get("codex", "proj1"); got != "%10" {
		t.Fatalf("expected %%10 after reload, got %q", got)
	}
	entry := r2.GetEntry("gemini", "proj2")
	if entry == nil || entry.SessionID != "sess-xyz" {
		t.Fatal("expected entry with session sess-xyz after reload")
	}
}

func TestPaneRegistryLegacyMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	// Write legacy format
	legacyData := map[string]string{
		"codex:proj1":  "%10",
		"gemini:proj2": "%20",
	}
	data, _ := json.MarshalIndent(legacyData, "", "  ")
	os.WriteFile(path, data, 0644)

	// Load should auto-migrate
	r := NewPaneRegistry(path)

	if got := r.Get("codex", "proj1"); got != "%10" {
		t.Fatalf("expected %%10 after migration, got %q", got)
	}
	if got := r.Get("gemini", "proj2"); got != "%20" {
		t.Fatalf("expected %%20 after migration, got %q", got)
	}

	// Verify new format was written
	data2, _ := os.ReadFile(path)
	var newData RegistryData
	if err := json.Unmarshal(data2, &newData); err != nil {
		t.Fatalf("failed to parse migrated file: %v", err)
	}
	if newData.Version != registryVersion {
		t.Fatalf("expected version %d, got %d", registryVersion, newData.Version)
	}
	if newData.Legacy != nil {
		t.Fatal("legacy data should be nil after migration")
	}
}

func TestPaneRegistryAllEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	r := NewPaneRegistry(path)
	r.Set("codex", "p1", "%1")
	r.Set("gemini", "p2", "%2")
	r.Set("claude", "p3", "%3")

	all := r.AllEntries()
	if len(all) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(all))
	}
}

func TestSessionResolverFromEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	r := NewPaneRegistry(path)
	r.Upsert("claude", "proj1", &PaneEntry{
		PaneID:      "%5",
		SessionID:   "env-session-123",
		SessionPath: "/tmp/log.jsonl",
	})

	resolver := NewSessionResolver(r, nil)

	t.Setenv("CCB_SESSION_ID", "env-session-123")

	result, err := resolver.Resolve("/some/dir")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Source != "env" {
		t.Fatalf("expected source 'env', got %q", result.Source)
	}
	if result.PaneID != "%5" {
		t.Fatalf("expected pane %%5, got %q", result.PaneID)
	}
}

func TestSessionResolverFromSessionFile(t *testing.T) {
	dir := t.TempDir()

	// Create .ccb_config directory with session file
	ccbDir := filepath.Join(dir, ".ccb_config")
	os.MkdirAll(ccbDir, 0755)
	os.WriteFile(filepath.Join(ccbDir, ".claude-session"), []byte("%42"), 0644)

	regPath := filepath.Join(dir, "registry.json")
	r := NewPaneRegistry(regPath)
	resolver := NewSessionResolver(r, nil)

	// Clear env to avoid stage 1
	t.Setenv("CCB_SESSION_ID", "")
	t.Setenv("TMUX_PANE", "")
	t.Setenv("WEZTERM_PANE", "")

	result, err := resolver.Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Source != "session_file" {
		t.Fatalf("expected source 'session_file', got %q", result.Source)
	}
	if result.PaneID != "%42" {
		t.Fatalf("expected pane %%42, got %q", result.PaneID)
	}
}
