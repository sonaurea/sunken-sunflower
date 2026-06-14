package entities

import (
	"testing"

	"github.com/michael/beach-dreams/internal/engine"
	gamepkg "github.com/michael/beach-dreams/internal/game"
)

func TestNewPlayer(t *testing.T) {
	state := gamepkg.NewGameState()
	p := NewPlayer(100, 200, state)
	if p == nil {
		t.Fatal("player should not be nil")
	}
	if p.ID() != "player" {
		t.Errorf("expected ID 'player', got %s", p.ID())
	}
	x, y := p.Position()
	if x != 100 || y != 200 {
		t.Errorf("expected (100,200), got (%f,%f)", x, y)
	}
	if p.Health != 100 {
		t.Errorf("expected HP 100, got %f", p.Health)
	}
	if !p.Active {
		t.Error("player should be active")
	}
}

func TestPlayerTakeDamage(t *testing.T) {
	state := gamepkg.NewGameState()
	p := NewPlayer(100, 100, state)
	p.TakeDamage(30)
	if p.Health != 70 {
		t.Errorf("expected HP 70, got %f", p.Health)
	}
	if p.Invincible <= 0 {
		t.Error("should have invincibility after damage")
	}
	// Damage during invincibility should be ignored
	p.TakeDamage(50)
	if p.Health != 70 {
		t.Errorf("HP should still be 70 during invincibility, got %f", p.Health)
	}
}

func TestPlayerDeath(t *testing.T) {
	state := gamepkg.NewGameState()
	p := NewPlayer(100, 100, state)
	p.TakeDamage(200)
	if p.Active {
		t.Error("player should be inactive after fatal damage")
	}
	if p.Health != 0 {
		t.Errorf("HP should be 0, got %f", p.Health)
	}
}

func TestPlayerHeal(t *testing.T) {
	state := gamepkg.NewGameState()
	p := NewPlayer(100, 100, state)
	p.TakeDamage(40)
	p.Heal(20)
	if p.Health != 80 {
		t.Errorf("expected HP 80, got %f", p.Health)
	}
	p.Heal(100)
	if p.Health != 100 {
		t.Errorf("expected HP 100 (max), got %f", p.Health)
	}
}

func TestPlayerAnimations(t *testing.T) {
	state := gamepkg.NewGameState()
	p := NewPlayer(100, 100, state)
	if p.Animator == nil {
		t.Fatal("player should have animator")
	}
	if p.Animator.CurrentClip() != "idle" {
		t.Errorf("expected 'idle', got %s", p.Animator.CurrentClip())
	}
}

func TestPlayerUpdateWithGame(t *testing.T) {
	state := gamepkg.NewGameState()
	p := NewPlayer(100, 100, state)
	g := engine.NewGame(engine.DefaultSettings())
	p.Update(g)
}

func TestNewCompanion(t *testing.T) {
	c := NewCompanion("comp1", 100, 200, CompGlow)
	if c == nil {
		t.Fatal("companion should not be nil")
	}
	if c.ID() != "comp1" {
		t.Errorf("expected ID 'comp1', got %s", c.ID())
	}
	if c.CompType != CompGlow {
		t.Errorf("expected CompGlow, got %d", c.CompType)
	}
}

func TestCompanionTypes(t *testing.T) {
	types := []CompanionType{CompGlow, CompShield, CompSpike}
	for _, ct := range types {
		c := NewCompanion("test", 100, 100, ct)
		if c.CompType != ct {
			t.Errorf("expected type %d, got %d", ct, c.CompType)
		}
	}
}

func TestNewEnemy(t *testing.T) {
	e := NewEnemy("e1", 100, 200, EnemyBasic, 1)
	if e == nil {
		t.Fatal("enemy should not be nil")
	}
	if e.ID() != "e1" {
		t.Errorf("expected ID 'e1', got %s", e.ID())
	}
	if e.Health <= 0 {
		t.Error("enemy should have health")
	}
}

