package scenes

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/michael/beach-dreams/internal/engine"
	"github.com/michael/beach-dreams/internal/entities"
	gamepkg "github.com/michael/beach-dreams/internal/game"
)

// ─── NPC Types ──────────────────────────────────────────────────────

type NPCType int

const (
	NPCShopkeeper NPCType = iota
	NPCQuestGiver
	NPCUpgradeMaster
	NPCCompanionTamer
)

// NPC represents a town NPC with interactivity.
type NPC struct {
	ID       string
	Name     string
	Type     NPCType
	X, Y     float64
	Sprite   *ebiten.Image
	Dialogue []string
	Color    color.Color
	Radius   float64
}

// NPCInteraction represents the current interaction state.
type NPCInteraction struct {
	Active   bool
	Npc      *NPC
	Menu     int
	Selected int
	Page     int
	Timer    float64
}

// ─── Town Scene ─────────────────────────────────────────────────────
// Inspired by Sonaurea's lyrics — a love letter set to code.
// "You're my sunrise" — the sunflower's journey begins in Sunken Beach.

type TownScene struct {
	engine.BaseScene
	camera      *gamepkg.Camera
	player      *entities.Player
	state       *gamepkg.GameState
	story       *gamepkg.StoryManager
	npcs        []*NPC
	interaction *NPCInteraction
	tweens      *engine.TweenManager
	shake       *engine.ScreenShake
	lightRays   []engine.LightRay
	damageNums  *engine.DamageNumberManager
	floatText   *engine.FloatingTextManager
	particles   *engine.ParticleEmitter
	gameTime    float64
}

func NewTownScene(state *gamepkg.GameState, story *gamepkg.StoryManager) *TownScene {
	return &TownScene{
		camera:     gamepkg.NewCamera(),
		state:      state,
		story:      story,
		tweens:     engine.NewTweenManager(),
		damageNums: engine.NewDamageNumberManager(),
		floatText:  engine.NewFloatingTextManager(),
		particles:  engine.NewParticleEmitter(),
		lightRays:  engine.GenerateLightRays(6, engine.ScreenWidth, engine.ScreenHeight),
	}
}

func (s *TownScene) Enter(g *engine.Game) {
	spawnX := 400.0
	spawnY := 300.0

	existing := g.EntityManager().Get("player")
	if existing == nil {
		s.player = entities.NewPlayer(spawnX, spawnY, s.state)
		g.EntityManager().Add(s.player)
	} else {
		s.player = existing.(*entities.Player)
		s.player.SetPosition(spawnX, spawnY)
		s.player.SetActive(true)
		s.state.PlayerHealth = s.player.Health
	}

	// NPCs named after Sonaurea's lyrical themes
	// "You're my sunrise" — Shopkeeper Sonaurea
	// "Like royalty, we're kings and queens" — Quest giver Royal
	// "Synergy, it's vibrating in me" — Craftswoman Synergy
	// "Your twin flame energy" — Companion Tamer Twin
	s.npcs = []*NPC{
		{
			ID: "shopkeeper", Name: "Sonaurea",
			Type: NPCShopkeeper, X: 130, Y: 200,
			Color: engine.ColSunflower,
			Radius: 42,
			Dialogue: []string{
				"You're my sunrise... welcome to my shop!",
				"I can't seem to find the words to use,",
				"but these goods speak for themselves.",
				"Don't stress it, we'll get through whatever it is.",
			},
		},
		{
			ID: "quest_giver", Name: "Royal",
			Type: NPCQuestGiver, X: 750, Y: 200,
			Color: engine.ColOcean,
			Radius: 42,
			Dialogue: []string{
				"You're the queen and I'm the king in royal blue.",
				"Like royalties, our love is always due.",
				"The well holds treasures fit for kings and queens.",
				"Bring me proof of your depth, and I'll reward you.",
			},
		},
		{
			ID: "upgrade_master", Name: "Synergy",
			Type: NPCUpgradeMaster, X: 250, Y: 450,
			Color: engine.ColSunset,
			Radius: 42,
			Dialogue: []string{
				"I feel the power... it's deep inside of you.",
				"Your twin flame energy is vibrating.",
				"Bring me resources, I'll unlock your potential.",
				"Like alchemy, we'll turn dust into gold.",
			},
		},
		{
			ID: "companion_tamer", Name: "Twin",
			Type: NPCCompanionTamer, X: 850, Y: 450,
			Color: engine.ColBio,
			Radius: 42,
			Dialogue: []string{
				"The creatures here have twin flame energy.",
				"They're looking for someone to complete them.",
				"Let me teach you to bond — you fill the void in them.",
				"The synergy between you will be beautiful.",
			},
		},
	}

	for _, compID := range s.state.ActiveCompanions {
		if g.EntityManager().Get(compID) == nil {
			comp := entities.NewCompanion(compID, spawnX-50, spawnY, entities.CompGlow)
			g.EntityManager().Add(comp)
		}
	}

	if !s.story.HasFlag("entered_town") {
		s.floatText.Add("* Sunken Beach welcomes you, my sunrise *", 300, 200, engine.ColSunflower)
		s.story.SetFlag("entered_town")
	}
}

