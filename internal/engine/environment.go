package engine

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ─── Dynamic Environment System ────────────────────────────────────
// Foliage, dynamic lighting, fireflies, swaying trees, atmosphere.

// ─── Light Source ──────────────────────────────────────────────────

type LightSource struct {
	X, Y     float64
	Radius   float64
	Color    color.Color
	Intensity float64 // 0-1
	Flicker  bool     // candle-like flicker
	Phase    float64
	Speed    float64
}

func NewLightSource(x, y, radius float64, clr color.Color, flicker bool) *LightSource {
	return &LightSource{
		X: x, Y: y, Radius: radius,
		Color: clr, Intensity: 1.0,
		Flicker: flicker, Phase: rand.Float64() * math.Pi * 2,
		Speed: 2 + rand.Float64()*3,
	}
}

func (ls *LightSource) Update(dt float64) {
	ls.Phase += dt * ls.Speed
	if ls.Flicker {
		ls.Intensity = 0.8 + 0.2*math.Sin(ls.Phase)
	}
}

func (ls *LightSource) Draw(screen *ebiten.Image) {
	if ls.Intensity <= 0 {
		return
	}
	r := ls.Radius * ls.Intensity
	cr := ls.Color.(color.RGBA)

	// Layered glow
	for i := 4; i > 0; i-- {
		t := float64(i) / 4.0
		alpha := uint8(float64(cr.A) * ls.Intensity * (1 - t) * 0.4)
		DrawCircle(screen, ls.X, ls.Y, r*t, color.RGBA{cr.R, cr.G, cr.B, alpha})
	}
}

// ─── Foliage / Plant ───────────────────────────────────────────────

type FoliageType int

const (
	FoliagePalm      FoliageType = iota
	FoliageFern
	FoliageFlower
	FoliageSeaOats
	FoliageMangrove
)

type Foliage struct {
	X, Y   float64
	Type   FoliageType
	Scale  float64
	Sway   float64 // current sway offset
	SwaySpeed float64
	SwayAmount float64
	Color  color.Color
	Alpha  float64
}

func NewFoliage(x, y float64, fType FoliageType) *Foliage {
	clr := color.RGBA{60, 120, 40, 255}
	switch fType {
	case FoliagePalm:
		clr = color.RGBA{40, 100, 30, 255}
	case FoliageFern:
		clr = color.RGBA{50, 130, 50, 255}
	case FoliageFlower:
		flowerColors := []color.RGBA{
			{255, 150, 200, 255}, {255, 200, 100, 255}, {255, 100, 150, 255},
		}
		clr = flowerColors[rand.Intn(3)]
	case FoliageSeaOats:
		clr = color.RGBA{180, 170, 100, 255}
	case FoliageMangrove:
		clr = color.RGBA{30, 80, 40, 255}
	}
	return &Foliage{
		X: x, Y: y, Type: fType,
		Scale: 0.5 + rand.Float64()*1.0,
		SwaySpeed: 0.5 + rand.Float64()*1.5,
		SwayAmount: 2 + rand.Float64()*5,
		Color: clr, Alpha: 0.8 + rand.Float64()*0.2,
	}
}

func (f *Foliage) Update(dt float64, wind float64) {
	f.Sway += dt * f.SwaySpeed
	// Wind affects sway
	windInfluence := wind * 3
	f.SwayAmount = (2 + rand.Float64()*5) + windInfluence
}

