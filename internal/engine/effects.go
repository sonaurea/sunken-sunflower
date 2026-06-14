package engine

import (
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ─── Effect Factory ─────────────────────────────────────────────────
// Inspired by Animal Well's masterful procedural atmosphere:
//   - Deep darkness punctuated by strategic light
//   - Floating dust motes in illuminated beams
//   - Layered environmental particles
//   - Minimalist but deeply atmospheric
// Everything is procedurally generated — no sprites, all math + particles.

// ─── Tween / Easing ─────────────────────────────────────────────────

type EasingFunc func(t float64) float64

func EaseOutBounce(t float64) float64 {
	if t < 1/2.75 {
		return 7.5625 * t * t
	} else if t < 2/2.75 {
		t -= 1.5 / 2.75
		return 7.5625*t*t + 0.75
	} else if t < 2.5/2.75 {
		t -= 2.25 / 2.75
		return 7.5625*t*t + 0.9375
	} else {
		t -= 2.625 / 2.75
		return 7.5625*t*t + 0.984375
	}
}

func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

func EaseOutElastic(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	return math.Pow(2, -10*t)*math.Sin((t*10-0.75)*math.Pi*2/3) + 1
}

func EaseInBack(t float64) float64 {
	c1 := 1.70158
	c3 := c1 + 1
	return c3*t*t*t - c1*t*t
}

// ─── Tween Animation ────────────────────────────────────────────────

type Tween struct {
	Duration float64
	Elapsed  float64
	From, To float64
	Easing   EasingFunc
	OnUpdate func(value float64)
	OnDone   func()
	Done     bool
}

func NewTween(duration, from, to float64, easing EasingFunc, onUpdate func(float64), onDone func()) *Tween {
	return &Tween{
		Duration: duration,
		From:     from,
		To:       to,
		Easing:   easing,
		OnUpdate: onUpdate,
		OnDone:   onDone,
	}
}

func (t *Tween) Update(dt float64) {
	if t.Done {
		return
	}
	t.Elapsed += dt
	progress := t.Elapsed / t.Duration
	if progress >= 1 {
		progress = 1
		t.Done = true
		if t.OnUpdate != nil {
			t.OnUpdate(t.To)
		}
		if t.OnDone != nil {
			t.OnDone()
		}
		return
	}
	val := t.From + (t.To-t.From)*t.Easing(progress)
	if t.OnUpdate != nil {
		t.OnUpdate(val)
	}
}

type TweenManager struct {
	tweens []*Tween
}

func NewTweenManager() *TweenManager { return &TweenManager{} }

func (tm *TweenManager) Add(t *Tween) {
	tm.tweens = append(tm.tweens, t)
}

func (tm *TweenManager) Update(dt float64) {
	alive := tm.tweens[:0]
	for _, t := range tm.tweens {
		t.Update(dt)
		if !t.Done {
			alive = append(alive, t)
		}
	}
	tm.tweens = alive
}

func (tm *TweenManager) Clear() { tm.tweens = nil }

// ─── Screen Shake ───────────────────────────────────────────────────
// Animal Well uses subtle, organic shake — not bouncy but grounded.

type ScreenShake struct {
	Intensity float64
	Duration  float64
	Elapsed   float64
	OffsetX   float64
	OffsetY   float64
	Decay     float64
}

func NewScreenShake(intensity, duration float64) *ScreenShake {
	return &ScreenShake{Intensity: intensity, Duration: duration, Decay: 2.0}
}

func (ss *ScreenShake) Update(dt float64) {
	if ss.Elapsed >= ss.Duration {
		ss.OffsetX, ss.OffsetY = 0, 0
		return
	}
	ss.Elapsed += dt
	remaining := 1 - (ss.Elapsed / ss.Duration)
	currentIntensity := ss.Intensity * remaining * remaining
	ss.OffsetX = (rand.Float64()*2 - 1) * currentIntensity
	ss.OffsetY = (rand.Float64()*2 - 1) * currentIntensity
}

func (ss *ScreenShake) IsActive() bool { return ss.Elapsed < ss.Duration }

// ─── Damage Number ──────────────────────────────────────────────────

type DamageNumber struct {
	X, Y      float64
	Text      string
	Color     color.Color
	Life      float64
	MaxLife   float64
	VelocityX float64
	VelocityY float64
}

type DamageNumberManager struct {
	Numbers []*DamageNumber
}

func NewDamageNumberManager() *DamageNumberManager { return &DamageNumberManager{} }

func (dnm *DamageNumberManager) Add(x, y float64, text string, clr color.Color) {
	dnm.Numbers = append(dnm.Numbers, &DamageNumber{
		X: x, Y: y,
		Text:      text,
		Color:     clr,
		Life:      1.2,
		MaxLife:   1.2,
		VelocityX: (rand.Float64()*2 - 1) * 20,
		VelocityY: -60 - rand.Float64()*40,
	})
}

func (dnm *DamageNumberManager) Update(dt float64) {
	alive := dnm.Numbers[:0]
	for _, n := range dnm.Numbers {
		n.Life -= dt
		if n.Life <= 0 {
			continue
		}
		n.X += n.VelocityX * dt
		n.Y += n.VelocityY * dt
		n.VelocityY += 120 * dt
		alive = append(alive, n)
	}
	dnm.Numbers = alive
}

func (dnm *DamageNumberManager) Draw(screen *ebiten.Image, offsetX, offsetY float64) {
	for _, dn := range dnm.Numbers {
		alpha := uint8(255 * (dn.Life / dn.MaxLife))
		cr := dn.Color.(color.RGBA)
		c := color.RGBA{cr.R, cr.G, cr.B, alpha}
		DrawText(screen, dn.Text, int(dn.X+offsetX), int(dn.Y+offsetY), c)
	}
}

// ─── Flash Overlay ──────────────────────────────────────────────────

type FlashOverlay struct {
	Color  color.Color
	Alpha  float64
	Decay  float64
	Active bool
}

func NewFlashOverlay(clr color.Color, initialAlpha, decay float64) *FlashOverlay {
	return &FlashOverlay{Color: clr, Alpha: initialAlpha, Decay: decay, Active: true}
}

func (f *FlashOverlay) Update(dt float64) {
	if !f.Active {
		return
	}
	f.Alpha -= f.Decay * dt
	if f.Alpha <= 0 {
		f.Alpha, f.Active = 0, false
	}
}

func (f *FlashOverlay) Draw(screen *ebiten.Image) {
	if !f.Active || f.Alpha <= 0 {
		return
	}
	cr := f.Color.(color.RGBA)
	c := color.RGBA{cr.R, cr.G, cr.B, uint8(float64(cr.A) * f.Alpha)}
	DrawRect(screen, 0, 0, float64(ScreenWidth), float64(ScreenHeight), c)
}

// ─── Status Effect Indicator ────────────────────────────────────────

type StatusEffect struct {
	Type      string
	Duration  float64
	Elapsed   float64
	IconColor color.Color
	Pulse     float64
}

type StatusEffectManager struct {
	effects []*StatusEffect
}

func NewStatusEffectManager() *StatusEffectManager { return &StatusEffectManager{} }

func (sem *StatusEffectManager) Add(effectType string, duration float64) {
	for _, e := range sem.effects {
		if e.Type == effectType {
			e.Duration, e.Elapsed = duration, 0
			return
		}
	}
	clr := color.RGBA{255, 255, 255, 255}
	switch effectType {
	case "poison":
		clr = color.RGBA{80, 255, 80, 255}
	case "burn":
		clr = color.RGBA{255, 100, 0, 255}
	case "shield":
		clr = color.RGBA{0, 150, 255, 255}
	case "stun":
		clr = color.RGBA{255, 255, 0, 255}
	case "heal":
		clr = color.RGBA{0, 255, 100, 255}
	case "buff":
		clr = ColSunflower
	}
	sem.effects = append(sem.effects, &StatusEffect{Type: effectType, Duration: duration, IconColor: clr})
}

func (sem *StatusEffectManager) Update(dt float64) {
	alive := sem.effects[:0]
	for _, e := range sem.effects {
		e.Elapsed += dt
		e.Pulse += dt * 4
		if e.Elapsed < e.Duration {
			alive = append(alive, e)
		}
	}
	sem.effects = alive
}

func (sem *StatusEffectManager) Has(effectType string) bool {
	for _, e := range sem.effects {
		if e.Type == effectType {
			return true
		}
	}
	return false
}

func (sem *StatusEffectManager) Draw(screen *ebiten.Image, x, y float64) {
	for i, e := range sem.effects {
		label := ""
		switch e.Type {
		case "poison":
			label = "☠"
		case "burn":
			label = "🔥"
		case "shield":
			label = "🛡"
		case "stun":
			label = "⚡"
		case "heal":
			label = "💚"
		case "buff":
			label = "✨"
		default:
			label = "?"
		}
		DrawText(screen, label, int(x)+int(float64(i)*20), int(y), e.IconColor)
	}
}

// ─── Transition Effect ──────────────────────────────────────────────

type TransitionDirection int

const (
	TransitionFade TransitionDirection = iota
	TransitionSlideUp
	TransitionSlideDown
	TransitionSlideLeft
	TransitionSlideRight
	TransitionZoom
)

type Transition struct {
	Direction TransitionDirection
	Progress  float64
	Duration  float64
	Elapsed   float64
	Active    bool
	OnHalf    func()
	Color     color.Color
}

func NewTransition(dir TransitionDirection, duration float64, onHalf func()) *Transition {
	return &Transition{
		Direction: dir, Duration: duration,
		Active: true, OnHalf: onHalf,
		Color: ColBlack,
	}
}

func (t *Transition) Update(dt float64) {
	if !t.Active || t.Progress >= 1 {
		t.Active = false
		return
	}
	t.Elapsed += dt
	t.Progress = t.Elapsed / t.Duration
	if t.Progress >= 0.5 && t.OnHalf != nil {
		t.OnHalf()
		t.OnHalf = nil
	}
}

func (t *Transition) Draw(screen *ebiten.Image) {
	if !t.Active {
		return
	}
	p := t.Progress
	if p > 1 {
		p = 1
	}
	cr := t.Color.(color.RGBA)

	switch t.Direction {
	case TransitionFade:
		alpha := uint8(255 * p)
		c := color.RGBA{cr.R, cr.G, cr.B, alpha}
		if p > 0.5 {
			alpha = uint8(255 * (1 - p) * 2)
			c = color.RGBA{cr.R, cr.G, cr.B, alpha}
		}
		DrawRect(screen, 0, 0, float64(ScreenWidth), float64(ScreenHeight), c)
	case TransitionSlideUp:
		offset := (1 - p) * float64(ScreenHeight)
		if p > 0.5 {
			offset = -(p - 0.5) * 2 * float64(ScreenHeight)
		}
		DrawRect(screen, 0, offset, float64(ScreenWidth), float64(ScreenHeight), t.Color)
	case TransitionSlideDown:
		offset := -(1 - p) * float64(ScreenHeight)
		if p > 0.5 {
			offset = (p - 0.5) * 2 * float64(ScreenHeight)
		}
		DrawRect(screen, 0, offset, float64(ScreenWidth), float64(ScreenHeight), t.Color)
	case TransitionZoom:
		radius := p * float64(ScreenWidth)
		if p > 0.5 {
			radius = (1 - p) * 2 * float64(ScreenWidth)
		}
		cx := float64(ScreenWidth) / 2
		cy := float64(ScreenHeight) / 2
		DrawCircle(screen, cx, cy, radius, t.Color)
		for i := 0; i < 3; i++ {
			r := radius * (0.7 + float64(i)*0.1)
			a := uint8(50 * (1 - p))
			DrawCircle(screen, cx, cy, r, color.RGBA{cr.R, cr.G, cr.B, a})
		}
	}
}

// ─── Particle Variety ───────────────────────────────────────────────
// Animal Well-style: dust motes, fireflies, caustics, shadows

func (pe *ParticleEmitter) EmitExplosion(x, y float64, count int, clr color.Color, speed, life float64) {
	if !pe.Active {
		return
	}
	for i := 0; i < count; i++ {
		angle := math.Pi * 2 * float64(i) / float64(count)
		spd := speed * (0.3 + rand.Float64()*0.7)
		pe.Particles = append(pe.Particles, &Particle{
			X: x, Y: y,
			VX: math.Cos(angle) * spd,
			VY: math.Sin(angle) * spd,
			Life:    life * (0.5 + rand.Float64()*0.5),
			MaxLife: life,
			Size:    1.5 + rand.Float64()*3,
			Color:   clr,
		})
	}
}

func (pe *ParticleEmitter) EmitTrail(x, y float64, clr color.Color, speed float64) {
	if !pe.Active || rand.Float64() > 0.3 {
		return
	}
	pe.Particles = append(pe.Particles, &Particle{
		X: x, Y: y,
		VX: (rand.Float64()*2 - 1) * 10,
		VY: (rand.Float64()*2 - 1) * 10,
		Life:    0.3 + rand.Float64()*0.4,
		MaxLife: 0.7,
		Size:    2 + rand.Float64()*2,
		Color:   clr,
	})
}

func (pe *ParticleEmitter) EmitSparkle(x, y float64, clr color.Color) {
	if !pe.Active {
		return
	}
	for i := 0; i < 3; i++ {
		angle := rand.Float64() * math.Pi * 2
		spd := 15 + rand.Float64()*25
		pe.Particles = append(pe.Particles, &Particle{
			X: x, Y: y,
			VX: math.Cos(angle) * spd,
			VY: math.Sin(angle) * spd,
			Life:    0.5 + rand.Float64()*0.3,
			MaxLife: 0.8,
			Size:    1 + rand.Float64()*2,
			Color:   clr,
		})
	}
}

// EmitDustMotes creates ambient floating dust — signature Animal Well atmosphere.
func (pe *ParticleEmitter) EmitDustMotes(x, y, radius float64, clr color.Color) {
	if !pe.Active || rand.Float64() > 0.1 {
		return
	}
	angle := rand.Float64() * math.Pi * 2
	dist := rand.Float64() * radius
	pe.Particles = append(pe.Particles, &Particle{
		X: x + math.Cos(angle)*dist,
		Y: y + math.Sin(angle)*dist,
		VX: (rand.Float64() - 0.5) * 5,
		VY: -(rand.Float64() * 3),
		Life:    2.0 + rand.Float64()*3.0,
		MaxLife: 5.0,
		Size:    0.5 + rand.Float64()*1.5,
		Color:   clr,
	})
}

// ─── Floating Text Pool ─────────────────────────────────────────────

type FloatingText struct {
	Text    string
	X, Y    float64
	Color   color.Color
	Life    float64
	MaxLife float64
	OffsetY float64
}

type FloatingTextManager struct {
	texts []*FloatingText
}

func NewFloatingTextManager() *FloatingTextManager { return &FloatingTextManager{} }

func (ftm *FloatingTextManager) Add(text string, x, y float64, clr color.Color) {
	ftm.texts = append(ftm.texts, &FloatingText{
		Text: text, X: x, Y: y, Color: clr,
		Life: 1.5, MaxLife: 1.5,
	})
}

func (ftm *FloatingTextManager) Update(dt float64) {
	alive := ftm.texts[:0]
	for _, ft := range ftm.texts {
		ft.Life -= dt
		if ft.Life <= 0 {
			continue
		}
		ft.OffsetY += dt * 30
		alive = append(alive, ft)
	}
	ftm.texts = alive
}

func (ftm *FloatingTextManager) Draw(screen *ebiten.Image) {
	for _, ft := range ftm.texts {
		alpha := uint8(255 * (ft.Life / ft.MaxLife))
		cr := ft.Color.(color.RGBA)
		c := color.RGBA{cr.R, cr.G, cr.B, alpha}
		DrawText(screen, ft.Text, int(ft.X), int(ft.Y-ft.OffsetY), c)
	}
}

// ─── Glow Ring — Animal Well-style pulsing light ────────────────────

func DrawGlowRing(screen *ebiten.Image, cx, cy, radius, time float64, clr color.Color) {
	pulse := math.Sin(time*3) * 0.3
	r := radius * (1 + pulse)
	alpha := uint8(60 + int(40*math.Sin(time*3+1)))
	cr := clr.(color.RGBA)
	DrawCircle(screen, cx, cy, r, color.RGBA{cr.R, cr.G, cr.B, alpha})
	DrawCircle(screen, cx, cy, r*0.7, color.RGBA{cr.R, cr.G, cr.B, alpha / 2})
}

// ─── Torchlight / Player Radiance ───────────────────────────────────
// Animal Well's signature: darkness with a moving light source.
// DrawPlayerLight creates a radial gradient that illuminates around the player.

func DrawPlayerLight(screen *ebiten.Image, cx, cy, radius float64) {
	// Layered circles of decreasing darkness to simulate torchlight
	// Center: fully transparent (bright)
	// Edge: fully opaque (dark)
	layers := 20
	for i := layers; i > 0; i-- {
		t := float64(i) / float64(layers)
		r := radius * t
		alpha := uint8(255 * (1 - t) * 0.6)
		c := color.RGBA{0, 0, 0, alpha}
		DrawCircle(screen, cx, cy, r, c)
	}
}

// DrawDarkness fills the screen with darkness, leaving a torchlit circle.
func DrawDarkness(screen *ebiten.Image, cx, cy, lightRadius float64) {
	// Fill with black first
	DrawRect(screen, 0, 0, float64(ScreenWidth), float64(ScreenHeight), color.RGBA{0, 0, 0, 200})
	// Then "cut out" the light circle by drawing a bright circle
	// (This creates the illusion of a dark screen with a lit area)
	layers := 15
	for i := 0; i < layers; i++ {
		t := float64(i) / float64(layers)
		r := lightRadius * t
		alpha := uint8(200 * (1 - t))
		// Use the screen's current pixel underneath — we're "erasing" darkness
		DrawCircle(screen, cx, cy, r, color.RGBA{0, 0, 0, alpha})
	}
}

// ─── Light Rays ─────────────────────────────────────────────────────

type LightRay struct {
	X, Y   float64
	Width  float64
	Height float64
	Angle  float64
	Speed  float64
	Alpha  float64
	Color  color.Color
}

func GenerateLightRays(count int, screenW, screenH float64) []LightRay {
	rays := make([]LightRay, count)
	for i := range rays {
		rays[i] = LightRay{
			X:      rand.Float64() * screenW,
			Y:      -rand.Float64() * screenH * 0.3,
			Width:  20 + rand.Float64()*60,
			Height: screenH * (0.5 + rand.Float64()*0.5),
			Angle:  -0.3 + rand.Float64()*0.6,
			Speed:  10 + rand.Float64()*20,
			Alpha:  0.03 + rand.Float64()*0.05,
			Color:  ColSunflower,
		}
	}
	return rays
}

func DrawLightRays(screen *ebiten.Image, rays []LightRay, time float64) {
	for i := range rays {
		r := &rays[i]
		r.Y += r.Speed * 0.016
		if r.Y > float64(ScreenHeight)+r.Height {
			r.Y = -r.Height
			r.X = rand.Float64() * float64(ScreenWidth)
		}
		cr := r.Color.(color.RGBA)
		alpha := uint8(float64(cr.A) * r.Alpha * (0.5 + 0.5*math.Sin(time+r.X*0.01)))
		for j := 0; j < int(r.Width); j += 4 {
			x := r.X + float64(j)*math.Cos(r.Angle)
			y := r.Y + float64(j)*math.Sin(r.Angle) + time*5
			luminance := 1.0 - float64(j)/r.Width
			c := color.RGBA{cr.R, cr.G, cr.B, uint8(float64(alpha) * luminance)}
			DrawRect(screen, x, y+r.Height*float64(j)/r.Width*0.2, 2, r.Height*0.3, c)
		}
	}
}

// ─── Ambient Dust (Animal Well atmospheric motes) ───────────────────

type DustMote struct {
	X, Y  float64
	VX, VY float64
	Size  float64
	Alpha float64
	Phase float64 // for drifting
}

type AmbientDust struct {
	Motes  []DustMote
	Bounds [2]float64 // width, height
}

func NewAmbientDust(count int, w, h float64) *AmbientDust {
	ad := &AmbientDust{Bounds: [2]float64{w, h}}
	ad.Motes = make([]DustMote, count)
	for i := range ad.Motes {
		ad.Motes[i] = DustMote{
			X:     rand.Float64() * w,
			Y:     rand.Float64() * h,
			VX:    (rand.Float64() - 0.5) * 4,
			VY:    -(rand.Float64() * 2),
			Size:  0.5 + rand.Float64()*1.5,
			Alpha: 0.1 + rand.Float64()*0.3,
			Phase: rand.Float64() * math.Pi * 2,
		}
	}
	return ad
}

func (ad *AmbientDust) Update(dt float64) {
	for i := range ad.Motes {
		m := &ad.Motes[i]
		m.X += m.VX * dt
		m.Y += m.VY * dt
		m.Phase += dt * 0.5
		if m.Y < -10 {
			m.Y = ad.Bounds[1] + 10
			m.X = rand.Float64() * ad.Bounds[0]
		}
		if m.X < -10 {
			m.X = ad.Bounds[0] + 10
		}
		if m.X > ad.Bounds[0]+10 {
			m.X = -10
		}
	}
}

func (ad *AmbientDust) Draw(screen *ebiten.Image) {
	for _, m := range ad.Motes {
		a := uint8(m.Alpha * 255 * (0.5 + 0.5*math.Sin(m.Phase)))
		c := color.RGBA{200, 200, 180, a}
		DrawCircle(screen, m.X, m.Y, m.Size, c)
	}
}

// ─── UI Pulse ───────────────────────────────────────────────────────

func DrawPulsingBorder(screen *ebiten.Image, x, y, w, h, time float64, clr color.Color, thickness float64) {
	pulse := 0.5 + 0.5*math.Sin(time*3)
	alpha := uint8(100 + int(155*pulse))
	cr := clr.(color.RGBA)
	c := color.RGBA{cr.R, cr.G, cr.B, alpha}
	DrawRect(screen, x, y, w, thickness, c)
	DrawRect(screen, x, y, thickness, h, c)
	DrawRect(screen, x+w-thickness, y, thickness, h+thickness, c)
	DrawRect(screen, x, y+h-thickness, w+thickness, thickness, c)
}

// ─── Enemy Spawn Animation ──────────────────────────────────────────

type SpawnAnim struct {
	X, Y     float64
	Progress float64
	Radius   float64
	Color    color.Color
	Done     bool
}

func NewSpawnAnim(x, y float64, clr color.Color) *SpawnAnim {
	return &SpawnAnim{X: x, Y: y, Radius: 5, Color: clr}
}

func (sa *SpawnAnim) Update(dt float64) {
	if sa.Done {
		return
	}
	sa.Progress += dt * 2
	if sa.Progress >= 1 {
		sa.Done = true
		return
	}
	if sa.Progress < 0.5 {
		sa.Radius = 5 + sa.Progress*2*40
	} else {
		sa.Radius = 45 - (sa.Progress-0.5)*2*40
	}
}

func (sa *SpawnAnim) Draw(screen *ebiten.Image) {
	if sa.Done {
		return
	}
	alpha := uint8(255 * (1 - sa.Progress))
	cr := sa.Color.(color.RGBA)
	c := color.RGBA{cr.R, cr.G, cr.B, alpha}
	DrawCircle(screen, sa.X, sa.Y, sa.Radius, c)
	DrawCircle(screen, sa.X, sa.Y, sa.Radius*0.5, color.RGBA{255, 255, 255, alpha / 2})
}

// ─── Orbital Indicator ──────────────────────────────────────────────

func DrawOrbitIndicator(screen *ebiten.Image, cx, cy, radius, time float64, clr color.Color) {
	for i := 0; i < 3; i++ {
		angle := time*2 + float64(i)*math.Pi*2/3
		x := cx + math.Cos(angle)*radius
		y := cy + math.Sin(angle)*radius
		alpha := uint8(80 + int(80*math.Sin(time+float64(i))))
		cr := clr.(color.RGBA)
		DrawCircle(screen, x, y, 3, color.RGBA{cr.R, cr.G, cr.B, alpha})
	}
}

// ─── Sort for rendering order ───────────────────────────────────────

func SortParticlesByY(particles []*Particle) {
	sort.Slice(particles, func(i, j int) bool {
		return particles[i].Y < particles[j].Y
	})
}