func (s *TownScene) Exit(g *engine.Game) {}

func (s *TownScene) Update(g *engine.Game) {
	dt := g.DeltaTime()
	s.gameTime += dt

	s.tweens.Update(dt)
	s.damageNums.Update(dt)
	s.floatText.Update(dt)
	s.particles.Update(dt)
	if s.shake != nil && s.shake.IsActive() {
		s.shake.Update(dt)
	}

	input := g.Input()

	if s.interaction != nil && s.interaction.Active {
		s.handleInteraction(input, g)
		return
	}

	s.player.Update(g)

	// NPC proximity checks
	for _, npc := range s.npcs {
		dx := s.player.X - npc.X
		dy := s.player.Y - npc.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < npc.Radius {
			s.floatText.Add(fmt.Sprintf("[E] %s", npc.Name), npc.X-20, npc.Y-30, npc.Color)
			if input.ActionJustPressed {
				s.startInteraction(npc)
				return
			}
		}
	}

	// Well entrance — "The sunset feels like it's about to fade away"
	wellX, wellY := 600.0, 500.0
	dx := s.player.X - wellX
	dy := s.player.Y - wellY
	if math.Sqrt(dx*dx+dy*dy) < 50 {
		s.floatText.Add("[E] Descend into the depths", wellX-50, wellY-50, engine.ColMidnight)
		if input.ActionJustPressed {
			s.state.PlayTimeSeconds = s.gameTime
			g.SceneManager().SwitchTo("dungeon", g)
			return
		}
	}

	if !s.player.Active {
		s.player.Health = s.player.MaxHealth * 0.5
		s.player.Active = true
		s.state.PlayerHealth = s.player.Health
		g.SceneManager().SwitchTo("title", g)
	}

	s.camera.Follow(s.player.X, s.player.Y, engine.ScreenWidth, engine.ScreenHeight, 1280, 720)
}

func (s *TownScene) startInteraction(npc *NPC) {
	s.interaction = &NPCInteraction{
		Active: true,
		Npc:    npc,
		Menu:   0,
	}
	if !s.story.HasFlag("met_" + npc.ID) {
		s.story.SetFlag("met_" + npc.ID)
	}
}

func (s *TownScene) handleInteraction(input *engine.Input, g *engine.Game) {
	if input.CancelJustPressed {
		s.interaction = nil
		return
	}
	npc := s.interaction.Npc
	switch npc.Type {
	case NPCShopkeeper:
		s.handleShopInteraction(input, g)
	case NPCQuestGiver:
		s.handleQuestInteraction(input, g)
	case NPCUpgradeMaster:
		s.handleUpgradeInteraction(input, g)
	case NPCCompanionTamer:
		s.handleTamerInteraction(input, g)
	}
}

