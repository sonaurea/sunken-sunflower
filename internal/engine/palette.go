package engine

import "image/color"

// Named color constants for the Sunken Sunflower palette.
// All procedural graphics reference these — mods can override via JSON.
var (
	ColSunflower   = color.RGBA{255, 215, 0, 255}   // #FFD700 — player, seeds, positive effects
	ColSunset      = color.RGBA{255, 107, 53, 255}  // #FF6B35 — sky gradients, warnings
	ColBeachPink   = color.RGBA{255, 140, 148, 255} // #FF8C94 — sunset transitions, UI accents
	ColDreamPurple = color.RGBA{123, 45, 142, 255}  // #7B2D8E — dream sequences, magic
	ColOcean       = color.RGBA{46, 196, 182, 255}  // #2EC4B6 — water, bioluminescence
	ColMidnight    = color.RGBA{26, 26, 62, 255}    // #1A1A3E — night sky, deep dungeon
	ColBio         = color.RGBA{0, 255, 204, 255}   // #00FFCC — glowing effects, companions
	ColWhite       = color.RGBA{255, 255, 255, 255}
	ColBlack       = color.RGBA{0, 0, 0, 255}
	ColRed         = color.RGBA{255, 50, 50, 255}
	ColGreen       = color.RGBA{50, 255, 50, 255}
	ColBrown       = color.RGBA{139, 69, 19, 255}
	ColSand        = color.RGBA{194, 178, 128, 255}
)

// ColorRGBA is a shorthand to create an RGBA color inline.
func ColorRGBA(r, g, b, a uint8) color.RGBA {
	return color.RGBA{r, g, b, a}
}