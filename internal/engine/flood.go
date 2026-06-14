package engine

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ─── Flood System ──────────────────────────────────────────────────
// When it rains above ground, the dungeon floods!
// Each turn, water rises from the entrance, filling rooms behind you.
// Players must push forward or drown.

type FloodState struct {
	Active      bool    `json:"active"`
	WaterLevel  int     `json:"level"`   // how many rows are flooded
	MaxWater    int     `json:"max"`     // max water level
	RiseRate    int     `json:"rate"`    // rows per turn
	TurnCount   int     `json:"turns"`   // turns since flooding started
	DamagePerTurn float64 `json:"dmg"`   // damage per turn when in water
	Puddles     []Puddle `json:"-"`
	Color       color.Color
}

type Puddle struct {
	X, Y   float64
	Size   float64
	Alpha  float64
	Phase  float64
}

func NewFloodState() *FloodState {
	return &FloodState{
		Active:        false,
		RiseRate:      1,
		DamagePerTurn: 5,
		Color:         color.RGBA{0, 100, 150, 180},
	}
}

// StartFlood begins the flooding when rain hits the surface.
func (fs *FloodState) StartFlood(mapHeight int) {
	fs.Active = true
	fs.WaterLevel = 0
	fs.MaxWater = mapHeight - 2
	fs.TurnCount = 0
	fs.Puddles = nil
	// Initial puddles
	for i := 0; i < 10; i++ {
		fs.Puddles = append(fs.Puddles, Puddle{
			X: rand.Float64()*500 + 50,
			Y: rand.Float64()*200,
			Size: 5 + rand.Float64()*15,
			Alpha: 0.3 + rand.Float64()*0.4,
			Phase: rand.Float64() * math.Pi * 2,
		})
	}
}

// AdvanceTurn advances the flood by one turn.
func (fs *FloodState) AdvanceTurn() {
	if !fs.Active {
		return
	}
	fs.TurnCount++
	fs.WaterLevel += fs.RiseRate
	if fs.WaterLevel > fs.MaxWater {
		fs.WaterLevel = fs.MaxWater
	}
	// Add more puddles as water rises
	for i := 0; i < 5; i++ {
		fs.Puddles = append(fs.Puddles, Puddle{
			X: rand.Float64()*700 - 100,
			Y: float64(fs.WaterLevel)*32 + rand.Float64()*100,
			Size: 8 + rand.Float64()*20,
			Alpha: 0.4 + rand.Float64()*0.5,
			Phase: rand.Float64() * math.Pi * 2,
		})
	}
}

// IsTileFlooded checks if a specific grid position is underwater.
func (fs *FloodState) IsTileFlooded(gridY int) bool {
	if !fs.Active {
		return false
	}
	return gridY <= fs.WaterLevel
}

// GetWaterDamage returns damage dealt to entities in flooded tiles.
func (fs *FloodState) GetWaterDamage(turnsInWater int) float64 {
	return fs.DamagePerTurn * float64(turnsInWater)
}

// GetFloodProgress returns how far the flood has progressed (0-1).
func (fs *FloodState) GetFloodProgress() float64 {
	if !fs.Active || fs.MaxWater == 0 {
		return 0
	}
	return float64(fs.WaterLevel) / float64(fs.MaxWater)
}

// UpdatePuddles animates the puddle visuals.
func (fs *FloodState) UpdatePuddles(dt float64) {
	for i := range fs.Puddles {
		fs.Puddles[i].Phase += dt * 2
		fs.Puddles[i].Alpha = 0.3 + 0.5*math.Abs(math.Sin(fs.Puddles[i].Phase))
	}
}

// Draw renders the flood overlay on the isometric map.
func (fs *FloodState) Draw(screen *ebiten.Image, isoCfg IsoConfig, camX, camY float64) {
	if !fs.Active {
		return
	}

	// Draw puddles
	for _, p := range fs.Puddles {
		alpha := uint8(p.Alpha * 180)
		c := color.RGBA{0, 120, 180, alpha}
		DrawCircle(screen, p.X, p.Y, p.Size, c)
		// Ripple ring
		ringSize := p.Size + float64(math.Sin(p.Phase))*3
		DrawCircle(screen, p.X, p.Y, ringSize*1.5, color.RGBA{100, 200, 255, alpha / 3})
	}

	// Water edge line (shows where water is)
	if fs.WaterLevel > 0 {
		waterY := float64(fs.WaterLevel) * isoCfg.TileHeight
		// Wave line across the map
		for x := 0; x < ScreenWidth; x += 4 {
			waveY := waterY + math.Sin(float64(x)*0.05+float64(fs.TurnCount)*0.5)*5
			alpha := uint8(100 + 80*math.Abs(math.Sin(float64(fs.TurnCount)*0.3+float64(x)*0.02)))
			DrawCircle(screen, float64(x), waveY, 2, color.RGBA{0, 150, 200, alpha})
		}
	}
}

// DrawFloodWarning renders a warning HUD element.
func (fs *FloodState) DrawFloodWarning(screen *ebiten.Image) {
	if !fs.Active {
		return
	}
	progress := fs.GetFloodProgress()

	// Warning bar at top
	barW := 200.0
	barH := 10.0
	barX := float64(ScreenWidth)/2 - barW/2
	barY := 60.0

	// Background
	DrawRect(screen, barX, barY, barW, barH, color.RGBA{0, 0, 0, 150})

	// Flood fill
	fillW := barW * progress
	clr := color.RGBA{0, 100, 200, 200}
	if progress > 0.7 {
		clr = color.RGBA{200, 50, 50, 200}
	} else if progress > 0.4 {
		clr = color.RGBA{200, 150, 50, 200}
	}
	DrawRect(screen, barX, barY, fillW, barH, clr)

	// Label
	label := "FLOOD"
	if progress > 0.7 {
		label = "FLOOD! EVACUATE!"
	}
	DrawText(screen, label, int(barX)+int(barW)/2-len(label)*4, int(barY)-5, clr)

	// Turn counter
	turnStr := sprintf("Turns: %d", fs.TurnCount)
	DrawText(screen, turnStr, int(barX)+int(barW)/2-len(turnStr)*4, int(barY)+int(barH)+5, ColWhite)
}

// IsFlooded checks if the entire map is flooded.
func (fs *FloodState) IsFullyFlooded() bool {
	return fs.Active && fs.WaterLevel >= fs.MaxWater
}

// StopFlood ends the flooding (when rain stops or players escape).
func (fs *FloodState) StopFlood() {
	fs.Active = false
	fs.Puddles = nil
}

// ─── sprintf helper (avoids fmt import) ────────────────────────────
func sprintf(format string, args ...interface{}) string {
	// Simple numeric formatting helper
	result := ""
	argIdx := 0
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			switch format[i+1] {
			case 'd':
				if argIdx < len(args) {
					if n, ok := args[argIdx].(int); ok {
						result += itoa(n)
					}
					argIdx++
				}
				i++
			default:
				result += string(format[i])
			}
		} else {
			result += string(format[i])
		}
	}
	return result
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	if neg {
		digits = "-" + digits
	}
	return digits
}