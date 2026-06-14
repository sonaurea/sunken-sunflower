package entities

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/michael/beach-dreams/internal/engine"
)

// ─── Companion AI ───────────────────────────────────────────────────

type CompanionType int

const (
	CompGlow CompanionType = iota
	CompShield
	CompSpike
)

// Companion is an AI-driven ally that follows the player.
type Companion struct {
	engine.BaseEntity
	engine.AnimatedEntity
	CompType    CompanionType
	Health      float64
	MaxHealth   float64
	FollowDist  float64
	OrbitAngle  float64
	TargetX, TargetY float64
	Emitter     *engine.ParticleEmitter
}

// NewCompanion creates a companion at the given position.
func NewCompanion(id string, x, y float64, compType CompanionType) *Companion {
	var baseClr color.Color
	switch compType {
	case CompGlow:
		baseClr = engine.ColBio
	case CompShield:
		baseClr = engine.ColOcean
	case CompSpike:
		baseClr = engine.ColRed
	default:
		baseClr = engine.ColBio
	}

	c := &Companion{
		BaseEntity: engine.BaseEntity{
			IDValue:  id,
			X:        x,
			Y:        y,
			Active:   true,
			Speed:    100,
			Direction: 0,
		},
		CompType:   compType,
		Health:     50,
		MaxHealth:  50,
		FollowDist: 40,
		OrbitAngle: 0,
		Emitter:    engine.NewParticleEmitter(),
	}

	// Animation setup
	c.AnimatedEntity = *engine.NewAnimatedEntity()
	idleFrames := engine.GenerateIdleFrames(20, baseClr)
	walkFrames := engine.GenerateWalkFrames(20, baseClr)
	hurtFrames := engine.GenerateHurtFrames(20, baseClr)

	allFrames := append(append(idleFrames, walkFrames...), hurtFrames...)
	c.Frames = allFrames
	c.Alpha = 1.0
	c.Scale = 0.8

	c.Animator.AddClip(engine.PresetIdle())
	c.Animator.AddClip(engine.PresetWalk())
	c.Animator.AddClip(engine.PresetHurt())

	c.Animator.Clip("idle").FrameAnim = engine.NewFrameAnim([]int{0, 1, 2, 1}, 0.4, true, false)
	c.Animator.Clip("walk").FrameAnim = engine.NewFrameAnim([]int{4, 5, 6, 7, 8, 9}, 0.12, true, false)
	c.Animator.Clip("hurt").FrameAnim = engine.NewFrameAnim([]int{10, 11}, 0.1, false, false)

	// Set sprite sheet reference
	if len(allFrames) > 0 {
		c.SetSpriteSheet(allFrames[0], 20, 20)
	}

	c.Animator.Play("idle", true)

	return c
}

func (c *Companion) Update(g *engine.Game) {
	if !c.Active {
		return
	}

	dt := g.DeltaTime()

	// Update animation
	c.AnimatedEntity.Update(dt)

	// Follow the player
	playerEnt := g.EntityManager().Get("player")
	if playerEnt == nil || !playerEnt.IsActive() {
		return
	}
	px, py := playerEnt.Position()

	// Orbiting around player
	c.OrbitAngle += dt * 2.0
	offsetX := math.Cos(c.OrbitAngle+float64(c.CompType)*2.0) * c.FollowDist
	offsetY := math.Sin(c.OrbitAngle+float64(c.CompType)*2.0) * c.FollowDist

	targetX := px + offsetX
	targetY := py + offsetY - 10

	dx := targetX - c.X
	dy := targetY - c.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	// Determine if moving
	isMoving := dist > 5
	currentClip := c.Animator.CurrentClip()
	if isMoving {
		if currentClip != "walk" && currentClip != "hurt" {
			c.Animator.Play("walk", true)
		}
		c.FlipX = dx < 0
	} else if currentClip != "idle" && currentClip != "hurt" {
		c.Animator.Play("idle", true)
	}

	if dist > 2 {
		dx /= dist
		dy /= dist
		c.X += dx * c.Speed * dt
		c.Y += dy * c.Speed * dt
		c.TargetX = targetX
		c.TargetY = targetY
	}

	// Emit glow particles
	if math.Mod(g.GameTime(), 1.5) < dt {
		c.Emitter.EmitSparkle(c.X+10, c.Y+10, engine.ColBio)
	}

	// Heal player if Glow type
	if c.CompType == CompGlow {
		player, ok := playerEnt.(*Player)
		if ok && player.Health < player.MaxHealth {
			if math.Mod(g.GameTime(), 0.5) < dt {
				player.Heal(0.5)
			}
		}
	}

	c.Emitter.Update(dt)
}

func (c *Companion) Draw(screen *ebiten.Image, g *engine.Game) {
	if !c.Active {
		return
	}

	// Animated draw
	c.AnimatedEntity.Draw(screen, c.X+10, c.Y+10)
	c.Emitter.Draw(screen)

	// Type indicator
	switch c.CompType {
	case CompGlow:
		engine.DrawCircle(screen, c.X+10, c.Y+20, 2, engine.ColGreen)
	case CompShield:
		engine.DrawCircle(screen, c.X+10, c.Y+20, 2, engine.ColOcean)
	case CompSpike:
		engine.DrawCircle(screen, c.X+10, c.Y+20, 2, engine.ColRed)
	}
}