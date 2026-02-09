package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/claude_code_bridge/internal/terminal"
)

const (
	registryTTL     = 7 * 24 * time.Hour // 7 days
	registryVersion = 2
)

// PaneRegistry manages pane ID registrations for providers.
// Supports nested provider schema with metadata and alive verification.
type PaneRegistry struct {
	mu       sync.RWMutex
	filePath string
	data     *RegistryData
	backend  terminal.Backend
}

// RegistryData is the top-level registry structure.
type RegistryData struct {
	Providers map[string]map[string]*PaneEntry `json:"providers"` // provider → projectID → entry
	Version   int                              `json:"version"`
	// Legacy flat keys for migration
	Legacy map[string]string `json:"legacy,omitempty"`
}

// PaneEntry holds registration data for a single provider+project combination.
type PaneEntry struct {
	PaneID         string `json:"pane_id"`
	SessionID      string `json:"session_id,omitempty"`
	ClaudePane     string `json:"claude_pane,omitempty"`
	PaneTitleMarker string `json:"pane_title_marker,omitempty"`
	SessionPath    string `json:"session_path,omitempty"`
	WorkDir        string `json:"work_dir,omitempty"`
	Terminal       string `json:"terminal,omitempty"`
	UpdatedAt      int64  `json:"updated_at"`
}

// NewPaneRegistry creates a new PaneRegistry backed by a JSON file.
func NewPaneRegistry(filePath string) *PaneRegistry {
	r := &PaneRegistry{
		filePath: filePath,
		data: &RegistryData{
			Providers: make(map[string]map[string]*PaneEntry),
			Version:   registryVersion,
		},
	}
	r.load()
	return r
}

// SetBackend sets the terminal backend for alive checks.
func (r *PaneRegistry) SetBackend(b terminal.Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backend = b
}

// Get returns the pane ID for a provider and project.
func (r *PaneRegistry) Get(provider, projectID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if provMap, ok := r.data.Providers[provider]; ok {
		if entry, ok := provMap[projectID]; ok {
			return entry.PaneID
		}
	}
	// Fallback to legacy
	return r.data.Legacy[key(provider, projectID)]
}

// GetEntry returns the full PaneEntry for a provider and project.
func (r *PaneRegistry) GetEntry(provider, projectID string) *PaneEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if provMap, ok := r.data.Providers[provider]; ok {
		if entry, ok := provMap[projectID]; ok {
			return entry
		}
	}
	return nil
}

// Set registers a pane ID for a provider and project (simple form).
func (r *PaneRegistry) Set(provider, projectID, paneID string) {
	r.Upsert(provider, projectID, &PaneEntry{
		PaneID:    paneID,
		UpdatedAt: time.Now().Unix(),
	})
}

// Upsert updates or inserts a full PaneEntry for a provider and project.
func (r *PaneRegistry) Upsert(provider, projectID string, entry *PaneEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry.UpdatedAt == 0 {
		entry.UpdatedAt = time.Now().Unix()
	}

	if _, ok := r.data.Providers[provider]; !ok {
		r.data.Providers[provider] = make(map[string]*PaneEntry)
	}
	r.data.Providers[provider][projectID] = entry
	r.saveLocked()
}

// Remove removes a pane registration.
func (r *PaneRegistry) Remove(provider, projectID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if provMap, ok := r.data.Providers[provider]; ok {
		delete(provMap, projectID)
		if len(provMap) == 0 {
			delete(r.data.Providers, provider)
		}
	}
	delete(r.data.Legacy, key(provider, projectID))
	r.saveLocked()
}

// GetByProvider returns all pane entries for a given provider.
func (r *PaneRegistry) GetByProvider(provider string) map[string]*PaneEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]*PaneEntry)
	if provMap, ok := r.data.Providers[provider]; ok {
		for k, v := range provMap {
			result[k] = v
		}
	}
	return result
}

// GetBySessionID finds a provider and entry by session ID.
func (r *PaneRegistry) GetBySessionID(sessionID string) (string, *PaneEntry) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for provider, provMap := range r.data.Providers {
		for _, entry := range provMap {
			if entry.SessionID == sessionID {
				return provider, entry
			}
		}
	}
	return "", nil
}

// GetByClaudePane finds a provider and entry by Claude pane ID.
func (r *PaneRegistry) GetByClaudePane(claudePane string) (string, *PaneEntry) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for provider, provMap := range r.data.Providers {
		for _, entry := range provMap {
			if entry.ClaudePane == claudePane {
				return provider, entry
			}
		}
	}
	return "", nil
}

// GetByProjectAndProvider returns the entry for a specific project and provider.
func (r *PaneRegistry) GetByProjectAndProvider(projectID, provider string) *PaneEntry {
	return r.GetEntry(provider, projectID)
}

