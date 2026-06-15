package scenes

import (
	"testing"

	gamepkg "github.com/michael/beach-dreams/internal/game"
	"github.com/michael/beach-dreams/internal/net"
)

// ─── DreamScene: generateDreamText ────────────────────────────────────
// Tests the poetic dream text generation based on dungeon depth.

func TestDreamText_Depth1(t *testing.T) {
	state := gamepkg.NewGameState()
	state.CurrentDepth = 1
	story := gamepkg.NewStoryManager()
	s := NewDreamScene(state, story)

	// Trigger text generation and verify
	s.dreamText = s.generateDreamText()
	if len(s.dreamText) < 5 {
		t.Errorf("depth 1: expected at least 5 lines, got %d", len(s.dreamText))
	}
}

func TestDreamText_VariousDepths(t *testing.T) {
	tests := []struct {
		depth    int
		minLines int
	}{
		{1, 5},    // depth 1: 5 lines + "..."
		{2, 4},    // depth <=3: 4 lines + "..."
		{3, 4},    // depth <=3
		{5, 4},    // depth <=6: 4 lines + "..."
		{6, 4},    // depth <=6
		{8, 4},    // depth <=10: 4 lines + "..."
		{10, 4},   // depth <=10
		{12, 4},   // depth <=15: 4 lines + "..."
		{15, 4},   // depth <=15
		{20, 6},   // default: 6 lines + "..."
		{100, 6},  // default
	}

	for _, tt := range tests {
		state := gamepkg.NewGameState()
		state.CurrentDepth = tt.depth
		story := gamepkg.NewStoryManager()
		s := NewDreamScene(state, story)

		// trigger text generation
		s.dreamText = s.generateDreamText()
		if len(s.dreamText) < tt.minLines {
			t.Errorf("depth %d: expected at least %d lines, got %d",
				tt.depth, tt.minLines, len(s.dreamText))
		}
		if len(s.dreamText) == 0 {
			t.Errorf("depth %d: dream text should not be empty", tt.depth)
		}
		// Last line should be the awakening prompt
		lastLine := s.dreamText[len(s.dreamText)-1]
		if lastLine != "Press SPACE to awaken." {
			t.Errorf("depth %d: last line should be awakening prompt, got %q",
				tt.depth, lastLine)
		}
	}
}

func TestDreamText_DifferentDepthsProduceDifferentText(t *testing.T) {
	state1 := gamepkg.NewGameState()
	state1.CurrentDepth = 1
	story1 := gamepkg.NewStoryManager()
	s1 := NewDreamScene(state1, story1)
	s1.dreamText = s1.generateDreamText()

	state10 := gamepkg.NewGameState()
	state10.CurrentDepth = 10
	story10 := gamepkg.NewStoryManager()
	s10 := NewDreamScene(state10, story10)
	s10.dreamText = s10.generateDreamText()

	// They should have different text
	same := true
	if len(s1.dreamText) != len(s10.dreamText) {
		same = false
	} else {
		for i := range s1.dreamText {
			if s1.dreamText[i] != s10.dreamText[i] {
				same = false
				break
			}
		}
	}
	if same {
		t.Error("different depths should produce different dream text")
	}
}

// ─── TitleScene: Menu States ─────────────────────────────────────────
// Tests the menu state machine and transitions.

func TestTitleScene_InitialState(t *testing.T) {
	s := NewTitleScene()
	if s == nil {
		t.Fatal("title scene should not be nil")
	}
	if s.menuState != MenuMain {
		t.Errorf("expected MenuMain state, got %d", s.menuState)
	}
	if s.selectedItem != 0 {
		t.Errorf("expected selected item 0, got %d", s.selectedItem)
	}
	if s.netMgr == nil {
		t.Error("network manager should be initialized")
	}
}

func TestTitleScene_MenuStateValues(t *testing.T) {
	if MenuMain != 0 {
		t.Errorf("expected MenuMain=0, got %d", MenuMain)
	}
	if MenuMultiplayer != 1 {
		t.Errorf("expected MenuMultiplayer=1, got %d", MenuMultiplayer)
	}
	if MenuHost != 2 {
		t.Errorf("expected MenuHost=2, got %d", MenuHost)
	}
	if MenuJoin != 3 {
		t.Errorf("expected MenuJoin=3, got %d", MenuJoin)
	}
	if MenuConnecting != 4 {
		t.Errorf("expected MenuConnecting=4, got %d", MenuConnecting)
	}
}

func TestTitleScene_NetManagerInitalized(t *testing.T) {
	s := NewTitleScene()
	if s.netMgr.Mode() != net.ModeOffline {
		t.Errorf("expected offline mode, got %d", s.netMgr.Mode())
	}
	if s.netMgr.IsConnected() {
		t.Error("should not be connected on init")
	}
}

// ─── DungeonScene: Basic Properties ───────────────────────────────────
// Tests dungeon scene creation (full gameplay requires Ebitengine).

func TestNewDungeonScene(t *testing.T) {
	state := gamepkg.NewGameState()
	story := gamepkg.NewStoryManager()
	s := NewDungeonScene(state, story)

	if s == nil {
		t.Fatal("dungeon scene should not be nil")
	}
	if s.state != state {
		t.Error("state should be the passed instance")
	}
	if s.story != story {
		t.Error("story should be the passed instance")
	}
	if s.depth != 1 {
		t.Errorf("expected depth 1, got %d", s.depth)
	}
	if s.hits == nil {
		t.Error("hitbox manager should be initialized")
	}
	if s.dialogue == nil {
		t.Error("dialogue manager should be initialized")
	}
	if s.isoCamera == nil {
		t.Error("iso camera should be initialized")
	}
	if s.isoDrawer == nil {
		t.Error("iso drawer should be initialized")
	}
}

func TestDungeonScene_DepthFromState(t *testing.T) {
	state := gamepkg.NewGameState()
	state.CurrentDepth = 5
	story := gamepkg.NewStoryManager()
	s := NewDungeonScene(state, story)

	// Before Enter, depth is from constructor
	if s.depth != 1 {
		t.Errorf("expected initial depth 1, got %d", s.depth)
	}
}

// ─── TownScene: Basic Properties ──────────────────────────────────────

func TestNewTownScene(t *testing.T) {
	state := gamepkg.NewGameState()
	story := gamepkg.NewStoryManager()
	s := NewTownScene(state, story)

	if s == nil {
		t.Fatal("town scene should not be nil")
	}
	if s.state != state {
		t.Error("state should be the passed instance")
	}
	if s.story != story {
		t.Error("story should be the passed instance")
	}
	if s.tweens == nil {
		t.Error("tween manager should be initialized")
	}
	if s.particles == nil {
		t.Error("particle emitter should be initialized")
	}
}

// ─── NPC Types ────────────────────────────────────────────────────────

func TestNPCTypeValues(t *testing.T) {
	if NPCShopkeeper != 0 {
		t.Errorf("expected NPCShopkeeper=0, got %d", NPCShopkeeper)
	}
	if NPCQuestGiver != 1 {
		t.Errorf("expected NPCQuestGiver=1, got %d", NPCQuestGiver)
	}
	if NPCUpgradeMaster != 2 {
		t.Errorf("expected NPCUpgradeMaster=2, got %d", NPCUpgradeMaster)
	}
	if NPCCompanionTamer != 3 {
		t.Errorf("expected NPCCompanionTamer=3, got %d", NPCCompanionTamer)
	}
}