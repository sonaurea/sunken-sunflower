package engine

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// ─── Animation System ───────────────────────────────────────────────
// A flexible, composable animation system designed for:
//   1. Developers — clean Go API with builder pattern
//   2. Modders — JSON-definable animation clips
//   3. Everything — frame anims, property tweens, state machines

// ─── Frame Animation ────────────────────────────────────────────────

// FrameAnim cycles through sprite frames.
type FrameAnim struct {
	Frames        []int     // frame indices into a sprite sheet
	FrameDuration float64   // seconds per frame
	Loop          bool
	PingPong      bool      // play forward then backward
	current       int
	dir           int       // 1 forward, -1 backward (for ping-pong)
	timer         float64
}

// NewFrameAnim creates a frame animation. frames are indices, frameDuration in seconds.
func NewFrameAnim(frames []int, frameDuration float64, loop, pingPong bool) *FrameAnim {
	return &FrameAnim{
		Frames:        frames,
		FrameDuration: frameDuration,
		Loop:          loop,
		PingPong:      pingPong,
		dir:           1,
	}
}

func (fa *FrameAnim) Update(dt float64) {
	if len(fa.Frames) == 0 {
		return
	}
	fa.timer += dt
	for fa.timer >= fa.FrameDuration {
		fa.timer -= fa.FrameDuration
		fa.current += fa.dir

		if fa.current >= len(fa.Frames) {
			if fa.PingPong {
				fa.dir = -1
				fa.current = len(fa.Frames) - 2
				if fa.current < 0 {
					fa.current = 0
				}
			} else if fa.Loop {
				fa.current = 0
			} else {
				fa.current = len(fa.Frames) - 1
			}
		}
		if fa.current < 0 && fa.PingPong {
			fa.dir = 1
			fa.current = 1
			if fa.current >= len(fa.Frames) {
				fa.current = 0
			}
		}
	}
}

func (fa *FrameAnim) CurrentFrame() int {
	if len(fa.Frames) == 0 {
		return 0
	}
	return fa.Frames[fa.current]
}

func (fa *FrameAnim) IsDone() bool {
	return !fa.Loop && fa.current >= len(fa.Frames)-1
}

func (fa *FrameAnim) Reset() {
	fa.current = 0
	fa.dir = 1
	fa.timer = 0
}

// ─── Property Animation (Tween any float field) ─────────────────────

// PropAccessor gets/sets a named property on an animated object.
type PropAccessor interface {
	GetProp(name string) float64
	SetProp(name string, val float64)
}

// PropAnim tweens a single property from→to over duration with easing.
type PropAnim struct {
	Property string
	From, To float64
	Duration float64
	Elapsed  float64
	Easing   EasingFunc
	done     bool
	OnUpdate func(val float64)
	OnDone   func()
}

func NewPropAnim(property string, from, to, duration float64, easing EasingFunc) *PropAnim {
	return &PropAnim{
		Property: property,
		From:     from, To: to,
		Duration: duration,
		Easing:   easing,
	}
}

func (pa *PropAnim) Update(dt float64, target PropAccessor) {
	if pa.done {
		return
	}
	pa.Elapsed += dt
	t := pa.Elapsed / pa.Duration
	if t >= 1 {
		t = 1
		pa.done = true
		if target != nil {
			target.SetProp(pa.Property, pa.To)
		}
		if pa.OnUpdate != nil {
			pa.OnUpdate(pa.To)
		}
		if pa.OnDone != nil {
			pa.OnDone()
		}
		return
	}
	val := pa.From + (pa.To-pa.From)*pa.Easing(t)
	if target != nil {
		target.SetProp(pa.Property, val)
	}
	if pa.OnUpdate != nil {
		pa.OnUpdate(val)
	}
}

func (pa *PropAnim) Reset() {
	pa.Elapsed = 0
	pa.done = false
}

// ─── Animation Clip (composite) ─────────────────────────────────────

