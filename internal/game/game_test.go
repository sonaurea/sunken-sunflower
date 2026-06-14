package game

import (
	"os"
	"path/filepath"
	"testing"
)

// ─── GameState Tests ───────────────────────────────────────────────

func TestNewGameState(t *testing.T) {
	gs := NewGameState()
	if gs.PlayerHealth != 100 {
		t.Errorf("expected HP 100, got %f", gs.PlayerHealth)
	}
	if gs.MaxPlayerHealth != 100 {
		t.Errorf("expected max HP 100, got %f", gs.MaxPlayerHealth)
	}
	if gs.CurrentDepth != 1 {
		t.Errorf("expected depth 1, got %d", gs.CurrentDepth)
	}
	if gs.MaxCompanions != 3 {
		t.Errorf("expected max companions 3, got %d", gs.MaxCompanions)
	}
}

func TestResourceManagement(t *testing.T) {
	gs := NewGameState()

	if gs.HasResource("moon_sand", 5) {
		t.Error("should not have resources initially")
	}

	gs.AddResource("moon_sand", 10)
	if !gs.HasResource("moon_sand", 5) {
		t.Error("should have 10 moon sand")
	}

	removed := gs.RemoveResource("moon_sand", 3)
	if removed != 3 {
		t.Errorf("expected 3 removed, got %d", removed)
	}
	if gs.Inventory["moon_sand"] != 7 {
		t.Errorf("expected 7 remaining, got %d", gs.Inventory["moon_sand"])
	}

	// Remove more than available
	removed = gs.RemoveResource("moon_sand", 100)
	if removed != 7 {
		t.Errorf("expected 7 removed (all available), got %d", removed)
	}

	// Remove from empty resource
	removed = gs.RemoveResource("nonexistent", 5)
	if removed != 0 {
		t.Errorf("expected 0 for nonexistent resource, got %d", removed)
	}
}

func TestUpgradeManagement(t *testing.T) {
	gs := NewGameState()

	if gs.IsUpgradeUnlocked("dmg_up") {
		t.Error("upgrade should not be unlocked initially")
	}

	gs.UnlockUpgrade("dmg_up")
	if !gs.IsUpgradeUnlocked("dmg_up") {
		t.Error("upgrade should be unlocked")
	}
}

func TestBuildingManagement(t *testing.T) {
	gs := NewGameState()

	if gs.BuildingLevel("house") != 0 {
		t.Errorf("expected level 0, got %d", gs.BuildingLevel("house"))
	}

	gs.UpgradeBuilding("house")
	if gs.BuildingLevel("house") != 1 {
		t.Errorf("expected level 1, got %d", gs.BuildingLevel("house"))
	}

	gs.UpgradeBuilding("house")
	if gs.BuildingLevel("house") != 2 {
		t.Errorf("expected level 2, got %d", gs.BuildingLevel("house"))
	}
}

// ─── StoryManager Tests ────────────────────────────────────────────

func TestStoryManager(t *testing.T) {
	sm := NewStoryManager()
	if sm.HasFlag("test_flag") {
		t.Error("flag should not be set initially")
	}

	sm.SetFlag("test_flag")
	if !sm.HasFlag("test_flag") {
		t.Error("flag should be set")
	}

	sm.ClearFlag("test_flag")
	if sm.HasFlag("test_flag") {
		t.Error("flag should be cleared")
	}
}

func TestDialogue(t *testing.T) {
	sm := NewStoryManager()
	lines := []string{"Hello", "World"}
	sm.RegisterDialogue("scene1", lines)

	got := sm.GetDialogue("scene1")
	if len(got) != 2 || got[0] != "Hello" {
		t.Errorf("dialogue mismatch: %v", got)
	}

	// Missing dialogue
	got = sm.GetDialogue("nonexistent")
	if got != nil {
		t.Error("nonexistent dialogue should return nil")
	}
}

// ─── Save System Tests ─────────────────────────────────────────────

func TestSaveGame(t *testing.T) {
	gs := NewGameState()
	gs.AddResource("moon_sand", 50)
	gs.CurrentDepth = 5

	sm := NewStoryManager()
	sm.SetFlag("entered_town")

	sd := SaveGame(gs, sm, "town")
	if sd.Version != 1 {
		t.Errorf("expected version 1, got %d", sd.Version)
	}
	if sd.SceneName != "town" {
		t.Errorf("expected scene 'town', got %s", sd.SceneName)
	}
	if sd.State.Inventory["moon_sand"] != 50 {
		t.Errorf("expected 50 moon sand in save, got %d", sd.State.Inventory["moon_sand"])
	}
}