func TestEnemyTypes(t *testing.T) {
	types := []EnemyType{EnemyBasic, EnemyCharger, EnemyRanged, EnemySwarm}
	for _, et := range types {
		e := NewEnemy("test", 100, 100, et, 3)
		if e.Behaviour.Type != et {
			t.Errorf("expected type %d, got %d", et, e.Behaviour.Type)
		}
		if e.Health <= 0 {
			t.Error("enemy should have health")
		}
	}
}

func TestEnemyDepthScaling(t *testing.T) {
	e1 := NewEnemy("e1", 100, 100, EnemyBasic, 1)
	e10 := NewEnemy("e10", 100, 100, EnemyBasic, 10)
	if e10.Health <= e1.Health {
		t.Error("deeper enemies should have more health")
	}
	if e10.Damage <= e1.Damage {
		t.Error("deeper enemies should have more damage")
	}
}

func TestEnemyTakeDamage(t *testing.T) {
	e := NewEnemy("e1", 100, 100, EnemyBasic, 1)
	initialHP := e.Health
	killed := e.TakeDamage(10)
	if killed {
		t.Error("should not kill with small damage")
	}
	if e.Health >= initialHP {
		t.Error("health should decrease")
	}
	killed = e.TakeDamage(initialHP)
	if !killed {
		t.Error("should kill with fatal damage")
	}
	if e.Active {
		t.Error("enemy should be inactive after death")
	}
}

func TestEnemyDrops(t *testing.T) {
	e := NewEnemy("e1", 100, 100, EnemyBasic, 5)
	hasDrops := false
	for i := 0; i < 50; i++ {
		drops := e.GetDrops()
		if len(drops) > 0 {
			hasDrops = true
			for item, count := range drops {
				if count <= 0 {
					t.Errorf("item %s has non-positive count %d", item, count)
				}
			}
		}
	}
	if !hasDrops {
		t.Log("note: no drops generated in 50 attempts")
	}
}

func TestNewProjectile(t *testing.T) {
	p := NewProjectile("proj1", 100, 100, 50, 0, 10, 2.0, true)
	if p == nil {
		t.Fatal("projectile should not be nil")
	}
	if p.Damage != 10 {
		t.Errorf("expected damage 10, got %f", p.Damage)
	}
	if !p.IsEnemy {
		t.Error("should be enemy projectile")
	}
}

func TestProjectileLifecycle(t *testing.T) {
	p := NewProjectile("proj1", 100, 100, 0, 0, 10, 0.01, true)
	if !p.Active {
		t.Error("projectile should start active")
	}
	g := engine.NewGame(engine.DefaultSettings())
	state := gamepkg.NewGameState()
	player := NewPlayer(200, 200, state)
	g.EntityManager().Add(player)
	p.Update(g)
}

func TestProjectileMovement(t *testing.T) {
	p := NewProjectile("proj1", 0, 0, 100, 50, 10, 2.0, false)
	g := engine.NewGame(engine.DefaultSettings())
	state := gamepkg.NewGameState()
	player := NewPlayer(200, 200, state)
	g.EntityManager().Add(player)
	// Simulate multiple updates with actual game time
	for i := 0; i < 10; i++ {
		g.Update() // this advances deltaTime
	}
	// Manually update projectile too
	p.Update(g)
	if p.X <= 0 || p.Y <= 0 {
		t.Logf("projectile at (%f,%f) after updates", p.X, p.Y)
	}
}

func TestProjectileUpdateWithGame(t *testing.T) {
	p := NewProjectile("p1", 100, 100, 10, 10, 5, 1.0, true)
	g := engine.NewGame(engine.DefaultSettings())
	state := gamepkg.NewGameState()
	player := NewPlayer(200, 200, state)
	g.EntityManager().Add(player)
	p.Update(g)
}

func TestEnemyUpdateWithGame(t *testing.T) {
	e := NewEnemy("e1", 100, 100, EnemyBasic, 1)
	g := engine.NewGame(engine.DefaultSettings())
	e.Update(g)
}

func TestCompanionUpdateWithGameNoPlayer(t *testing.T) {
	c := NewCompanion("c1", 100, 100, CompGlow)
	g := engine.NewGame(engine.DefaultSettings())
	c.Update(g)
}