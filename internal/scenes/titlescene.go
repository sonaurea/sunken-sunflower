package scenes

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/michael/beach-dreams/internal/engine"
	"github.com/michael/beach-dreams/internal/net"
)

// ─── Title Scene with Multiplayer Menu ─────────────────────────────

type MenuState int

const (
	MenuMain     MenuState = iota
	MenuMultiplayer
	MenuHost
	MenuJoin
	MenuConnecting
)

type TitleScene struct {
	engine.BaseScene
	stars        []engine.Star
	titlePulse   float64
	menuState    MenuState
	selectedItem int
	promptTimer  float64
	hosting      bool
	statusText   string
	playersOnline int
	netMgr       *net.NetworkManager
}

func NewTitleScene() *TitleScene {
	return &TitleScene{
		stars:  engine.GenerateStarField(100, engine.ScreenWidth, engine.ScreenHeight),
		netMgr: net.NewNetworkManager(),
	}
}

func (s *TitleScene) Enter(g *engine.Game) {
	s.titlePulse = 0
	s.menuState = MenuMain
	s.selectedItem = 0
	s.promptTimer = 0
	s.statusText = ""
	s.playersOnline = 0
}

func (s *TitleScene) Exit(g *engine.Game) {
	if s.netMgr.IsConnected() {
		s.netMgr.Disconnect()
	}
}

func (s *TitleScene) Update(g *engine.Game) {
	dt := g.DeltaTime()
	s.titlePulse += dt
	s.promptTimer += dt
	input := g.Input()

	switch s.menuState {
	case MenuMain:
		s.updateMainMenu(input, g)
	case MenuMultiplayer:
		s.updateMultiplayerMenu(input, g)
	case MenuHost:
		s.updateHost(input, g)
	case MenuJoin:
		s.updateJoin(input, g)
	case MenuConnecting:
		s.updateConnecting(input, g)
	}
}

func (s *TitleScene) updateMainMenu(input *engine.Input, g *engine.Game) {
	if input.UpJustPressed || input.DownJustPressed {
		s.selectedItem = 1 - s.selectedItem // toggle between 2 items
	}

	if input.ActionJustPressed {
		switch s.selectedItem {
		case 0: // Single Player
			g.SceneManager().SwitchTo("town", g)
		case 1: // Multiplayer
			s.menuState = MenuMultiplayer
			s.selectedItem = 0
		}
	}
}

func (s *TitleScene) updateMultiplayerMenu(input *engine.Input, g *engine.Game) {
	if input.UpJustPressed || input.DownJustPressed {
		maxItems := 2
		if s.selectedItem == 0 {
			s.selectedItem = maxItems
		} else {
			s.selectedItem = 0
		}
	}

	if input.ActionJustPressed {
		switch s.selectedItem {
		case 0: // Host Game
			s.menuState = MenuHost
			s.statusText = "Starting host..."
			if err := s.netMgr.StartHost(0); err != nil {
				s.statusText = fmt.Sprintf("Failed: %v", err)
				s.menuState = MenuMultiplayer
			} else {
				s.hosting = true
				s.statusText = fmt.Sprintf("Hosting on port %d", net.DefaultPort)
				s.menuState = MenuMain // go to game as host
				g.SceneManager().SwitchTo("town", g)
			}
		case 1: // Join Game
			s.menuState = MenuJoin
			s.selectedItem = 0
			s.statusText = "Enter host IP or press ENTER for LAN"
		}
	}

	if input.CancelJustPressed {
		s.menuState = MenuMain
		s.selectedItem = 0
	}
}

func (s *TitleScene) updateHost(input *engine.Input, g *engine.Game) {
	if input.CancelJustPressed {
		s.menuState = MenuMultiplayer
	}
}

func (s *TitleScene) updateJoin(input *engine.Input, g *engine.Game) {
	if input.ActionJustPressed {
		s.menuState = MenuConnecting
		s.statusText = "Scanning LAN for hosts..."
		go func() {
			hosts, err := net.DiscoverLAN(2)
			if err != nil || len(hosts) == 0 {
				s.statusText = "No hosts found. Enter IP manually."
				s.menuState = MenuJoin
				return
			}
			host := hosts[0]
			s.statusText = fmt.Sprintf("Found host at %s:%d", host.Address, host.Port)
			if err := s.netMgr.Connect(host.Address, host.Port); err != nil {
				s.statusText = fmt.Sprintf("Connection failed: %v", err)
			} else {
				s.hosting = false
				s.playersOnline = host.PlayerCount + 1
				g.SceneManager().SwitchTo("town", g)
			}
			s.menuState = MenuMain
		}()
	}
	if input.CancelJustPressed {
		s.menuState = MenuMultiplayer
	}
}