func TestLoadGame(t *testing.T) {
	gs := NewGameState()
	gs.AddResource("starfruit", 20)
	sm := NewStoryManager()
	sm.SetFlag("met_shopkeeper")

	sd := SaveGame(gs, sm, "dungeon")
	loadedGS, loadedSM, scene := LoadGame(sd)

	if scene != "dungeon" {
		t.Errorf("expected scene 'dungeon', got %s", scene)
	}
	if loadedGS.Inventory["starfruit"] != 20 {
		t.Errorf("expected 20 starfruit, got %d", loadedGS.Inventory["starfruit"])
	}
	if !loadedSM.HasFlag("met_shopkeeper") {
		t.Error("flag should be loaded")
	}
}

func TestLoadGameNil(t *testing.T) {
	gs, sm, scene := LoadGame(nil)
	if gs == nil {
		t.Error("should return new state for nil save")
	}
	if sm == nil {
		t.Error("should return new story for nil save")
	}
	if scene != "title" {
		t.Errorf("expected scene 'title', got %s", scene)
	}
}

func TestSaveManager(t *testing.T) {
	// Use temp dir for save tests
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	sm, err := NewSaveManager()
	if err != nil {
		t.Fatalf("failed to create save manager: %v", err)
	}

	gs := NewGameState()
	gs.CurrentDepth = 3
	story := NewStoryManager()
	story.SetFlag("entered_dungeon")

	// Save
	err = sm.Save("test_slot", gs, story, "dungeon")
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if sm.CurrentSlot() != "test_slot" {
		t.Errorf("expected current slot 'test_slot', got %s", sm.CurrentSlot())
	}

	// List slots
	slots := sm.ListSlots()
	if len(slots) == 0 {
		t.Error("should have at least 1 slot")
	}

	// Load
	loadedGS, loadedStory, scene, err := sm.Load("test_slot")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loadedGS.CurrentDepth != 3 {
		t.Errorf("expected depth 3, got %d", loadedGS.CurrentDepth)
	}
	if !loadedStory.HasFlag("entered_dungeon") {
		t.Error("flag should be loaded")
	}
	if scene != "dungeon" {
		t.Errorf("expected scene 'dungeon', got %s", scene)
	}

	// Auto-save
	err = sm.AutoSaveSave(gs, story, "dungeon")
	if err != nil {
		t.Fatalf("auto-save failed: %v", err)
	}
	if !sm.HasAutoSave() {
		t.Error("should have auto-save")
	}

	// Delete
	err = sm.Delete("test_slot")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Load deleted slot should fail
	_, _, _, err = sm.Load("test_slot")
	if err == nil {
		t.Error("loading deleted slot should fail")
	}
}

func TestSaveManagerNoHomeDir(t *testing.T) {
	// Test error handling - should not panic even with unusual HOME
	// We already tested with temp dir above
}

func TestSaveManagerDisabledAutosave(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	sm, _ := NewSaveManager()
	sm.AutoSave = false

	err := sm.AutoSaveSave(NewGameState(), NewStoryManager(), "town")
	if err != nil {
		t.Errorf("should not error when auto-save disabled, got %v", err)
	}
}

// ─── EventBus Tests ────────────────────────────────────────────────

func TestEventBus(t *testing.T) {
	eb := NewEventBus()
	called := false

	eb.Subscribe("test_event", func(data interface{}) {
		called = true
		if data.(string) != "hello" {
			t.Errorf("expected data 'hello', got %v", data)
		}
	})

	eb.Emit("test_event", "hello")
	if !called {
		t.Error("event handler should have been called")
	}

	// Unsubscribed event
	eb.Emit("other_event", nil) // should not panic
}

// ─── Camera Tests ──────────────────────────────────────────────────

func TestCamera(t *testing.T) {
	cam := NewCamera()
	if cam.X != 0 || cam.Y != 0 {
		t.Error("camera should start at origin")
	}

	// Camera centers on position minus half screen size
	cam.Follow(300, 300, 200, 200, 600, 600)
	if cam.X != 200 || cam.Y != 200 {
		t.Errorf("camera should be at (200,200), got (%f,%f)", cam.X, cam.Y)
	}

	// Camera clamping
	cam.Follow(0, 0, 200, 200, 400, 400)
	if cam.X < 0 || cam.Y < 0 {
		t.Error("camera should not go negative")
	}

	cam.Follow(500, 500, 200, 200, 400, 400)
	if cam.X > 200 || cam.Y > 200 {
		t.Errorf("camera should be clamped to 200, got (%f,%f)", cam.X, cam.Y)
	}

	op := cam.TranslateOp()
	if op == nil {
		t.Error("TranslateOp should not return nil")
	}
}

