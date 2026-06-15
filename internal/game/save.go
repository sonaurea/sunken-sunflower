package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ─── Save System ────────────────────────────────────────────────────
// Full save/load with multiple slots, auto-save, and metadata.

const (
	SaveVersion        = 1
	SaveDirName        = ".sunken-sunflower"
	AutoSaveSlot       = "auto"
	MaxManualSlots     = 10
	SaveFileExtension  = ".save"
)

// SaveSlot represents one save slot with metadata.
type SaveSlot struct {
	SlotID    string `json:"slot_id"`
	Label     string `json:"label"`
	Timestamp int64  `json:"timestamp"`
	Depth     int    `json:"depth"`
	PlayTime  float64 `json:"play_time"`
	Level     int    `json:"level"`
	Version   int    `json:"version"`
}

// SaveManager handles all save/load operations.
type SaveManager struct {
	mu        sync.Mutex
	SaveDir   string
	AutoSave  bool
	Slots     map[string]*SaveSlot
	currentID string
}

// NewSaveManager creates a save manager in the user's config directory.
func NewSaveManager() (*SaveManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find home directory: %w", err)
	}
	saveDir := filepath.Join(home, SaveDirName)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create save directory: %w", err)
	}

	sm := &SaveManager{
		SaveDir:  saveDir,
		AutoSave: true,
		Slots:    make(map[string]*SaveSlot),
	}

	// Load existing slot metadata
	sm.scanSlots()
	return sm, nil
}

func (sm *SaveManager) scanSlots() {
	pattern := filepath.Join(sm.SaveDir, "*"+SaveFileExtension)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var slot SaveSlot
		if err := json.Unmarshal(data, &slot); err != nil {
			continue
		}
		sm.Slots[slot.SlotID] = &slot
	}
}

// Save persists the current game state to a save slot.
func (sm *SaveManager) Save(slotID string, state *GameState, story *StoryManager, sceneName string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sd := SaveGame(state, story, sceneName)

	// Build save file with metadata header + data
	saveData := struct {
		Slot     SaveSlot   `json:"slot"`
		SaveData *SaveData  `json:"save_data"`
	}{
		Slot: SaveSlot{
			SlotID:    slotID,
			Timestamp: time.Now().Unix(),
			Depth:     state.CurrentDepth,
			PlayTime:  state.PlayTimeSeconds,
			Level:     1, // Could add level system reference
			Version:   SaveVersion,
		},
		SaveData: sd,
	}

	if slotID == AutoSaveSlot {
		saveData.Slot.Label = "Auto-Save"
	} else {
		saveData.Slot.Label = fmt.Sprintf("Save %s", slotID)
	}

	data, err := json.MarshalIndent(saveData, "", "  ")
	if err != nil {
		return fmt.Errorf("save marshal error: %w", err)
	}

	path := filepath.Join(sm.SaveDir, slotID+SaveFileExtension)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("save write error: %w", err)
	}

	sm.Slots[slotID] = &saveData.Slot
	sm.currentID = slotID
	return nil
}

// Load restores game state from a save slot.
func (sm *SaveManager) Load(slotID string) (*GameState, *StoryManager, string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	path := filepath.Join(sm.SaveDir, slotID+SaveFileExtension)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, "", fmt.Errorf("load read error: %w", err)
	}

	var container struct {
		Slot     SaveSlot  `json:"slot"`
		SaveData *SaveData `json:"save_data"`
	}
	if err := json.Unmarshal(data, &container); err != nil {
		// Try legacy format (just SaveData)
		var sd SaveData
		if err2 := json.Unmarshal(data, &sd); err2 != nil {
			return nil, nil, "", fmt.Errorf("load unmarshal error: %w", err)
		}
		state, story, scene := LoadGame(&sd)
		// Update slot metadata
		sm.Slots[slotID] = &SaveSlot{
			SlotID:    slotID,
			Timestamp: time.Now().Unix(),
			Depth:     state.CurrentDepth,
		}
		sm.currentID = slotID
		return state, story, scene, nil
	}

	state, story, scene := LoadGame(container.SaveData)
	sm.currentID = slotID
	return state, story, scene, nil
}

// Delete removes a save slot.
func (sm *SaveManager) Delete(slotID string) error {
	path := filepath.Join(sm.SaveDir, slotID+SaveFileExtension)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	delete(sm.Slots, slotID)
	return nil
}

// AutoSaveSave performs an automatic save to the auto-save slot.
func (sm *SaveManager) AutoSaveSave(state *GameState, story *StoryManager, sceneName string) error {
	if !sm.AutoSave {
		return nil
	}
	return sm.Save(AutoSaveSlot, state, story, sceneName)
}

// ListSlots returns all available save slots.
func (sm *SaveManager) ListSlots() []*SaveSlot {
	slots := make([]*SaveSlot, 0, len(sm.Slots))
	for _, slot := range sm.Slots {
		slots = append(slots, slot)
	}
	return slots
}