func (f *Foliage) Draw(screen *ebiten.Image) {
	sx := math.Sin(f.Sway) * f.SwayAmount

	switch f.Type {
	case FoliagePalm:
		// Trunk
		DrawLine(screen, f.X, f.Y, f.X+sx*0.3, f.Y-20*f.Scale, color.RGBA{80, 50, 20, 200})
		// Fronds
		for i := 0; i < 5; i++ {
			angle := float64(i)*math.Pi*2/5 + f.Sway*0.2
			fx := f.X + math.Cos(angle)*15*f.Scale
			fy := f.Y - 20*f.Scale + math.Sin(angle)*5*f.Scale
			DrawLine(screen, f.X+sx*0.3, f.Y-20*f.Scale, fx+sx, fy, f.Color)
			// Leaf tip
			DrawCircle(screen, fx+sx*1.5, fy, 3*f.Scale, f.Color)
		}

	case FoliageFern:
		// Stalk
		stalkH := 15 * f.Scale
		DrawLine(screen, f.X, f.Y, f.X+sx*0.5, f.Y-stalkH, color.RGBA{40, 80, 30, 200})
		// Fronds
		for i := 0; i < 4; i++ {
			ty := f.Y - float64(i)*stalkH/4
			tx := f.X + sx*0.5*float64(i)/4
			DrawLine(screen, tx, ty, tx-8*f.Scale+sx, ty-2, f.Color)
			DrawLine(screen, tx, ty, tx+8*f.Scale+sx, ty-2, f.Color)
		}

	case FoliageFlower:
		// Stem
		DrawLine(screen, f.X, f.Y, f.X+sx*0.3, f.Y-10*f.Scale, color.RGBA{40, 100, 30, 200})
		// Petals
		for i := 0; i < 5; i++ {
			angle := float64(i)*math.Pi*2/5
			px := f.X + math.Cos(angle)*5*f.Scale + sx*0.3
			py := f.Y - 10*f.Scale + math.Sin(angle)*5*f.Scale
			DrawCircle(screen, px, py, 3*f.Scale, f.Color)
		}
		// Center
		DrawCircle(screen, f.X+sx*0.3, f.Y-10*f.Scale, 2*f.Scale, ColSunflower)

	case FoliageSeaOats:
		// Tall grass stalks
		for i := 0; i < 3; i++ {
			ox := float64(i-1) * 4
			DrawLine(screen, f.X+ox, f.Y, f.X+ox+sx, f.Y-18*f.Scale, f.Color)
			// Seed head
			DrawCircle(screen, f.X+ox+sx*1.2, f.Y-18*f.Scale, 2, color.RGBA{200, 190, 120, 200})
		}

	case FoliageMangrove:
		// Arching roots
		for i := 0; i < 3; i++ {
			ox := float64(i-1) * 6
			DrawLine(screen, f.X+ox, f.Y, f.X+ox+sx, f.Y-12*f.Scale, color.RGBA{60, 40, 20, 200})
		}
		// Canopy
		DrawCircle(screen, f.X+sx*0.5, f.Y-14*f.Scale, 8*f.Scale, f.Color)
		DrawCircle(screen, f.X+sx*0.8, f.Y-12*f.Scale, 6*f.Scale, f.Color)
	}
}

// ─── Firefly System ────────────────────────────────────────────────

type Firefly struct {
	X, Y     float64
	VX, VY   float64
	Brightness float64
	Phase    float64
	Speed    float64
	Size     float64
}

type FireflySwarm struct {
	Fireflies []Firefly
	Bounds    [2]float64
}

func NewFireflySwarm(count int, w, h float64) *FireflySwarm {
	fs := &FireflySwarm{Bounds: [2]float64{w, h}}
	fs.Fireflies = make([]Firefly, count)
	for i := range fs.Fireflies {
		fs.Fireflies[i] = Firefly{
			X: rand.Float64() * w, Y: rand.Float64() * h,
			VX: (rand.Float64() - 0.5) * 10, VY: (rand.Float64() - 0.5) * 10,
			Brightness: rand.Float64(), Phase: rand.Float64() * math.Pi * 2,
			Speed: 1 + rand.Float64()*3, Size: 1 + rand.Float64()*2,
		}
	}
	return fs
}

func (fs *FireflySwarm) Update(dt float64, time float64) {
	for i := range fs.Fireflies {
		f := &fs.Fireflies[i]
		f.Phase += dt * f.Speed
		f.Brightness = 0.3 + 0.7*math.Abs(math.Sin(f.Phase))

		// Random wander
		f.VX += (rand.Float64() - 0.5) * 20 * dt
		f.VY += (rand.Float64() - 0.5) * 20 * dt
		// Dampen
		f.VX *= 0.99
		f.VY *= 0.99

		f.X += f.VX * dt
		f.Y += f.VY * dt

		// Bounce off bounds
		if f.X < 0 {
			f.X = 0
			f.VX = -f.VX
		}
		if f.X > fs.Bounds[0] {
			f.X = fs.Bounds[0]
			f.VX = -f.VX
		}
		if f.Y < 0 {
			f.Y = 0
			f.VY = -f.VY
		}
		if f.Y > fs.Bounds[1] {
			f.Y = fs.Bounds[1]
			f.VY = -f.VY
		}
	}
}

