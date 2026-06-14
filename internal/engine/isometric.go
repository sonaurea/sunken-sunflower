package engine

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ─── Isometric Rendering System ─────────────────────────────────────
// Inspired by Bastion's gorgeous isometric perspective:
//   - 3/4 top-down view with diamond tiles
//   - Depth-sorted entity rendering
//   - World-relative positioning with camera following
//   - Dynamic tile elevation (ground rises as you approach)

// IsoConfig holds the isometric projection parameters.
type IsoConfig struct {
	TileWidth    float64 // width of a tile in pixels (e.g. 64)
	TileHeight   float64 // height of a tile in pixels (e.g. 32)
	BaseY        float64 // screen Y offset for the top of the map
	ElevationScale float64 // how much height differences affect Y
}

// DefaultIsoConfig returns sensible isometric defaults.
func DefaultIsoConfig() IsoConfig {
	return IsoConfig{
		TileWidth:      64,
		TileHeight:     32,
		BaseY:          ScreenHeight * 0.2,
		ElevationScale: 16,
	}
}

// ─── Coordinate Conversion ──────────────────────────────────────────

// WorldToIso converts world (grid) coordinates to isometric screen space.
// World X increases to the right, world Y increases downward.
// Iso X points right-down, Iso Y points left-down.
func WorldToIso(wx, wy float64, cfg IsoConfig) (float64, float64) {
	sx := (wx - wy) * cfg.TileWidth / 2
	sy := (wx + wy) * cfg.TileHeight / 2
	return sx, sy
}

// IsoToWorld converts isometric screen coords back to world coordinates.
func IsoToWorld(sx, sy float64, cfg IsoConfig) (float64, float64) {
	wx := (sx/cfg.TileWidth + sy/cfg.TileHeight)
	wy := (sy/cfg.TileHeight - sx/cfg.TileWidth)
	return wx, wy
}

// WorldToScreen converts world coords to screen coords with camera offset.
func WorldToScreen(wx, wy, elev float64, cfg IsoConfig, camX, camY float64) (float64, float64) {
	ix, iy := WorldToIso(wx, wy, cfg)
	sx := ix - camX + float64(ScreenWidth)/2
	sy := iy - camY + cfg.BaseY - elev*cfg.ElevationScale
	return sx, sy
}

// ─── Isometric Tile ─────────────────────────────────────────────────

// IsoTile represents a single isometric floor tile with optional features.
type IsoTile struct {
	WorldX, WorldY float64   // grid position
	Elevation     float64   // height (0 = ground)
	Color         color.Color
	HasWall       bool
	WallColor     color.Color
	WallHeight    float64
	Occupied      bool
	IsExit        bool
	IsChest       bool
	IsWell        bool
}

// IsoMap is a grid of isometric tiles forming the dungeon floor.
type IsoMap struct {
	Tiles    [][]*IsoTile
	Width    int
	Height   int
	Config   IsoConfig
}

// NewIsoMap creates a new isometric map with the given dimensions.
func NewIsoMap(w, h int, cfg IsoConfig) *IsoMap {
	m := &IsoMap{
		Tiles:  make([][]*IsoTile, w),
		Width:  w,
		Height: h,
		Config: cfg,
	}
	for x := 0; x < w; x++ {
		m.Tiles[x] = make([]*IsoTile, h)
		for y := 0; y < h; y++ {
			clr := color.RGBA{40, 40, 55, 255}
			if (x+y)%2 == 0 {
				clr = color.RGBA{50, 50, 65, 255}
			}
			m.Tiles[x][y] = &IsoTile{
				WorldX:    float64(x),
				WorldY:    float64(y),
				Color:     clr,
				WallHeight: 1.0,
			}
		}
	}
	return m
}

// TileAt returns the tile at grid position, or nil if out of bounds.
func (m *IsoMap) TileAt(x, y int) *IsoTile {
	if x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return nil
	}
	return m.Tiles[x][y]
}