// AnimEvent fires a callback at a specific time in the clip.
type AnimEvent struct {
	Time     float64 // normalized 0.0–1.0
	Callback func()
	fired    bool
}

// AnimationClip combines frame and property animations with events.
type AnimationClip struct {
	Name       string
	FrameAnim  *FrameAnim
	PropAnims  []*PropAnim
	Events     []AnimEvent
	Duration   float64 // total duration (overrides individual if set)
	Loop       bool
	NextClip   string // auto-transition to this clip when done
	target     PropAccessor
	elapsed    float64
	done       bool
	mu         sync.Mutex
}

func NewAnimationClip(name string) *AnimationClip {
	return &AnimationClip{
		Name:     name,
		PropAnims: []*PropAnim{},
		Events:   []AnimEvent{},
	}
}

// SetTarget sets the property accessor for this clip.
func (ac *AnimationClip) SetTarget(target PropAccessor) {
	ac.target = target
}

// Update advances the clip by dt seconds.
func (ac *AnimationClip) Update(dt float64) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.done {
		return
	}

	ac.elapsed += dt

	// Calculate normalized progress
	clipDur := ac.Duration
	if clipDur == 0 {
		// Derive from longest prop anim
		clipDur = 1.0
		for _, pa := range ac.PropAnims {
			if pa.Duration > clipDur {
				clipDur = pa.Duration
			}
		}
	}
	progress := ac.elapsed / clipDur

	// Update frame animation
	if ac.FrameAnim != nil {
		ac.FrameAnim.Update(dt)
	}

	// Update property animations
	for _, pa := range ac.PropAnims {
		pa.Update(dt, ac.target)
	}

	// Fire events
	for i := range ac.Events {
		e := &ac.Events[i]
		if !e.fired && progress >= e.Time {
			e.fired = true
			if e.Callback != nil {
				e.Callback()
			}
		}
	}

	// Check completion
	if progress >= 1 {
		if ac.Loop {
			ac.elapsed = 0
			ac.ResetAnims()
		} else {
			ac.done = true
		}
	}
}

func (ac *AnimationClip) ResetAnims() {
	if ac.FrameAnim != nil {
		ac.FrameAnim.Reset()
	}
	for _, pa := range ac.PropAnims {
		pa.Reset()
	}
	for i := range ac.Events {
		ac.Events[i].fired = false
	}
}

func (ac *AnimationClip) Reset() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.elapsed = 0
	ac.done = false
	ac.ResetAnims()
}

func (ac *AnimationClip) IsDone() bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	return ac.done
}

// AddProp adds a property animation to this clip.
func (ac *AnimationClip) AddProp(property string, from, to, duration float64, easing EasingFunc) *AnimationClip {
	ac.PropAnims = append(ac.PropAnims, NewPropAnim(property, from, to, duration, easing))
	return ac
}

// AddEvent adds a time-triggered callback.
func (ac *AnimationClip) AddEvent(timeNorm float64, cb func()) *AnimationClip {
	ac.Events = append(ac.Events, AnimEvent{Time: timeNorm, Callback: cb})
	return ac
}

// ─── Animator (State Machine) ───────────────────────────────────────
// Animator manages animation clips as states with transitions.

type Animator struct {
	clips     map[string]*AnimationClip
	current   string
	previous  string
	timer     float64
	Speed     float64 // playback speed multiplier (1.0 = normal)
	OnStateChange func(from, to string)
	mu        sync.RWMutex
}

func NewAnimator() *Animator {
	return &Animator{
		clips: make(map[string]*AnimationClip),
		Speed: 1.0,
	}
}

// AddClip registers an animation clip.
func (a *Animator) AddClip(clip *AnimationClip) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.clips[clip.Name] = clip
}

// Play starts playing a clip by name, optionally resetting it.
func (a *Animator) Play(name string, reset bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.clips[name]; !ok {
		return
	}
	if name == a.current && !reset {
		return
	}

	a.previous = a.current
	a.current = name
	if reset {
		a.clips[name].Reset()
	}
	if a.OnStateChange != nil {
		a.OnStateChange(a.previous, name)
	}
}