// VerifyAlive checks if the pane for a provider+project is still alive.
func (r *PaneRegistry) VerifyAlive(provider, projectID string) bool {
	r.mu.RLock()
	b := r.backend
	r.mu.RUnlock()

	entry := r.GetEntry(provider, projectID)
	if entry == nil || entry.PaneID == "" {
		return false
	}

	if b == nil {
		return true // can't verify without backend, assume alive
	}

	return b.IsAlive(entry.PaneID)
}

// PruneStalePanes removes entries older than the given TTL.
// Returns the number of entries removed.
func (r *PaneRegistry) PruneStalePanes(ttl time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ttl == 0 {
		ttl = registryTTL
	}

	cutoff := time.Now().Add(-ttl).Unix()
	removed := 0

	for provider, provMap := range r.data.Providers {
		for projectID, entry := range provMap {
			if entry.UpdatedAt > 0 && entry.UpdatedAt < cutoff {
				delete(provMap, projectID)
				removed++
			}
		}
		if len(provMap) == 0 {
			delete(r.data.Providers, provider)
		}
	}

	if removed > 0 {
		r.saveLocked()
	}

	return removed
}

// PruneDeadPanes removes entries whose panes are no longer alive.
// Returns the number of entries removed.
func (r *PaneRegistry) PruneDeadPanes() int {
	r.mu.Lock()
	b := r.backend
	r.mu.Unlock()

	if b == nil {
		return 0
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	removed := 0
	for provider, provMap := range r.data.Providers {
		for projectID, entry := range provMap {
			if entry.PaneID != "" && !b.IsAlive(entry.PaneID) {
				delete(provMap, projectID)
				removed++
			}
		}
		if len(provMap) == 0 {
			delete(r.data.Providers, provider)
		}
	}

	if removed > 0 {
		r.saveLocked()
	}

	return removed
}

// AllEntries returns all entries across all providers.
func (r *PaneRegistry) AllEntries() map[string]map[string]*PaneEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]map[string]*PaneEntry)
	for provider, provMap := range r.data.Providers {
		result[provider] = make(map[string]*PaneEntry)
		for k, v := range provMap {
			result[provider][k] = v
		}
	}
	return result
}

// MigrateLegacy migrates legacy flat-key entries to the nested providers schema.
func (r *PaneRegistry) MigrateLegacy() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.data.Legacy) == 0 {
		return
	}

	for k, paneID := range r.data.Legacy {
		parts := strings.SplitN(k, ":", 2)
		if len(parts) != 2 {
			continue
		}
		provider := parts[0]
		projectID := parts[1]

		if _, ok := r.data.Providers[provider]; !ok {
			r.data.Providers[provider] = make(map[string]*PaneEntry)
		}

		// Don't overwrite existing entries
		if _, exists := r.data.Providers[provider][projectID]; !exists {
			r.data.Providers[provider][projectID] = &PaneEntry{
				PaneID:    paneID,
				UpdatedAt: time.Now().Unix(),
			}
		}
	}

	// Clear legacy data after migration
	r.data.Legacy = nil
	r.data.Version = registryVersion
	r.saveLocked()
}

// key builds the legacy registry key.
func key(provider, projectID string) string {
	return provider + ":" + projectID
}

// load reads the registry from disk.
func (r *PaneRegistry) load() {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return
	}

	// Try new format first
	var newData RegistryData
	if err := json.Unmarshal(data, &newData); err == nil && newData.Version >= 2 {
		if newData.Providers == nil {
			newData.Providers = make(map[string]map[string]*PaneEntry)
		}
		r.data = &newData
		return
	}

	// Try legacy flat format
	var legacyData map[string]string
	if err := json.Unmarshal(data, &legacyData); err == nil {
		r.data = &RegistryData{
			Providers: make(map[string]map[string]*PaneEntry),
			Version:   1,
			Legacy:    legacyData,
		}
		// Auto-migrate
		r.migrateLegacyLocked()
		return
	}
}

// migrateLegacyLocked migrates legacy data (caller must hold lock).
func (r *PaneRegistry) migrateLegacyLocked() {
	if len(r.data.Legacy) == 0 {
		return
	}

	for k, paneID := range r.data.Legacy {
		parts := strings.SplitN(k, ":", 2)
		if len(parts) != 2 {
			continue
		}
		provider := parts[0]
		projectID := parts[1]

		if _, ok := r.data.Providers[provider]; !ok {
			r.data.Providers[provider] = make(map[string]*PaneEntry)
		}
		r.data.Providers[provider][projectID] = &PaneEntry{
			PaneID:    paneID,
			UpdatedAt: time.Now().Unix(),
		}
	}

	r.data.Legacy = nil
	r.data.Version = registryVersion
	r.saveLocked()
}

// save writes the registry to disk.
func (r *PaneRegistry) save() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveLocked()
}

// saveLocked writes the registry to disk (caller must hold lock).
func (r *PaneRegistry) saveLocked() {
	dir := filepath.Dir(r.filePath)
	os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return
	}
	tmpFile := r.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return
	}
	os.Rename(tmpFile, r.filePath)
}
