package engine

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// ─── Font System ────────────────────────────────────────────────────

// DefaultFace is the built-in 7x13 pixel bitmap font used throughout the game.
// Replace with TTF loading when custom fonts are desired.
var DefaultFace font.Face = basicfont.Face7x13

// DrawText draws text at (x, y) using the default font. This wraps text.Draw
// so scenes don't need to import text/font packages directly.
func DrawText(screen *ebiten.Image, str string, x, y int, clr color.Color) {
	text.Draw(screen, str, DefaultFace, x, y, clr)
}

// textBound returns the bounding rectangle for a string using the default font.
func textBound(str string) (int, int) {
	b := text.BoundString(DefaultFace, str)
	return b.Dx(), b.Dy()
}

// CenterX returns the x-coordinate to center a string horizontally in a given width.
func CenterX(str string, screenWidth int) int {
	w, _ := textBound(str)
	return (screenWidth - w) / 2
}

// TextWidth returns the pixel width of the given string.
func TextWidth(str string) int {
	w, _ := textBound(str)
	return w
}

// TextHeight returns the pixel height of the given string.
func TextHeight(str string) int {
	_, h := textBound(str)
	return h
}