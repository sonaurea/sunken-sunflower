package scenes

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/michael/beach-dreams/internal/engine"
	"github.com/michael/beach-dreams/internal/entities"
	gamepkg "github.com/michael/beach-dreams/internal/game"
)

// ─── Dungeon Scene ──────────────────────────────────────────────────

// Room represents a single dungeon room.
type Room struct {
	X, Y        float64
	W, H        float64
	Enemies     []*entities.Enemy
	Cleared     bool
	HasExit     bool // leads to next floor
	HasChest    bool
	ChestLooted bool
	Type        string // "combat", "treasure", "exit", "boss"
}

// DungeonScene is the procedural dungeon crawling scene.
type DungeonScene struct {
	engine.BaseScene
	camera        *gamepkg.Camera
	player        *entities.Player
	state         *gamepkg.GameState
	story         *gamepkg.StoryManager
	rooms         []*Room
	currentRoom   int
	tileSize      float64
	floorComplete bool
	exitX, exitY  float64
}

func NewDungeonScene(state *gamepkg.GameState, story *gamepkg.StoryManager) *DungeonScene {
	return &DungeonScene{
		camera:   gamepkg.NewCamera(),
		state:    state,
		story:    story,
		tileSize: 32,
	}
}

func (s *DungeonScene) Enter(g *engine.Game) {
	// Get or create player
	existing := g.EntityManager().Get("player")
	if existing != nil {
		s.player = existing.(*entities.Player)
		s.player.SetActive(true)
	} else {
		s.player = entities.NewPlayer(64, 64, s.state)
		g.EntityManager().Add(s.player)
	}

	// Generate dungeon floor
	s.generateFloor()

	// Position player at start
	s.player.SetPosition(64, 64)
	s.currentRoom = 0
	s.floorComplete = false

	// Room entry narration
	if s.state.CurrentDepth == 1 {
		s.story.SetFlag("entered_dungeon")
	}
}

func (s *DungeonScene) Exit(g *engine.Game) {
	// Keep player for scene transitions
}

func (s *DungeonScene) generateFloor() {
	s.rooms = nil
	numRooms := 3 + s.state.CurrentDepth
	if numRooms > 8 {
		numRooms = 8
	}

	roomW := 320.0
	roomH := 240.0
	spacing := 40.0

	for i := 0; i < numRooms; i++ {
		room := &Room{
			X:   float64(i) * (roomW + spacing),
			Y:   60.0 + math.Mod(float64(i)*1.5, 3)*20,
			W:   roomW,
			H:   roomH,
			HasExit:  i == numRooms-1, // last room has exit
			HasChest: rand.Float64() < 0.3 && i > 0,
			Type:     "combat",
		}

		// Spawn enemies based on depth
		if i < numRooms-1 || rand.Float64() < 0.7 {
			numEnemies := 1 + rand.Intn(2+s.state.CurrentDepth/2)
			if numEnemies > 5 {
				numEnemies = 5
			}
			for j := 0; j < numEnemies; j++ {
				ex := room.X + 40 + rand.Float64()*(roomW-80)
				ey := room.Y + 40 + rand.Float64()*(roomH-80)
				var etype entities.EnemyType
				switch rand.Intn(4) {
				case 0:
					etype = entities.EnemyBasic
				case 1:
					etype = entities.EnemyCharger
				case 2:
					etype = entities.EnemyRanged
				case 3:
					etype = entities.EnemySwarm
				}
				enemy := entities.NewEnemy(
					fmt.Sprintf("enemy_%d_%d", i, j),
					ex, ey, etype, s.state.CurrentDepth,
				)
				room.Enemies = append(room.Enemies, enemy)
			}
		}

		if room.HasExit {
			room.Type = "exit"
		}
		if room.HasChest && room.Type != "exit" {
			room.Type = "treasure"
		}

		s.rooms = append(s.rooms, room)
	}

	// Exit position
	last := s.rooms[len(s.rooms)-1]
	s.exitX = last.X + last.W/2
	s.exitY = last.Y + last.H/2
}

