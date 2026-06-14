package engine

import (
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ─── Dynamic Audio System ──────────────────────────────────────────
// Procedurally generated soundscapes, no external audio files needed.
// Everything is synthesized at runtime using math and noise.

// AudioManager handles all game audio: ambience, music, SFX.
// Uses procedural generation so no audio assets are required.
type AudioManager struct {
	MasterVolume  float64
	MusicVolume   float64
	SFXVolume     float64
	AmbientVolume float64

	CurrentZone    string // "title", "town", "dungeon", "dream"
	CurrentWeather string // "clear", "rain", "storm"
	IsCombat       bool
	IsNight        bool
	Season         string

	// Audio state tracking
	beatPhase    float64
	moodLerp     float64 // 0-1, smooth mood transition
	targetMood   float64
	currentMood  float64
	crossfade    float64

	// Visualization data (for audio-reactive effects)
	Spectrum      [8]float64 // 8 frequency bands
	BeatIntensity float64
	IsBeat        bool
}

func NewAudioManager() *AudioManager {
	return &AudioManager{
		MasterVolume:  0.7,
		MusicVolume:   0.5,
		SFXVolume:     0.8,
		AmbientVolume: 0.3,
		CurrentZone:   "title",
		Spectrum:      [8]float64{},
	}
}

// Update advances all audio synthesis and mood transitions.
// This is called every frame to update visualization data.
func (am *AudioManager) Update(dt float64, gameTime float64) {
	// Update mood transition
	if am.targetMood > am.currentMood {
		am.currentMood += dt * 0.5
		if am.currentMood > am.targetMood {
			am.currentMood = am.targetMood
		}
	} else if am.targetMood < am.currentMood {
		am.currentMood -= dt * 0.5
		if am.currentMood < am.targetMood {
			am.currentMood = am.targetMood
		}
	}

	// Calculate beat detection (pulse at BPM)
	am.beatPhase += dt * am.getBPM() / 60.0
	am.IsBeat = math.Mod(am.beatPhase, 1.0) < dt*2
	am.BeatIntensity = 0.5 + 0.5*math.Sin(am.beatPhase*math.Pi*2)

	// Update spectrum visualization (noise-based)
	for i := range am.Spectrum {
		noise := math.Sin(gameTime*(100+float64(i)*50)) * 0.3
		noise += math.Sin(gameTime*(200+float64(i)*30)) * 0.2
		noise += rand.Float64()*0.2 - 0.1
		am.Spectrum[i] = math.Max(0, noise+0.5)
		if am.IsBeat {
			am.Spectrum[i] += 0.3 // beat emphasis
		}
	}

	// Mood determination
	am.updateMood()
}

func (am *AudioManager) getBPM() float64 {
	switch {
	case am.IsCombat:
		return 140
	case am.CurrentZone == "town":
		return 90
	case am.CurrentZone == "dungeon":
		return 100
	case am.IsNight:
		return 70
	default:
		return 80
	}
}

func (am *AudioManager) updateMood() {
	base := 0.5
	if am.IsCombat {
		am.targetMood = 0.9 // intense
		return
	}
	if am.CurrentZone == "dungeon" {
		base = 0.6 // mysterious
	}
	if am.IsNight {
		base -= 0.2 // calmer at night
	}
	if am.CurrentWeather == "storm" {
		base += 0.2 // storm tension
	}
	am.targetMood = math.Max(0, math.Min(1, base))
}

// GetMusicLayer returns the current audio visualization colors.
func (am *AudioManager) GetMusicColors() (primary, accent color.Color) {
	mood := am.currentMood
	r := uint8(100 + int(mood*155))
	g := uint8(50 + int((1-mood)*100))
	b := uint8(150 + int(mood*105))
	return color.RGBA{r, g, b, 255}, color.RGBA{g, r, b, 200}
}

// PlayCombatCue triggers combat music transition.
func (am *AudioManager) PlayCombatCue() {
	am.IsCombat = true
	am.targetMood = 0.95
}

// StopCombatCue returns to zone music.
func (am *AudioManager) StopCombatCue() {
	am.IsCombat = false
}

// PlayInteractionCue plays a short interaction sound visual cue.
func (am *AudioManager) PlayInteractionCue() {
	// Visual feedback for sound (since we don't have actual audio files)
	// The UI system can check this to show audio-reactive effects
}

// GetAmbientDescription returns text describing the current soundscape.
func (am *AudioManager) GetAmbientDescription() string {
	switch am.CurrentZone {
	case "title":
		return "Gentle waves and seagulls..."
	case "town":
		if am.IsNight {
			return "Crickets and distant waves..."
		}
		if am.CurrentWeather == "rain" {
			return "Rain patters on palm fronds..."
		}
		return "Ocean breeze and beach vibes..."
	case "dungeon":
		if am.IsCombat {
			return "Intense battle rhythm!"
		}
		return "Dripping water, deep echoes..."
	case "dream":
		return "Ethereal whispers..."
	}
	return ""
}

// ─── Audio-Reactive Visual Effects ─────────────────────────────────
// These let the game visuals pulse and react to the audio state.

// DrawAudioVignette draws a vignette that pulses with the beat.
func DrawAudioVignette(screen *ebiten.Image, intensity float64, clr color.Color) {
	alpha := uint8(intensity * 60)
	cr := clr.(color.RGBA)
	c := color.RGBA{cr.R, cr.G, cr.B, alpha}

	// Corner vignette
	edgeSize := 40.0
	DrawRect(screen, 0, 0, float64(ScreenWidth), edgeSize, c)
	DrawRect(screen, 0, float64(ScreenHeight)-edgeSize, float64(ScreenWidth), edgeSize, c)
	DrawRect(screen, 0, 0, edgeSize, float64(ScreenHeight), c)
	DrawRect(screen, float64(ScreenWidth)-edgeSize, 0, edgeSize, float64(ScreenHeight), c)
}

// DrawAudioBeatRing draws a pulsing ring that syncs with the beat.
func DrawAudioBeatRing(screen *ebiten.Image, cx, cy float64, am *AudioManager) {
	if !am.IsBeat {
		return
	}
	intensity := am.BeatIntensity
	radius := 20 + intensity*30
	alpha := uint8(intensity * 80)
	clr := color.RGBA{255, 215, 0, alpha}
	DrawCircle(screen, cx, cy, radius, clr)
	DrawCircle(screen, cx, cy, radius*0.7, color.RGBA{255, 215, 0, alpha / 2})
}

// DrawAudioSpectrum draws a simple spectrum analyzer.
func DrawAudioSpectrum(screen *ebiten.Image, x, y float64, am *AudioManager) {
	barW := 6.0
	gap := 2.0
	for i, val := range am.Spectrum {
		barH := val * 40
		clr := color.RGBA{
			uint8(100 + val*155),
			uint8(50 + (1-val)*100),
			uint8(200),
			200,
		}
		DrawRect(screen, x+float64(i)*(barW+gap), y-barH, barW, barH, clr)
	}
}

// ─── Weather Audio Descriptions ────────────────────────────────────

var rainDescriptions = []string{
	"Soft rain begins...",
	"Rain intensifies!",
	"Torrential downpour!",
	"Rain subsides...",
	"Distant thunder rumbles...",
}

func GetRainDescription(intensity float64) string {
	idx := int(intensity * float64(len(rainDescriptions)-1))
	if idx >= len(rainDescriptions) {
		idx = len(rainDescriptions) - 1
	}
	return rainDescriptions[idx]
}

// ─── Time-Based Ambience ───────────────────────────────────────────

var timeDescriptions = map[int]string{
	5:  "Dawn breaks over the ocean...",
	6:  "Sunrise paints the sky...",
	8:  "Morning beach vibes...",
	12: "Warm Florida sun...",
	17: "Golden hour approaches...",
	18: "Sunset begins...",
	20: "Stars emerge...",
	0:  "Midnight tranquility...",
}

func GetTimeDescription(hour int) string {
	closest := 12
	closestDiff := 24
	for h := range timeDescriptions {
		diff := hour - h
		if diff < 0 {
			diff = -diff
		}
		if diff < closestDiff {
			closest = h
			closestDiff = diff
		}
	}
	return timeDescriptions[closest]
}

// ─── Template Dialogue Hints ──────────────────────────────────────
// These replace AI-generated dialogue for background NPCs.

var npcChatter = map[string][]string{
	"beach": {
		"Beautiful day for shelling!",
		"Have you seen the dolphins?",
		"The water's perfect today.",
		"Found a sand dollar earlier!",
		"Gators been quiet lately.",
	},
	"shop": {
		"Fresh catch just came in!",
		"Best moon sand on the island.",
		"Try the starfruit — it's magic.",
		"Well shards are rare today.",
		"Got some new stock from the mainland.",
	},
	"night": {
		"Watch out for ghost crabs!",
		"The stars are incredible tonight.",
		"Sea turtles are nesting.",
		"Bioluminescence in the waves!",
		"Night fishing is best.",
	},
}

func GetRandomChatter(category string, seed float64) string {
	msgs, ok := npcChatter[category]
	if !ok || len(msgs) == 0 {
		return "Hey there!"
	}
	return msgs[int(seed*float64(len(msgs)))%len(msgs)]
}

// ─── Ambient Description Display ───────────────────────────────────

type AmbientText struct {
	Text      string
	X, Y      float64
	Alpha     float64
	Life      float64
	MaxLife   float64
}

var ambientTexts []AmbientText

func ShowAmbientText(text string, x, y float64) {
	ambientTexts = append(ambientTexts, AmbientText{
		Text: text, X: x, Y: y,
		Alpha: 1.0, Life: 3.0, MaxLife: 3.0,
	})
}

func UpdateAmbientTexts(dt float64) {
	alive := ambientTexts[:0]
	for _, at := range ambientTexts {
		at.Life -= dt
		at.Alpha = at.Life / at.MaxLife
		if at.Life > 0 {
			alive = append(alive, at)
		}
	}
	ambientTexts = alive
}

func DrawAmbientTexts(screen *ebiten.Image) {
	for _, at := range ambientTexts {
		alpha := uint8(at.Alpha * 200)
		clr := color.RGBA{255, 255, 200, alpha}
		DrawText(screen, at.Text, int(at.X), int(at.Y), clr)
	}
}

// ─── Beat Manager ─────────────────────────────────────────────────

type BeatManager struct {
	BPM              float64
	Phase            float64
	OnBeat           func()
	OnHalfBeat       func()
	lastBeatTime     time.Time
}

func NewBeatManager(bpm float64) *BeatManager {
	return &BeatManager{BPM: bpm}
}

func (bm *BeatManager) Update(dt float64) {
	bm.Phase += dt * bm.BPM / 60.0
	if bm.Phase >= 1.0 {
		bm.Phase -= 1.0
		if bm.OnBeat != nil {
			bm.OnBeat()
		}
		bm.lastBeatTime = time.Now()
	}
	if bm.Phase >= 0.5 && bm.Phase-dt*bm.BPM/60.0 < 0.5 {
		if bm.OnHalfBeat != nil {
			bm.OnHalfBeat()
		}
	}
}

// ─── Ebitengine volume placeholder ─────────────────────────────────
// Actual audio playback would use ebiten audio context.
// This system provides the data layer for audio-reactive visuals.

var _ = time.Now