// SetElevation sets the elevation of a tile and its neighbors for smooth transitions.
func (m *IsoMap) SetElevation(cx, cy int, elev float64, radius int) {
	for dx := -radius; dx <= radius; dx++ {
		for dy := -radius; dy <= radius; dy++ {
			t := m.TileAt(cx+dx, cy+dy)
			if t == nil {
				continue
			}
			dist := math.Sqrt(float64(dx*dx + dy*dy))
			falloff := 1.0 - dist/float64(radius+1)
			if falloff > 0 {
				t.Elevation = elev * falloff
			}
		}
	}
}

// ─── Draw Isometric Map ─────────────────────────────────────────────
// Renders the map with proper depth sorting (painter's algorithm).

type drawableTile struct {
	screenX, screenY float64
	tile             *IsoTile
	depth            float64
}

func (m *IsoMap) Draw(screen *ebiten.Image, camX, camY float64) {
	cfg := m.Config
	var drawables []drawableTile

	// Collect all visible tiles
	for x := 0; x < m.Width; x++ {
		for y := 0; y < m.Height; y++ {
			tile := m.Tiles[x][y]
			sx, sy := WorldToScreen(tile.WorldX, tile.WorldY, 0, cfg, camX, camY)

			// Frustum culling
			if sx < -cfg.TileWidth || sx > ScreenWidth+cfg.TileWidth ||
				sy < -cfg.TileHeight || sy > ScreenHeight+cfg.TileHeight {
				continue
			}

			// Depth = worldX + worldY (for painter's algorithm)
			depth := tile.WorldX + tile.WorldY
			drawables = append(drawables, drawableTile{
				screenX: sx,
				screenY: sy,
				tile:    tile,
				depth:   depth,
			})
		}
	}

	// Sort back to front (painter's algorithm)
	sort.Slice(drawables, func(i, j int) bool {
		return drawables[i].depth < drawables[j].depth
	})

	// Draw sorted tiles
	for _, d := range drawables {
		tile := d.tile
		sx, sy := d.screenX, d.screenY

		// Draw floor diamond
		drawIsoDiamond(screen, sx, sy, cfg.TileWidth, cfg.TileHeight, tile.Color)

		// Elevation walls (if tile is raised)
		if tile.Elevation > 0 {
			elevPx := tile.Elevation * cfg.ElevationScale
			// Draw side walls (left and right faces)
			wallClr := color.RGBA{30, 30, 45, 255}
			// Left wall
			drawIsoWallLeft(screen, sx, sy, cfg.TileWidth, cfg.TileHeight, elevPx, wallClr)
			// Right wall
			drawIsoWallRight(screen, sx, sy, cfg.TileWidth, cfg.TileHeight, elevPx, wallClr)
			// Top face (slightly brighter)
			topClr := tile.Color
			drawIsoDiamond(screen, sx, sy-elevPx, cfg.TileWidth, cfg.TileHeight, topClr)
		}

		// Wall on tile
		if tile.HasWall {
			drawIsoWall(screen, sx, sy, cfg.TileWidth, cfg.TileHeight, tile.WallHeight*cfg.ElevationScale, tile.WallColor)
		}

		// Features
		if tile.IsChest {
			drawIsoChest(screen, sx, sy, cfg)
		}
		if tile.IsExit {
			drawIsoExit(screen, sx, sy, cfg)
		}
		if tile.IsWell {
			drawIsoWell(screen, sx, sy, cfg)
		}
	}
}

// ─── Isometric Drawing Primitives ───────────────────────────────────

// drawIsoDiamond draws a filled isometric diamond (tile floor).
func drawIsoDiamond(screen *ebiten.Image, cx, cy, w, h float64, clr color.Color) {
	hw := w/2
	hh := h / 2
	// Build a diamond out of triangles (approximate with rects and circles)
	// Top triangle
	drawIsoTriangle(screen, cx, cy-hh, cx-hw, cy, cx, cy+hh, clr)
	drawIsoTriangle(screen, cx, cy-hh, cx+hw, cy, cx, cy+hh, clr)
}

// drawIsoTriangle draws a filled triangle (for isometric faces).
func drawIsoTriangle(screen *ebiten.Image, x1, y1, x2, y2, x3, y3 float64, clr color.Color) {
	// Approximate triangle with lines
	DrawLine(screen, x1, y1, x2, y2, clr)
	DrawLine(screen, x2, y2, x3, y3, clr)
	DrawLine(screen, x3, y3, x1, y1, clr)
	// Fill with a circle at centroid
	centX := (x1 + x2 + x3) / 3
	centY := (y1 + y2 + y3) / 3
	DrawCircle(screen, centX, centY, 4, clr)
}

