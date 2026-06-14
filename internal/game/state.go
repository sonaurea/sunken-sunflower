package game

import "github.com/hajimehoshi/ebiten/v2"

// ─── Global Game State ──────────────────────────────────────────────

// GameState holds the persistent state of the player's entire playthrough.
type GameState struct {
	// Player stats
	PlayerHealth    float64
	MaxPlayerHealth float64
	PlayerSpeed     float64
	PlayerDamage    float64
	DashCooldown    float64

	// Resources
	Inventory map[string]int // resource ID -> count

	// Progression
	CurrentDepth    int     // which dungeon floor
	MaxDepth        int     // deepest floor reached
	TotalGold       int
	TotalKills      int
	PlayTimeSeconds float64

	// Village state
	VillageBuildings map[string]int // building ID -> level
	UnlockedUpgrades map[string]bool

	// Companions
	ActiveCompanions []string // companion entity IDs
	MaxCompanions    int

	// Game flags
	HasWellKey  bool
	HasLantern  bool
	NPCMet      map[string]bool
	StoryFlags  map[string]bool
}

// NewGameState creates a fresh game state.
func NewGameState() *GameState {
	return &GameState{
		PlayerHealth:      100,
		MaxPlayerHealth:   100,
		PlayerSpeed:       120,
		PlayerDamage:      15,
		DashCooldown:      0.5,
		Inventory:         make(map[string]int),
		CurrentDepth:      1,
		MaxDepth:          1,
		VillageBuildings:  make(map[string]int),
		UnlockedUpgrades:  make(map[string]bool),
		ActiveCompanions:  make([]string, 0),
		MaxCompanions:     3,
		NPCMet:            make(map[string]bool),
		StoryFlags:        make(map[string]bool),
	}
}

// HasResource returns true if the player has at least `count` of the given resource.
func (gs *GameState) HasResource(resource string, count int) bool {
	return gs.Inventory[resource] >= count
}

// AddResource adds `count` to the given resource.
func (gs *GameState) AddResource(resource string, count int) {
	gs.Inventory[resource] += count
}

// RemoveResource removes up to `count` of the given resource. Returns actual amount removed.
func (gs *GameState) RemoveResource(resource string, count int) int {
	available := gs.Inventory[resource]
	if available <= 0 {
		return 0
	}
	removed := count
	if available < count {
		removed = available
	}
	gs.Inventory[resource] -= removed
	return removed
}

// IsUpgradeUnlocked checks if a permanent upgrade is active.
func (gs *GameState) IsUpgradeUnlocked(id string) bool {
	return gs.UnlockedUpgrades[id]
}

// UnlockUpgrade permanently unlocks an upgrade.
func (gs *GameState) UnlockUpgrade(id string) {
	gs.UnlockedUpgrades[id] = true
}

// BuildingLevel returns the current level of a village building.
func (gs *GameState) BuildingLevel(id string) int {
	return gs.VillageBuildings[id]
}

// UpgradeBuilding increments a building's level.
func (gs *GameState) UpgradeBuilding(id string) {
	gs.VillageBuildings[id]++
}

// ─── Story / Progression ────────────────────────────────────────────

// StoryManager tracks narrative flags and triggers.
type StoryManager struct {
	Flags    map[string]bool
	Dialogue map[string][]string // scene ID -> dialogue lines
}

func NewStoryManager() *StoryManager {
	return &StoryManager{
		Flags:    make(map[string]bool),
		Dialogue: make(map[string][]string),
	}
}

func (sm *StoryManager) SetFlag(flag string)              { sm.Flags[flag] = true }
func (sm *StoryManager) HasFlag(flag string) bool          { return sm.Flags[flag] }
func (sm *StoryManager) ClearFlag(flag string)             { delete(sm.Flags, flag) }

func (sm *StoryManager) RegisterDialogue(sceneID string, lines []string) {
	sm.Dialogue[sceneID] = lines
}

func (sm *StoryManager) GetDialogue(sceneID string) []string {
	return sm.Dialogue[sceneID]
}

// ─── Save / Load ────────────────────────────────────────────────────

// SaveData is the serializable snapshot for save files.
type SaveData struct {
	Version   int
	State     *GameState
	Story     map[string]bool
	SceneName string // scene to load into
}

// SaveGame serializes the current game state into a SaveData structure.
func SaveGame(state *GameState, story *StoryManager, sceneName string) *SaveData {
	sd := &SaveData{
		Version:   1,
		State:     state,
		Story:     make(map[string]bool),
		SceneName: sceneName,
	}
	for k, v := range story.Flags {
		sd.Story[k] = v
	}
	return sd
}

// LoadGame restores game state from a SaveData.
func LoadGame(sd *SaveData) (*GameState, *StoryManager, string) {
	if sd == nil {
		return NewGameState(), NewStoryManager(), "title"
	}
	state := sd.State
	story := NewStoryManager()
	for k, v := range sd.Story {
		story.Flags[k] = v
	}
	if state == nil {
		state = NewGameState()
	}
	if state.Inventory == nil {
		state.Inventory = make(map[string]int)
	}
	if state.VillageBuildings == nil {
		state.VillageBuildings = make(map[string]int)
	}
	if state.UnlockedUpgrades == nil {
		state.UnlockedUpgrades = make(map[string]bool)
	}
	if state.NPCMet == nil {
		state.NPCMet = make(map[string]bool)
	}
	if state.StoryFlags == nil {
		state.StoryFlags = make(map[string]bool)
	}
	return state, story, sd.SceneName
}

// Serialized sprite reference — for save files we just store entity IDs.
type SavedEntity struct {
	ID       string
	X, Y     float64
	Health   float64
	Type     string // "player", "companion", "enemy"
}

// ─── Event Registry (simple event bus) ──────────────────────────────

type EventHandler func(data interface{})

type EventBus struct {
	handlers map[string][]EventHandler
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]EventHandler),
	}
}

func (eb *EventBus) Subscribe(event string, handler EventHandler) {
	eb.handlers[event] = append(eb.handlers[event], handler)
}

func (eb *EventBus) Emit(event string, data interface{}) {
	if handlers, ok := eb.handlers[event]; ok {
		for _, h := range handlers {
			h(data)
		}
	}
}

// ─── Camera ─────────────────────────────────────────────────────────

// Camera provides viewport offset for scrolling scenes.
type Camera struct {
	X, Y float64
}

func NewCamera() *Camera {
	return &Camera{X: 0, Y: 0}
}

func (c *Camera) Follow(targetX, targetY, screenW, screenH, mapW, mapH float64) {
	c.X = targetX - screenW/2
	c.Y = targetY - screenH/2

	// Clamp to map bounds
	if c.X < 0 {
		c.X = 0
	}
	if c.Y < 0 {
		c.Y = 0
	}
	if c.X+screenW > mapW {
		c.X = mapW - screenW
	}
	if c.Y+screenH > mapH {
		c.Y = mapH - screenH
	}
}

func (c *Camera) TranslateOp() *ebiten.DrawImageOptions {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-c.X, -c.Y)
	return op
}