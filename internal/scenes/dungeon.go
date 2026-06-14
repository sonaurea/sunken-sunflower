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

// ─── Isometric Dungeon Scene ───────────────────────────────────────
// Bright, visible, with rich character sprites and clear text.

type DungeonScene struct {
	engine.BaseScene
	isoMap      *engine.IsoMap
	isoConfig   engine.IsoConfig
	isoCamera   *engine.IsoCamera
	isoDrawer   *engine.IsoEntityDrawer
	player      *entities.Player
	state       *gamepkg.GameState
	story       *gamepkg.StoryManager
	enemies     []*entities.Enemy
	hits        *engine.HitboxManager
	dialogue    *engine.DialogueManager
	gameTime    float64
	depth       int
	exitReached bool
}

func NewDungeonScene(state *gamepkg.GameState, story *gamepkg.StoryManager) *DungeonScene {
	cfg := engine.DefaultIsoConfig()
	return &DungeonScene{
		isoConfig: cfg,
		isoCamera: engine.NewIsoCamera(cfg),
		isoDrawer: engine.NewIsoEntityDrawer(),
		state:     state,
		story:     story,
		depth:     1,
		hits:      engine.NewHitboxManager(),
		dialogue:  engine.NewDialogueManager(),
	}
}

func (s *DungeonScene) Enter(g *engine.Game) {
	s.depth = s.state.CurrentDepth
	s.exitReached = false
	s.gameTime = 0

	existing := g.EntityManager().Get("player")
	if existing != nil {
		s.player = existing.(*entities.Player)
		s.player.SetActive(true)
	} else {
		s.player = entities.NewPlayer(1, 1, s.state)
		g.EntityManager().Add(s.player)
	}

	s.generateDungeon()
	s.player.SetPosition(3, 3)
	s.isoCamera.Follow(3, 3)
	s.isoCamera.Smoothing = 0.12

	if s.depth == 1 {
		s.story.SetFlag("entered_dungeon")
	}
}

func (s *DungeonScene) Exit(g *engine.Game) {}

func (s *DungeonScene) generateDungeon() {
	mapW := 15 + s.depth
	mapH := 10 + s.depth/2
	if mapW > 35 {
		mapW = 35
	}
	if mapH > 25 {
		mapH = 25
	}

	s.isoMap = engine.NewIsoMap(mapW, mapH, s.isoConfig)
	s.enemies = nil

	// Floor colors — bright and readable
	brightFloor := color.RGBA{60, 55, 45, 255}
	altFloor := color.RGBA{70, 65, 50, 255}
	wallClr := color.RGBA{90, 60, 40, 255}

	for x := 0; x < mapW; x++ {
		for y := 0; y < mapH; y++ {
			tile := s.isoMap.TileAt(x, y)
			if x == 0 || y == 0 || x == mapW-1 || y == mapH-1 {
				tile.HasWall = true
				tile.WallColor = wallClr
				tile.Color = color.RGBA{40, 35, 30, 255}
			} else {
				if (x+y)%2 == 0 {
					tile.Color = brightFloor
				} else {
					tile.Color = altFloor
				}
			}
		}
	}

	// Rooms with elevation visible to player
	numRooms := 2 + s.depth/2
	if numRooms > 6 {
		numRooms = 6
	}
	for i := 0; i < numRooms; i++ {
		rx := 3 + rand.Intn(mapW-6)
		ry := 2 + rand.Intn(mapH-4)
		rw := 3 + rand.Intn(4)
		rh := 2 + rand.Intn(3)

		for dx := -rw / 2; dx <= rw/2; dx++ {
			for dy := -rh / 2; dy <= rh/2; dy++ {
				t := s.isoMap.TileAt(rx+dx, ry+dy)
				if t != nil && !t.IsExit {
					t.HasWall = false
					t.Color = color.RGBA{75, 70, 55, 255}
					if (dx+dy)%2 == 0 {
						t.Color = color.RGBA{85, 80, 60, 255}
					}
					t.Elevation = 0
				}
			}
		}

		if i > 0 {
			numEnemies := 1 + rand.Intn(1+s.depth/3)
			if numEnemies > 3 {
				numEnemies = 3
			}
			for j := 0; j < numEnemies; j++ {
				ex := rx + rand.Intn(rw) - rw/2
				ey := ry + rand.Intn(rh) - rh/2
				etype := []entities.EnemyType{
					entities.EnemyBasic, entities.EnemyCharger,
					entities.EnemyRanged, entities.EnemySwarm,
				}[rand.Intn(4)]
				enemy := entities.NewEnemy(
					fmt.Sprintf("enemy_%d_%d", i, j),
					float64(ex), float64(ey), etype, s.depth,
				)
				s.enemies = append(s.enemies, enemy)
			}
		}
	}

	exitX := mapW - 3
	exitY := mapH / 2
	exitTile := s.isoMap.TileAt(exitX, exitY)
	if exitTile != nil {
		exitTile.IsExit = true
		exitTile.Color = color.RGBA{255, 215, 0, 255}
		exitTile.Elevation = 1.5
	}
}