// ─── Shop — "Don't stress it, we'll get through whatever it is" ──────

type ShopItem struct {
	Name     string
	Resource string
	Cost     int
	Count    int
}

var shopItems = []ShopItem{
	{Name: "Healing Petal", Resource: "health_potion", Cost: 5, Count: 1},
	{Name: "Moon Sand (x5)", Resource: "moon_sand", Cost: 10, Count: 5},
	{Name: "Starfruit (x3)", Resource: "starfruit", Cost: 8, Count: 3},
	{Name: "Well Shard", Resource: "well_shard", Cost: 20, Count: 1},
	{Name: "Lantern Oil", Resource: "lantern_oil", Cost: 15, Count: 1},
}

func (s *TownScene) handleShopInteraction(input *engine.Input, g *engine.Game) {
	if s.interaction.Menu == 0 {
		if input.ActionJustPressed {
			s.interaction.Menu = 1
		}
		return
	}
	if input.UpJustPressed {
		s.interaction.Selected--
		if s.interaction.Selected < 0 {
			s.interaction.Selected = len(shopItems) - 1
		}
	}
	if input.DownJustPressed {
		s.interaction.Selected++
		if s.interaction.Selected >= len(shopItems) {
			s.interaction.Selected = 0
		}
	}
	if input.ActionJustPressed {
		item := shopItems[s.interaction.Selected]
		if s.state.TotalGold >= item.Cost {
			s.state.TotalGold -= item.Cost
			s.state.AddResource(item.Resource, item.Count)
			s.floatText.Add(fmt.Sprintf("~ %s ~", item.Name), 400, 360, engine.ColGreen)
			s.particles.EmitExplosion(400, 360, 8, engine.ColSunflower, 50, 0.5)
		} else {
			s.floatText.Add("Not enough gold, my sunrise...", 400, 380, engine.ColRed)
		}
	}
}

// ─── Quests — "Like royalties, our love is always due" ──────────────

type Quest struct {
	ID          string
	Name        string
	Description string
	Resource    string
	Required    int
	RewardGold  int
	RewardItem  string
	RewardCount int
	Completed   bool
}

var quests = []Quest{
	{
		ID: "first_descent", Name: "First Light",
		Description: "Reach depth 3 — the sunrise awaits",
		Required:    3, RewardGold: 50,
		RewardItem: "starfruit", RewardCount: 5,
	},
	{
		ID: "shard_collector", Name: "Shattered Pieces",
		Description: "Collect 10 well shards — pieces of me with you",
		Resource:    "well_shard", Required: 10,
		RewardGold: 100, RewardItem: "well_key", RewardCount: 1,
	},
	{
		ID: "moon_sand_baron", Name: "Royal Sands",
		Description: "Collect 50 moon sand — royalty in our desert of jewels",
		Resource:    "moon_sand", Required: 50,
		RewardGold: 200,
	},
	{
		ID: "depth_explorer", Name: "Ocean Breeze",
		Description: "Reach depth 10 — you and I, we flow like the sea",
		Required:    10, RewardGold: 500,
		RewardItem: "ancient_seed", RewardCount: 1,
	},
}

