package engine

import (
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ─── Constants ──────────────────────────────────────────────────────

const (
	ScreenWidth  = 1280
	ScreenHeight = 720
	InternalTPS  = 60
	MaxDeltaTime = 0.1 // cap dt at 100ms to prevent physics weirdness
)

// ─── Game Settings ──────────────────────────────────────────────────

type GameSettings struct {
	Title               string
	Fullscreen          bool
	VSyncEnabled        bool
	RunnableOnUnfocused bool
	WindowResizingMode  bool
	FullscreenToggleKey ebiten.Key
}

// DefaultSettings returns sensible defaults tuned for Steam Deck.
func DefaultSettings() GameSettings {
	return GameSettings{
		Title:               "Sunken Sunflower",
		Fullscreen:          false,
		VSyncEnabled:        true,
		RunnableOnUnfocused: true,
		WindowResizingMode:  true,
		FullscreenToggleKey: ebiten.KeyF11,
	}
}

// ─── Scene Interface ────────────────────────────────────────────────

// Scene is the interface every game scene must implement.
type Scene interface {
	Enter(g *Game)
	Exit(g *Game)
	Update(g *Game)
	Draw(screen *ebiten.Image, g *Game)
}

// BaseScene provides no-op defaults so scenes only override what they need.
type BaseScene struct{}

func (b *BaseScene) Enter(g *Game) {}
func (b *BaseScene) Exit(g *Game)  {}

// ─── SceneManager ───────────────────────────────────────────────────

type SceneManager struct {
	scenes   map[string]Scene
	activeID string
	active   Scene
}

func NewSceneManager() *SceneManager {
	return &SceneManager{
		scenes: make(map[string]Scene),
	}
}

func (sm *SceneManager) Register(name string, scene Scene) {
	sm.scenes[name] = scene
}

func (sm *SceneManager) SwitchTo(name string, g *Game) {
	if s, ok := sm.scenes[name]; ok {
		if sm.active != nil {
			sm.active.Exit(g)
		}
		sm.activeID = name
		sm.active = s
		sm.active.Enter(g)
		log.Printf("[SceneManager] switched to '%s'", name)
	} else {
		log.Printf("[SceneManager] scene '%s' not registered", name)
	}
}

func (sm *SceneManager) Active() Scene    { return sm.active }
func (sm *SceneManager) ActiveID() string { return sm.activeID }

// ─── Entity Interface ───────────────────────────────────────────────

type Entity interface {
	ID() string
	Update(g *Game)
	Draw(screen *ebiten.Image, g *Game)
	Position() (x, y float64)
	SetPosition(x, y float64)
	IsActive() bool
	SetActive(active bool)
}

// BaseEntity provides defaults for entities.
type BaseEntity struct {
	IDValue   string
	X, Y      float64
	Active    bool
	Direction int // 0=down, 1=up, 2=left, 3=right
	Speed     float64
	Sprite    *ebiten.Image
}

func (e *BaseEntity) ID() string                   { return e.IDValue }
func (e *BaseEntity) Position() (float64, float64) { return e.X, e.Y }
func (e *BaseEntity) SetPosition(x, y float64)     { e.X, e.Y = x, y }
func (e *BaseEntity) IsActive() bool               { return e.Active }
func (e *BaseEntity) SetActive(active bool)        { e.Active = active }
func (e *BaseEntity) Update(g *Game)               {}
func (e *BaseEntity) Draw(screen *ebiten.Image, g *Game) {
	if e.Sprite != nil && e.Active {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(e.X, e.Y)
		screen.DrawImage(e.Sprite, op)
	}
}

// ─── EntityManager ──────────────────────────────────────────────────

type EntityManager struct {
	entities map[string]Entity
	order    []string // insertion order for deterministic iteration
}

func NewEntityManager() *EntityManager {
	return &EntityManager{
		entities: make(map[string]Entity),
	}
}

func (em *EntityManager) Add(entity Entity) {
	if entity.ID() == "" {
		log.Printf("[EntityManager] cannot add entity with empty ID")
		return
	}
	if _, exists := em.entities[entity.ID()]; !exists {
		em.order = append(em.order, entity.ID())
	}
	em.entities[entity.ID()] = entity
}

func (em *EntityManager) Remove(id string) {
	delete(em.entities, id)
	for i, oid := range em.order {
		if oid == id {
			em.order = append(em.order[:i], em.order[i+1:]...)
			break
		}
	}
}

func (em *EntityManager) Get(id string) Entity {
	return em.entities[id]
}

func (em *EntityManager) All() []Entity {
	all := make([]Entity, 0, len(em.order))
	for _, id := range em.order {
		if e, ok := em.entities[id]; ok {
			all = append(all, e)
		}
	}
	return all
}

func (em *EntityManager) Update(g *Game) {
	for _, e := range em.All() {
		if e.IsActive() {
			e.Update(g)
		}
	}
}

func (em *EntityManager) Draw(screen *ebiten.Image, g *Game) {
	for _, e := range em.All() {
		if e.IsActive() {
			e.Draw(screen, g)
		}
	}
}

func (em *EntityManager) Clear() {
	em.entities = make(map[string]Entity)
	em.order = nil
}

// ─── Particle System ────────────────────────────────────────────────

type Particle struct {
	X, Y    float64
	VX, VY  float64
	Life    float64 // remaining life in seconds
	MaxLife float64
	Size    float64
	Color   color.Color
}

type ParticleEmitter struct {
	Particles []*Particle
	Active    bool
}

func NewParticleEmitter() *ParticleEmitter {
	return &ParticleEmitter{Active: true}
}

func (pe *ParticleEmitter) Emit(x, y float64, count int, clr color.Color, speed, life float64) {
	if !pe.Active {
		return
	}
	for i := 0; i < count; i++ {
		angle := math.Pi * 2 * float64(i) / float64(count)
		spd := speed * (0.5 + math.Mod(float64(i)*0.7, 1.0)*0.5)
		pe.Particles = append(pe.Particles, &Particle{
			X: x, Y: y,
			VX: math.Cos(angle) * spd,
			VY: math.Sin(angle) * spd,
			Life:    life,
			MaxLife: life,
			Size:    2.0 + math.Mod(float64(i)*1.3, 4.0),
			Color:   clr,
		})
	}
}

func (pe *ParticleEmitter) Update(dt float64) {
	alive := pe.Particles[:0]
	for _, p := range pe.Particles {
		p.Life -= dt
		if p.Life <= 0 {
			continue
		}
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.VX *= 0.98 // friction
		p.VY *= 0.98
		alive = append(alive, p)
	}
	pe.Particles = alive
}

func (pe *ParticleEmitter) Draw(screen *ebiten.Image) {
	for _, p := range pe.Particles {
		alpha := uint8(255 * (p.Life / p.MaxLife))
		cr := p.Color.(color.RGBA)
		c := color.RGBA{cr.R, cr.G, cr.B, alpha}
		DrawCircle(screen, p.X, p.Y, p.Size, c)
	}
}

// ─── ParticleManager ────────────────────────────────────────────────

type ParticleManager struct {
	emitters []*ParticleEmitter
}

func NewParticleManager() *ParticleManager {
	return &ParticleManager{}
}

func (pm *ParticleManager) AddEmitter(e *ParticleEmitter) {
	pm.emitters = append(pm.emitters, e)
}

func (pm *ParticleManager) Update(dt float64) {
	for _, e := range pm.emitters {
		e.Update(dt)
	}
}

func (pm *ParticleManager) Draw(screen *ebiten.Image) {
	for _, e := range pm.emitters {
		e.Draw(screen)
	}
}

// ─── Game (Main Loop) ───────────────────────────────────────────────

// Game is the top-level struct implementing ebiten.Game.
type Game struct {
	settings      GameSettings
	sceneManager  *SceneManager
	entityManager *EntityManager
	particleMgr   *ParticleManager
	input         *Input

	gameTime  float64 // total elapsed seconds
	deltaTime float64 // seconds since last tick
}

func NewGame(settings GameSettings) *Game {
	return &Game{
		settings:      settings,
		sceneManager:  NewSceneManager(),
		entityManager: NewEntityManager(),
		particleMgr:   NewParticleManager(),
		input:         &Input{},
	}
}

// ─── ebiten.Game Interface ──────────────────────────────────────────

func (g *Game) Update() error {
	// Calculate delta time
	dt := 1.0 / InternalTPS
	if ebiten.ActualTPS() > 0 {
		dt = 1.0 / ebiten.ActualTPS()
	}
	if dt > MaxDeltaTime {
		dt = MaxDeltaTime
	}
	g.deltaTime = dt
	g.gameTime += dt

	// Read input
	g.input.Update()

	// Toggle fullscreen with F11
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		g.settings.Fullscreen = !g.settings.Fullscreen
		ebiten.SetFullscreen(g.settings.Fullscreen)
	}

	// Update active scene
	if g.sceneManager.Active() != nil {
		g.sceneManager.Active().Update(g)
	}

	// Update entities
	g.entityManager.Update(g)

	// Update particles
	g.particleMgr.Update(g.deltaTime)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Clear to black
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// Draw active scene
	if g.sceneManager.Active() != nil {
		g.sceneManager.Active().Draw(screen, g)
	}

	// Draw entities on top of scene
	g.entityManager.Draw(screen, g)

	// Draw particles on top of everything
	g.particleMgr.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

// ─── Accessors ──────────────────────────────────────────────────────

func (g *Game) SceneManager() *SceneManager     { return g.sceneManager }
func (g *Game) EntityManager() *EntityManager   { return g.entityManager }
func (g *Game) ParticleManager() *ParticleManager { return g.particleMgr }
func (g *Game) Input() *Input                   { return g.input }
func (g *Game) DeltaTime() float64              { return g.deltaTime }
func (g *Game) GameTime() float64               { return g.gameTime }
func (g *Game) Settings() GameSettings          { return g.settings }