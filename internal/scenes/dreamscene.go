package scenes

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/michael/beach-dreams/internal/engine"
	gamepkg "github.com/michael/beach-dreams/internal/game"
)

// ─── Dream Scene ────────────────────────────────────────────────────
// "The sunset feels like it's about to fade away"
// "You and me, we're free to explore the possibilities"

// DreamScene is a surreal narrative sequence between dungeon runs,
// inspired by Sonaurea's lyrics about love, depth, and transformation.
type DreamScene struct {
	engine.BaseScene
	stars     []engine.Star
	dreamText []string
	textIndex int
	textTimer float64
	showText  bool
	fadeAlpha float64
	complete  bool
	state     *gamepkg.GameState
	story     *gamepkg.StoryManager
	rays      []engine.LightRay
	particles *engine.ParticleEmitter
}

func NewDreamScene(state *gamepkg.GameState, story *gamepkg.StoryManager) *DreamScene {
	return &DreamScene{
		stars:     engine.GenerateStarField(120, engine.ScreenWidth, engine.ScreenHeight),
		state:     state,
		story:     story,
		rays:      engine.GenerateLightRays(8, engine.ScreenWidth, engine.ScreenHeight),
		particles: engine.NewParticleEmitter(),
	}
}

func (s *DreamScene) Enter(g *engine.Game) {
	s.fadeAlpha = 0
	s.textIndex = 0
	s.complete = false
	s.showText = false
	s.textTimer = 0
	s.dreamText = s.generateDreamText()
	s.particles.EmitExplosion(
		float64(engine.ScreenWidth)/2, float64(engine.ScreenHeight)/2,
		20, engine.ColDreamPurple, 100, 2.0,
	)
}

func (s *DreamScene) generateDreamText() []string {
	depth := s.state.CurrentDepth
	lines := []string{}

	switch {
	case depth == 1:
		lines = []string{
			"The sunset feels like it's about to fade away...",
			"You and me, we're free to explore the possibilities.",
			"There's something about the cyclical nature of the universe.",
			"The sunrise, the sunset, life and death.",
			"But you and me, we're free.",
		}
	case depth <= 3:
		lines = []string{
			"I can't seem to find the words to use...",
			"But the well speaks in colours, in vibes, in energy.",
			"Your twin flame energy is vibrating in me.",
			"I feel the power — it's deep inside of you.",
		}
	case depth <= 6:
		lines = []string{
			"The toxicity in me... let the demons die by the bedside.",
			"I know my demons get too dark in the night.",
			"But you see beauty even when I cry.",
			"Show me all your problems, I don't mind.",
		}
	case depth <= 10:
		lines = []string{
			"Like royalty, we're kings and queens in the depths.",
			"We can make gold from anything.",
			"We'll turn all our dust into gold.",
			"We'll rule this world, up on our throne.",
		}
	case depth <= 15:
		lines = []string{
			"Just like Amsterdam, you get me high.",
			"We can get high, we can get by.",
			"We can take a trip around the world.",
			"We got no limits.",
		}
	default:
		lines = []string{
			"You're the queen and I'm the king in royal blue.",
			"Bury us together, die with all the jewels.",
			"We can be royalty of a place like Timbuktu.",
			"Like alchemy, we'll turn everything into gold.",
			"At the core of the well, a seed awaits.",
			"The sunflower remembers what the sun forgot.",
		}
	}

	lines = append(lines,
		"...",
		"Press SPACE to awaken.")
	return lines
}

func (s *DreamScene) Exit(g *engine.Game) {}

