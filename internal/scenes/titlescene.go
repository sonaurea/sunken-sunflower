package scenes

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/michael/beach-dreams/internal/engine"
)

// ─── Title Scene ────────────────────────────────────────────────────

// TitleScene shows the game title with a beach sunset background.
type TitleScene struct {
	engine.BaseScene
	stars       []engine.Star
	titlePulse  float64
	showPrompt  bool
	promptTimer float64
}

func NewTitleScene() *TitleScene {
	return &TitleScene{
		stars: engine.GenerateStarField(80, engine.ScreenWidth, engine.ScreenHeight),
	}
}

func (s *TitleScene) Enter(g *engine.Game) {
	s.titlePulse = 0
	s.promptTimer = 0
	s.showPrompt = true
}

func (s *TitleScene) Exit(g *engine.Game) {}

func (s *TitleScene) Update(g *engine.Game) {
	dt := g.DeltaTime()
	s.titlePulse += dt
	s.promptTimer += dt

	// Blink prompt every 1.5 seconds
	if s.promptTimer > 1.5 {
		s.showPrompt = !s.showPrompt
		s.promptTimer = 0
	}

	// Press Space or Enter to start
	input := g.Input()
	if input.ActionJustPressed {
		g.SceneManager().SwitchTo("town", g)
	}
}

func (s *TitleScene) Draw(screen *ebiten.Image, g *engine.Game) {
	// Sky gradient (sunset)
	engine.DrawGradient(screen, engine.ColBeachPink, engine.ColSunset)

	// Ocean waves at bottom
	engine.DrawOceanWaves(screen, g.GameTime())

	// Stars (top half)
	engine.DrawStarField(screen, s.stars[:40], engine.ColWhite)

	// Title text
	title := "Sunken Sunflower"
	subtitle := "A Roguelike Dungeon-Crawler"

	// Draw title with glow effect
	cx := float64(engine.ScreenWidth) / 2
	cy := float64(engine.ScreenHeight) * 0.3

	glowSize := 80.0 + math.Sin(s.titlePulse*2)*20
	engine.DrawGlow(screen, cx, cy, glowSize, engine.ColSunflower)

	// Draw title text (centered)
	titleX := engine.CenterX(title, engine.ScreenWidth)
	titleY := int(cy) - 10
	engine.DrawText(screen, title, titleX, titleY, engine.ColSunflower)

	// Subtitle
	subX := engine.CenterX(subtitle, engine.ScreenWidth)
	subY := titleY + 40
	engine.DrawText(screen, subtitle, subX, subY, engine.ColWhite)

	// Draw decorative elements
	// Beach sand strip at bottom
	engine.DrawRect(screen, 0, float64(engine.ScreenHeight)-60,
		float64(engine.ScreenWidth), 60, engine.ColSand)

	// Well silhouette (simple circle)
	wellX := float64(engine.ScreenWidth) * 0.5
	wellY := float64(engine.ScreenHeight) - 60
	engine.DrawCircle(screen, wellX, wellY, 30, engine.ColMidnight)
	engine.DrawCircle(screen, wellX, wellY, 22, color.RGBA{10, 10, 30, 255})

	// Prompt
	if s.showPrompt {
		prompt := "Press SPACE or ENTER to begin"
		promptX := engine.CenterX(prompt, engine.ScreenWidth)
		promptY := int(float64(engine.ScreenHeight) * 0.8)
		engine.DrawText(screen, prompt, promptX, promptY, engine.ColSunflower)
	}

	// Version info
	versionStr := fmt.Sprintf("v0.1.0 | Depth: %d", 1)
	engine.DrawText(screen, versionStr, 10, engine.ScreenHeight-10, engine.ColWhite)
}