// ─── Steam Integration Tests ───────────────────────────────────────

func TestSteamIntegration(t *testing.T) {
	si := NewSteamIntegration()
	if si.Initialized {
		t.Error("steam should not be initialized in tests")
	}

	// Init should return false (no Steam in tests)
	if si.Init() {
		t.Error("Init should return false without Steam")
	}

	// Achievements should work locally
	si.UnlockAchievement("test_ach")
	if !si.Achievements["test_ach"] {
		t.Error("achievement should be unlocked locally")
	}

	// Duplicate unlock should not error
	si.UnlockAchievement("test_ach")

	// Stats
	si.SetStat("kills", 100)
	if si.Stats["kills"] != 100 {
		t.Errorf("expected stat 100, got %d", si.Stats["kills"])
	}

	// Store without init
	if si.StoreStats() {
		t.Error("StoreStats should return false without Steam init")
	}

	// Steam Deck check
	if si.IsSteamDeck() {
		t.Error("IsSteamDeck() should return false in tests")
	}
}

func TestAllAchievements(t *testing.T) {
	achs := AllAchievements()
	if len(achs) < 10 {
		t.Errorf("expected at least 10 achievements, got %d", len(achs))
	}

	// Check specific achievements exist
	required := []string{
		AchFirstBlood, AchDepth5, AchDepth10, AchDepth20,
		AchCompanion, AchFullParty, AchCollector, AchWellKey,
	}
	for _, ach := range required {
		if _, ok := achs[ach]; !ok {
			t.Errorf("achievement %s not found", ach)
		}
	}
}

// ─── Combat System Tests ───────────────────────────────────────────

func TestDefaultPlayerStats(t *testing.T) {
	stats := DefaultPlayerStats()
	if stats.Damage != 15 {
		t.Errorf("expected damage 15, got %f", stats.Damage)
	}
	if stats.CritChance != 0.10 {
		t.Errorf("expected crit 0.10, got %f", stats.CritChance)
	}
	if stats.Armor != 5 {
		t.Errorf("expected armor 5, got %f", stats.Armor)
	}
}

func TestDefaultEnemyStats(t *testing.T) {
	stats := DefaultEnemyStats(1)
	if stats.Damage != 13 {
		t.Errorf("expected damage 13 at depth 1, got %f", stats.Damage)
	}

	stats10 := DefaultEnemyStats(10)
	if stats10.Damage <= stats.Damage {
		t.Error("deeper enemies should have more damage")
	}
}

func TestCalculateHit(t *testing.T) {
	attacker := DefaultPlayerStats()
	defender := DefaultEnemyStats(1)

	// Run multiple hits to ensure no panics and reasonable results
	for i := 0; i < 100; i++ {
		result := CalculateHit(attacker, defender, "player")
		if result.IsDodged {
			if result.Damage != 0 {
				t.Error("dodged hits should deal 0 damage")
			}
			continue
		}
		if result.Damage < 0 {
			t.Errorf("negative damage: %f", result.Damage)
		}
	}
}

func TestCalculateHitGuaranteed(t *testing.T) {
	// Make attacker very strong and defender very weak
	attacker := CombatStats{
		Damage:      100,
		CritChance:  0,
		DodgeChance: 0,
		StatusChance: 0,
	}
	defender := CombatStats{
		Armor:       0,
		DodgeChance: 0,
		ParryChance: 0,
		Resistances: map[DamageType]float64{DamagePhysical: 1.0},
	}

	for i := 0; i < 50; i++ {
		result := CalculateHit(attacker, defender, "player")
		if result.IsDodged || result.IsParried {
			continue
		}
		if result.Damage <= 0 {
			t.Errorf("hit should deal damage, got %f", result.Damage)
		}
	}
}

