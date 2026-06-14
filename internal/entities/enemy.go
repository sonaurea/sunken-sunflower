package entities

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/michael/beach-dreams/internal/engine"
)

// ─── Enemy Types ────────────────────────────────────────────────────

type EnemyType int

const (
	EnemyBasic    EnemyType = iota
	EnemyCharger
	EnemyRanged
	EnemySwarm
)

type EnemyBehaviour struct {
	Type     EnemyType
	Cooldown float64
	Range    float64
	Timer    float64
}

// Enemy is an AI-driven hostile entity with full animation support.
type Enemy struct {
	engine.BaseEntity
	engine.AnimatedEntity
	Health    float64
	MaxHealth float64
	Damage    float64
	Behaviour EnemyBehaviour
	Color     color.Color
	DropTable []DropEntry
	Emitter   *engine.ParticleEmitter
	isMoving  bool
}

type DropEntry struct {
	Item         string
	Chance       float64
	Min, Max     int
}

// NewEnemy creates a new enemy at the given position.
func NewEnemy(id string, x, y float64, etype EnemyType, depth int) *Enemy {
	speed := 60.0
	health := 30.0
	damage := 10.0
	clr := color.RGBA{255, 68, 68, 255}
	behaviour := EnemyBehaviour{Type: etype, Range: 150, Cooldown: 2.0}

	switch etype {
	case EnemyBasic:
		speed = 60
		health = 30 + float64(depth)*5
		damage = 10 + float64(depth)*2
		clr = color.RGBA{200, 50, 50, 255}
	case EnemyCharger:
		speed = 40
		health = 20 + float64(depth)*3
		damage = 20 + float64(depth)*3
		behaviour.Range = 250
		behaviour.Cooldown = 3.0
		clr = color.RGBA{255, 100, 0, 255}
	case EnemyRanged:
		speed = 50
		health = 15 + float64(depth)*3
		damage = 8 + float64(depth)*2
		behaviour.Range = 300
		behaviour.Cooldown = 2.5
		clr = color.RGBA{100, 50, 200, 255}
	case EnemySwarm:
		speed = 100
		health = 10 + float64(depth)*2
		damage = 5 + float64(depth)*1
		clr = color.RGBA{200, 200, 50, 255}
	}

	e := &Enemy{
		BaseEntity: engine.BaseEntity{
			IDValue:   id,
			X:         x,
			Y:         y,
			Active:    true,
			Speed:     speed,
			Direction: 0,
		},
		Health:    health,
		MaxHealth: health,
		Damage:    damage,
		Behaviour: behaviour,
		Color:     clr,
		DropTable: defaultDrops(depth),
		Emitter:   engine.NewParticleEmitter(),
	}

	// Animation setup
	e.AnimatedEntity = *engine.NewAnimatedEntity()
	idleFrames := engine.GenerateIdleFrames(24, clr)
	walkFrames := engine.GenerateWalkFrames(24, clr)
	attackFrames := engine.GenerateAttackFrames(24, clr)
	hurtFrames := engine.GenerateHurtFrames(24, clr)

	allFrames := append(append(append(idleFrames, walkFrames...), attackFrames...), hurtFrames...)
	e.Frames = allFrames
	e.Alpha = 1.0
	e.Scale = 1.0

	e.Animator.AddClip(engine.PresetIdle())
	e.Animator.AddClip(engine.PresetWalk())
	e.Animator.AddClip(engine.PresetAttack())
	e.Animator.AddClip(engine.PresetHurt())
	e.Animator.AddClip(engine.PresetDeath())

	e.Animator.Clip("idle").FrameAnim = engine.NewFrameAnim([]int{0, 1, 2, 1}, 0.35, true, false)
	e.Animator.Clip("walk").FrameAnim = engine.NewFrameAnim([]int{4, 5, 6, 7, 8, 9}, 0.12, true, false)
	e.Animator.Clip("attack").FrameAnim = engine.NewFrameAnim([]int{10, 11, 12, 13}, 0.08, false, false)
	e.Animator.Clip("hurt").FrameAnim = engine.NewFrameAnim([]int{14, 15}, 0.1, false, false)
	e.Animator.Clip("death").FrameAnim = engine.NewFrameAnim([]int{14, 15, 14, 15}, 0.2, false, false)

	e.Animator.Clip("attack").NextClip = "idle"
	e.Animator.Clip("hurt").NextClip = "idle"

	// Set sprite sheet
	if len(allFrames) > 0 {
		e.SetSpriteSheet(allFrames[0], 24, 24)
	}

	e.Animator.Play("idle", true)
	return e
}

func defaultDrops(depth int) []DropEntry {
	return []DropEntry{
		{Item: "moon_sand", Chance: 0.6, Min: 1, Max: 3},
		{Item: "starfruit", Chance: 0.3, Min: 1, Max: 1},
		{Item: "well_shard", Chance: 0.1 + float64(depth)*0.02, Min: 1, Max: 1},
	}
}

