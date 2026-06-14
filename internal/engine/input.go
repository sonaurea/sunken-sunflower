package engine

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ─── Input Abstraction ──────────────────────────────────────────────

// Input stores the current frame's input state for keyboard and gamepad.
type Input struct {
	// Directional
	LeftPressed  bool
	RightPressed bool
	UpPressed    bool
	DownPressed  bool

	// Actions
	ActionPressed  bool // interact / confirm
	CancelPressed  bool // cancel / back
	DashPressed    bool
	CompanionCmd   bool // issue companion command

	// JustPressed variants (triggered once on key down)
	LeftJustPressed  bool
	RightJustPressed bool
	UpJustPressed    bool
	DownJustPressed  bool
	ActionJustPressed  bool
	CancelJustPressed  bool
	DashJustPressed    bool

	// Mouse
	MouseX, MouseY int
	MousePressed   bool

	// Raw axis values for gamepad (-1 to 1)
	GamepadLX, GamepadLY float64
}

// Update reads current input state from keyboard and first connected gamepad.
func (i *Input) Update() {
	// Reset frame-specific state
	i.LeftPressed = false
	i.RightPressed = false
	i.UpPressed = false
	i.DownPressed = false
	i.ActionPressed = false
	i.CancelPressed = false
	i.DashPressed = false
	i.CompanionCmd = false

	i.MouseX, i.MouseY = ebiten.CursorPosition()
	i.MousePressed = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// Keyboard
	i.LeftPressed = ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft)
	i.RightPressed = ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight)
	i.UpPressed = ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp)
	i.DownPressed = ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown)

	i.ActionPressed = ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsKeyPressed(ebiten.KeyE)
	i.CancelPressed = ebiten.IsKeyPressed(ebiten.KeyEscape) || ebiten.IsKeyPressed(ebiten.KeyQ)
	i.DashPressed = ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)
	i.CompanionCmd = ebiten.IsKeyPressed(ebiten.KeyF)

	// Just-pressed
	i.LeftJustPressed = inpututil.IsKeyJustPressed(ebiten.KeyA) || inpututil.IsKeyJustPressed(ebiten.KeyLeft)
	i.RightJustPressed = inpututil.IsKeyJustPressed(ebiten.KeyD) || inpututil.IsKeyJustPressed(ebiten.KeyRight)
	i.UpJustPressed = inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsKeyJustPressed(ebiten.KeyUp)
	i.DownJustPressed = inpututil.IsKeyJustPressed(ebiten.KeyS) || inpututil.IsKeyJustPressed(ebiten.KeyDown)
	i.ActionJustPressed = inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyE)
	i.CancelJustPressed = inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyQ)
	i.DashJustPressed = inpututil.IsKeyJustPressed(ebiten.KeyShiftLeft) || inpututil.IsKeyJustPressed(ebiten.KeyShiftRight)

	// Gamepad (first player)
	i.pollGamepad()
}

func (i *Input) pollGamepad() {
	const axisThreshold = 0.3

	// Get list of connected gamepad IDs
	gamepadIDs := ebiten.AppendGamepadIDs(nil)
	if len(gamepadIDs) == 0 {
		return
	}

	// Use the first gamepad
	id := gamepadIDs[0]

	i.GamepadLX = ebiten.GamepadAxis(id, 0)
	i.GamepadLY = ebiten.GamepadAxis(id, 1)

	// Standard gamepad button layout (Xbox-style):
	// 0=A(south), 1=B(east), 2=X(west), 3=Y(north)
	// 4=LB, 5=RB, 6=Select/Back, 7=Start
	// 8=Guide, 9=LT, 10=RT, 11=LS(click), 12=RS(click)
	// 13=DPadUp, 14=DPadDown, 15=DPadLeft, 16=DPadRight

	// D-pad / analog stick as directional
	if ebiten.IsGamepadButtonPressed(id, 15) || i.GamepadLX < -axisThreshold {
		i.LeftPressed = true
	}
	if ebiten.IsGamepadButtonPressed(id, 16) || i.GamepadLX > axisThreshold {
		i.RightPressed = true
	}
	if ebiten.IsGamepadButtonPressed(id, 13) || i.GamepadLY < -axisThreshold {
		i.UpPressed = true
	}
	if ebiten.IsGamepadButtonPressed(id, 14) || i.GamepadLY > axisThreshold {
		i.DownPressed = true
	}

	// Action buttons
	if ebiten.IsGamepadButtonPressed(id, 0) || ebiten.IsGamepadButtonPressed(id, 5) {
		i.ActionPressed = true
	}
	if ebiten.IsGamepadButtonPressed(id, 1) || ebiten.IsGamepadButtonPressed(id, 6) {
		i.CancelPressed = true
	}
	if ebiten.IsGamepadButtonPressed(id, 4) {
		i.DashPressed = true
	}
}