package mods

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// ─── Mod Syncing System ────────────────────────────────────────────
// When a client connects to a host, the host sends its mod manifest.
// The client can choose to download/load the host's mods.
// This ensures everyone in the session has the same mods enabled.

const ModsDir = "mods"

// ModInfo describes a single mod file.
type ModInfo struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Filename string `json:"filename"`
	Hash     string `json:"hash"` // SHA256 for integrity
	Size     int64  `json:"size"`
	Enabled  bool   `json:"enabled"`
}

// ModManifest is the list of mods the host has enabled.
type ModManifest struct {
	HostID    string    `json:"host_id"`
	ModCount  int       `json:"mod_count"`
	Mods      []ModInfo `json:"mods"`
}

// ModSyncManager handles mod discovery, syncing, and loading.
type ModSyncManager struct {
	mu          sync.RWMutex
	ModsDir     string
	LocalMods   map[string]*ModInfo // filename -> info
	EnabledMods map[string]bool     // filename -> enabled
	HostManifest *ModManifest       // received from host
	SyncedMods  []string            // mods downloaded from host
	AutoAccept  bool                // auto-accept host mods

	// When true, Steam achievements are permanently disabled
	// for this play session to prevent cheating.
	AchievementsLocked bool
}

func NewModSyncManager() *ModSyncManager {
	msm := &ModSyncManager{
		ModsDir:     ModsDir,
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	msm.scanLocalMods()
	return msm
}

// scanLocalMods scans the mods directory and indexes all mods.
// If ANY mods are found, achievements are locked for this session.
func (msm *ModSyncManager) scanLocalMods() {
	entries, err := os.ReadDir(msm.ModsDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(msm.ModsDir, entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		hash, _ := hashFile(path)
		mod := &ModInfo{
			Filename: entry.Name(),
			Hash:     hash,
			Size:     info.Size(),
			Enabled:  true, // default enabled
		}
		// Try to extract name and version from the mod file
		if data, err := os.ReadFile(path); err == nil {
			var parsed struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			}
			if json.Unmarshal(data, &parsed) == nil {
				mod.Name = parsed.Name
				mod.Version = parsed.Version
			}
		}
			msm.LocalMods[entry.Name()] = mod
			msm.EnabledMods[entry.Name()] = true
			// Found a mod — lock achievements for this session
			msm.AchievementsLocked = true
		}
}

// hashFile computes SHA256 of a file.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// GetHostManifest creates the manifest to send to clients.
func (msm *ModSyncManager) GetHostManifest(hostID string) *ModManifest {
	msm.mu.RLock()
	defer msm.mu.RUnlock()

	manifest := &ModManifest{
		HostID:   hostID,
		ModCount: 0,
	}
	for filename, info := range msm.LocalMods {
		if msm.EnabledMods[filename] {
			manifest.Mods = append(manifest.Mods, *info)
			manifest.ModCount++
		}
	}
	sort.Slice(manifest.Mods, func(i, j int) bool {
		return manifest.Mods[i].Filename < manifest.Mods[j].Filename
	})
	return manifest
}

// ReceiveHostManifest stores the host's mod manifest.
// Returns mods the client needs to download.
func (msm *ModSyncManager) ReceiveHostManifest(manifest *ModManifest) []ModInfo {
	msm.mu.Lock()
	defer msm.mu.Unlock()

	msm.HostManifest = manifest
	var needed []ModInfo

	for _, hostMod := range manifest.Mods {
		localMod, exists := msm.LocalMods[hostMod.Filename]
		if !exists || localMod.Hash != hostMod.Hash {
			needed = append(needed, hostMod)
		}
	}
	return needed
}

// AcceptHostMods accepts the host's mod configuration.
// Returns the list of mods that need to be loaded/updated.
func (msm *ModSyncManager) AcceptHostMods() []string {
	msm.mu.Lock()
	defer msm.mu.Unlock()

	if msm.HostManifest == nil {
		return nil
	}

	// Disable all local mods not in host manifest
	hostMods := make(map[string]bool)
	for _, m := range msm.HostManifest.Mods {
		hostMods[m.Filename] = true
	}
	for filename := range msm.EnabledMods {
		if !hostMods[filename] {
			msm.EnabledMods[filename] = false
		}
	}

	// Enable all host mods
	var toLoad []string
	for _, m := range msm.HostManifest.Mods {
		msm.EnabledMods[m.Filename] = true
		toLoad = append(toLoad, m.Filename)
	}
	msm.SyncedMods = toLoad
	return toLoad
}

// ReceiveModData saves a mod file received from the host.
func (msm *ModSyncManager) ReceiveModData(filename string, data []byte) error {
	path := filepath.Join(msm.ModsDir, filename)
	if err := os.MkdirAll(msm.ModsDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	// Update local index
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	msm.mu.Lock()
	msm.LocalMods[filename] = &ModInfo{
		Filename: filename,
		Hash:     hash,
		Size:     int64(len(data)),
		Enabled:  true,
	}
	msm.mu.Unlock()
	return nil
}

// GetEnabledMods returns the list of mod files to load.
func (msm *ModSyncManager) GetEnabledMods() []string {
	msm.mu.RLock()
	defer msm.mu.RUnlock()

	var enabled []string
	for filename, active := range msm.EnabledMods {
		if active {
			enabled = append(enabled, filename)
		}
	}
	return enabled
}

// IsModEnabled checks if a specific mod is enabled.
func (msm *ModSyncManager) IsModEnabled(filename string) bool {
	msm.mu.RLock()
	defer msm.mu.RUnlock()
	return msm.EnabledMods[filename]
}

// ToggleMod enables or disables a local mod.
func (msm *ModSyncManager) ToggleMod(filename string) bool {
	msm.mu.Lock()
	defer msm.mu.Unlock()
	current := msm.EnabledMods[filename]
	msm.EnabledMods[filename] = !current
	return !current
}

// ModCount returns the number of local mods.
func (msm *ModSyncManager) ModCount() int {
	msm.mu.RLock()
	defer msm.mu.RUnlock()
	return len(msm.LocalMods)
}

// ─── Network Integration ───────────────────────────────────────────
// These messages integrate with internal/net package.

// ModSyncMessage wraps mod sync data for network transmission.
type ModSyncMessage struct {
	Type     string          `json:"type"` // "manifest", "mod_data", "accept"
	Manifest *ModManifest    `json:"manifest,omitempty"`
	Filename string          `json:"filename,omitempty"`
	Data     []byte          `json:"data,omitempty"`
	Hash     string          `json:"hash,omitempty"`
}

// SerializeManifest serializes the host manifest for sending.
func SerializeManifest(manifest *ModManifest) ([]byte, error) {
	return json.Marshal(manifest)
}

// DeserializeManifest deserializes a received manifest.
func DeserializeManifest(data []byte) (*ModManifest, error) {
	var manifest ModManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// ModConflict describes a mod version mismatch between host and client.
type ModConflict struct {
	Filename     string `json:"filename"`
	HostVersion  string `json:"host_version"`
	ClientVersion string `json:"client_version"`
	AutoResolved bool   `json:"auto_resolved"`
}

// ResolveModConflicts checks for mod conflicts and suggests resolution.
func (msm *ModSyncManager) ResolveModConflicts() []ModConflict {
	msm.mu.RLock()
	defer msm.mu.RUnlock()

	if msm.HostManifest == nil {
		return nil
	}

	var conflicts []ModConflict
	for _, hostMod := range msm.HostManifest.Mods {
		if localMod, exists := msm.LocalMods[hostMod.Filename]; exists {
			if localMod.Hash != hostMod.Hash {
				conflicts = append(conflicts, ModConflict{
					Filename:     hostMod.Filename,
					HostVersion:  hostMod.Version,
					ClientVersion: localMod.Version,
				})
			}
		}
	}
	return conflicts
}

// Init function prints mod system info on startup.
func init() {
	log.Printf("[modsync] Mod syncing system initialized")
	log.Printf("[modsync] Place .json mod files in ./%s/", ModsDir)
	log.Printf("[modsync] Hosts automatically sync mods to connected clients")
}