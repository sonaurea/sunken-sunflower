package engine

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ─── Drawing Primitives ─────────────────────────────────────────────

// DrawCircle fills a circle on the target image at (cx, cy) with given radius and color.
func DrawCircle(target *ebiten.Image, cx, cy, radius float64, clr color.Color) {
	if radius <= 0 {
		return
	}
	r := float32(radius)
	vector.DrawFilledCircle(target, float32(cx), float32(cy), r, clr, true)
}

// DrawRect fills a rectangle on the target.
func DrawRect(target *ebiten.Image, x, y, w, h float64, clr color.Color) {
	vector.DrawFilledRect(target, float32(x), float32(y), float32(w), float32(h), clr, true)
}

// DrawLine draws a 1px-thick line on the target.
func DrawLine(target *ebiten.Image, x1, y1, x2, y2 float64, clr color.Color) {
	vector.StrokeLine(target, float32(x1), float32(y1), float32(x2), float32(y2), 1, clr, true)
}

// ─── Gradient ───────────────────────────────────────────────────────

// DrawGradient fills the entire target with a vertical gradient from topClr to bottomClr.
func DrawGradient(target *ebiten.Image, topClr, bottomClr color.Color) {
	w, h := target.Bounds().Dx(), target.Bounds().Dy()
	if w == 0 || h == 0 {
		return
	}
	// Simple banded gradient — 16 bands is enough for a smooth look.
	bands := 64
	bandH := h / bands
	if bandH < 1 {
		bandH = 1
	}
	for i := 0; i < bands; i++ {
		t := float64(i) / float64(bands-1)
		r := lerpU8(topClr.(color.RGBA).R, bottomClr.(color.RGBA).R, t)
		g := lerpU8(topClr.(color.RGBA).G, bottomClr.(color.RGBA).G, t)
		b := lerpU8(topClr.(color.RGBA).B, bottomClr.(color.RGBA).B, t)
		a := lerpU8(topClr.(color.RGBA).A, bottomClr.(color.RGBA).A, t)
		bandClr := color.RGBA{r, g, b, a}
		y := i * bandH
		DrawRect(target, 0, float64(y), float64(w), float64(bandH), bandClr)
	}
}

// ─── Glow ───────────────────────────────────────────────────────────

// DrawGlow draws a multi-layer glow effect centered at (cx, cy) with given radius.
// The glow fades from centerClr outward.
func DrawGlow(target *ebiten.Image, cx, cy, radius float64, centerClr color.Color) {
	layers := 6
	for i := layers; i > 0; i-- {
		t := float64(i) / float64(layers)
		r := float64(radius) * t
		// Overlay with decreasing alpha
		cr := centerClr.(color.RGBA)
		alpha := uint8(float64(cr.A) * (1.0 - float64(i-1)/float64(layers)) * 0.3)
		DrawCircle(target, cx, cy, r, color.RGBA{cr.R, cr.G, cr.B, alpha})
	}
}

// ─── Ocean Waves ────────────────────────────────────────────────────

// DrawOceanWaves animates sine-wave ocean across the target at given time offset.
func DrawOceanWaves(target *ebiten.Image, time float64) {
	w, h := target.Bounds().Dx(), target.Bounds().Dy()
	waveCount := 5
	waveHeight := 20.0
	for i := 0; i < waveCount; i++ {
		yBase := float64(h) - 40.0 - float64(i)*18.0
		alpha := uint8(60 + i*15)
		clr := color.RGBA{46, 196, 182, alpha}
		// Draw wave as series of horizontal lines with vertical offset
		for x := 0; x < w; x++ {
			offset := math.Sin(float64(x)*0.02+time*2.0+float64(i)*1.5) * waveHeight
			yy := yBase + offset
			if yy >= 0 && yy < float64(h) {
				target.Set(x, int(yy), clr)
			}
		}
	}
}

// ─── Star Field ─────────────────────────────────────────────────────

type Star struct {
	X, Y float64
	Size float64
}