// HasAutoSave checks if an auto-save exists.
func (sm *SaveManager) HasAutoSave() bool {
	_, ok := sm.Slots[AutoSaveSlot]
	return ok
}

// CurrentSlot returns the ID of the last used save slot.
func (sm *SaveManager) CurrentSlot() string {
	return sm.currentID
}

// ─── Steam Integration (Stub) ─────────────────────────────────────
// This provides hooks for Steamworks SDK integration.
// Replace stubs with actual Steam API calls when building with Steam support.

type SteamIntegration struct {
	Initialized bool
	UserID      string
	Achievements map[string]bool
	Stats        map[string]int
}

func NewSteamIntegration() *SteamIntegration {
	return &SteamIntegration{
		Achievements: make(map[string]bool),
		Stats:        make(map[string]int),
	}
}

// Init attempts to initialize the Steamworks API.
// Returns false if Steam is not available (standalone build).
func (si *SteamIntegration) Init() bool {
	// Stub: In production, call SteamAPI_Init() via CGo
	// For now, we just return false (standalone mode)
	si.Initialized = false
	return si.Initialized
}

// UnlockAchievement unlocks a Steam achievement.
// In standalone mode, this just logs locally.
func (si *SteamIntegration) UnlockAchievement(id string) bool {
	if si.Achievements[id] {
		return true // already unlocked
	}
	si.Achievements[id] = true
	if si.Initialized {
		// TODO: SteamAPI_UserStats()->SetAchievement(id)
		_ = si.Initialized // placeholder: will call Steam API here
	}
	return true
}

// SetStat sets a Steam stat value.
func (si *SteamIntegration) SetStat(name string, value int) bool {
	si.Stats[name] = value
	if si.Initialized {
		// TODO: SteamAPI_UserStats()->SetStat(name, value)
		_ = si.Initialized // placeholder: will call Steam API here
	}
	return true
}

// StoreStats flushes stats/achievements to Steam.
func (si *SteamIntegration) StoreStats() bool {
	if !si.Initialized {
		return false
	}
	// TODO: SteamAPI_UserStats()->StoreStats()
	return true
}

// IsSteamDeck returns true if running on a Steam Deck.
func (si *SteamIntegration) IsSteamDeck() bool {
	// Stub: In production, check via SteamAPI or DMI/device info
	// On Linux, check for /sys/devices/virtual/dmi/id/product_name == "Galileo"
	return false
}

// ─── Achievement Definitions ────────────────────────────────────────

const (
	AchFirstBlood     = "ACH_FIRST_BLOOD"
	AchDepth5         = "ACH_DEPTH_5"
	AchDepth10        = "ACH_DEPTH_10"
	AchDepth20        = "ACH_DEPTH_20"
	AchCompanion      = "ACH_COMPANION"
	AchFullParty      = "ACH_FULL_PARTY"
	AchCollector      = "ACH_COLLECTOR"
	AchWellKey        = "ACH_WELL_KEY"
	AchNoHitRun       = "ACH_NO_HIT_RUN"
	AchSpeedRun       = "ACH_SPEED_RUN"
	AchVillageMax     = "ACH_VILLAGE_MAX"
	AchDreamer        = "ACH_DREAMER"
	AchSunflowerMax   = "ACH_SUNFLOWER_MAX"
)

// AllAchievements returns the full list of achievements with descriptions.
func AllAchievements() map[string]string {
	return map[string]string{
		AchFirstBlood:   "Defeat your first enemy",
		AchDepth5:       "Reach well depth 5",
		AchDepth10:      "Reach well depth 10",
		AchDepth20:      "Reach well depth 20",
		AchCompanion:    "Gain your first companion",
		AchFullParty:    "Have 3 companions at once",
		AchCollector:    "Collect 100 of a single resource",
		AchWellKey:      "Find the well key",
		AchNoHitRun:     "Complete a depth without taking damage",
		AchSpeedRun:     "Reach depth 5 within 10 minutes",
		AchVillageMax:   "Fully upgrade the village",
		AchDreamer:      "Experience all dream sequences",
		AchSunflowerMax: "Reach maximum power",
	}
}

// ─── Controller Input (Steam-aware) ────────────────────────────────
// The input system in engine/input.go already handles gamepad via
// ebiten's standard API. Steam Input (SDL) is mapped automatically
// by Steam when running through Steam, so the standard gamepad
// handling covers Steam Deck controls.
//
// Key mappings (Steam Deck / standard controller):
//   - Left stick / D-pad: Movement
//   - A (South button): Action/Interact
//   - B (East button): Cancel/Back
//   - LB (Left bumper): Dash
//   - RB (Right bumper): Companion command
//   - Start (Menu): Pause/Menu
//   - Back/Select: Inventory
//   - Right stick: Aim (future)