func (s *TitleScene) updateConnecting(input *engine.Input, g *engine.Game) {
	if input.CancelJustPressed {
		s.menuState = MenuMultiplayer
	}
}

func (s *TitleScene) Draw(screen *ebiten.Image, g *engine.Game) {
	// Sky — sunset gradient
	engine.DrawGradient(screen, engine.ColBeachPink, engine.ColSunset)

	// Ocean
	engine.DrawOceanWaves(screen, g.GameTime())

	// Stars top half
	engine.DrawStarField(screen, s.stars[:50], engine.ColWhite)

	// Sunflower glow in center
	cx := float64(engine.ScreenWidth) / 2
	cy := float64(engine.ScreenHeight) * 0.22
	glowSize := 80.0 + math.Sin(s.titlePulse*2)*20
	engine.DrawGlow(screen, cx, cy, glowSize, engine.ColSunflower)

	// Title
	title := "Sunken Sunflower"
	titleX := engine.CenterX(title, engine.ScreenWidth)
	titleY := int(cy)
	engine.DrawText(screen, title, titleX, titleY, engine.ColSunflower)

	// Subtitle
	sub := "A Roguelike Dungeon-Crawler"
	subX := engine.CenterX(sub, engine.ScreenWidth)
	engine.DrawText(screen, sub, subX, titleY+30, engine.ColWhite)

	// Decorative beach elements
	engine.DrawRect(screen, 0, float64(engine.ScreenHeight)-60,
		float64(engine.ScreenWidth), 60, engine.ColSand)
	wellX := float64(engine.ScreenWidth) * 0.5
	wellY := float64(engine.ScreenHeight) - 60
	engine.DrawCircle(screen, wellX, wellY, 30, engine.ColMidnight)
	engine.DrawCircle(screen, wellX, wellY, 22, color.RGBA{10, 10, 30, 255})

	// ─── Menu ──────────────────────────────────────────────────────
	menuStartY := engine.ScreenHeight * 11 / 20
	menuClr := engine.ColWhite

	switch s.menuState {
	case MenuMain:
		items := []string{"  Single Player", "  Multiplayer"}
		for i, item := range items {
			y := menuStartY + i*35
			clr := menuClr
			if i == s.selectedItem {
				clr = engine.ColSunflower
				items[i] = "▶ " + item[2:]
			}
			engine.DrawText(screen, items[i], engine.CenterX(items[i], engine.ScreenWidth), y, clr)
		}

	case MenuMultiplayer:
		engine.DrawText(screen, "— Multiplayer —", engine.CenterX("— Multiplayer —", engine.ScreenWidth),
			menuStartY-10, engine.ColSunflower)
		items := []string{"  Host Game", "  Join Game"}
		for i, item := range items {
			y := menuStartY + 30 + i*35
			clr := menuClr
			if i == s.selectedItem {
				clr = engine.ColSunflower
				items[i] = "▶ " + item[2:]
			}
			engine.DrawText(screen, items[i], engine.CenterX(items[i], engine.ScreenWidth), y, clr)
		}
		engine.DrawText(screen, "[ESC] Back", engine.ScreenWidth-100, engine.ScreenHeight-10, engine.ColBeachPink)

	case MenuHost:
		engine.DrawText(screen, s.statusText, engine.CenterX(s.statusText, engine.ScreenWidth),
			menuStartY, engine.ColSunflower)
		engine.DrawText(screen, "Starting session...", engine.CenterX("Starting session...", engine.ScreenWidth),
			menuStartY+30, engine.ColWhite)

	case MenuJoin:
		engine.DrawText(screen, s.statusText, engine.CenterX(s.statusText, engine.ScreenWidth),
			menuStartY, engine.ColSunflower)
		engine.DrawText(screen, "[SPACE] Scan LAN | [ESC] Back",
			engine.CenterX("[SPACE] Scan LAN | [ESC] Back", engine.ScreenWidth),
			menuStartY+40, engine.ColWhite)

	case MenuConnecting:
		engine.DrawText(screen, s.statusText, engine.CenterX(s.statusText, engine.ScreenWidth),
			menuStartY, engine.ColSunflower)
	}

	// Version info
	ver := fmt.Sprintf("v0.1.0 | %d players max", net.MaxPlayers)
	engine.DrawText(screen, ver, 10, engine.ScreenHeight-10, engine.ColWhite)

	// Network status indicator
	if s.netMgr.IsConnected() {
		mode := "Host"
		if !s.hosting {
			mode = "Client"
		}
		status := fmt.Sprintf("🟢 %s | %d players", mode, s.netMgr.PlayerCount())
		engine.DrawText(screen, status, engine.ScreenWidth-engine.TextWidth(status)-10, 10, engine.ColGreen)
	}
}