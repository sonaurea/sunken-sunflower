package engine

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// ─── Procedural Audio Synthesizer ──────────────────────────────────
// Generates all game audio in real-time using math synthesis.
// No external audio files required — full dynamic control.
//
// Architecture:
//   - AudioContext: Ebitengine audio context
//   - Synth: Generates PCM samples for different instruments
//   - Mixer: Blends multiple sound layers
//   - Sequencer: Triggers notes/events based on game state

const (
	SampleRate    = 44100
	ChannelCount  = 2 // stereo
	BitDepth      = 16
	BaseFrequency = 440.0 // A4
)

// SynthEngine is the master audio synthesizer.
type SynthEngine struct {
	ctx          *audio.Context
	mu           sync.Mutex
	MasterVolume float64

	// Active sound layers
	ambient      *SynthLayer
	music        *SynthLayer
	sfx          *SynthLayer
	noise        *SynthLayer

	// Dynamic parameters (modified by game state)
	BPM          float64
	Key          string   // musical key: "C", "Am", etc.
	Mood         float64  // 0=calm, 1=intense
	BassIntensity float64
	MelodyActive bool

	// Active notes
	notes     []ActiveNote
	time      float64
}

type SynthLayer struct {
	Volume     float64
	Pan        float64 // -1 left, 0 center, 1 right
	Active     bool
	Waveform   WaveType
	FilterCutoff float64 // 0-1
	baseFreq   float64
}

type WaveType int

const (
	WaveSine     WaveType = iota
	WaveSquare
	WaveSawtooth
	WaveTriangle
	WaveNoise
	WaveOrgan
)

type ActiveNote struct {
	Frequency float64
	Velocity  float64 // 0-1
	StartTime float64
	Duration  float64
	Layer     int // 0=ambient, 1=music, 2=sfx
	Release   bool
}

// ─── Musical Scales ────────────────────────────────────────────────

var scales = map[string][]float64{
	"C_major": {261.63, 293.66, 329.63, 349.23, 392.00, 440.00, 493.88, 523.25},
	"A_minor": {220.00, 246.94, 261.63, 293.66, 329.63, 349.23, 392.00, 440.00},
	"G_major": {196.00, 220.00, 246.94, 261.63, 293.66, 329.63, 369.99, 392.00},
	"E_minor": {164.81, 185.00, 196.00, 220.00, 246.94, 261.63, 293.66, 329.63},
	"F_major": {174.61, 196.00, 220.00, 233.08, 261.63, 293.66, 311.13, 349.23},
}

// NewSynthEngine creates a new audio synthesizer.
func NewSynthEngine() *SynthEngine {
	ctx := audio.NewContext(SampleRate)
	se := &SynthEngine{
		ctx:          ctx,
		MasterVolume: 0.5,
		BPM:          90,
		Key:          "C_major",
		ambient:      &SynthLayer{Volume: 0.3, Active: true, Waveform: WaveSine, FilterCutoff: 0.3},
		music:        &SynthLayer{Volume: 0.4, Active: true, Waveform: WaveOrgan, FilterCutoff: 0.8},
		sfx:          &SynthLayer{Volume: 0.6, Active: true, Waveform: WaveSquare, FilterCutoff: 1.0},
		noise:        &SynthLayer{Volume: 0.1, Active: true, Waveform: WaveNoise, FilterCutoff: 0.5},
	}
	return se
}

// Update advances the synthesizer and generates audio for this frame.
// Call this every game tick with the delta time.
func (se *SynthEngine) Update(dt float64) {
	se.mu.Lock()
	defer se.mu.Unlock()

	se.time += dt

	// Ambient pad — always playing soft chords
	if se.ambient.Active && int(se.time*2)%4 == 0 {
		scale := scales[se.Key]
		note := scale[rand.Intn(len(scale)-2)+2] // mid range
		se.playNote(note, se.ambient.Volume*0.3, 2.0, 0)
	}

	// Music — more active when mood is high
	if se.music.Active && se.Mood > 0.3 {
		if int(se.time*se.BPM/60)%2 == 0 && rand.Float64() < 0.3 {
			scale := scales[se.Key]
			octave := 1
			if se.Mood > 0.7 {
				octave = 2
			}
			idx := rand.Intn(len(scale))
			freq := scale[idx] * float64(octave)
			vel := se.Mood * se.music.Volume
			dur := 0.5 + se.Mood*0.5
			se.playNote(freq, vel, dur, 1)
		}
	}

	// Bass — heartbeat at night or combat
	if se.BassIntensity > 0 && int(se.time*2)%2 == 0 {
		scale := scales[se.Key]
		freq := scale[0] * 0.5 // bass octave
		se.playNote(freq, se.BassIntensity*0.5, 0.5, 1)
	}

	// Clean up finished notes
	var alive []ActiveNote
	for _, n := range se.notes {
		if se.time-n.StartTime < n.Duration {
			alive = append(alive, n)
		}
	}
	se.notes = alive
}

func (se *SynthEngine) playNote(freq, vel, dur float64, layer int) {
	se.notes = append(se.notes, ActiveNote{
		Frequency: freq,
		Velocity:  vel,
		StartTime: se.time,
		Duration:  dur,
		Layer:     layer,
	})
}