// Update advances the current clip.
func (a *Animator) Update(dt float64) {
	a.mu.RLock()
	current := a.current
	speed := a.Speed
	a.mu.RUnlock()

	if current == "" {
		return
	}

	a.mu.RLock()
	clip, ok := a.clips[current]
	a.mu.RUnlock()

	if !ok {
		return
	}

	clip.Update(dt * speed)

	// Auto-transition to next clip
	if clip.IsDone() && clip.NextClip != "" {
		a.Play(clip.NextClip, true)
	}
}

// CurrentClip returns the name of the currently playing clip.
func (a *Animator) CurrentClip() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.current
}

// Clip returns a registered clip by name.
func (a *Animator) Clip(name string) *AnimationClip {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.clips[name]
}

// ─── AnimatedEntity ─────────────────────────────────────────────────
// Embed this in any entity to give it full animation capabilities.

type AnimatedEntity struct {
	Animator  *Animator
	Sprite    *ebiten.Image
	FrameW    int // width of one frame in sprite sheet
	FrameH    int // height of one frame
	Frames    []*ebiten.Image // cached frame sub-images
	FlipX     bool
	FlipY     bool
	Alpha     float64
	Scale     float64
	Rotation  float64
	ColorOver color.Color
}

func NewAnimatedEntity() *AnimatedEntity {
	return &AnimatedEntity{
		Animator: NewAnimator(),
		Alpha:    1.0,
		Scale:    1.0,
		ColorOver: color.RGBA{255, 255, 255, 255},
	}
}

// SetSpriteSheet sets a sprite sheet and splits it into frames.
func (ae *AnimatedEntity) SetSpriteSheet(sheet *ebiten.Image, frameW, frameH int) {
	ae.Sprite = sheet
	ae.FrameW = frameW
	ae.FrameH = frameH
	ae.Frames = nil

	if sheet == nil {
		return
	}

	sw := sheet.Bounds().Dx()
	sh := sheet.Bounds().Dy()
	cols := sw / frameW
	rows := sh / frameH

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			frame := sheet.SubImage(
				imageRect(x*frameW, y*frameH, (x+1)*frameW, (y+1)*frameH),
			).(*ebiten.Image)
			ae.Frames = append(ae.Frames, frame)
		}
	}
}

// CurrentFrameImage returns the current frame from the active animation.
func (ae *AnimatedEntity) CurrentFrameImage() *ebiten.Image {
	clip := ae.Animator.clips[ae.Animator.current]
	if clip == nil || clip.FrameAnim == nil {
		if len(ae.Frames) > 0 {
			return ae.Frames[0]
		}
		return nil
	}
	idx := clip.FrameAnim.CurrentFrame()
	if idx >= 0 && idx < len(ae.Frames) {
		return ae.Frames[idx]
	}
	if len(ae.Frames) > 0 {
		return ae.Frames[0]
	}
	return nil
}

func (ae *AnimatedEntity) Update(dt float64) {
	ae.Animator.Update(dt)
}

// DrawOpts returns the DrawImageOptions with current transform applied.
func (ae *AnimatedEntity) DrawOpts(x, y float64) *ebiten.DrawImageOptions {
	op := &ebiten.DrawImageOptions{}
	// Center of frame
	fw := float64(ae.FrameW)
	fh := float64(ae.FrameH)
	op.GeoM.Translate(-fw/2, -fh/2) // rotate around center
	if ae.FlipX {
		op.GeoM.Scale(-1, 1)
	}
	if ae.FlipY {
		op.GeoM.Scale(1, -1)
	}
	op.GeoM.Scale(ae.Scale, ae.Scale)
	op.GeoM.Rotate(ae.Rotation)
	op.GeoM.Translate(x, y)
	// Alpha
	op.ColorScale.Scale(1, 1, 1, float32(ae.Alpha))
	return op
}