func (s *DungeonScene) Update(g *engine.Game) {
	dt := g.DeltaTime()
	s.gameTime += dt

	if s.exitReached {
		s.state.MaxDepth = s.depth
		s.state.CurrentDepth = s.depth + 1
		g.SceneManager().SwitchTo("town", g)
		return
	}

	// Dialogue takes priority
	if s.dialogue.Active {
		s.dialogue.Update(dt)
		if g.Input().ActionJustPressed {
			s.dialogue.Advance()
		}
		return
	}

	s.player.Update(g)

	if !s.player.Active {
		s.player.Health = s.player.MaxHealth * 0.5
		s.player.Active = true
		g.SceneManager().SwitchTo("dream", g)
		return
	}

	// Update enemies
	allDead := true
	for _, enemy := range s.enemies {
		if enemy.IsActive() {
			enemy.Update(g)
			allDead = false
		}
	}

	if allDead && len(s.enemies) > 0 {
		for _, enemy := range s.enemies {
			drops := enemy.GetDrops()
			for item, count := range drops {
				s.state.AddResource(item, count)
			}
		}
		s.state.TotalKills += len(s.enemies)
		for x := 0; x < s.isoMap.Width; x++ {
			for y := 0; y < s.isoMap.Height; y++ {
				t := s.isoMap.TileAt(x, y)
				if t != nil && t.IsExit {
					t.Elevation = 2.5
				}
			}
		}
		s.dialogue.Enqueue([]engine.DialogueLine{
			{Speaker: "Narrator", Text: "All enemies defeated. The exit glows ahead.", Color: engine.ColSunflower, PortraitClr: engine.ColSunflower},
		})
	}

	px, py := s.player.Position()
	exitX, exitY := s.isoMap.Width-3, s.isoMap.Height/2
	exitTile := s.isoMap.TileAt(exitX, exitY)
	if exitTile != nil && exitTile.IsExit {
		dx := px - float64(exitX)
		dy := py - float64(exitY)
		if math.Sqrt(dx*dx+dy*dy) < 2.5 {
			s.exitReached = true
		}
	}

	// Clamp
	if px < 1 {
		px = 1
	}
	if py < 1 {
		py = 1
	}
	if px > float64(s.isoMap.Width)-2 {
		px = float64(s.isoMap.Width) - 2
	}
	if py > float64(s.isoMap.Height)-2 {
		py = float64(s.isoMap.Height) - 2
	}
	s.player.SetPosition(px, py)

	s.isoCamera.Follow(px, py)
	s.isoCamera.Update()
}

func (s *DungeonScene) Draw(screen *ebiten.Image, g *engine.Game) {
	// Warm dungeon background
	screen.Fill(color.RGBA{25, 22, 18, 255})

	camX, camY := s.isoCamera.ScreenOffset()

	// Draw isometric map — bright and visible
	if s.isoMap != nil {
		s.isoMap.Draw(screen, camX, camY)
	}

	// Draw enemies with isometric depth
	s.isoDrawer = engine.NewIsoEntityDrawer()
	for _, enemy := range s.enemies {
		if !enemy.IsActive() {
			continue
		}
		ex, ey := enemy.Position()
		sx, sy := engine.WorldToScreen(ex, ey, 0.3, s.isoConfig, camX, camY)
		enemyCopy := enemy
		s.isoDrawer.Add(sx, sy, ex+ey, func(img *ebiten.Image) {
			enemyCopy.Draw(img, g)
		})
	}

	// Draw player
	px, py := s.player.Position()
	psx, psy := engine.WorldToScreen(px, py, 0.6, s.isoConfig, camX, camY)
	s.isoDrawer.Add(psx, psy, px+py+1, func(img *ebiten.Image) {
		s.player.Draw(img, g)
	})
	s.isoDrawer.Draw(screen)

	// HUD — ALWAYS on top, dark background for readability
	engine.DrawRect(screen, 0, 0, 550, 28, color.RGBA{0, 0, 0, 180})
	hudStr := fmt.Sprintf("Well: Depth %d | HP: %.0f/%.0f | Enemies: %d",
		s.depth, s.player.Health, s.player.MaxHealth, s.countAlive())
	engine.DrawText(screen, hudStr, 10, 20, engine.ColWhite)

	// Controls at bottom — dark background
	engine.DrawRect(screen, 0, engine.ScreenHeight-22, engine.ScreenWidth, 22, color.RGBA{0, 0, 0, 180})
	engine.DrawText(screen, "WASD: Move | Shift: Dash | Defeat all enemies → Golden Exit",
		10, engine.ScreenHeight-7, engine.ColWhite)

	// Mini-map
	if s.isoMap != nil {
		engine.DrawIsoMinimap(screen, s.isoMap, px, py)
	}

	// Dialogue on top of everything
	s.dialogue.Draw(screen)
}

func (s *DungeonScene) countAlive() int {
	c := 0
	for _, e := range s.enemies {
		if e.IsActive() {
			c++
		}
	}
	return c
}