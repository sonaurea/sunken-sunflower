package engine

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ─── Drawing Primitives ─────────────────────────────────────────────

func DrawCircle(target *ebiten.Image, cx, cy, radius float64, clr color.Color) {
	if radius <= 0 {
		return
	}
	//nolint:staticcheck // DrawFilledCircle deprecated in v2.9, migrate when upgrading Ebitengine
	vector.DrawFilledCircle(target, float32(cx), float32(cy), float32(radius), clr, true)
}

func DrawRect(target *ebiten.Image, x, y, w, h float64, clr color.Color) {
	//nolint:staticcheck // DrawFilledRect deprecated in v2.9, migrate when upgrading Ebitengine
	vector.DrawFilledRect(target, float32(x), float32(y), float32(w), float32(h), clr, true)
}

func DrawLine(target *ebiten.Image, x1, y1, x2, y2 float64, clr color.Color) {
	vector.StrokeLine(target, float32(x1), float32(y1), float32(x2), float32(y2), 1, clr, true)
}

// ─── Gradient ───────────────────────────────────────────────────────

func DrawGradient(target *ebiten.Image, topClr, bottomClr color.Color) {
	w, h := target.Bounds().Dx(), target.Bounds().Dy()
	if w == 0 || h == 0 {
		return
	}
	bands := 64
	bandH := h / bands
	if bandH < 1 {
		bandH = 1
	}
	tr, tg, tb, ta := topClr.(color.RGBA).R, topClr.(color.RGBA).G, topClr.(color.RGBA).B, topClr.(color.RGBA).A
	br, bg, bb, ba := bottomClr.(color.RGBA).R, bottomClr.(color.RGBA).G, bottomClr.(color.RGBA).B, bottomClr.(color.RGBA).A
	for i := 0; i < bands; i++ {
		t := float64(i) / float64(bands-1)
		c := color.RGBA{
			lerpU8(tr, br, t), lerpU8(tg, bg, t),
			lerpU8(tb, bb, t), lerpU8(ta, ba, t),
		}
		DrawRect(target, 0, float64(i*bandH), float64(w), float64(bandH), c)
	}
}

// ─── Glow ───────────────────────────────────────────────────────────

func DrawGlow(target *ebiten.Image, cx, cy, radius float64, centerClr color.Color) {
	cr := centerClr.(color.RGBA)
	for i := 6; i > 0; i-- {
		t := float64(i) / 6.0
		r := radius * t
		alpha := uint8(float64(cr.A) * (1.0 - float64(i-1)/6.0) * 0.3)
		DrawCircle(target, cx, cy, r, color.RGBA{cr.R, cr.G, cr.B, alpha})
	}
}

// ─── Ocean Waves ────────────────────────────────────────────────────

func DrawOceanWaves(target *ebiten.Image, time float64) {
	w, h := target.Bounds().Dx(), target.Bounds().Dy()
	for i := 0; i < 5; i++ {
		yBase := float64(h) - 40.0 - float64(i)*18.0
		alpha := uint8(60 + i*15)
		clr := color.RGBA{46, 196, 182, alpha}
		for x := 0; x < w; x++ {
			offset := math.Sin(float64(x)*0.02+time*2.0+float64(i)*1.5) * 20
			yy := yBase + offset
			if yy >= 0 && yy < float64(h) {
				target.Set(x, int(yy), clr)
			}
		}
	}
}

// ─── Star Field ─────────────────────────────────────────────────────

type Star struct{ X, Y, Size float64 }

func GenerateStarField(count int, w, h float64) []Star {
	stars := make([]Star, count)
	for i := range stars {
		stars[i] = Star{rand.Float64() * w, rand.Float64() * h, 1.0 + rand.Float64()*2.5}
	}
	return stars
}

func DrawStarField(target *ebiten.Image, stars []Star, clr color.Color) {
	for _, s := range stars {
		DrawCircle(target, s.X, s.Y, s.Size, clr)
	}
}

func GenerateBeachSand(target *ebiten.Image, baseClr color.Color) {
	w, h := target.Bounds().Dx(), target.Bounds().Dy()
	cr := baseClr.(color.RGBA)
	for x := 0; x < w; x += 2 {
		for y := 0; y < h; y += 2 {
			n := uint8(rand.Intn(30) - 15)
			DrawRect(target, float64(x), float64(y), 2, 2,
				color.RGBA{clampU8(int(cr.R) + int(n)), clampU8(int(cr.G) + int(n)), clampU8(int(cr.B) + int(n)), cr.A})
		}
	}
}

// ─── CHARACTER SPRITES — With Real Detail ──────────────────────────