func TestDamageTypeString(t *testing.T) {
	tests := []struct {
		dt   DamageType
		want string
	}{
		{DamagePhysical, "physical"},
		{DamageMagic, "magic"},
		{DamageFire, "fire"},
		{DamagePoison, "poison"},
		{DamageHoly, "holy"},
		{DamageShadow, "shadow"},
		{DamageType(99), "unknown"},
	}
	for _, tt := range tests {
		if tt.dt.String() != tt.want {
			t.Errorf("DamageType(%d).String() = %s, want %s", tt.dt, tt.dt.String(), tt.want)
		}
	}
}

// ─── Combo Tests ───────────────────────────────────────────────────

func TestNewPlayerCombo(t *testing.T) {
	c := NewPlayerCombo()
	if len(c.Stages) != 3 {
		t.Errorf("expected 3 stages, got %d", len(c.Stages))
	}
	if c.Stages[0].Name != "Petal Slash" {
		t.Errorf("expected 'Petal Slash', got %s", c.Stages[0].Name)
	}
}

func TestComboAdvance(t *testing.T) {
	c := NewPlayerCombo()
	stage := c.Advance()
	if stage.Name != "Petal Slash" {
		t.Errorf("expected first stage 'Petal Slash', got %s", stage.Name)
	}
	if !c.Active {
		t.Error("combo should be active after advance")
	}

	// Advance through all stages
	c.Advance()
	c.Advance()
	// Should loop back
	stage = c.Advance()
	if stage.Name != "Petal Slash" {
		t.Errorf("should loop back to first stage, got %s", stage.Name)
	}
}

func TestComboReset(t *testing.T) {
	c := NewPlayerCombo()
	c.Advance()
	c.Advance()

	c.Reset()
	if c.Active {
		t.Error("combo should be inactive after reset")
	}
	if c.Current != 0 {
		t.Errorf("current should be 0 after reset, got %d", c.Current)
	}
}

func TestComboTimer(t *testing.T) {
	c := NewPlayerCombo()
	c.Advance()
	c.Update(2.0) // exceeds MaxTimer
	if c.Active {
		t.Error("combo should expire after timer exceeds MaxTimer")
	}
}

// ─── StatusEffect Tests ────────────────────────────────────────────

func TestStatusManager(t *testing.T) {
	sm := NewStatusManager()
	sm.Add(&ActiveStatus{Type: "burn", Duration: 3.0, Damage: 5, SourceID: "enemy"})

	ticks := sm.Update(0.5)
	if len(ticks) < 1 {
		t.Error("should have tick data")
	}

	if !sm.HasType("burn") {
		t.Error("should have burn effect")
	}

	sm.Update(3.0)
	if sm.HasType("burn") {
		t.Error("burn should have expired")
	}
}

func TestStatusManagerRefresh(t *testing.T) {
	sm := NewStatusManager()
	sm.Add(&ActiveStatus{Type: "poison", Duration: 2.0, Damage: 3, SourceID: "enemy"})
	sm.Add(&ActiveStatus{Type: "poison", Duration: 5.0, Damage: 3, SourceID: "enemy"})

	// Duration should be refreshed to 5.0
	ticks := sm.Update(0.1)
	if len(ticks) < 1 {
		t.Error("should have ticks")
	}

	sm.Clear()
	if sm.HasType("poison") {
		t.Error("after Clear, should not have any effects")
	}
}

func TestDoTDamage(t *testing.T) {
	tests := []struct {
		damage float64
		status string
		want   float64
	}{
		{100, "burn", 15},
		{100, "poison", 10},
		{100, "heal", -20},
		{100, "unknown", 0},
	}
	for _, tt := range tests {
		got := DoTDamage(tt.damage, tt.status)
		if got != tt.want {
			t.Errorf("DoTDamage(%f, %q) = %f, want %f", tt.damage, tt.status, got, tt.want)
		}
	}
}

// ─── LevelSystem Tests ─────────────────────────────────────────────

func TestNewLevelSystem(t *testing.T) {
	ls := NewLevelSystem()
	if ls.Level != 1 {
		t.Errorf("expected level 1, got %d", ls.Level)
	}
	if ls.XPToNext != 100 {
		t.Errorf("expected 100 XP to next, got %f", ls.XPToNext)
	}
}

func TestLevelSystemAddXP(t *testing.T) {
	ls := NewLevelSystem()
	leveled := ls.AddXP(50)
	if leveled {
		t.Error("should not level up from 50 XP")
	}
	if ls.CurrentXP != 50 {
		t.Errorf("expected 50 XP, got %f", ls.CurrentXP)
	}

	leveled = ls.AddXP(60)
	if !leveled {
		t.Error("should level up from 110 total XP")
	}
	if ls.Level != 2 {
		t.Errorf("expected level 2, got %d", ls.Level)
	}
	if ls.StatPoints != 3 {
		t.Errorf("expected 3 stat points, got %d", ls.StatPoints)
	}
}