func (s *DreamScene) Update(g *engine.Game) {
	dt := g.DeltaTime()

	s.particles.Update(dt)

	// Fade in
	if s.fadeAlpha < 1 && !s.complete {
		s.fadeAlpha += dt * 0.5
		if s.fadeAlpha > 1 {
			s.fadeAlpha = 1
		}
	}

	// Text progression
	if s.textIndex >= len(s.dreamText) {
		s.complete = true
	}

	if !s.complete {
		s.textTimer += dt
		if s.textTimer > 2.0 {
			s.showText = true
		}
		if g.Input().ActionJustPressed && s.showText {
			s.textIndex++
			s.textTimer = 0
			s.showText = false
			// Particle burst on text advance
			s.particles.EmitSparkle(
				float64(engine.ScreenWidth)/2, float64(engine.ScreenHeight)*0.7,
				engine.ColDreamPurple,
			)
		}
	}

	if s.complete && g.Input().ActionJustPressed {
		s.story.SetFlag("dream_seen")
		g.SceneManager().SwitchTo("town", g)
	}

	if g.Input().CancelJustPressed {
		g.SceneManager().SwitchTo("town", g)
	}
}

func (s *DreamScene) Draw(screen *ebiten.Image, g *engine.Game) {
	// Deep night sky — "The sunset fades away"
	engine.DrawGradient(screen, engine.ColMidnight, engine.ColDreamPurple)

	// Stars above
	engine.DrawStarField(screen, s.stars, engine.ColWhite)

	// Light rays filtering through
	engine.DrawLightRays(screen, s.rays, g.GameTime())

	// Floating magical particles
	for i := 0; i < 25; i++ {
		offset := g.GameTime()*5 + float64(i)*2.3
		x := math.Mod(float64(i)*137.5+math.Sin(offset)*30, float64(engine.ScreenWidth))
		y := math.Mod(float64(i)*89.3+math.Cos(offset)*20, float64(engine.ScreenHeight))
		alpha := uint8(30 + int(math.Sin(offset)*15))
		clr := engine.ColorRGBA(123, 45, 142, alpha)
		engine.DrawCircle(screen, x, y, 2, clr)
	}

	// Central sunflower glow
	cx := float64(engine.ScreenWidth) / 2
	cy := float64(engine.ScreenHeight) / 2
	pulse := math.Sin(g.GameTime()*1.5) * 0.3
	engine.DrawGlow(screen, cx, cy, 100+pulse*40, engine.ColDreamPurple)

	// Inner glow ring
	engine.DrawGlowRing(screen, cx, cy, 60, g.GameTime(), engine.ColSunflower)

	// Particles
	s.particles.Draw(screen)

	// Dream text
	if s.textIndex < len(s.dreamText) {
		line := s.dreamText[s.textIndex]
		tx := engine.CenterX(line, engine.ScreenWidth)
		ty := engine.ScreenHeight * 7 / 10

		if s.showText {
			// Text appears with a subtle glow behind it
			engine.DrawGlow(screen, float64(tx)+float64(engine.TextWidth(line))/2, float64(ty), 80, engine.ColorRGBA(255, 215, 0, 30))
			engine.DrawText(screen, line, tx, ty, engine.ColWhite)
		} else {
			ellipsis := ""
			for i := 0; i < int(g.GameTime()*2)%4; i++ {
				ellipsis += "."
			}
			engine.DrawText(screen, "*"+ellipsis, tx, ty, engine.ColDreamPurple)
		}
	} else {
		awake := "The dream fades, but the love remains..."
		tx := engine.CenterX(awake, engine.ScreenWidth)
		ty := engine.ScreenHeight * 7 / 10
		engine.DrawText(screen, awake, tx, ty, engine.ColWhite)
	}

	// Fade overlay
	if s.fadeAlpha < 1 {
		fadeClr := engine.ColorRGBA(0, 0, 0, uint8(255*(1-s.fadeAlpha)))
		engine.DrawRect(screen, 0, 0, engine.ScreenWidth, engine.ScreenHeight, fadeClr)
	}

	// Depth indicator
	depthStr := fmt.Sprintf("~ Depth %d ~", s.state.CurrentDepth)
	engine.DrawText(screen, depthStr, 20, engine.ScreenHeight-20, engine.ColDreamPurple)

	// Lyric credit
	credit := "— Sonaurea"
	engine.DrawText(screen, credit, engine.ScreenWidth-engine.TextWidth(credit)-20, engine.ScreenHeight-20, color.RGBA{255, 255, 255, 60})
}