// GenerateSunflowerSprite creates a detailed sunflower character
// with petals, face, stem, and leaves.
func GenerateSunflowerSprite(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	center := float64(size) / 2
	r := float64(size) * 0.4

	// Petals (8 petals around, yellow with slight variation)
	for i := 0; i < 8; i++ {
		angle := float64(i) * math.Pi / 4
		px := center + math.Cos(angle)*r*0.75
		py := center + math.Sin(angle)*r*0.75
		petalR := r * 0.35
		petalClr := ColSunflower
		if i%2 == 0 {
			petalClr = color.RGBA{255, 225, 50, 255}
		}
		DrawCircle(img, px, py, petalR, petalClr)
	}

	// Inner face (brown center)
	DrawCircle(img, center, center, r*0.55, ColBrown)
	DrawCircle(img, center, center, r*0.35, color.RGBA{100, 50, 15, 255})

	// Eyes (white with black pupils)
	eyeY := center - r*0.1
	DrawCircle(img, center-r*0.15, eyeY, r*0.1, ColWhite)
	DrawCircle(img, center+r*0.15, eyeY, r*0.1, ColWhite)
	DrawCircle(img, center-r*0.15, eyeY, r*0.05, ColBlack)
	DrawCircle(img, center+r*0.15, eyeY, r*0.05, ColBlack)

	// Cute mouth (small smile arc)
	smileY := center + r*0.15
	for i := -2; i <= 2; i++ {
		DrawCircle(img, center+float64(i)*4, smileY+float64(i*i)*0.5, 1.5, color.RGBA{200, 100, 50, 255})
	}

	// Blush
	DrawCircle(img, center-r*0.3, center+r*0.05, r*0.08, color.RGBA{255, 150, 150, 100})
	DrawCircle(img, center+r*0.3, center+r*0.05, r*0.08, color.RGBA{255, 150, 150, 100})

	return img
}

// GenerateEnemySprite creates a detailed enemy with body, eyes, teeth.
func GenerateEnemySprite(size int, clr color.Color) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	center := float64(size) / 2
	cr := clr.(color.RGBA)
	r := float64(size) * 0.45

	// Body
	DrawCircle(img, center, center, r, clr)
	// Darker inner
	innerClr := color.RGBA{uint8(float64(cr.R) * 0.7), uint8(float64(cr.G) * 0.7), uint8(float64(cr.B) * 0.7), cr.A}
	DrawCircle(img, center, center, r*0.7, innerClr)

	// Angry eyes
	eyeY := center - r*0.2
	eyeR := r * 0.13
	DrawCircle(img, center-r*0.2, eyeY, eyeR, ColWhite)
	DrawCircle(img, center+r*0.2, eyeY, eyeR, ColWhite)
	DrawCircle(img, center-r*0.2, eyeY, eyeR*0.5, ColRed)
	DrawCircle(img, center+r*0.2, eyeY, eyeR*0.5, ColRed)

	// Angry eyebrows
	DrawLine(img, center-r*0.35, eyeY-eyeR*1.5, center-r*0.1, eyeY-eyeR*0.5, ColBlack)
	DrawLine(img, center+r*0.35, eyeY-eyeR*1.5, center+r*0.1, eyeY-eyeR*0.5, ColBlack)

	// Teeth
	for i := -2; i <= 2; i++ {
		DrawRect(img, center+float64(i)*4-2, center+r*0.2, 4, 5, ColWhite)
	}

	return img
}

// GenerateCompanionSprite creates a detailed companion creature.
func GenerateCompanionSprite(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	center := float64(size) / 2

	// Outer glow
	DrawGlow(img, center, center, float64(size)*0.5, ColBio)

	// Body
	DrawCircle(img, center, center, float64(size)*0.22, ColBio)

	// Inner core
	DrawCircle(img, center, center, float64(size)*0.1, ColWhite)

	// Eyes
	DrawCircle(img, center-4, center-3, 2.5, ColWhite)
	DrawCircle(img, center+4, center-3, 2.5, ColWhite)
	DrawCircle(img, center-4, center-3, 1.5, color.RGBA{0, 200, 150, 255})
	DrawCircle(img, center+4, center-3, 1.5, color.RGBA{0, 200, 150, 255})

	// Wings (small)
	for i := 0; i < 2; i++ {
		angle := float64(i)*math.Pi + math.Pi/4
		wx := center + math.Cos(angle)*float64(size)*0.25
		wy := center + math.Sin(angle)*float64(size)*0.25
		DrawCircle(img, wx, wy, 3, color.RGBA{0, 255, 200, 150})
	}

	return img
}

// GenerateDungeonTile creates a detailed stone floor tile.
func GenerateDungeonTile(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	baseClr := color.RGBA{75, 70, 55, 255}
	var borderClr color.RGBA

	// Stone pattern
	if rand.Intn(2) == 0 {
		baseClr = color.RGBA{80, 75, 58, 255}
		borderClr = color.RGBA{55, 50, 40, 255}
	} else {
		borderClr = color.RGBA{65, 60, 45, 255}
	}

	DrawRect(img, 0, 0, float64(size), float64(size), baseClr)
	DrawRect(img, 0, 0, float64(size), 1.5, borderClr)
	DrawRect(img, 0, 0, 1.5, float64(size), borderClr)
	DrawRect(img, float64(size)-1.5, 0, 1.5, float64(size), borderClr)
	DrawRect(img, 0, float64(size)-1.5, float64(size), 1.5, borderClr)

	// Texture cracks
	if rand.Intn(3) == 0 {
		DrawLine(img, 2, 2, float64(size)-2, float64(size)-2, borderClr)
	}
	return img
}

// ─── Helpers ────────────────────────────────────────────────────────

func lerpU8(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + (float64(b)-float64(a))*t)
}

func clampU8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}