func (s *TownScene) handleQuestInteraction(input *engine.Input, g *engine.Game) {
	if s.interaction.Menu == 0 {
		if input.ActionJustPressed {
			s.interaction.Menu = 1
		}
		return
	}
	if input.UpJustPressed {
		s.interaction.Selected--
		if s.interaction.Selected < 0 {
			s.interaction.Selected = len(quests) - 1
		}
	}
	if input.DownJustPressed {
		s.interaction.Selected++
		if s.interaction.Selected >= len(quests) {
			s.interaction.Selected = 0
		}
	}
	if input.ActionJustPressed {
		q := &quests[s.interaction.Selected]
		if q.Completed {
			return
		}
		completed := false
		if q.Resource != "" {
			if s.state.Inventory[q.Resource] >= q.Required {
				s.state.RemoveResource(q.Resource, q.Required)
				completed = true
			}
		} else if q.Required > 0 {
			if s.state.MaxDepth >= q.Required {
				completed = true
			}
		}
		if completed {
			q.Completed = true
			s.state.TotalGold += q.RewardGold
			if q.RewardItem != "" {
				s.state.AddResource(q.RewardItem, q.RewardCount)
			}
			s.floatText.Add(fmt.Sprintf("* Royalty achieved! +%d gold *", q.RewardGold), 400, 360, engine.ColSunflower)
			s.particles.EmitExplosion(400, 360, 10, engine.ColSunflower, 60, 0.6)
		} else {
			s.floatText.Add("Keep exploring, my king/queen...", 400, 380, engine.ColBeachPink)
		}
	}
}

// ─── Upgrades — "Like alchemy, we'll turn everything into gold" ─────

type Upgrade struct {
	ID          string
	Name        string
	Description string
	Cost        map[string]int
	StatEffect  string
	EffectValue float64
	MaxLevel    int
}

var upgrades = []Upgrade{
	{
		ID: "health_up", Name: "Vitality Bloom",
		Description: "+25 Max HP — the power inside of you",
		Cost:        map[string]int{"moon_sand": 10, "starfruit": 5},
		StatEffect: "max_health", EffectValue: 25, MaxLevel: 5,
	},
	{
		ID: "dmg_up", Name: "Sunflare Edge",
		Description: "+5 Damage — your twin flame energy",
		Cost:        map[string]int{"well_shard": 3, "moon_sand": 15},
		StatEffect: "damage", EffectValue: 5, MaxLevel: 5,
	},
	{
		ID: "speed_up", Name: "Zephyr Step",
		Description: "+10 Speed — like the ocean breeze",
		Cost:        map[string]int{"starfruit": 8, "lantern_oil": 2},
		StatEffect: "speed", EffectValue: 10, MaxLevel: 3,
	},
	{
		ID: "armor_up", Name: "Shell of Light",
		Description: "+5 Armor — we turn dust into gold",
		Cost:        map[string]int{"well_shard": 5, "moon_sand": 20},
		StatEffect: "armor", EffectValue: 5, MaxLevel: 3,
	},
	{
		ID: "crit_up", Name: "Synergy Strike",
		Description: "+5% Crit — the synergy vibrating in me",
		Cost:        map[string]int{"starfruit": 10, "ancient_seed": 1},
		StatEffect: "crit", EffectValue: 0.05, MaxLevel: 3,
	},
}

func (s *TownScene) handleUpgradeInteraction(input *engine.Input, g *engine.Game) {
	if s.interaction.Menu == 0 {
		if input.ActionJustPressed {
			s.interaction.Menu = 1
		}
		return
	}
	if input.UpJustPressed {
		s.interaction.Selected--
		if s.interaction.Selected < 0 {
			s.interaction.Selected = len(upgrades) - 1
		}
	}
	if input.DownJustPressed {
		s.interaction.Selected++
		if s.interaction.Selected >= len(upgrades) {
			s.interaction.Selected = 0
		}
	}
	if input.ActionJustPressed {
		upg := &upgrades[s.interaction.Selected]
		level := s.state.BuildingLevel(upg.ID)
		if level >= upg.MaxLevel {
			s.floatText.Add("Already at full power!", 400, 360, engine.ColRed)
			return
		}
		canAfford := true
		for res, cost := range upg.Cost {
			if s.state.Inventory[res] < cost {
				canAfford = false
				break
			}
		}
		if canAfford {
			for res, cost := range upg.Cost {
				s.state.RemoveResource(res, cost)
			}
			s.state.UpgradeBuilding(upg.ID)
			s.floatText.Add(fmt.Sprintf("* %s ~", upg.Name), 400, 360, engine.ColSunset)
			s.particles.EmitExplosion(400, 360, 12, engine.ColSunset, 70, 0.5)
			switch upg.StatEffect {
			case "max_health":
				s.state.MaxPlayerHealth += upg.EffectValue
				s.state.PlayerHealth += upg.EffectValue
			case "damage":
				s.state.PlayerDamage += upg.EffectValue
			case "speed":
				s.state.PlayerSpeed += upg.EffectValue
			}
		} else {
			s.floatText.Add("Not enough synergy...", 400, 380, engine.ColRed)
		}
	}
}

