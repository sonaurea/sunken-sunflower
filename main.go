package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/michael/beach-dreams/internal/engine"
	gamepkg "github.com/michael/beach-dreams/internal/game"
	"github.com/michael/beach-dreams/internal/scenes"
)

func main() {
	settings := engine.DefaultSettings()

	// Create global game state
	state := gamepkg.NewGameState()
	story := gamepkg.NewStoryManager()

	// Initialize save manager for auto-save & manual saves
	saveMgr, err := gamepkg.NewSaveManager()
	if err != nil {
		log.Printf("[main] Save manager warning: %v", err)
		// Non-fatal — continue without saves
	}

	// Try to load auto-save if it exists
	loadScene := "title"
	if saveMgr != nil && saveMgr.HasAutoSave() {
		loadedState, loadedStory, scene, err := saveMgr.Load("auto")
		if err == nil {
			state = loadedState
			story = loadedStory
			loadScene = scene
			// Only load into town from auto-save
			if scene != "town" && scene != "dungeon" {
				loadScene = "title"
			}
			log.Printf("[main] Auto-save loaded (scene: %s, depth: %d)", scene, state.CurrentDepth)
		}
	}

	// Initialize Steam integration (stub)
	steam := gamepkg.NewSteamIntegration()
	steam.Init()

	// Create the game engine
	game := engine.NewGame(settings)

	// Register scenes
	game.SceneManager().Register("title", scenes.NewTitleScene())
	game.SceneManager().Register("town", scenes.NewTownScene(state, story))
	game.SceneManager().Register("dungeon", scenes.NewDungeonScene(state, story))
	game.SceneManager().Register("dream", scenes.NewDreamScene(state, story))

	// Start at loaded or title screen
	game.SceneManager().SwitchTo(loadScene, game)

	// Configure Ebitengine window
	ebiten.SetWindowTitle(settings.Title)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(settings.Fullscreen)
	ebiten.SetRunnableOnUnfocused(settings.RunnableOnUnfocused)
	ebiten.SetTPS(engine.InternalTPS)
	ebiten.SetVsyncEnabled(settings.VSyncEnabled)

	// Steam Deck optimization: hide cursor in fullscreen
	if steam.IsSteamDeck() || settings.Fullscreen {
		ebiten.SetCursorMode(ebiten.CursorModeHidden)
	}

	// Set up auto-save ticker via the Update loop
	// We wrap the game in a save-aware adapter
	saveGame := &SaveAwareGame{
		Game:    game,
		State:   state,
		Story:   story,
		SaveMgr: saveMgr,
		Steam:   steam,
	}

	log.Println("[main] Sunken Sunflower starting...")
	log.Printf("[main] Steam: %v, Auto-save: %v", steam.Initialized, saveMgr != nil)

	if err := ebiten.RunGame(saveGame); err != nil {
		log.Fatal(err)
	}
}

// ─── Save-Aware Game Wrapper ────────────────────────────────────────
// Wraps the engine.Game to add auto-save and achievement checks.

type SaveAwareGame struct {
	*engine.Game
	State   *gamepkg.GameState
	Story   *gamepkg.StoryManager
	SaveMgr *gamepkg.SaveManager
	Steam   *gamepkg.SteamIntegration
	tick    float64
}

func (sg *SaveAwareGame) Update() error {
	// Call original update
	if err := sg.Game.Update(); err != nil {
		return err
	}

	dt := sg.Game.DeltaTime()
	sg.tick += dt

	// Auto-save every 30 seconds if in town or dungeon
	if sg.SaveMgr != nil && sg.tick > 30 {
		sg.tick = 0
		scene := sg.Game.SceneManager().ActiveID()
		if scene == "town" || scene == "dungeon" {
			sg.State.PlayTimeSeconds += dt
			if err := sg.SaveMgr.AutoSaveSave(sg.State, sg.Story, scene); err != nil {
				log.Printf("[save] Auto-save error: %v", err)
			}
		}
	}

	// Achievement checks (Steam integration)
	sg.checkAchievements()

	return nil
}

func (sg *SaveAwareGame) checkAchievements() {
	state := sg.State

	// Depth achievements
	if state.MaxDepth >= 5 {
		sg.Steam.UnlockAchievement(gamepkg.AchDepth5)
	}
	if state.MaxDepth >= 10 {
		sg.Steam.UnlockAchievement(gamepkg.AchDepth10)
	}

	// Collector achievement
	for _, count := range state.Inventory {
		if count >= 100 {
			sg.Steam.UnlockAchievement(gamepkg.AchCollector)
			break
		}
	}

	// Companion achievement
	if len(state.ActiveCompanions) >= 1 {
		sg.Steam.UnlockAchievement(gamepkg.AchCompanion)
	}
	if len(state.ActiveCompanions) >= 3 {
		sg.Steam.UnlockAchievement(gamepkg.AchFullParty)
	}

	// First kill
	if state.TotalKills >= 1 {
		sg.Steam.UnlockAchievement(gamepkg.AchFirstBlood)
	}

	// Well key
	if state.HasWellKey {
		sg.Steam.UnlockAchievement(gamepkg.AchWellKey)
	}
}