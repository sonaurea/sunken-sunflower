package entities

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/michael/beach-dreams/internal/engine"
	gamepkg "github.com/michael/beach-dreams/internal/game"
)

// ─── Player ─────────────────────────────────────────────────────────

// Player is the sunflower character controlled by the player.
type Player struct {
	engine.BaseEntity
	engine.AnimatedEntity
	Health       float64
	MaxHealth    float64
	Damage       float64
	DashTimer    float64
	DashCooldown float64
	IsDashing    bool
	Invincible   float64
	State        *gamepkg.GameState
	Emitter      *engine.ParticleEmitter
	isMoving     bool
}

// NewPlayer creates the player entity at the given position.
func NewPlayer(x, y float64, state *gamepkg.GameState) *Player {
	p := &Player{
		BaseEntity: engine.BaseEntity{
			IDValue: "player",
			X:       x,
			Y:       y,
			Active:  true,
			Speed:   120,
		},
		Health:    state.PlayerHealth,
		MaxHealth: state.MaxPlayerHealth,
		Damage:    state.PlayerDamage,
		DashCooldown: state.DashCooldown,
		State:     state,
		Emitter:   engine.NewParticleEmitter(),
	}

	// Setup animation system
	p.AnimatedEntity = *engine.NewAnimatedEntity()

	// Procedurally generate sprite sheet frames
	idleFrames := engine.GenerateIdleFrames(32, engine.ColSunflower)
	walkFrames := engine.GenerateWalkFrames(32, engine.ColSunflower)
	attackFrames := engine.GenerateAttackFrames(32, engine.ColSunflower)
	hurtFrames := engine.GenerateHurtFrames(32, engine.ColSunflower)

	// Build sprite sheet from all frames combined
	allFrames := append(append(append(idleFrames, walkFrames...), attackFrames...), hurtFrames...)
	p.SetSpriteSheet(allFrames[0], 32, 32)
	p.Frames = allFrames
	p.Alpha = 1.0
	p.Scale = 1.0

	// Register animation clips
	p.Animator.AddClip(engine.PresetIdle())
	p.Animator.AddClip(engine.PresetWalk())
	p.Animator.AddClip(engine.PresetAttack())
	p.Animator.AddClip(engine.PresetHurt())
	p.Animator.AddClip(engine.PresetDeath())

	// Frame indices for each animation
	p.Animator.Clip("idle").FrameAnim = engine.NewFrameAnim([]int{0, 1, 2, 1}, 0.3, true, false)
	p.Animator.Clip("walk").FrameAnim = engine.NewFrameAnim([]int{4, 5, 6, 7, 8, 9}, 0.1, true, false)
	p.Animator.Clip("attack").FrameAnim = engine.NewFrameAnim([]int{10, 11, 12, 13}, 0.06, false, false)
	p.Animator.Clip("hurt").FrameAnim = engine.NewFrameAnim([]int{14, 15}, 0.1, false, false)
	p.Animator.Clip("death").FrameAnim = engine.NewFrameAnim([]int{14, 15, 14, 15}, 0.15, false, false)

	p.Animator.Play("idle", true)

	return p
}