// ─── Companion Tamer — "You fill the void in me" ────────────────────

type TameOption struct {
	Name string
	Type entities.CompanionType
	Cost map[string]int
}

var tameOptions = []TameOption{
	{
		Name: "Glow Pup (Heals)", Type: entities.CompGlow,
		Cost: map[string]int{"starfruit": 5, "moon_sand": 10},
	},
	{
		Name: "Shell Crab (Shield)", Type: entities.CompShield,
		Cost: map[string]int{"well_shard": 3, "moon_sand": 15},
	},
	{
		Name: "Spike Finch (Attack)", Type: entities.CompSpike,
		Cost: map[string]int{"starfruit": 8, "well_shard": 5},
	},
}

func (s *TownScene) handleTamerInteraction(input *engine.Input, g *engine.Game) {
	if s.interaction.Menu == 0 {
		if input.ActionJustPressed {
			s.interaction.Menu = 1
		}
		return
	}
	if input.UpJustPressed {
		s.interaction.Selected--
		if s.interaction.Selected < 0 {
			s.interaction.Selected = len(tameOptions) - 1
		}
	}
	if input.DownJustPressed {
		s.interaction.Selected++
		if s.interaction.Selected >= len(tameOptions) {
			s.interaction.Selected = 0
		}
	}
	if input.ActionJustPressed {
		if len(s.state.ActiveCompanions) >= s.state.MaxCompanions {
			s.floatText.Add("Your heart is full (max 3 companions)", 400, 360, engine.ColRed)
			return
		}
		opt := tameOptions[s.interaction.Selected]
		canAfford := true
		for res, cost := range opt.Cost {
			if s.state.Inventory[res] < cost {
				canAfford = false
				break
			}
		}
		if canAfford {
			for res, cost := range opt.Cost {
				s.state.RemoveResource(res, cost)
			}
			id := fmt.Sprintf("companion_%d", len(s.state.ActiveCompanions))
			comp := entities.NewCompanion(id, s.player.X-40, s.player.Y, opt.Type)
			g.EntityManager().Add(comp)
			s.state.ActiveCompanions = append(s.state.ActiveCompanions, id)
			s.floatText.Add(fmt.Sprintf("* %s bonded with you *", opt.Name), 400, 360, engine.ColBio)
			s.particles.EmitExplosion(400, 360, 15, engine.ColBio, 80, 0.6)
		} else {
			s.floatText.Add("Gather more resources for the bond...", 400, 380, engine.ColRed)
		}
	}
}

// ─── Draw ───────────────────────────────────────────────────────────
// "The sunset feels like it's about to fade away"
// "You and I, we flow like the sea"