// drawIsoWall draws a wall block on a tile.
func drawIsoWall(screen *ebiten.Image, cx, cy, w, h, wallH float64, clr color.Color) {
	hw, hh := w/2, h/2
	// Front face
	drawIsoQuad(screen, cx-hw, cy, cx, cy+hh, cx, cy+hh-wallH, cx-hw, cy-wallH, clr)
	// Right face (darker)
	rClr := color.RGBA{
		R: uint8(float64(clr.(color.RGBA).R) * 0.7),
		G: uint8(float64(clr.(color.RGBA).G) * 0.7),
		B: uint8(float64(clr.(color.RGBA).B) * 0.7),
		A: clr.(color.RGBA).A,
	}
	drawIsoQuad(screen, cx, cy+hh, cx+hw, cy, cx+hw, cy-wallH, cx, cy+hh-wallH, rClr)
	// Top face (brighter)
	tClr := color.RGBA{
		R: uint8(float64(clr.(color.RGBA).R) * 1.3),
		G: uint8(float64(clr.(color.RGBA).G) * 1.3),
		B: uint8(float64(clr.(color.RGBA).B) * 1.3),
		A: clr.(color.RGBA).A,
	}
	drawIsoDiamond(screen, cx, cy-wallH, w, h, tClr)
}

// drawIsoWallLeft draws the left face of an elevated tile.
func drawIsoWallLeft(screen *ebiten.Image, cx, cy, w, h, elev float64, clr color.Color) {
	hw, hh := w/2, h/2
	drawIsoQuad(screen, cx-hw, cy, cx, cy+hh, cx, cy+hh-elev, cx-hw, cy-elev, clr)
}

// drawIsoWallRight draws the right face of an elevated tile.
func drawIsoWallRight(screen *ebiten.Image, cx, cy, w, h, elev float64, clr color.Color) {
	hw, hh := w/2, h/2
	drawIsoQuad(screen, cx, cy+hh, cx+hw, cy, cx+hw, cy-elev, cx, cy+hh-elev, clr)
}

// drawIsoQuad draws a filled isometric quadrilateral (approximate with lines and fill).
func drawIsoQuad(screen *ebiten.Image, x1, y1, x2, y2, x3, y3, x4, y4 float64, clr color.Color) {
	DrawLine(screen, x1, y1, x2, y2, clr)
	DrawLine(screen, x2, y2, x3, y3, clr)
	DrawLine(screen, x3, y3, x4, y4, clr)
	DrawLine(screen, x4, y4, x1, y1, clr)
	// Fill centroid
	cx := (x1 + x2 + x3 + x4) / 4
	cy := (y1 + y2 + y3 + y4) / 4
	DrawCircle(screen, cx, cy, 3, clr)
}

// ─── Feature Drawings ───────────────────────────────────────────────

func drawIsoChest(screen *ebiten.Image, sx, sy float64, cfg IsoConfig) {
	hw := cfg.TileWidth * 0.3
	hh := cfg.TileHeight * 0.3
	DrawRect(screen, sx-hw, sy-hh*2, hw*2, hh*2, ColSunflower)
	DrawRect(screen, sx-hw*0.6, sy-hh*2.3, hw*1.2, hh*0.5, ColBrown)
	DrawCircle(screen, sx, sy-hh*2.3, 2, ColSunflower)
}

func drawIsoExit(screen *ebiten.Image, sx, sy float64, cfg IsoConfig) {
	DrawGlow(screen, sx, sy-cfg.TileHeight*0.5, cfg.TileWidth*0.5, ColBio)
}

func drawIsoWell(screen *ebiten.Image, sx, sy float64, cfg IsoConfig) {
	r := cfg.TileWidth * 0.35
	DrawCircle(screen, sx, sy-r*0.5, r, ColMidnight)
	DrawCircle(screen, sx, sy-r*0.5, r*0.7, ColBlack)
	DrawRect(screen, sx-r, sy-r*0.8, r*2, r*0.2, ColBrown)
}

