package engine

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ─── Turn-Based System ─────────────────────────────────────────────
// AP (Action Point) system inspired by Pox Nora / tactics games.
// Player and enemies take turns on an isometric grid.

const (
	MaxAP          = 6   // maximum action points per turn
	MoveAPCost     = 1   // AP to move one tile
	AttackAPCost   = 2   // AP to attack
	SpecialAPCost  = 3   // AP to use special ability
	DashAPCost     = 2   // AP to dash
	EndTurnAPCost  = 0   // AP remaining to end turn (any amount)
	APRegenPerTurn = MaxAP // full AP refresh each turn
)

// TurnPhase tracks whose turn it is and what phase.
type TurnPhase int

const (
	PhasePlayerTurn TurnPhase = iota
	PhaseEnemyTurn
	PhaseWin
	PhaseLose
)

// TurnManager handles the turn-based flow.
type TurnManager struct {
	Phase          TurnPhase
	CurrentAP      int
	MaxAP          int
	TurnNumber     int
	EnemyIndex     int // which enemy is currently acting
	EnemyActing    bool
	WaitTimer      float64
	MoveRange      []GridPos // tiles player can move to
	AttackRange    []GridPos // tiles player can attack
	SelectedTile   GridPos
	HasMoved       bool
	HasAttacked    bool
	Log            []string
}

// GridPos is a position on the isometric grid.
type GridPos struct {
	X, Y int
}

// NewTurnManager creates a turn manager.
func NewTurnManager() *TurnManager {
	return &TurnManager{
		Phase:     PhasePlayerTurn,
		CurrentAP: MaxAP,
		MaxAP:     MaxAP,
		TurnNumber: 1,
	}
}

// StartPlayerTurn begins the player's turn.
func (tm *TurnManager) StartPlayerTurn() {
	tm.Phase = PhasePlayerTurn
	tm.CurrentAP = MaxAP
	tm.HasMoved = false
	tm.HasAttacked = false
	tm.MoveRange = nil
	tm.AttackRange = nil
	tm.Log = append(tm.Log, "✦ Your turn")
}

// EndPlayerTurn ends the player's turn and starts enemy phase.
func (tm *TurnManager) EndPlayerTurn() {
	tm.Phase = PhaseEnemyTurn
	tm.EnemyIndex = 0
	tm.EnemyActing = true
	tm.Log = append(tm.Log, "Enemy turn...")
}

// CanMove checks if the player can move to a tile.
func (tm *TurnManager) CanMove(cost int) bool {
	return tm.CurrentAP >= cost
}

// SpendAP spends AP and returns true if affordable.
func (tm *TurnManager) SpendAP(cost int) bool {
	if tm.CurrentAP < cost {
		return false
	}
	tm.CurrentAP -= cost
	return true
}

// CanEndTurn checks if the player can end their turn.
func (tm *TurnManager) CanEndTurn() bool {
	return tm.Phase == PhasePlayerTurn
}

// CalculateMoveRange returns tiles within movement range from a position.
// Uses isometric distance (tile-based).
func CalculateMoveRange(startX, startY float64, ap int, mapW, mapH int) []GridPos {
	var tiles []GridPos
	sx, sy := int(math.Round(startX)), int(math.Round(startY))
	maxDist := ap / MoveAPCost
	if maxDist < 1 {
		maxDist = 1
	}
	for dx := -maxDist; dx <= maxDist; dx++ {
		for dy := -maxDist; dy <= maxDist; dy++ {
			dist := int(math.Abs(float64(dx)) + math.Abs(float64(dy)))
			if dist > 0 && dist <= maxDist {
				nx, ny := sx+dx, sy+dy
				if nx >= 0 && nx < mapW && ny >= 0 && ny < mapH {
					tiles = append(tiles, GridPos{nx, ny})
				}
			}
		}
	}
	return tiles
}

// CalculateAttackRange returns tiles within attack range.
func CalculateAttackRange(startX, startY float64, rangeDist int, mapW, mapH int) []GridPos {
	var tiles []GridPos
	sx, sy := int(math.Round(startX)), int(math.Round(startY))
	for dx := -rangeDist; dx <= rangeDist; dx++ {
		for dy := -rangeDist; dy <= rangeDist; dy++ {
			dist := int(math.Abs(float64(dx)) + math.Abs(float64(dy)))
			if dist > 0 && dist <= rangeDist {
				nx, ny := sx+dx, sy+dy
				if nx >= 0 && nx < mapW && ny >= 0 && ny < mapH {
					tiles = append(tiles, GridPos{nx, ny})
				}
			}
		}
	}
	return tiles
}

// TileDistance calculates Manhattan distance between two tiles.
func TileDistance(x1, y1, x2, y2 int) int {
	return int(math.Abs(float64(x1-x2)) + math.Abs(float64(y1-y2)))
}

// ─── Turn UI Drawing ───────────────────────────────────────────────