func (ae *AnimatedEntity) Draw(screen *ebiten.Image, x, y float64) {
	img := ae.CurrentFrameImage()
	if img == nil {
		return
	}
	op := ae.DrawOpts(x, y)
	screen.DrawImage(img, op)
}

// ─── Procedural Sprite Sheet Generator ──────────────────────────────
// Generates animation frames procedurally — no asset files needed.

// GenerateIdleFrames creates a 4-frame idle animation (gentle bob).
func GenerateIdleFrames(size int, baseClr color.Color) []*ebiten.Image {
	frames := make([]*ebiten.Image, 4)
	for i := 0; i < 4; i++ {
		frames[i] = ebiten.NewImage(size, size)
		offset := math.Sin(float64(i)*math.Pi/2) * 1.5
		center := float64(size) / 2
		// Base body
		DrawCircle(frames[i], center, center+offset, float64(size)*0.4, baseClr)
		// Eyes
		eyeR := float64(size) * 0.08
		DrawCircle(frames[i], center-float64(size)*0.15, center-float64(size)*0.1+offset, eyeR, ColWhite)
		DrawCircle(frames[i], center+float64(size)*0.15, center-float64(size)*0.1+offset, eyeR, ColWhite)
		DrawCircle(frames[i], center-float64(size)*0.15, center-float64(size)*0.1+offset, eyeR*0.5, ColBlack)
		DrawCircle(frames[i], center+float64(size)*0.15, center-float64(size)*0.1+offset, eyeR*0.5, ColBlack)
	}
	return frames
}

// GenerateWalkFrames creates a 6-frame walk cycle.
func GenerateWalkFrames(size int, baseClr color.Color) []*ebiten.Image {
	frames := make([]*ebiten.Image, 6)
	for i := 0; i < 6; i++ {
		frames[i] = ebiten.NewImage(size, size)
		phase := float64(i) / 6.0 * math.Pi * 2
		bob := math.Sin(phase) * 2
		squash := 1.0 + math.Sin(phase)*0.1
		center := float64(size) / 2
		// Squash and stretch the body
		r := float64(size) * 0.4 * squash
		DrawCircle(frames[i], center, center+bob, r, baseClr)
		// Legs (simple lines at bottom)
		legOffset := math.Sin(phase) * float64(size) * 0.15
		DrawLine(frames[i], center-5, center+float64(size)*0.3+bob, center-5-legOffset, center+float64(size)*0.45, ColBlack)
		DrawLine(frames[i], center+5, center+float64(size)*0.3+bob, center+5+legOffset, center+float64(size)*0.45, ColBlack)
		// Eyes
		eyeR := float64(size) * 0.07
		DrawCircle(frames[i], center-float64(size)*0.12, center-float64(size)*0.08+bob, eyeR, ColWhite)
		DrawCircle(frames[i], center+float64(size)*0.12, center-float64(size)*0.08+bob, eyeR, ColWhite)
		DrawCircle(frames[i], center-float64(size)*0.12, center-float64(size)*0.08+bob, eyeR*0.5, ColBlack)
		DrawCircle(frames[i], center+float64(size)*0.12, center-float64(size)*0.08+bob, eyeR*0.5, ColBlack)
	}
	return frames
}

// GenerateAttackFrames creates a 4-frame strike animation.
func GenerateAttackFrames(size int, baseClr color.Color) []*ebiten.Image {
	frames := make([]*ebiten.Image, 4)
	for i := 0; i < 4; i++ {
		frames[i] = ebiten.NewImage(size*2, size) // wider for attack extension
		phase := float64(i) / 4.0
		centerX := float64(size) / 2
		centerY := float64(size) / 2
		// Body
		DrawCircle(frames[i], centerX, centerY, float64(size)*0.35, baseClr)
		// Attack arm extends
		armLen := phase * float64(size) * 0.8
		armClr := baseClr
		if i == 3 {
			armClr = ColSunflower // flash on hit
		}
		DrawRect(frames[i], centerX+float64(size)*0.2, centerY-3, armLen, 6, armClr)
		// Eyes (angry)
		eyeR := float64(size) * 0.07
		DrawCircle(frames[i], centerX-8, centerY-6, eyeR, ColWhite)
		DrawCircle(frames[i], centerX+8, centerY-6, eyeR, ColWhite)
		DrawCircle(frames[i], centerX-8, centerY-6, eyeR*0.5, ColRed)
		DrawCircle(frames[i], centerX+8, centerY-6, eyeR*0.5, ColRed)
	}
	return frames
}