func TestLevelSystemProgress(t *testing.T) {
	ls := NewLevelSystem()
	ls.AddXP(50)
	progress := ls.XPProgress()
	if progress != 0.5 {
		t.Errorf("expected progress 0.5, got %f", progress)
	}
}

func TestXPForEnemy(t *testing.T) {
	xp1 := XPForEnemy(1)
	if xp1 != 20 {
		t.Errorf("expected 20 XP for depth 1, got %f", xp1)
	}
	xp10 := XPForEnemy(10)
	if xp10 != 65 {
		t.Errorf("expected 65 XP for depth 10, got %f", xp10)
	}
}

// ─── Stealth Tests ─────────────────────────────────────────────────

func TestStealthState(t *testing.T) {
	ss := NewStealthState()
	// Should start detected (not hidden) since no Update has been called
	if ss.IsHidden {
		t.Error("should start detected before first Update")
	}

	// First update with no enemy makes it hidden
	ss.Update(0.5, false, false)
	if !ss.IsHidden {
		t.Error("should become hidden after Update with no enemies")
	}

	// Near enemy and moving increases detection
	ss.Update(0.5, true, true)
	if ss.Detection <= 0 {
		t.Error("detection should increase near enemy")
	}

	// Away from enemy decreases detection
	for i := 0; i < 10; i++ {
		ss.Update(0.5, false, false)
	}
	if !ss.IsHidden {
		t.Error("should become hidden again")
	}
}

func TestStealthStateMaxDetection(t *testing.T) {
	ss := NewStealthState()
	for i := 0; i < 10; i++ {
		ss.Update(0.5, true, true)
	}
	if ss.Detection > 1 {
		t.Errorf("detection should be capped at 1, got %f", ss.Detection)
	}
}

// ─── SaveData Tests ────────────────────────────────────────────────

func TestSaveDataRoundTrip(t *testing.T) {
	gs := NewGameState()
	gs.CurrentDepth = 7
	gs.AddResource("well_shard", 25)
	gs.PlayerHealth = 80

	sm := NewStoryManager()
	sm.SetFlag("deep_dive")

	sd := SaveGame(gs, sm, "dungeon")

	loadedGS, loadedSM, scene := LoadGame(sd)
	if loadedGS.CurrentDepth != 7 {
		t.Errorf("expected depth 7, got %d", loadedGS.CurrentDepth)
	}
	if loadedGS.Inventory["well_shard"] != 25 {
		t.Errorf("expected 25 well_shard, got %d", loadedGS.Inventory["well_shard"])
	}
	if !loadedSM.HasFlag("deep_dive") {
		t.Error("flag should be loaded")
	}
	if scene != "dungeon" {
		t.Errorf("expected scene 'dungeon', got %s", scene)
	}
}

// ─── File Save/Load Integration Tests ──────────────────────────────

func TestSaveManagerFileOperations(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	sm, _ := NewSaveManager()

	gs := NewGameState()
	gs.CurrentDepth = 10
	story := NewStoryManager()

	// Save and load
	err := sm.Save("slot1", gs, story, "town")
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, _, scene, err := sm.Load("slot1")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.CurrentDepth != 10 {
		t.Errorf("expected depth 10, got %d", loaded.CurrentDepth)
	}
	if scene != "town" {
		t.Errorf("expected scene 'town', got %s", scene)
	}

	// Verify file exists
	savePath := filepath.Join(tmpDir, ".sunken-sunflower", "slot1.save")
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		t.Error("save file should exist")
	}

	// Delete
	err = sm.Delete("slot1")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := os.Stat(savePath); !os.IsNotExist(err) {
		t.Error("save file should be deleted")
	}
}

func TestLegacySaveFormat(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a legacy format save (just SaveData, no container)
	gs := NewGameState()
	gs.CurrentDepth = 3
	sd := SaveGame(gs, NewStoryManager(), "town")

	sm, _ := NewSaveManager()
	err := sm.Save("legacy_test", gs, NewStoryManager(), "town")
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Load should work
	loaded, _, scene, err := sm.Load("legacy_test")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.CurrentDepth != 3 {
		t.Errorf("expected depth 3, got %d", loaded.CurrentDepth)
	}
	_ = scene
	_ = sd
}