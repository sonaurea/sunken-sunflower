package engine

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ─── Dialogue System ───────────────────────────────────────────────
// Full-featured dialogue with typing animation, portraits, choices,
// and queue management. Works in both 2D and isometric modes.

// DialogueLine is a single line of dialogue with metadata.
type DialogueLine struct {
	Speaker string
	Text    string
	Color   color.Color  // speaker name color
	PortraitClr color.Color // portrait color
	SoundID string       // optional sound effect
}

// DialogueChoice is a player-selectable option.
type DialogueChoice struct {
	Text    string
	NextLine int // line index to jump to, -1 to end
}

// DialogueBox renders and manages a dialogue box with typing animation.
type DialogueBox struct {
	Lines      []DialogueLine
	Choices    []DialogueChoice
	CurrentLine int
	Visible    bool
	Finished   bool

	// Typing animation
	typingBuffer string
	typingIndex  int
	charTimer    float64
	charSpeed    float64 // seconds per character
	doneTyping   bool

	// Box dimensions
	BoxX, BoxY   float64
	BoxW, BoxH   float64
	Padding      float64

	// Portrait
	ShowPortrait bool
	PortraitSize float64

	// Choice selection
	SelectedChoice int
	ChoiceActive   bool

	// Timing
	startTime time.Time
	onComplete func()
	onChoice   func(choiceIndex int)
}

// NewDialogueBox creates a dialogue box with default styling.
func NewDialogueBox() *DialogueBox {
	return &DialogueBox{
		BoxX:         80,
		BoxY:         ScreenHeight - 180,
		BoxW:         ScreenWidth - 160,
		BoxH:         150,
		Padding:      15,
		PortraitSize: 48,
		charSpeed:    0.035, // ~29 chars/sec
	}
}

// SetLines sets the dialogue lines and resets the box.
func (db *DialogueBox) SetLines(lines []DialogueLine) {
	db.Lines = lines
	db.CurrentLine = 0
	db.Choices = nil
	db.Visible = true
	db.Finished = false
	db.startLine()
}

// SetChoices sets selectable choices for the current line.
func (db *DialogueBox) SetChoices(choices []DialogueChoice) {
	db.Choices = choices
	db.ChoiceActive = len(choices) > 0
	db.SelectedChoice = 0
}

// startLine initializes typing for the current line.
func (db *DialogueBox) startLine() {
	if db.CurrentLine >= len(db.Lines) {
		db.Finished = true
		db.Visible = false
		if db.onComplete != nil {
			db.onComplete()
		}
		return
	}
	db.typingBuffer = ""
	db.typingIndex = 0
	db.charTimer = 0
	db.doneTyping = false
}

// Update advances the typing animation.
func (db *DialogueBox) Update(dt float64) {
	if !db.Visible || db.Finished {
		return
	}

	if db.CurrentLine >= len(db.Lines) {
		db.Finished = true
		return
	}

	line := db.Lines[db.CurrentLine]

	if !db.doneTyping {
		db.charTimer += dt
		for db.charTimer >= db.charSpeed && db.typingIndex < len(line.Text) {
			db.typingBuffer += string(line.Text[db.typingIndex])
			db.typingIndex++
			db.charTimer -= db.charSpeed
		}
		if db.typingIndex >= len(line.Text) {
			db.doneTyping = true
		}
	}
}

// Advance advances to the next line or completes.
// Returns true if the dialogue is still going, false if finished.
func (db *DialogueBox) Advance() bool {
	if !db.Visible || db.Finished {
		return false
	}

	// If still typing, complete instantly
	if !db.doneTyping {
		line := db.Lines[db.CurrentLine]
		db.typingBuffer = line.Text
		db.typingIndex = len(line.Text)
		db.doneTyping = true
		return true
	}

	// If choices are active, don't advance with regular advance
	if db.ChoiceActive {
		return true
	}

	// Move to next line
	db.CurrentLine++
	if db.CurrentLine >= len(db.Lines) {
		db.Finished = true
		db.Visible = false
		if db.onComplete != nil {
			db.onComplete()
		}
		return false
	}
	db.startLine()
	return true
}

// SelectChoice moves the choice selector up/down.
func (db *DialogueBox) SelectChoice(direction int) {
	if !db.ChoiceActive || len(db.Choices) == 0 {
		return
	}
	db.SelectedChoice += direction
	if db.SelectedChoice < 0 {
		db.SelectedChoice = len(db.Choices) - 1
	}
	if db.SelectedChoice >= len(db.Choices) {
		db.SelectedChoice = 0
	}
}

// ConfirmChoice confirms the currently selected choice.
func (db *DialogueBox) ConfirmChoice() int {
	if !db.ChoiceActive || db.SelectedChoice < 0 || db.SelectedChoice >= len(db.Choices) {
		return -1
	}
	choice := db.Choices[db.SelectedChoice]
	if db.onChoice != nil {
		db.onChoice(db.SelectedChoice)
	}
	if choice.NextLine < 0 {
		db.Finished = true
		db.Visible = false
		if db.onComplete != nil {
			db.onComplete()
		}
		return -1
	}
	db.CurrentLine = choice.NextLine
	db.Choices = nil
	db.ChoiceActive = false
	db.startLine()
	return db.SelectedChoice
}