// ─── Entity Rendering on Isometric Map ──────────────────────────────

// IsoEntityDrawer handles depth-sorted entity rendering.
type IsoEntityDrawer struct {
	entities []IsoDrawable
}

type IsoDrawable struct {
	ScreenX, ScreenY float64
	Depth            float64
	DrawFn           func(screen *ebiten.Image)
}

func NewIsoEntityDrawer() *IsoEntityDrawer {
	return &IsoEntityDrawer{}
}

func (ied *IsoEntityDrawer) Add(sx, sy, depth float64, drawFn func(*ebiten.Image)) {
	ied.entities = append(ied.entities, IsoDrawable{
		ScreenX: sx,
		ScreenY: sy,
		Depth:   depth,
		DrawFn:  drawFn,
	})
}

func (ied *IsoEntityDrawer) Draw(screen *ebiten.Image) {
	sort.Slice(ied.entities, func(i, j int) bool {
		return ied.entities[i].Depth < ied.entities[j].Depth
	})
	for _, e := range ied.entities {
		e.DrawFn(screen)
	}
	ied.entities = nil
}

// ─── Isometric Camera ──────────────────────────────────────────────
// Follows the player in world coordinates, converting to iso screen space.

type IsoCamera struct {
	WorldX, WorldY float64
	TargetX, TargetY float64
	Smoothing      float64 // 0-1, how fast camera follows
	Config         IsoConfig
}

func NewIsoCamera(cfg IsoConfig) *IsoCamera {
	return &IsoCamera{
		Config:    cfg,
		Smoothing: 0.08,
	}
}

func (ic *IsoCamera) Follow(targetWX, targetWY float64) {
	ic.TargetX = targetWX * ic.Config.TileWidth
	ic.TargetY = targetWY * ic.Config.TileHeight
}

func (ic *IsoCamera) Update() {
	// Smooth interpolation
	ic.WorldX += (ic.TargetX - ic.WorldX) * ic.Smoothing
	ic.WorldY += (ic.TargetY - ic.WorldY) * ic.Smoothing
}

// ScreenOffset returns the camera offset for rendering.
func (ic *IsoCamera) ScreenOffset() (float64, float64) {
	return ic.WorldX, ic.WorldY
}

// ─── Isometric Grid Highlighting ────────────────────────────────────

// DrawIsoHighlight draws a glowing highlight ring around the tile at world coords.
func DrawIsoHighlight(screen *ebiten.Image, wx, wy float64, cfg IsoConfig, camX, camY float64, clr color.Color) {
	sx, sy := WorldToScreen(wx, wy, 0, cfg, camX, camY)
	DrawGlowRing(screen, sx, sy, cfg.TileWidth*0.3, 0, clr)
}

// ─── Isometric Mini-Map ────────────────────────────────────────────

func DrawIsoMinimap(screen *ebiten.Image, m *IsoMap, playerWX, playerWY float64) {
	mmSize := 80.0
	mmX := float64(ScreenWidth) - mmSize - 10
	mmY := float64(ScreenHeight) - mmSize - 10
	tileS := mmSize / float64(max(m.Width, m.Height))

	// Background
	DrawRect(screen, mmX, mmY, mmSize, mmSize, color.RGBA{0, 0, 0, 150})

	for x := 0; x < m.Width; x++ {
		for y := 0; y < m.Height; y++ {
			t := m.Tiles[x][y]
			clr := color.RGBA{40, 40, 60, 200}
			if t.IsExit {
				clr = ColBio
			} else if t.IsChest {
				clr = ColSunflower
			} else if t.HasWall {
				clr = color.RGBA{80, 80, 100, 200}
			} else if t.Elevation > 0 {
				clr = color.RGBA{60, 60, 80, 200}
			}
			DrawRect(screen, mmX+float64(x)*tileS, mmY+float64(y)*tileS, tileS, tileS, clr)
		}
	}

	// Player dot
	DrawCircle(screen, mmX+playerWX*tileS+mmSize/float64(m.Width)/2, mmY+playerWY*tileS+mmSize/float64(m.Height)/2, 3, ColSunflower)
}