func (fs *FireflySwarm) Draw(screen *ebiten.Image) {
	for _, f := range fs.Fireflies {
		if f.Brightness < 0.1 {
			continue
		}
		alpha := uint8(f.Brightness * 255)
		clr := color.RGBA{180, 255, 100, alpha}
		DrawCircle(screen, f.X, f.Y, f.Size*f.Brightness, clr)
		// Outer glow
		if f.Brightness > 0.5 {
			DrawCircle(screen, f.X, f.Y, f.Size*3, color.RGBA{100, 200, 50, alpha / 3})
		}
	}
}

// ─── Dynamic Lighting Overlay ──────────────────────────────────────
// Renders dynamic shadows and light emissive effects.

type DynamicLighting struct {
	AmbientColor color.Color // the base ambient light (darker at night)
	LightSources []*LightSource
	SunAngle     float64
	SunHeight    float64 // 0=horizon, 1=noon
}

func NewDynamicLighting() *DynamicLighting {
	return &DynamicLighting{
		AmbientColor: color.RGBA{255, 255, 255, 255},
		SunAngle:     0,
		SunHeight:    0.5,
	}
}

func (dl *DynamicLighting) Update(dt float64, hour int) {
	// Sun position based on hour
	dl.SunAngle = float64(hour) / 24.0 * math.Pi * 2
	dl.SunHeight = math.Sin(dl.SunAngle)
	if dl.SunHeight < 0 {
		dl.SunHeight = 0 // night
	}

	// Ambient color changes based on sun
	nightFactor := 1.0 - dl.SunHeight
	r := uint8(255 - int(nightFactor*200))
	g := uint8(255 - int(nightFactor*200))
	b := uint8(255 - int(nightFactor*100))
	dl.AmbientColor = color.RGBA{r, g, b, 255}

	for _, ls := range dl.LightSources {
		ls.Update(dt)
	}
}

func (dl *DynamicLighting) Draw(screen *ebiten.Image) {
	// Draw light sources
	for _, ls := range dl.LightSources {
		ls.Draw(screen)
	}
}

// ─── Weather Particles ─────────────────────────────────────────────

type WeatherParticle struct {
	X, Y   float64
	VX, VY float64
	Life   float64
	Size   float64
	Color  color.Color
}

type WeatherSystem struct {
	Particles  []WeatherParticle
	Type       string // "rain", "storm", "clear"
	Intensity  float64
	Wind       float64
}

func NewWeatherSystem() *WeatherSystem {
	return &WeatherSystem{Type: "clear"}
}

func (ws *WeatherSystem) Update(dt float64, weatherType string) {
	ws.Type = weatherType
	switch weatherType {
	case "storm":
		ws.Intensity = 0.8
		ws.Wind = 15
	case "rain":
		ws.Intensity = 0.5
		ws.Wind = 5
	case "rainbow":
		ws.Intensity = 0.1
		ws.Wind = 2
	default:
		ws.Intensity = 0
		ws.Wind = 2
	}

	// Spawn new particles
	if ws.Intensity > 0 && rand.Float64() < ws.Intensity*0.3 {
		ws.Particles = append(ws.Particles, WeatherParticle{
			X: rand.Float64() * float64(ScreenWidth),
			Y: -5,
			VX: -ws.Wind + (rand.Float64()-0.5)*5,
			VY: 100 + rand.Float64()*50,
			Life: 2.0, Size: 1.5,
			Color: color.RGBA{150, 180, 255, 100},
		})
	}

	// Update particles
	alive := ws.Particles[:0]
	for _, p := range ws.Particles {
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.Life -= dt
		if p.Life > 0 && p.Y < float64(ScreenHeight)+10 {
			alive = append(alive, p)
		}
	}
	ws.Particles = alive
}

func (ws *WeatherSystem) Draw(screen *ebiten.Image) {
	for _, p := range ws.Particles {
		DrawRect(screen, p.X, p.Y, 1.5, p.Size*3, p.Color)
	}

	// Rainbow arc
	if ws.Type == "rainbow" {
		cx := float64(ScreenWidth) * 0.5
		cy := float64(ScreenHeight) * 0.8
		colors := []color.Color{
			color.RGBA{255, 0, 0, 30}, color.RGBA{255, 165, 0, 30},
			color.RGBA{255, 255, 0, 30}, color.RGBA{0, 255, 0, 30},
			color.RGBA{0, 150, 255, 30}, color.RGBA{128, 0, 128, 30},
		}
		for i, c := range colors {
			r := float64(200 + i*15)
			DrawCircle(screen, cx, cy, r, c)
		}
	}
}

// ─── Wind Effect (sways foliage) ───────────────────────────────────