// DrawTurnUI renders the turn-based HUD: AP bar, turn info, etc.
func DrawTurnUI(screen *ebiten.Image, tm *TurnManager) {
	// AP bar (top center)
	barX := ScreenWidth/2 - 120
	barY := 5
	barW := 240
	barH := 22

	// Background
	DrawRect(screen, float64(barX), float64(barY), float64(barW), float64(barH), color.RGBA{0, 0, 0, 180})

	// AP segments
	segW := float64(barW-2) / float64(tm.MaxAP)
	for i := 0; i < tm.MaxAP; i++ {
		x := float64(barX) + 1 + float64(i)*segW
		clr := color.RGBA{80, 80, 80, 255} // spent
		if i < tm.CurrentAP {
			clr = ColSunflower // available
		}
		DrawRect(screen, x, float64(barY)+2, segW-2, float64(barH)-4, clr)
	}

	// AP text
	apStr := fmt.Sprintf("AP: %d/%d", tm.CurrentAP, tm.MaxAP)
	DrawText(screen, apStr, barX+barW/2-len(apStr)*4, barY+barH-5, ColWhite)

	// Turn info
	turnStr := fmt.Sprintf("Turn %d", tm.TurnNumber)
	if tm.Phase == PhaseEnemyTurn {
		turnStr += " — Enemy Phase"
	}
	DrawText(screen, turnStr, 10, 6, ColWhite)
}

// DrawMoveRange visualizes the available movement tiles.
func DrawMoveRange(screen *ebiten.Image, tiles []GridPos, isoCfg IsoConfig, camX, camY float64) {
	for _, t := range tiles {
		sx, sy := WorldToScreen(float64(t.X), float64(t.Y), 0, isoCfg, camX, camY)
		clr := color.RGBA{100, 200, 255, 80}
		drawIsoDiamond(screen, sx, sy, isoCfg.TileWidth, isoCfg.TileHeight, clr)
	}
}

// DrawAttackRange visualizes attackable tiles.
func DrawAttackRange(screen *ebiten.Image, tiles []GridPos, isoCfg IsoConfig, camX, camY float64) {
	for _, t := range tiles {
		sx, sy := WorldToScreen(float64(t.X), float64(t.Y), 0, isoCfg, camX, camY)
		clr := color.RGBA{255, 80, 80, 80}
		drawIsoDiamond(screen, sx, sy, isoCfg.TileWidth, isoCfg.TileHeight, clr)
	}
}

// DrawSelectedTile highlights the currently selected tile.
func DrawSelectedTile(screen *ebiten.Image, pos GridPos, isoCfg IsoConfig, camX, camY float64) {
	sx, sy := WorldToScreen(float64(pos.X), float64(pos.Y), 0.3, isoCfg, camX, camY)
	DrawGlowRing(screen, sx, sy, isoCfg.TileWidth*0.4, 0, ColSunflower)
}

// ─── End Turn Button ───────────────────────────────────────────────

func DrawEndTurnButton(screen *ebiten.Image, tm *TurnManager) {
	if tm.Phase != PhasePlayerTurn {
		return
	}
	btnX := ScreenWidth - 140
	btnY := ScreenHeight - 45
	btnW := 130
	btnH := 35

	// Button
	clr := color.RGBA{50, 50, 80, 220}
	if tm.CurrentAP == 0 || tm.HasAttacked {
		clr = color.RGBA{200, 150, 50, 220}
	}
	DrawRect(screen, float64(btnX), float64(btnY), float64(btnW), float64(btnH), clr)
	DrawPulsingBorder(screen, float64(btnX), float64(btnY), float64(btnW), float64(btnH), 0, ColSunflower, 2)

	label := "END TURN [E]"
	DrawText(screen, label, btnX+btnW/2-len(label)*4, btnY+btnH-8, ColWhite)
}

// DrawActionPanel shows available actions.
func DrawActionPanel(screen *ebiten.Image, tm *TurnManager) {
	if tm.Phase != PhasePlayerTurn {
		return
	}
	x := 10
	y := ScreenHeight - 100
	DrawRect(screen, float64(x-5), float64(y-5), 260, 100, color.RGBA{0, 0, 0, 160})

	actions := []string{
		fmt.Sprintf("[Click] Move (%d AP)", MoveAPCost),
		fmt.Sprintf("[A] Attack (%d AP)", AttackAPCost),
		fmt.Sprintf("[S] Special (%d AP)", SpecialAPCost),
		fmt.Sprintf("[E] End Turn (any AP)" ),
	}
	for i, a := range actions {
		clr := ColWhite
		if i == 0 && !tm.CanMove(MoveAPCost) {
			clr = color.RGBA{100, 100, 100, 255}
		}
		if i == 1 && !tm.CanMove(AttackAPCost) {
			clr = color.RGBA{100, 100, 100, 255}
		}
		DrawText(screen, a, x, y+i*22, clr)
	}
}

// ─── Floating Combat Text ──────────────────────────────────────────

type CombatLogEntry struct {
	Text string
	X, Y float64
	Life float64
	Color color.Color
}

var CombatLog []CombatLogEntry

func AddCombatLog(text string, x, y float64, clr color.Color) {
	CombatLog = append(CombatLog, CombatLogEntry{
		Text: text, X: x, Y: y,
		Life: 2.0, Color: clr,
	})
}

func UpdateCombatLog(dt float64) {
	alive := CombatLog[:0]
	for _, e := range CombatLog {
		e.Life -= dt
		if e.Life > 0 {
			e.Y -= dt * 20
			alive = append(alive, e)
		}
	}
	CombatLog = alive
}

func DrawCombatLog(screen *ebiten.Image) {
	for _, e := range CombatLog {
		alpha := uint8(255 * (e.Life / 2.0))
		cr := e.Color.(color.RGBA)
		c := color.RGBA{cr.R, cr.G, cr.B, alpha}
		DrawText(screen, e.Text, int(e.X), int(e.Y), c)
	}
}