func (s *DungeonScene) Update(g *engine.Game) {
	dt := g.DeltaTime()

	if s.floorComplete {
		// Walk to exit
		px, py := s.player.Position()
		dx := s.exitX - px
		dy := s.exitY - py
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 10 {
			s.player.SetPosition(
				px+(dx/dist)*s.player.Speed*dt,
				py+(dy/dist)*s.player.Speed*dt,
			)
		}

		// Exit area reached — go back to town or next floor
		if dist < 30 {
			s.state.MaxDepth = s.state.CurrentDepth
			g.SceneManager().SwitchTo("town", g)
		}
		return
	}

	// Update player
	s.player.Update(g)

	// Check player death
	if !s.player.Active {
		s.player.Health = s.player.MaxHealth * 0.5
		s.player.Active = true
		s.state.CurrentDepth = 1
		g.SceneManager().SwitchTo("town", g)
		return
	}

	// Update enemies in current room
	current := s.getCurrentRoom()
	if current != nil && !current.Cleared {
		allDead := true
		for _, enemy := range current.Enemies {
			if enemy.IsActive() {
				enemy.Update(g)
				allDead = false
			}
		}

		// Check if room is cleared
		if allDead && len(current.Enemies) > 0 {
			current.Cleared = true
			s.state.TotalKills += len(current.Enemies)

			// Drop resources
			for _, enemy := range current.Enemies {
				drops := enemy.GetDrops()
				for item, count := range drops {
					s.state.AddResource(item, count)
				}
			}

			// Chest in treasure rooms
			if current.HasChest && !current.ChestLooted {
				s.chestLoot(current)
			}
		}
	}

	// Move to next room if at right edge
	px, py := s.player.Position()
	if current != nil && px > current.X+current.W-20 {
		if s.currentRoom < len(s.rooms)-1 {
			s.currentRoom++
			s.player.SetPosition(s.rooms[s.currentRoom].X+10, py)
		}
	}
	// Move to previous room
	if current != nil && px < current.X+10 {
		if s.currentRoom > 0 {
			s.currentRoom--
			s.player.SetPosition(s.rooms[s.currentRoom].X+s.rooms[s.currentRoom].W-20, py)
		}
	}

	// Clamp player to current room
	if current != nil {
		px, py := s.player.Position()
		if px < current.X {
			px = current.X
		}
		if px > current.X+current.W {
			px = current.X + current.W
		}
		if py < current.Y {
			py = current.Y
		}
		if py > current.Y+current.H {
			py = current.Y + current.H
		}
		s.player.SetPosition(px, py)
	}

	// Camera follows player
	mapWidth := s.rooms[len(s.rooms)-1].X + s.rooms[len(s.rooms)-1].W + 100
	mapHeight := 500.0
	s.camera.Follow(px, py, engine.ScreenWidth, engine.ScreenHeight, mapWidth, mapHeight)

	// Check if all rooms cleared and exit reached
	if current != nil && current.HasExit && current.Cleared {
		s.floorComplete = true
		s.state.CurrentDepth++
	}
}

func (s *DungeonScene) Draw(screen *ebiten.Image, g *engine.Game) {
	// Dungeon background
	screen.Fill(color.RGBA{10, 10, 30, 255}) // deep dark

	// Draw rooms
	for _, room := range s.rooms {
		s.drawRoom(screen, room, g)
	}

	// Draw enemies in current room
	current := s.getCurrentRoom()
	if current != nil {
		for _, enemy := range current.Enemies {
			if enemy.IsActive() {
				enemy.Draw(screen, g)
			}
		}
	}

	// Draw exit marker
	if s.floorComplete {
		engine.DrawGlow(screen, s.exitX, s.exitY, 30, engine.ColBio)
		engine.DrawText(screen, "EXIT → TOWN", int(s.exitX)-40, int(s.exitY)-30, engine.ColBio)
	}

	// HUD
	depthStr := fmt.Sprintf("Well Depth: %d | Room: %d/%d | HP: %.0f/%.0f",
		s.state.CurrentDepth, s.currentRoom+1, len(s.rooms),
		s.player.Health, s.player.MaxHealth)
	engine.DrawRect(screen, 0, 0, 600, 25, engine.ColMidnight)
	engine.DrawText(screen, depthStr, 10, 18, engine.ColWhite)

	// Controls hint
	engine.DrawText(screen, "WASD: Move | Shift: Dash | Clear all rooms to descend",
		10, engine.ScreenHeight-10, engine.ColWhite)

	// Inventory quick view
	invY := 30
	for res, count := range s.state.Inventory {
		if count > 0 {
			invStr := fmt.Sprintf("%s: %d", res, count)
			engine.DrawText(screen, invStr, 10, invY+18, engine.ColSunflower)
			invY += 18
			if invY > 150 {
				break
			}
		}
	}
}