func (s *TownScene) Draw(screen *ebiten.Image, g *engine.Game) {
	shakeX, shakeY := 0.0, 0.0
	if s.shake != nil && s.shake.IsActive() {
		shakeX, shakeY = s.shake.OffsetX, s.shake.OffsetY
	}

	// Sky gradient — "The sunset feels like it's about to fade away"
	engine.DrawGradient(screen, engine.ColOcean, engine.ColSand)

	// Light rays — "You're my sunrise"
	engine.DrawLightRays(screen, s.lightRays, g.GameTime())

	// Ocean — "You and I, we flow like the sea"
	engine.DrawOceanWaves(screen, g.GameTime())

	// Sand texture dots
	for i := 0; i < 30; i++ {
		x := math.Mod(float64(i)*47.3+float64(i)*13.1, float64(engine.ScreenWidth))
		y := math.Mod(float64(i)*89.7+float64(i)*7.3, float64(engine.ScreenHeight))
		engine.DrawCircle(screen, x, y, 2, color.RGBA{220, 200, 160, 30})
	}

	// === BUILDINGS ===
	// Sonaurea's Shop — "You're my sunrise"
	engine.DrawRect(screen, 100+shakeX, 150+shakeY, 100, 90, engine.ColBrown)
	engine.DrawRect(screen, 130+shakeX, 170+shakeY, 40, 40, engine.ColSunflower)
	engine.DrawPulsingBorder(screen, 100, 150, 100, 90, g.GameTime(), engine.ColSunflower, 2)
	engine.DrawText(screen, "* SUNRISE *", 108, 140, engine.ColSunflower)
	engine.DrawOrbitIndicator(screen, 150, 240, 8, g.GameTime(), engine.ColSunflower)

	// Royal's Quest Board — "Like royalties, we're kings and queens"
	engine.DrawRect(screen, 720+shakeX, 150+shakeY, 80, 100, engine.ColMidnight)
	engine.DrawRect(screen, 730+shakeX, 170+shakeY, 60, 40, color.RGBA{200, 180, 100, 255})
	engine.DrawPulsingBorder(screen, 720, 150, 80, 100, g.GameTime()*0.7, engine.ColOcean, 2)
	engine.DrawText(screen, "ROYAL", 730, 140, engine.ColOcean)
	engine.DrawOrbitIndicator(screen, 760, 295, 8, g.GameTime()*1.3, engine.ColOcean)

	// Synergy's Forge — "I feel the power"
	engine.DrawRect(screen, 220+shakeX, 430+shakeY, 90, 80, color.RGBA{80, 40, 20, 255})
	engine.DrawCircle(screen, 265+shakeX, 470+shakeY, 15, engine.ColSunset)
	engine.DrawPulsingBorder(screen, 220, 430, 90, 80, g.GameTime()*0.5, engine.ColSunset, 2)
	engine.DrawText(screen, "SYNERGY", 230, 420, engine.ColSunset)
	engine.DrawOrbitIndicator(screen, 265, 555, 8, g.GameTime()*0.9, engine.ColSunset)

	// Twin's Grove — "Your twin flame energy"
	engine.DrawRect(screen, 820+shakeX, 430+shakeY, 90, 80, color.RGBA{20, 60, 30, 255})
	engine.DrawCircle(screen, 865+shakeX, 470+shakeY, 15, engine.ColBio)
	engine.DrawPulsingBorder(screen, 820, 430, 90, 80, g.GameTime()*0.8, engine.ColBio, 2)
	engine.DrawText(screen, "TWIN", 840, 420, engine.ColBio)
	engine.DrawOrbitIndicator(screen, 865, 555, 8, g.GameTime()*1.1, engine.ColBio)

	// The Well — "The sunset fades away"
	wellX, wellY := 600.0, 500.0
	engine.DrawGlow(screen, wellX+shakeX, wellY+shakeY, 40, engine.ColMidnight)
	engine.DrawCircle(screen, wellX+shakeX, wellY+shakeY, 28, engine.ColMidnight)
	engine.DrawCircle(screen, wellX+shakeX, wellY+shakeY, 20, engine.ColBlack)
	engine.DrawRect(screen, wellX-30+shakeX, wellY+5+shakeY, 60, 8, engine.ColBrown)
	engine.DrawGlowRing(screen, wellX+shakeX, wellY+shakeY, 32, g.GameTime(), engine.ColDreamPurple)
	engine.DrawText(screen, "THE DEPTHS", int(wellX)-42+int(shakeX), int(wellY)+50+int(shakeY), engine.ColWhite)

	// Bioluminescent plants
	for i := 0; i < 5; i++ {
		bx := 450 + float64(i)*80 + math.Sin(g.GameTime()+float64(i))*10
		by := 550 + math.Cos(g.GameTime()*0.7+float64(i))*5
		engine.DrawCircle(screen, bx, by, 4+math.Sin(g.GameTime()+float64(i)*2), engine.ColBio)
	}

	// NPC sprites
	for _, npc := range s.npcs {
		engine.DrawGlowRing(screen, npc.X+shakeX, npc.Y+shakeY, 20, g.GameTime()+float64(npc.Type), npc.Color)
		engine.DrawCircle(screen, npc.X+shakeX, npc.Y+shakeY, 10, npc.Color)
		engine.DrawText(screen, npc.Name, int(npc.X)-20+int(shakeX), int(npc.Y)-25+int(shakeY), npc.Color)
	}

	s.particles.Draw(screen)
	s.floatText.Draw(screen)
	s.damageNums.Draw(screen, shakeX, shakeY)

	if s.interaction != nil && s.interaction.Active {
		s.drawInteractionMenu(screen, g)
	}

	// HUD
	hudStr := fmt.Sprintf("HP: %.0f/%.0f | ❤ x%d | 💰 %d",
		s.player.Health, s.player.MaxHealth,
		len(s.state.ActiveCompanions), s.state.TotalGold)
	engine.DrawRect(screen, 0, 0, 350, 28, engine.ColMidnight)
	engine.DrawText(screen, hudStr, 10, 20, engine.ColWhite)

	timeStr := fmt.Sprintf("[Sun] Depth Record: %d", s.state.MaxDepth)
	engine.DrawText(screen, timeStr, engine.ScreenWidth-200, 20, engine.ColBeachPink)

	engine.DrawText(screen, "WASD: Move | Shift: Dash | E: Interact | ESC: Back",
		10, engine.ScreenHeight-10, engine.ColWhite)
}