// GenerateHurtFrames creates a 2-frame hurt/flinch animation.
func GenerateHurtFrames(size int, baseClr color.Color) []*ebiten.Image {
	frames := make([]*ebiten.Image, 2)
	for i := 0; i < 2; i++ {
		frames[i] = ebiten.NewImage(size, size)
		center := float64(size) / 2
		shake := float64(i) * 3
		clr := baseClr
		if i == 1 {
			clr = ColRed // flash red
		}
		DrawCircle(frames[i], center+shake, center, float64(size)*0.4, clr)
		// X eyes
		DrawLine(frames[i], center-10+shake, center-8, center-4+shake, center-2, ColBlack)
		DrawLine(frames[i], center-4+shake, center-8, center-10+shake, center-2, ColBlack)
		DrawLine(frames[i], center+4+shake, center-8, center+10+shake, center-2, ColBlack)
		DrawLine(frames[i], center+10+shake, center-8, center+4+shake, center-2, ColBlack)
	}
	return frames
}

// ─── AnimationBuilder (Fluent API for devs) ─────────────────────────

type AnimationBuilder struct {
	clip *AnimationClip
}

// Begin creates a new animation builder for the named clip.
func BeginAnim(name string) *AnimationBuilder {
	return &AnimationBuilder{clip: NewAnimationClip(name)}
}

// Frames sets the frame animation.
func (ab *AnimationBuilder) Frames(indices []int, frameDuration float64, loop, pingPong bool) *AnimationBuilder {
	ab.clip.FrameAnim = NewFrameAnim(indices, frameDuration, loop, pingPong)
	return ab
}

// Prop adds a property tween.
func (ab *AnimationBuilder) Prop(property string, from, to, duration float64, easing EasingFunc) *AnimationBuilder {
	ab.clip.AddProp(property, from, to, duration, easing)
	return ab
}

// Event adds a timed callback.
func (ab *AnimationBuilder) Event(timeNorm float64, cb func()) *AnimationBuilder {
	ab.clip.AddEvent(timeNorm, cb)
	return ab
}

// Duration sets the total clip duration.
func (ab *AnimationBuilder) Duration(d float64) *AnimationBuilder {
	ab.clip.Duration = d
	return ab
}

// Loop sets whether the clip loops.
func (ab *AnimationBuilder) Loop(l bool) *AnimationBuilder {
	ab.clip.Loop = l
	return ab
}

// NextClip sets the auto-transition clip.
func (ab *AnimationBuilder) NextClip(name string) *AnimationBuilder {
	ab.clip.NextClip = name
	return ab
}

// Build returns the completed AnimationClip.
func (ab *AnimationBuilder) Build() *AnimationClip {
	return ab.clip
}

// ─── JSON Animation Definition ──────────────────────────────────────
// For mods: define animations in JSON, loaded at runtime.

type JSONAnimation struct {
	Name        string         `json:"name"`
	FrameCount  int            `json:"frame_count"`
	FrameWidth  int            `json:"frame_width"`
	FrameHeight int            `json:"frame_height"`
	FrameIndices []int         `json:"frame_indices"`
	FrameRate   float64        `json:"frame_rate"`
	Loop        bool           `json:"loop"`
	PingPong    bool           `json:"ping_pong"`
	Props       []JSONPropAnim `json:"props"`
	NextClip    string         `json:"next_clip"`
}

type JSONPropAnim struct {
	Property string  `json:"property"`
	From     float64 `json:"from"`
	To       float64 `json:"to"`
	Duration float64 `json:"duration"`
	Easing   string  `json:"easing"` // "bounce", "quad", "elastic", "back"
}