func (p *Player) Update(g *engine.Game) {
	if !p.Active {
		return
	}

	dt := g.DeltaTime()
	input := g.Input()

	// Update animation
	p.AnimatedEntity.Update(dt)

	// === Movement ===
	var dx, dy float64
	p.isMoving = false
	if input.LeftPressed {
		dx -= 1
		p.Direction = 2
		p.isMoving = true
	}
	if input.RightPressed {
		dx += 1
		p.Direction = 3
		p.isMoving = true
	}
	if input.UpPressed {
		dy -= 1
		p.Direction = 1
		p.isMoving = true
	}
	if input.DownPressed {
		dy += 1
		p.Direction = 0
		p.isMoving = true
	}

	if dx != 0 && dy != 0 {
		invLen := 1.0 / math.Sqrt(2)
		dx *= invLen
		dy *= invLen
	}

	// Set flip based on direction
	p.FlipX = dx < 0

	// Switch animations based on state
	currentClip := p.Animator.CurrentClip()
	if p.Invincible > 0 {
		if currentClip != "hurt" {
			p.Animator.Play("hurt", true)
		}
	} else if p.isMoving {
		if currentClip != "walk" && currentClip != "dash" {
			p.Animator.Play("walk", true)
		}
	} else if currentClip != "idle" && currentClip != "attack" && currentClip != "hurt" && currentClip != "death" {
		p.Animator.Play("idle", true)
	}

	// === Dash ===
	if p.DashTimer > 0 {
		p.DashTimer -= dt
		if p.DashTimer <= 0 {
			p.IsDashing = false
		}
	}
	if p.DashCooldown > 0 {
		p.DashCooldown -= dt
	}

	if input.DashJustPressed && p.DashCooldown <= 0 && !p.IsDashing {
		p.IsDashing = true
		p.DashTimer = 0.15
		p.DashCooldown = 0.5
		p.Animator.Play("attack", true)

		dashSpeed := 400.0
		if dx != 0 || dy != 0 {
			dx *= dashSpeed / p.Speed
			dy *= dashSpeed / p.Speed
		} else {
			switch p.Direction {
			case 0:
				dy = 1
			case 1:
				dy = -1
			case 2:
				dx = -1
			case 3:
				dx = 1
			}
			dx *= dashSpeed / p.Speed
			dy *= dashSpeed / p.Speed
		}
		p.Emitter.EmitExplosion(p.X+16, p.Y+16, 8, engine.ColSunflower, 60, 0.3)
	}

	if p.IsDashing {
		speed := 400.0
		p.X += dx * speed * dt
		p.Y += dy * speed * dt
	} else {
		p.X += dx * p.Speed * dt
		p.Y += dy * p.Speed * dt
	}

	// Clamp
	if p.X < 0 {
		p.X = 0
	}
	if p.Y < 0 {
		p.Y = 0
	}
	if p.X > 1280-32 {
		p.X = 1280 - 32
	}
	if p.Y > 720-32 {
		p.Y = 720 - 32
	}

	// Invincibility frames
	if p.Invincible > 0 {
		p.Invincible -= dt
	}

	// Emit idle dust motes (Animal Well atmosphere)
	if !p.isMoving && p.Active {
		p.Emitter.EmitDustMotes(p.X+16, p.Y+16, 10, engine.ColSunflower)
	}
	if p.isMoving {
		p.Emitter.EmitTrail(p.X+16, p.Y+16, engine.ColSunflower, p.Speed)
	}

	p.Emitter.Update(dt)
	p.State.PlayerHealth = p.Health
}

func (p *Player) Draw(screen *ebiten.Image, g *engine.Game) {
	if !p.Active {
		return
	}

	// Blink during invincibility
	if p.Invincible > 0 && int(p.Invincible*10)%2 == 0 {
		// Draw with reduced alpha instead of skipping
		p.Alpha = 0.3
	} else {
		p.Alpha = 1.0
	}

	// Animated draw
	p.AnimatedEntity.Draw(screen, p.X+16, p.Y+16)

	p.Emitter.Draw(screen)

	// Health bar
	barW := 32.0
	barH := 4.0
	healthRatio := p.Health / p.MaxHealth
	engine.DrawRect(screen, p.X, p.Y-8, barW, barH, engine.ColRed)
	engine.DrawRect(screen, p.X, p.Y-8, barW*healthRatio, barH, engine.ColGreen)
}

func (p *Player) TakeDamage(amount float64) {
	if p.Invincible > 0 {
		return
	}
	p.Health -= amount
	p.Invincible = 0.5
	p.Animator.Play("hurt", true)
	p.Emitter.EmitExplosion(p.X+16, p.Y+16, 6, engine.ColRed, 80, 0.4)
	if p.Health <= 0 {
		p.Health = 0
		p.Active = false
		p.Animator.Play("death", true)
	}
}

func (p *Player) Heal(amount float64) {
	p.Health += amount
	if p.Health > p.MaxHealth {
		p.Health = p.MaxHealth
	}
	p.Emitter.EmitSparkle(p.X+16, p.Y+16, engine.ColGreen)
}