func (s *DungeonScene) drawRoom(screen *ebiten.Image, room *Room, g *engine.Game) {
	// Room floor
	floorClr := color.RGBA{30, 30, 50, 255}
	if room.Cleared {
		floorClr = color.RGBA{40, 40, 60, 255}
	}
	engine.DrawRect(screen, room.X, room.Y, room.W, room.H, floorClr)

	// Room border
	borderClr := color.RGBA{80, 80, 120, 255}
	if room.HasExit {
		borderClr = engine.ColBio
	}
	engine.DrawRect(screen, room.X, room.Y, room.W, 2, borderClr)
	engine.DrawRect(screen, room.X, room.Y, 2, room.H, borderClr)
	engine.DrawRect(screen, room.X+room.W-2, room.Y, 2, room.H, borderClr)
	engine.DrawRect(screen, room.X, room.Y+room.H-2, room.W, 2, borderClr)

	// Room label
	label := ""
	idx := s.getRoomIndex(room)
	switch room.Type {
	case "combat":
		if !room.Cleared {
			label = fmt.Sprintf("⚔ Room %d", idx+1)
		} else {
			label = fmt.Sprintf("✓ Room %d", idx+1)
		}
	case "treasure":
		if room.ChestLooted {
			label = fmt.Sprintf("□ Room %d (Looted)", idx+1)
		} else {
			label = fmt.Sprintf("■ Room %d (Treasure!)", idx+1)
		}
	case "exit":
		label = fmt.Sprintf("▼ Room %d (Exit)", idx+1)
	}
	engine.DrawText(screen, label, int(room.X)+10, int(room.Y)+15, engine.ColWhite)

	// Draw chest
	if room.HasChest && !room.ChestLooted {
		cx := room.X + room.W/2 - 12
		cy := room.Y + room.H/2 - 12
		engine.DrawRect(screen, cx, cy, 24, 20, engine.ColSunflower)
		engine.DrawRect(screen, cx+4, cy-4, 16, 6, engine.ColBrown)
	}

	// Room connection indicator (next room arrow)
	if idx < len(s.rooms)-1 && room.Cleared {
		arrowX := room.X + room.W - 16
		arrowY := room.Y + room.H/2
		engine.DrawCircle(screen, arrowX, arrowY, 6, engine.ColGreen)
		engine.DrawText(screen, "▶", int(arrowX)-4, int(arrowY)+5, engine.ColWhite)
	}
}

func (s *DungeonScene) getCurrentRoom() *Room {
	if s.currentRoom >= 0 && s.currentRoom < len(s.rooms) {
		return s.rooms[s.currentRoom]
	}
	return nil
}

func (s *DungeonScene) getRoomIndex(room *Room) int {
	for i, r := range s.rooms {
		if r == room {
			return i
		}
	}
	return -1
}

func (s *DungeonScene) chestLoot(room *Room) {
	room.ChestLooted = true
	// Random loot
	loot := []string{"moon_sand", "starfruit", "well_shard", "gold"}
	item := loot[rand.Intn(len(loot))]
	count := 1 + rand.Intn(5)
	s.state.AddResource(item, count)
	s.state.TotalGold += rand.Intn(20)
	_ = item // used for logging in future
}