func (s *TownScene) drawInteractionMenu(screen *ebiten.Image, g *engine.Game) {
	npc := s.interaction.Npc
	px, py := 300.0, 200.0
	w, h := 680.0, 320.0

	engine.DrawRect(screen, px, py, w, h, color.RGBA{10, 10, 30, 220})
	engine.DrawPulsingBorder(screen, px, py, w, h, g.GameTime(), npc.Color, 2)

	engine.DrawText(screen, npc.Name, int(px)+20, int(py)+30, npc.Color)

	switch npc.Type {
	case NPCShopkeeper:
		s.drawShopMenu(screen, px, py)
	case NPCQuestGiver:
		s.drawQuestMenu(screen, px, py)
	case NPCUpgradeMaster:
		s.drawUpgradeMenu(screen, px, py)
	case NPCCompanionTamer:
		s.drawTamerMenu(screen, px, py)
	}
	engine.DrawText(screen, "[ESC] Close", int(px+w-100), int(py+h-25), engine.ColWhite)
}

func (s *TownScene) drawShopMenu(screen *ebiten.Image, px, py float64) {
	engine.DrawText(screen, "— Sunrise Sundries —", int(px)+250, int(py)+25, engine.ColSunflower)
	engine.DrawText(screen, fmt.Sprintf("Gold: 💰 %d", s.state.TotalGold), int(px)+20, int(py)+55, engine.ColSunflower)

	for i, item := range shopItems {
		y := py + 80 + float64(i)*35
		marker := "  "
		if i == s.interaction.Selected {
			marker = "-> "
			engine.DrawText(screen, marker, int(px)+20, int(y), engine.ColSunflower)
		}
		engine.DrawText(screen, fmt.Sprintf("%s %s — 💰 %d", marker, item.Name, item.Cost),
			int(px)+40, int(y), engine.ColWhite)
	}
	engine.DrawText(screen, "[E] Buy | [W/S] Navigate", int(px)+20, int(py)+270, engine.ColBeachPink)
}

