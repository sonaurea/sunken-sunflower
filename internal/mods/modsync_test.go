package mods

import (
	"os"
	"path/filepath"
	"testing"
)

// ─── Test Helpers ─────────────────────────────────────────────────────

// tempModDir creates a temporary directory with test mod files.
func tempModDir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "mods-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create test mod files
	mods := map[string]string{
		"test_mod.json": `{"name": "Test Mod", "version": "1.0.0"}`,
		"combat_tweaks.json": `{"name": "Combat Tweaks", "version": "2.1.0"}`,
		"invalid.txt":  `this is not a json mod`,
		"empty.json":   `{}`,
	}

	for name, content := range mods {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	return dir, func() { os.RemoveAll(dir) }
}

// ─── NewModSyncManager ────────────────────────────────────────────────

func TestNewModSyncManager_NoMods(t *testing.T) {
	msm := NewModSyncManager()
	if msm == nil {
		t.Fatal("mod sync manager should not be nil")
	}
	if msm.ModCount() != 0 {
		t.Errorf("expected 0 mods, got %d", msm.ModCount())
	}
	if msm.AchievementsLocked {
		t.Error("achievements should not be locked without mods")
	}
}

func TestNewModSyncManager_CustomDir(t *testing.T) {
	dir, cleanup := tempModDir(t)
	defer cleanup()

	msm := &ModSyncManager{
		ModsDir:     dir,
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	msm.scanLocalMods()

	if msm.ModCount() != 3 {
		t.Errorf("expected 3 mod files (.json only), got %d", msm.ModCount())
	}
	if !msm.AchievementsLocked {
		t.Error("achievements should be locked when mods are present")
	}
}

// ─── ModInfo and Manifest ─────────────────────────────────────────────

func TestModDiscoveredCorrectly(t *testing.T) {
	dir, cleanup := tempModDir(t)
	defer cleanup()

	msm := &ModSyncManager{
		ModsDir:     dir,
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	msm.scanLocalMods()

	mod, exists := msm.LocalMods["test_mod.json"]
	if !exists {
		t.Fatal("test_mod.json should be discovered")
	}
	if mod.Name != "Test Mod" {
		t.Errorf("expected name 'Test Mod', got %q", mod.Name)
	}
	if mod.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", mod.Version)
	}
	if !mod.Enabled {
		t.Error("mod should be enabled by default")
	}
	if mod.Hash == "" {
		t.Error("mod hash should not be empty")
	}
	if mod.Size <= 0 {
		t.Error("mod size should be positive")
	}
}

// ─── GetHostManifest ──────────────────────────────────────────────────

func TestGetHostManifest(t *testing.T) {
	dir, cleanup := tempModDir(t)
	defer cleanup()

	msm := &ModSyncManager{
		ModsDir:     dir,
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	msm.scanLocalMods()

	manifest := msm.GetHostManifest("host-123")
	if manifest == nil {
		t.Fatal("manifest should not be nil")
	}
	if manifest.HostID != "host-123" {
		t.Errorf("expected host 'host-123', got %q", manifest.HostID)
	}
	if manifest.ModCount != 3 {
		t.Errorf("expected 3 mods in manifest, got %d", manifest.ModCount)
	}
	if len(manifest.Mods) != 3 {
		t.Errorf("expected 3 mod entries, got %d", len(manifest.Mods))
	}

	// Verify mods are sorted by filename
	if manifest.Mods[0].Filename > manifest.Mods[1].Filename {
		t.Error("mods should be sorted alphabetically by filename")
	}
}

func TestGetHostManifest_DisabledModsExcluded(t *testing.T) {
	dir, cleanup := tempModDir(t)
	defer cleanup()

	msm := &ModSyncManager{
		ModsDir:     dir,
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	msm.scanLocalMods()

	// Disable two mods, leave one enabled
	msm.EnabledMods["test_mod.json"] = false
	msm.EnabledMods["combat_tweaks.json"] = false

	manifest := msm.GetHostManifest("host-1")
	if manifest.ModCount != 1 {
		t.Errorf("expected 1 enabled mod, got %d", manifest.ModCount)
	}
}

// ─── ReceiveHostManifest ──────────────────────────────────────────────

func TestReceiveHostManifest(t *testing.T) {
	dir, cleanup := tempModDir(t)
	defer cleanup()

	msm := &ModSyncManager{
		ModsDir:     dir,
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	msm.scanLocalMods()

	// Host has a mod we don't have
	hostManifest := &ModManifest{
		HostID:   "host-1",
		ModCount: 1,
		Mods: []ModInfo{
			{Filename: "new_mod.json", Hash: "abc123", Size: 100},
		},
	}

	needed := msm.ReceiveHostManifest(hostManifest)
	if len(needed) != 1 {
		t.Fatalf("expected 1 needed mod, got %d", len(needed))
	}
	if needed[0].Filename != "new_mod.json" {
		t.Errorf("expected 'new_mod.json', got %q", needed[0].Filename)
	}
}

func TestReceiveHostManifest_MatchingMod(t *testing.T) {
	dir, cleanup := tempModDir(t)
	defer cleanup()

	msm := &ModSyncManager{
		ModsDir:     dir,
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	msm.scanLocalMods()

	localHash := msm.LocalMods["test_mod.json"].Hash

	// Host has same version of mod we have
	hostManifest := &ModManifest{
		HostID:   "host-1",
		ModCount: 1,
		Mods: []ModInfo{
			{Filename: "test_mod.json", Hash: localHash, Size: 100},
		},
	}

	needed := msm.ReceiveHostManifest(hostManifest)
	if len(needed) != 0 {
		t.Errorf("expected 0 needed mods (hash match), got %d", len(needed))
	}
}

// ─── AcceptHostMods ───────────────────────────────────────────────────

func TestAcceptHostMods(t *testing.T) {
	dir, cleanup := tempModDir(t)
	defer cleanup()

	msm := &ModSyncManager{
		ModsDir:     dir,
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	msm.scanLocalMods()

	// Receive host manifest first
	hostManifest := &ModManifest{
		HostID:   "host-1",
		ModCount: 1,
		Mods: []ModInfo{
			{Filename: "combat_tweaks.json"},
		},
	}
	msm.ReceiveHostManifest(hostManifest)

	toLoad := msm.AcceptHostMods()
	if len(toLoad) != 1 {
		t.Fatalf("expected 1 mod to load, got %d", len(toLoad))
	}
	if toLoad[0] != "combat_tweaks.json" {
		t.Errorf("expected 'combat_tweaks.json', got %q", toLoad[0])
	}

	// Other mod should be disabled
	if msm.IsModEnabled("test_mod.json") {
		t.Error("test_mod.json should be disabled after accepting host mods")
	}
}

func TestAcceptHostMods_NilManifest(t *testing.T) {
	msm := &ModSyncManager{
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	result := msm.AcceptHostMods()
	if result != nil {
		t.Errorf("expected nil with no manifest, got %v", result)
	}
}

// ─── ToggleMod ────────────────────────────────────────────────────────

func TestToggleMod(t *testing.T) {
	msm := &ModSyncManager{
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	msm.EnabledMods["test.json"] = true

	// Toggle: enabled (true) → disabled (false)
	newState := msm.ToggleMod("test.json")
	if newState {
		t.Error("expected ToggleMod to return false (new disabled state)")
	}
	if msm.IsModEnabled("test.json") {
		t.Error("mod should be disabled after toggle")
	}

	// Toggle: disabled (false) → enabled (true)
	newState = msm.ToggleMod("test.json")
	if !newState {
		t.Error("expected ToggleMod to return true (new enabled state)")
	}
	if !msm.IsModEnabled("test.json") {
		t.Error("mod should be enabled after second toggle")
	}
}

// ─── GetEnabledMods ───────────────────────────────────────────────────

func TestGetEnabledMods(t *testing.T) {
	msm := &ModSyncManager{
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	msm.EnabledMods["a.json"] = true
	msm.EnabledMods["b.json"] = false
	msm.EnabledMods["c.json"] = true

	enabled := msm.GetEnabledMods()
	if len(enabled) != 2 {
		t.Fatalf("expected 2 enabled, got %d", len(enabled))
	}
}

// ─── ResolveModConflicts ──────────────────────────────────────────────

func TestResolveModConflicts(t *testing.T) {
	dir, cleanup := tempModDir(t)
	defer cleanup()

	msm := &ModSyncManager{
		ModsDir:     dir,
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	msm.scanLocalMods()

	// Host has a different hash for test_mod.json
	hostManifest := &ModManifest{
		HostID:   "host-1",
		ModCount: 1,
		Mods: []ModInfo{
			{Filename: "test_mod.json", Hash: "differenthash", Version: "2.0.0"},
		},
	}
	msm.ReceiveHostManifest(hostManifest)

	conflicts := msm.ResolveModConflicts()
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Filename != "test_mod.json" {
		t.Errorf("expected conflict for 'test_mod.json', got %q", conflicts[0].Filename)
	}
}

// ─── ReceiveModData ───────────────────────────────────────────────────

func TestReceiveModData(t *testing.T) {
	dir, err := os.MkdirTemp("", "mods-test-receive-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	msm := &ModSyncManager{
		ModsDir:     dir,
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}

	data := []byte(`{"name": "Received Mod", "version": "1.0.0"}`)
	if err := msm.ReceiveModData("received_mod.json", data); err != nil {
		t.Fatalf("failed to receive mod data: %v", err)
	}

	// Verify it was saved
	path := filepath.Join(dir, "received_mod.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("mod file was not saved to disk")
	}

	// Verify it was indexed
	mod, exists := msm.LocalMods["received_mod.json"]
	if !exists {
		t.Fatal("received mod should be in local index")
	}
	if mod.Size != int64(len(data)) {
		t.Errorf("expected size %d, got %d", len(data), mod.Size)
	}
	if !mod.Enabled {
		t.Error("received mod should be enabled by default")
	}
}

// ─── SerializeManifest / DeserializeManifest ──────────────────────────

func TestSerializeDeserializeManifest(t *testing.T) {
	original := &ModManifest{
		HostID:   "host-1",
		ModCount: 2,
		Mods: []ModInfo{
			{Name: "Mod A", Version: "1.0", Filename: "a.json", Hash: "hash_a"},
			{Name: "Mod B", Version: "2.0", Filename: "b.json", Hash: "hash_b"},
		},
	}

	data, err := SerializeManifest(original)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	restored, err := DeserializeManifest(data)
	if err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	if restored.HostID != original.HostID {
		t.Errorf("expected host %q, got %q", original.HostID, restored.HostID)
	}
	if restored.ModCount != original.ModCount {
		t.Errorf("expected %d mods, got %d", original.ModCount, restored.ModCount)
	}
	if restored.Mods[0].Name != "Mod A" {
		t.Errorf("expected 'Mod A', got %q", restored.Mods[0].Name)
	}
}

// ─── Edge Cases ───────────────────────────────────────────────────────

func TestEmptyModsDir(t *testing.T) {
	dir, err := os.MkdirTemp("", "mods-test-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	msm := &ModSyncManager{
		ModsDir:     dir,
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	msm.scanLocalMods()

	if msm.ModCount() != 0 {
		t.Errorf("expected 0 mods in empty dir, got %d", msm.ModCount())
	}
	if msm.AchievementsLocked {
		t.Error("achievements should not be locked with empty mod dir")
	}
}

func TestNonExistentDir(t *testing.T) {
	msm := &ModSyncManager{
		ModsDir:     "/nonexistent/path",
		LocalMods:   make(map[string]*ModInfo),
		EnabledMods: make(map[string]bool),
	}
	msm.scanLocalMods() // should not panic

	if msm.ModCount() != 0 {
		t.Errorf("expected 0 mods for nonexistent dir, got %d", msm.ModCount())
	}
}