func easingFromName(name string) EasingFunc {
	switch name {
	case "bounce":
		return EaseOutBounce
	case "quad":
		return EaseInOutQuad
	case "elastic":
		return EaseOutElastic
	case "back":
		return EaseInBack
	default:
		return EaseInOutQuad
	}
}

// LoadAnimationFromJSON parses a JSON animation definition.
func LoadAnimationFromJSON(data []byte) (*AnimationClip, error) {
	var j JSONAnimation
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, fmt.Errorf("animation JSON parse error: %w", err)
	}

	clip := NewAnimationClip(j.Name)
	if len(j.FrameIndices) > 0 {
		clip.FrameAnim = NewFrameAnim(j.FrameIndices, 1.0/j.FrameRate, j.Loop, j.PingPong)
	}
	clip.NextClip = j.NextClip
	for _, p := range j.Props {
		clip.AddProp(p.Property, p.From, p.To, p.Duration, easingFromName(p.Easing))
	}
	return clip, nil
}

// ─── Built-in Animation Presets ─────────────────────────────────────

// PresetIdle creates a gentle bobbing idle animation for any entity.
func PresetIdle() *AnimationClip {
	return BeginAnim("idle").
		Frames([]int{0, 1, 2, 1}, 0.3, true, false).
		Prop("bob", 0, 2, 0.6, EaseInOutQuad).
		Prop("bob", 2, 0, 0.6, EaseInOutQuad).
		Duration(1.2).
		Loop(true).
		Build()
}

// PresetWalk creates a walk cycle animation.
func PresetWalk() *AnimationClip {
	return BeginAnim("walk").
		Frames([]int{0, 1, 2, 3, 4, 5}, 0.1, true, false).
		Prop("bob", 0, 3, 0.3, EaseInOutQuad).
		Prop("bob", 3, 0, 0.3, EaseInOutQuad).
		Duration(0.6).
		Loop(true).
		Build()
}

// PresetAttack creates a strike-and-recover animation.
func PresetAttack() *AnimationClip {
	return BeginAnim("attack").
		Frames([]int{0, 1, 2, 3}, 0.06, false, false).
		Prop("scale_x", 1.0, 1.3, 0.12, EaseOutBounce).
		Prop("scale_x", 1.3, 1.0, 0.12, EaseOutBounce).
		Prop("rotation", 0, 0.3, 0.12, EaseOutBounce).
		Prop("rotation", 0.3, 0, 0.12, EaseOutBounce).
		Duration(0.24).
		NextClip("idle").
		Build()
}

// PresetHurt creates a flinch/stagger animation.
func PresetHurt() *AnimationClip {
	return BeginAnim("hurt").
		Frames([]int{0, 1}, 0.1, false, false).
		Prop("alpha", 1.0, 0.5, 0.1, EaseInOutQuad).
		Prop("alpha", 0.5, 1.0, 0.3, EaseInOutQuad).
		Prop("rotation", 0, 0.2, 0.1, EaseOutElastic).
		Prop("rotation", 0.2, 0, 0.3, EaseOutElastic).
		Duration(0.4).
		NextClip("idle").
		Build()
}

// PresetDeath creates a fade-out death animation.
func PresetDeath() *AnimationClip {
	return BeginAnim("death").
		Frames([]int{0, 1, 0, 1}, 0.15, false, false).
		Prop("scale", 1.0, 0.1, 0.8, EaseInBack).
		Prop("alpha", 1.0, 0.0, 0.8, EaseInOutQuad).
		Prop("rotation", 0, 1.5, 0.8, EaseInBack).
		Duration(0.8).
		Build()
}

// ─── Helper: image.Rectangle without importing image ────────────────

func imageRect(x0, y0, x1, y1 int) image.Rectangle {
	return image.Rectangle{Min: image.Point{X: x0, Y: y0}, Max: image.Point{X: x1, Y: y1}}
}