// GenerateSamples fills a buffer with synthesized audio.
// This is called by Ebitengine's audio system.
func (se *SynthEngine) GenerateSamples(out []int16) {
	se.mu.Lock()
	defer se.mu.Unlock()

	for i := range out {
		t := float64(i) / SampleRate
		var sample float64

		// Mix all active notes
		for _, note := range se.notes {
			localT := se.time + t - note.StartTime
			if localT < 0 || localT > note.Duration {
				continue
			}
			env := envelope(localT, note.Duration)
			wave := generateWave(note.Frequency, t, getWaveformForLayer(note.Layer, se))
			sample += wave * note.Velocity * env
		}

		// Noise layer (ambient texture)
		if se.noise.Active {
			noise := (rand.Float64()*2 - 1) * se.noise.Volume * 0.1
			sample += noise
		}

		// Clamp and convert to int16
		sample *= se.MasterVolume
		if sample > 1.0 {
			sample = 1.0
		}
		if sample < -1.0 {
			sample = -1.0
		}
		out[i] = int16(sample * 32767)
	}
}

func getWaveformForLayer(layer int, se *SynthEngine) WaveType {
	switch layer {
	case 0:
		return se.ambient.Waveform
	case 1:
		return se.music.Waveform
	case 2:
		return se.sfx.Waveform
	default:
		return WaveSine
	}
}

// ─── Wave Generators ───────────────────────────────────────────────

func generateWave(freq, t float64, wav WaveType) float64 {
	phase := t * freq * 2 * math.Pi
	switch wav {
	case WaveSine:
		return math.Sin(phase)
	case WaveSquare:
		if math.Sin(phase) > 0 {
			return 1.0
		}
		return -1.0
	case WaveSawtooth:
		return 2.0*math.Mod(t*freq, 1.0) - 1.0
	case WaveTriangle:
		return 2.0*math.Abs(2.0*math.Mod(t*freq, 1.0)-1.0) - 1.0
	case WaveOrgan:
		// Rich organ: fundamental + harmonics
		return math.Sin(phase)*0.5 +
			math.Sin(phase*2)*0.25 +
			math.Sin(phase*3)*0.125 +
			math.Sin(phase*4)*0.0625
	case WaveNoise:
		return (rand.Float64()*2 - 1) * 0.3
	}
	return 0
}

// ─── ADSR Envelope ─────────────────────────────────────────────────

func envelope(t, duration float64) float64 {
	attack := 0.05
	release := 0.2
	totalLife := t / duration

	if totalLife < attack/duration {
		return totalLife / (attack / duration)
	}
	if totalLife < 1.0-release/duration {
		return 1.0
	}
	return 1.0 - (totalLife-(1.0-release/duration))/(release/duration)
}

// ─── SFX Generators ────────────────────────────────────────────────

// PlayHitSFX generates a short impact sound.
func (se *SynthEngine) PlayHitSFX() {
	se.mu.Lock()
	defer se.mu.Unlock()
	for i := 0; i < 3; i++ {
		freq := 200 + rand.Float64()*400
		se.playNote(freq, se.sfx.Volume*0.5, 0.1+rand.Float64()*0.1, 2)
	}
}

// PlayPickupSFX generates a pleasant pickup sound.
func (se *SynthEngine) PlayPickupSFX() {
	se.mu.Lock()
	defer se.mu.Unlock()
	scale := scales["C_major"]
	for i, freq := range []float64{scale[0], scale[2], scale[4]} {
		se.playNote(freq, se.sfx.Volume*0.6, 0.15+float64(i)*0.1, 2)
	}
}

// PlayStepSFX generates a footstep sound.
func (se *SynthEngine) PlayStepSFX() {
	se.mu.Lock()
	defer se.mu.Unlock()
	freq := 50 + rand.Float64()*30
	se.playNote(freq, se.sfx.Volume*0.2, 0.05, 2)
}

// PlayUISFX generates a UI interaction sound.
func (se *SynthEngine) PlayUISFX() {
	se.mu.Lock()
	defer se.mu.Unlock()
	scale := scales["C_major"]
	se.playNote(scale[4], se.sfx.Volume*0.4, 0.1, 2)
}

// PlayAlertSFX generates a warning sound.
func (se *SynthEngine) PlayAlertSFX() {
	se.mu.Lock()
	defer se.mu.Unlock()
	for i := 0; i < 3; i++ {
		freq := 800 + float64(i)*100
		se.playNote(freq, se.sfx.Volume*0.7, 0.15, 2)
	}
}

// ─── Zone Transitions ──────────────────────────────────────────────

// SetZone changes the musical key and mood for a game zone.
func (se *SynthEngine) SetZone(zone string) {
	se.mu.Lock()
	defer se.mu.Unlock()

	switch zone {
	case "title":
		se.Key = "C_major"
		se.BPM = 80
		se.Mood = 0.3
		se.BassIntensity = 0.2
	case "town":
		se.Key = "G_major"
		se.BPM = 90
		se.Mood = 0.5
		se.BassIntensity = 0.3
	case "dungeon":
		se.Key = "A_minor"
		se.BPM = 100
		se.Mood = 0.6
		se.BassIntensity = 0.5
	case "dream":
		se.Key = "E_minor"
		se.BPM = 70
		se.Mood = 0.2
		se.BassIntensity = 0.1
	}
}

// SetCombatMode adjusts the synthesizer for combat intensity.
func (se *SynthEngine) SetCombatMode(active bool) {
	se.mu.Lock()
	defer se.mu.Unlock()

	if active {
		se.BPM = 140
		se.Mood = 0.9
		se.BassIntensity = 0.8
		se.music.Waveform = WaveSawtooth
	} else {
		se.BPM = 100
		se.Mood = 0.5
		se.BassIntensity = 0.3
		se.music.Waveform = WaveOrgan
	}
}

// AudioContext returns the Ebitengine audio context for playback.
func (se *SynthEngine) AudioContext() *audio.Context {
	return se.ctx
}

// ─── Convenience functions ─────────────────────────────────────────

var _ = time.Now
var _ = audio.Context{}