func (s *TownScene) drawQuestMenu(screen *ebiten.Image, px, py float64) {
	engine.DrawText(screen, "— Royal Decrees —", int(px)+270, int(py)+25, engine.ColOcean)

	for i, q := range quests {
		y := py + 60 + float64(i)*45
		marker := "  "
		if i == s.interaction.Selected {
			marker = "-> "
		}
		status := ""
		if q.Completed {
			status = " ✓ Royal"
		}
		engine.DrawText(screen, fmt.Sprintf("%s%s%s", marker, q.Name, status),
			int(px)+40, int(y), engine.ColWhite)
		engine.DrawText(screen, q.Description,
			int(px)+60, int(y)+18, engine.ColBeachPink)
	}
	engine.DrawText(screen, "[E] Complete | [W/S] Navigate", int(px)+20, int(py)+270, engine.ColOcean)
}

func (s *TownScene) drawUpgradeMenu(screen *ebiten.Image, px, py float64) {
	engine.DrawText(screen, "— Forge of Synergy —", int(px)+250, int(py)+25, engine.ColSunset)

	for i, upg := range upgrades {
		y := py + 55 + float64(i)*40
		marker := "  "
		if i == s.interaction.Selected {
			marker = "-> "
		}
		level := s.state.BuildingLevel(upg.ID)
		maxStr := ""
		if level >= upg.MaxLevel {
			maxStr = " MAX POWER"
		}
		engine.DrawText(screen, fmt.Sprintf("%s%s Lv.%d/%d%s", marker, upg.Name, level, upg.MaxLevel, maxStr),
			int(px)+40, int(y), engine.ColWhite)
		engine.DrawText(screen, upg.Description,
			int(px)+60, int(y)+16, engine.ColBeachPink)
		costStr := ""
		for res, cost := range upg.Cost {
			costStr += fmt.Sprintf("%s:%d ", res, cost)
		}
		engine.DrawText(screen, "Cost: "+costStr,
			int(px)+60, int(y)+30, engine.ColGreen)
	}
	engine.DrawText(screen, "[E] Upgrade | [W/S] Navigate", int(px)+20, int(py)+270, engine.ColSunset)
}

func (s *TownScene) drawTamerMenu(screen *ebiten.Image, px, py float64) {
	engine.DrawText(screen, "— Twin Flame Grove —", int(px)+260, int(py)+25, engine.ColBio)
	engine.DrawText(screen, fmt.Sprintf("Bonds: %d/%d", len(s.state.ActiveCompanions), s.state.MaxCompanions),
		int(px)+20, int(py)+55, engine.ColBio)

	for i, opt := range tameOptions {
		y := py + 80 + float64(i)*40
		marker := "  "
		if i == s.interaction.Selected {
			marker = "-> "
		}
		engine.DrawText(screen, fmt.Sprintf("%s %s", marker, opt.Name),
			int(px)+40, int(y), engine.ColWhite)
		costStr := ""
		for res, cost := range opt.Cost {
			costStr += fmt.Sprintf("%s:%d ", res, cost)
		}
		engine.DrawText(screen, "Cost: "+costStr,
			int(px)+60, int(y)+16, engine.ColBeachPink)
	}
	engine.DrawText(screen, "[E] Bond | [W/S] Navigate", int(px)+20, int(py)+270, engine.ColBio)
}

func (s *TownScene) AddCompanion(g *engine.Game, compType entities.CompanionType) {
	id := fmt.Sprintf("companion_%d", len(s.state.ActiveCompanions))
	px, py := s.player.Position()
	comp := entities.NewCompanion(id, px-50, py, compType)
	g.EntityManager().Add(comp)
	s.state.ActiveCompanions = append(s.state.ActiveCompanions, id)
}

func GetQuests() []Quest { return quests }