// GenerateStarField creates a random distribution of stars.
func GenerateStarField(count int, w, h float64) []Star {
	stars := make([]Star, count)
	for i := range stars {
		stars[i] = Star{
			X:    rand.Float64() * w,
			Y:    rand.Float64() * h,
			Size: 1.0 + rand.Float64()*2.5,
		}
	}
	return stars
}

// DrawStarField renders the star field onto the target.
func DrawStarField(target *ebiten.Image, stars []Star, clr color.Color) {
	for _, s := range stars {
		DrawCircle(target, s.X, s.Y, s.Size, clr)
	}
}

// ─── Sand Texture ───────────────────────────────────────────────────

// GenerateBeachSand fills the target with a noisy sand-like grain.
func GenerateBeachSand(target *ebiten.Image, baseClr color.Color) {
	w, h := target.Bounds().Dx(), target.Bounds().Dy()
	br, bg, bb, ba := baseClr.(color.RGBA).R, baseClr.(color.RGBA).G, baseClr.(color.RGBA).B, baseClr.(color.RGBA).A
	for x := 0; x < w; x += 2 {
		for y := 0; y < h; y += 2 {
			noise := uint8(rand.Intn(30) - 15)
			c := color.RGBA{
				R: clampU8(int(br) + int(noise)),
				G: clampU8(int(bg) + int(noise)),
				B: clampU8(int(bb) + int(noise)),
				A: ba,
			}
			DrawRect(target, float64(x), float64(y), 2, 2, c)
		}
	}
}

// ─── Sunflower Sprite ───────────────────────────────────────────────

// GenerateSunflowerSprite procedurally draws a sunflower on a new image.
func GenerateSunflowerSprite(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	center := float64(size) / 2
	// Petals (yellow circles around center)
	for i := 0; i < 8; i++ {
		angle := float64(i) * math.Pi / 4
		px := center + math.Cos(angle)*float64(size)*0.3
		py := center + math.Sin(angle)*float64(size)*0.3
		DrawCircle(img, px, py, float64(size)*0.12, ColSunflower)
	}
	// Center (brown circle)
	DrawCircle(img, center, center, float64(size)*0.18, ColBrown)
	return img
}

// GenerateEnemySprite procedurally draws a generic enemy.
func GenerateEnemySprite(size int, clr color.Color) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	center := float64(size) / 2
	// Body
	DrawCircle(img, center, center, float64(size)*0.4, clr)
	// Eyes (white with black pupils)
	eyeR := float64(size) * 0.08
	DrawCircle(img, center-float64(size)*0.15, center-float64(size)*0.1, eyeR, ColWhite)
	DrawCircle(img, center+float64(size)*0.15, center-float64(size)*0.1, eyeR, ColWhite)
	DrawCircle(img, center-float64(size)*0.15, center-float64(size)*0.1, eyeR*0.5, ColBlack)
	DrawCircle(img, center+float64(size)*0.15, center-float64(size)*0.1, eyeR*0.5, ColBlack)
	return img
}

// GenerateCompanionSprite procedurally draws a bioluminescent companion.
func GenerateCompanionSprite(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	center := float64(size) / 2
	// Glow
	DrawGlow(img, center, center, float64(size)*0.5, ColBio)
	// Core
	DrawCircle(img, center, center, float64(size)*0.2, ColBio)
	return img
}

// ─── Procedural Tile ────────────────────────────────────────────────

// GenerateDungeonTile creates a simple stone floor tile.
func GenerateDungeonTile(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	baseClr := color.RGBA{60, 60, 80, 255}
	DrawRect(img, 0, 0, float64(size), float64(size), baseClr)
	// Stone pattern — subtle lines
	borderClr := color.RGBA{50, 50, 70, 255}
	DrawLine(img, 0, 0, float64(size), 0, borderClr)
	DrawLine(img, 0, 0, 0, float64(size), borderClr)
	// Random crack
	if rand.Intn(4) == 0 {
		crackClr := color.RGBA{40, 40, 60, 255}
		DrawLine(img, 2, 2, float64(size)-2, float64(size)-2, crackClr)
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