func CalculateWind(time float64) float64 {
	return math.Sin(time*0.3)*3 + math.Sin(time*0.7)*2 + math.Sin(time*1.1)*1
}

// ─── Rain Buff/Debuff System ──────────────────────────────────────
// Florida rain is short but intense. It empowers certain creatures
// and abilities while debuffing others.

// RainEffect describes a gameplay modifier during rain.
type RainEffect struct {
	Buff  []string  // things that get stronger
	Debuff []string // things that get weaker
	Duration float64 // minutes remaining
	Intensity float64 // 0-1
}

// NewRainEffect creates a rain effect for Florida-style short bursts.
func NewRainEffect() *RainEffect {
	return &RainEffect{
		Buff:    []string{"Glowpup", "Motesprite", "water_attacks", "Rootling_growth"},
		Debuff:  []string{"fire_attacks", "Spikefin", "Shellback_defense"},
		Duration: 15 + rand.Float64()*20, // 15-35 min Florida showers
		Intensity: 0.5 + rand.Float64()*0.5,
	}
}

// GetBuffMultiplier returns stat changes during rain.
func (re *RainEffect) GetBuffMultiplier(entityType string) float64 {
	for _, b := range re.Buff {
		if b == entityType {
			return 1.0 + re.Intensity*0.3 // +30% buff
		}
	}
	for _, d := range re.Debuff {
		if d == entityType {
			return 1.0 - re.Intensity*0.25 // -25% debuff
		}
	}
	return 1.0
}

// ─── NPC Schedule System ──────────────────────────────────────────
// NPCs have daily schedules that shift based on weather and season.

type NPCSchedule struct {
	NPCID     string
	HourlyPos map[int]struct{ X, Y float64 } // hour -> position
	RainHide  bool // hides during rain
	SeasonalActivity string // what they do in each season
}

// Default schedules for town NPCs
func DefaultSchedules() []NPCSchedule {
	return []NPCSchedule{
		{
			NPCID: "shopkeeper", RainHide: true,
			HourlyPos: map[int]struct{ X, Y float64 }{
				8:  {130, 200}, 12: {130, 200}, 17: {400, 300},
			},
		},
		{
			NPCID: "quest_giver", RainHide: false,
			HourlyPos: map[int]struct{ X, Y float64 }{
				6: {750, 200}, 10: {600, 500}, 14: {750, 200}, 18: {200, 100},
			},
		},
	}
}

// GetPosition returns the NPC's position at a given hour.
func (ns NPCSchedule) GetPosition(hour int) (float64, float64) {
	if pos, ok := ns.HourlyPos[hour]; ok {
		return pos.X, pos.Y
	}
	// Interpolate between known positions
	var closestBefore, closestAfter struct{ X, Y float64 }
	var beforeHour, afterHour int
	for h, pos := range ns.HourlyPos {
		if h <= hour && (beforeHour == 0 || h > beforeHour) {
			beforeHour = h
			closestBefore = pos
		}
		if h >= hour && (afterHour == 0 || h < afterHour) {
			afterHour = h
			closestAfter = pos
		}
	}
	if beforeHour == afterHour || afterHour == 0 {
		return closestBefore.X, closestBefore.Y
	}
	t := float64(hour-beforeHour) / float64(afterHour-beforeHour)
	return closestBefore.X + (closestAfter.X-closestBefore.X)*t,
		closestBefore.Y + (closestAfter.Y-closestBefore.Y)*t
}

// ─── Creature Weather Reactions ────────────────────────────────────
// Creatures react to rain and weather uniquely.

// GetCreatureWeatherMod returns weather modifiers for creature stats.
func GetCreatureWeatherMod(creatureType string, weather string) (buff, debuff string, mult float64) {
	if weather == "storm" || weather == "rain" {
		switch creatureType {
		case "Glowpup":
			return "Bioluminescence Boost", "none", 1.4
		case "Motesprite":
			return "Rain Dance", "none", 1.5
		case "Spikefin":
			return "none", "Waterlogged", 0.7
		case "Shellback":
			return "Mud Shield", "none", 1.2
		case "Rootling":
			return "Flash Growth", "none", 1.6
		case "Crystalisk":
			return "Storm Conduit", "none", 1.8
		}
	}
	if weather == "clear" {
		switch creatureType {
		case "Spikefin":
			return "Solar Charge", "none", 1.3
		case "Rootling":
			return "none", "Dry Spell", 0.8
		}
	}
	return "none", "none", 1.0
}