// Draw renders the dialogue box on screen.
func (db *DialogueBox) Draw(screen *ebiten.Image) {
	if !db.Visible || db.Finished || db.CurrentLine >= len(db.Lines) {
		return
	}

	line := db.Lines[db.CurrentLine]

	// Box background
	bgClr := color.RGBA{10, 10, 30, 220}
	DrawRect(screen, db.BoxX, db.BoxY, db.BoxW, db.BoxH, bgClr)

	// Pulsing border
	DrawPulsingBorder(screen, db.BoxX, db.BoxY, db.BoxW, db.BoxH,
		float64(time.Since(db.startTime).Milliseconds())/1000.0,
		line.Color, 2)

	// Portrait
	portraitX := db.BoxX + db.Padding
	portraitY := db.BoxY + db.Padding
	if db.ShowPortrait {
		DrawCircle(screen, portraitX+db.PortraitSize/2, portraitY+db.PortraitSize/2,
			db.PortraitSize/2, line.PortraitClr)
		// Inner circle (face)
		DrawCircle(screen, portraitX+db.PortraitSize/2, portraitY+db.PortraitSize/2,
			db.PortraitSize/3, lightenColor(line.PortraitClr, 0.3))
	}

	// Speaker name
	nameX := portraitX + db.PortraitSize + db.Padding
	if !db.ShowPortrait {
		nameX = db.BoxX + db.Padding
	}
	DrawText(screen, line.Speaker, int(nameX), int(portraitY+5), line.Color)

	// Dialogue text
	textX := int(nameX)
	textY := int(portraitY + 25)
	displayText := db.typingBuffer
	if db.doneTyping {
		displayText = line.Text
	}
	// Word wrap
	wrapWidth := int(db.BoxW - db.Padding*2)
	charWidth := 8
	lineY := textY
	lineX := textX
	for _, ch := range displayText {
		if ch == '\n' || lineX-textX >= wrapWidth {
			lineY += 16
			lineX = textX
		}
		if ch == '\n' {
			continue
		}
		DrawText(screen, string(ch), lineX, lineY, ColWhite)
		lineX += charWidth
	}

	// Continue indicator (blinking triangle)
	if db.doneTyping && !db.ChoiceActive {
		blink := math.Sin(float64(time.Since(db.startTime).Milliseconds())/200.0) > 0
		if blink {
			DrawText(screen, "v", int(db.BoxX+db.BoxW-30), int(db.BoxY+db.BoxH-20), ColSunflower)
		}
	}

	// Choices
	if db.ChoiceActive {
		choiceY := db.BoxY + db.BoxH - float64(len(db.Choices))*25 - 10
		for i, choice := range db.Choices {
			marker := "  "
			clr := ColWhite
			if i == db.SelectedChoice {
				marker = "-> "
				clr = ColSunflower
			}
			DrawText(screen, marker+choice.Text, int(db.BoxX+db.Padding+10), int(choiceY)+i*25, clr)
		}
	}
}

// ─── Dialogue Manager ──────────────────────────────────────────────
// Manages a queue of dialogue events across the game.

type DialogueManager struct {
	Queue    []DialogueLine
	Box      *DialogueBox
	Active   bool
}

func NewDialogueManager() *DialogueManager {
	return &DialogueManager{
		Box: NewDialogueBox(),
	}
}

// Enqueue adds dialogue lines to the queue.
func (dm *DialogueManager) Enqueue(lines []DialogueLine) {
	dm.Queue = append(dm.Queue, lines...)
	if !dm.Active {
		dm.showNext()
	}
}

// showNext shows the next dialogue in the queue.
func (dm *DialogueManager) showNext() {
	if len(dm.Queue) == 0 {
		dm.Active = false
		return
	}
	dm.Active = true
	var batch []DialogueLine
	// Take all lines with the same speaker as a batch
	speaker := dm.Queue[0].Speaker
	for len(dm.Queue) > 0 && dm.Queue[0].Speaker == speaker {
		batch = append(batch, dm.Queue[0])
		dm.Queue = dm.Queue[1:]
	}
	dm.Box.SetLines(batch)
}

// Update updates the dialogue manager.
func (dm *DialogueManager) Update(dt float64) {
	if !dm.Active {
		return
	}
	dm.Box.Update(dt)
	if dm.Box.Finished {
		dm.showNext()
	}
}

// Advance advances the current dialogue.
func (dm *DialogueManager) Advance() bool {
	if !dm.Active {
		return false
	}
	return dm.Box.Advance()
}

// Draw renders the active dialogue.
func (dm *DialogueManager) Draw(screen *ebiten.Image) {
	if !dm.Active {
		return
	}
	dm.Box.Draw(screen)
}

// ─── Helper ────────────────────────────────────────────────────────

func lightenColor(c color.Color, factor float64) color.Color {
	cr := c.(color.RGBA)
	return color.RGBA{
		R: uint8(float64(cr.R) + (255-float64(cr.R))*factor),
		G: uint8(float64(cr.G) + (255-float64(cr.G))*factor),
		B: uint8(float64(cr.B) + (255-float64(cr.B))*factor),
		A: cr.A,
	}
}