func (e *Enemy) Update(g *engine.Game) {
	if !e.Active {
		return
	}

	dt := g.DeltaTime()
	e.Behaviour.Timer -= dt
	e.AnimatedEntity.Update(dt)

	playerEnt := g.EntityManager().Get("player")
	if playerEnt == nil || !playerEnt.IsActive() {
		// No player — idle
		e.Animator.Play("idle", false)
		return
	}
	px, py := playerEnt.Position()

	dx := px - e.X
	dy := py - e.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	e.isMoving = false
	currentClip := e.Animator.CurrentClip()

	// Don't change animation if in hurt or attack
	if currentClip == "hurt" || currentClip == "death" {
		return
	}

	switch e.Behaviour.Type {
	case EnemyBasic:
		if dist > 20 {
			e.X += (dx / dist) * e.Speed * dt
			e.Y += (dy / dist) * e.Speed * dt
			e.isMoving = true
			e.FlipX = dx < 0
		}
		if dist < 24 {
			if player, ok := playerEnt.(*Player); ok {
				player.TakeDamage(e.Damage * dt)
			}
			e.Animator.Play("attack", true)
		}

	case EnemyCharger:
		if e.Behaviour.Timer <= 0 && dist < e.Behaviour.Range {
			chargeSpeed := e.Speed * 4
			e.X += (dx / dist) * chargeSpeed * dt
			e.Y += (dy / dist) * chargeSpeed * dt
			e.isMoving = true
			e.FlipX = dx < 0
			if dist < 24 {
				if player, ok := playerEnt.(*Player); ok {
					player.TakeDamage(e.Damage)
				}
				e.Behaviour.Timer = e.Behaviour.Cooldown
				e.Animator.Play("attack", true)
			}
		} else {
			e.X += (dx / dist) * e.Speed * 0.3 * dt
			e.Y += (dy / dist) * e.Speed * 0.3 * dt
			e.isMoving = true
		}

	case EnemyRanged:
		if dist < e.Behaviour.Range*0.5 {
			e.X -= (dx / dist) * e.Speed * dt
			e.Y -= (dy / dist) * e.Speed * dt
			e.isMoving = true
		}
		if e.Behaviour.Timer <= 0 && dist < e.Behaviour.Range {
			e.Emitter.EmitExplosion(e.X+12, e.Y+12, 1, e.Color, 150, 1.0)
			e.Behaviour.Timer = e.Behaviour.Cooldown
			e.Animator.Play("attack", true)
		}

	case EnemySwarm:
		e.X += (dx / dist) * e.Speed * dt
		e.Y += (dy / dist) * e.Speed * dt
		e.isMoving = true
		e.FlipX = dx < 0
		if dist < 20 {
			if player, ok := playerEnt.(*Player); ok {
				player.TakeDamage(e.Damage * dt)
			}
		}
	}

	// Animation state switching
	if e.isMoving && currentClip != "attack" {
		e.Animator.Play("walk", false)
	} else if !e.isMoving && currentClip != "attack" {
		e.Animator.Play("idle", false)
	}

	e.Emitter.Update(dt)
}

func (e *Enemy) Draw(screen *ebiten.Image, g *engine.Game) {
	if !e.Active {
		return
	}

	// Animated draw with center offset
	e.AnimatedEntity.Draw(screen, e.X+12, e.Y+12)
	e.Emitter.Draw(screen)

	// Health bar
	barW := 24.0
	barH := 3.0
	ratio := e.Health / e.MaxHealth
	engine.DrawRect(screen, e.X+2, e.Y-6, barW, barH, engine.ColRed)
	engine.DrawRect(screen, e.X+2, e.Y-6, barW*ratio, barH, engine.ColGreen)
}

func (e *Enemy) TakeDamage(amount float64) bool {
	e.Health -= amount
	e.Animator.Play("hurt", true)
	e.Emitter.EmitExplosion(e.X+12, e.Y+12, 4, engine.ColWhite, 60, 0.3)
	if e.Health <= 0 {
		e.Active = false
		e.Animator.Play("death", true)
		return true
	}
	return false
}

func (e *Enemy) GetDrops() map[string]int {
	drops := make(map[string]int)
	for _, entry := range e.DropTable {
		if rand.Float64() < entry.Chance {
			count := entry.Min
			if entry.Max > entry.Min {
				count += rand.Intn(entry.Max - entry.Min + 1)
			}
			drops[entry.Item] += count
		}
	}
	return drops
}

// ─── Projectile ─────────────────────────────────────────────────────

type Projectile struct {
	engine.BaseEntity
	Damage    float64
	VX, VY    float64
	Lifespan  float64
	IsEnemy   bool
}

func NewProjectile(id string, x, y, vx, vy, damage, lifespan float64, isEnemy bool) *Projectile {
	return &Projectile{
		BaseEntity: engine.BaseEntity{
			IDValue: id,
			X:       x,
			Y:       y,
			Active:  true,
		},
		Damage:   damage,
		VX:       vx,
		VY:       vy,
		Lifespan: lifespan,
		IsEnemy:  isEnemy,
	}
}

func (p *Projectile) Update(g *engine.Game) {
	if !p.Active {
		return
	}
	dt := g.DeltaTime()
	p.Lifespan -= dt
	if p.Lifespan <= 0 {
		p.Active = false
		return
	}
	p.X += p.VX * dt
	p.Y += p.VY * dt

	if p.IsEnemy {
		playerEnt := g.EntityManager().Get("player")
		if player, ok := playerEnt.(*Player); ok && player.Active {
			dx := player.X + 16 - p.X
			dy := player.Y + 16 - p.Y
			if math.Sqrt(dx*dx+dy*dy) < 20 {
				player.TakeDamage(p.Damage)
				p.Active = false
			}
		}
	}
}

func (p *Projectile) Draw(screen *ebiten.Image, g *engine.Game) {
	if !p.Active {
		return
	}
	clr := engine.ColRed
	if !p.IsEnemy {
		clr = engine.ColSunflower
	}
	engine.DrawCircle(screen, p.X, p.Y, 4, clr)
}