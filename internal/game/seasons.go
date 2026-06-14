package game

import (
	"fmt"
	"math/rand"
	"time"
)

// ─── Florida Seasons & Day/Night Cycle ────────────────────────────
// "Honeymoon Island, Florida — where the sun meets the sea."
//
// Seasons affect: creature spawns, plant growth, dungeon mood, loot tables.
// All players must sleep to advance to the next day.

type Season int

const (
	SeasonDry     Season = iota // Oct–May: snowbird season, clear skies
	SeasonWet                   // Jun–Sep: afternoon thunderstorms
	SeasonHurricane             // Aug–Oct: storm season (special events)
	SeasonTurtleNest            // Mar–Oct: sea turtle nesting (rare finds)
)

func (s Season) String() string {
	switch s {
	case SeasonDry:
		return "☀ Dry Season"
	case SeasonWet:
		return "🌧 Wet Season"
	case SeasonHurricane:
		return "🌀 Hurricane Season"
	case SeasonTurtleNest:
		return "🐢 Turtle Nesting"
	}
	return "Unknown"
}

// SeasonData holds the current season state.
type SeasonData struct {
	Current     Season  `json:"season"`
	Day         int     `json:"day"`
	Hour        int     `json:"hour"`
	Minute      int     `json:"minute"`
	IsNight     bool    `json:"night"`
	Temperature float64 `json:"temp"` // 60-100°F (Florida range)
	Humidity    float64 `json:"humidity"`
	Weather     string  `json:"weather"` // "clear", "cloudy", "storm", "rainbow"
	DaysInSeason int    `json:"days"`
	SeasonDay   int     `json:"season_day"` // day within current season
}

func NewSeasonData() *SeasonData {
	sd := &SeasonData{
		Day:          1,
		Hour:         8, // start at 8 AM
		Minute:       0,
		DaysInSeason: 28,
		Current:      SeasonDry,
		Temperature:  75,
		Humidity:     60,
		Weather:      "clear",
	}
	sd.updateWeather()
	return sd
}

// AdvanceTime advances the game time by minutes.
func (sd *SeasonData) AdvanceTime(minutes int) {
	sd.Minute += minutes
	for sd.Minute >= 60 {
		sd.Minute -= 60
		sd.Hour++
		if sd.Hour >= 24 {
			sd.Hour = 0
			sd.Day++
			sd.SeasonDay++
			if sd.SeasonDay >= sd.DaysInSeason {
				sd.advanceSeason()
			}
		}
	}
	sd.IsNight = sd.Hour < 6 || sd.Hour >= 20
	sd.updateTemperature()
	sd.updateWeather()
}

func (sd *SeasonData) advanceSeason() {
	sd.SeasonDay = 0
	switch sd.Current {
	case SeasonDry:
		sd.Current = SeasonWet
	case SeasonWet:
		if rand.Float64() < 0.4 {
			sd.Current = SeasonHurricane
		} else {
			sd.Current = SeasonTurtleNest
		}
	case SeasonHurricane:
		sd.Current = SeasonTurtleNest
	case SeasonTurtleNest:
		sd.Current = SeasonDry
	}
}

func (sd *SeasonData) updateTemperature() {
	base := 72.0
	switch sd.Current {
	case SeasonDry:
		base = 70
	case SeasonWet:
		base = 85
	case SeasonHurricane:
		base = 88
	case SeasonTurtleNest:
		base = 82
	}
	// Day/night variation
	if sd.IsNight {
		base -= 10
	}
	// Hour variation (hottest at 2PM)
	hourDiff := float64(sd.Hour - 14)
	base -= hourDiff * hourDiff * 0.05
	// Random Florida variance
	sd.Temperature = base + (rand.Float64()*2 - 1) * 5
}

func (sd *SeasonData) updateWeather() {
	switch sd.Current {
	case SeasonDry:
		if rand.Float64() < 0.1 {
			sd.Weather = "cloudy"
		} else {
			sd.Weather = "clear"
		}
		sd.Humidity = 45 + rand.Float64()*20
	case SeasonWet:
		if sd.Hour >= 14 && sd.Hour <= 17 && rand.Float64() < 0.7 {
			sd.Weather = "storm"
		} else if rand.Float64() < 0.2 {
			sd.Weather = "rainbow"
		} else {
			sd.Weather = "cloudy"
		}
		sd.Humidity = 75 + rand.Float64()*20
	case SeasonHurricane:
		if rand.Float64() < 0.3 {
			sd.Weather = "storm"
		} else {
			sd.Weather = "cloudy"
		}
		sd.Humidity = 90 + rand.Float64()*10
	case SeasonTurtleNest:
		sd.Weather = "clear"
		sd.Humidity = 60 + rand.Float64()*20
	}
}

// CreatureBoost returns which creature types are boosted this season.
func (sd *SeasonData) CreatureBoost() []string {
	switch sd.Current {
	case SeasonDry:
		return []string{"Shellback", "Rootling"}
	case SeasonWet:
		return []string{"Glowpup", "Motesprite"}
	case SeasonHurricane:
		return []string{"Spikefin", "Crystalisk"}
	case SeasonTurtleNest:
		return []string{"Rootling", "Glowpup"}
	}
	return nil
}

// LootMultiplier returns loot scaling based on season and night.
func (sd *SeasonData) LootMultiplier() float64 {
	mult := 1.0
	if sd.IsNight {
		mult *= 1.5 // night has better loot
	}
	if sd.Weather == "storm" {
		mult *= 1.3 // storms stir things up
	}
	if sd.Current == SeasonHurricane {
		mult *= 2.0 // hurricane season = rare loot
	}
	return mult
}

// ─── Sleep System ──────────────────────────────────────────────────
// All players must sleep to advance the day and recover.

type SleepSystem struct {
	IsSleeping    bool      `json:"sleeping"`
	SleepStart    time.Time `json:"-"`
	HoursSlept    float64   `json:"hours_slept"`
	HoursNeeded   float64   `json:"hours_needed"` // 8 hours for full recovery
	AllAsleep     bool      `json:"all_asleep"`
	BedLocation   string    `json:"bed"` // "home", "inn", "camp"
}

func NewSleepSystem() *SleepSystem {
	return &SleepSystem{
		HoursNeeded: 8,
		BedLocation: "home",
	}
}

// StartSleep begins the sleep cycle.
func (ss *SleepSystem) StartSleep() {
	ss.IsSleeping = true
	ss.SleepStart = time.Now()
	ss.HoursSlept = 0
}

// UpdateSleep progresses the sleep timer. Returns true when fully rested.
func (ss *SleepSystem) UpdateSleep(dtHours float64) bool {
	if !ss.IsSleeping {
		return false
	}
	ss.HoursSlept += dtHours
	if ss.HoursSlept >= ss.HoursNeeded {
		ss.IsSleeping = false
		return true // fully rested!
	}
	return false
}

// RecoveryPercent returns how much HP/AP is restored.
func (ss *SleepSystem) RecoveryPercent() float64 {
	return ss.HoursSlept / ss.HoursNeeded
}

// ─── Day Summary (shown after sleeping) ────────────────────────────

type DaySummary struct {
	Day           int
	Season        Season
	Weather       string
	Temperature   float64
	CreaturesFound []string
	ResourcesGathered map[string]int
	GoldEarned    int
	DepthReached  int
	CompanionsGained int
	Message       string
}

func NewDaySummary(day int, season SeasonData) *DaySummary {
	return &DaySummary{
		Day:         day,
		Season:      season.Current,
		Weather:     season.Weather,
		Temperature: season.Temperature,
		ResourcesGathered: make(map[string]int),
		Message:     randomFloridaMessage(season.Current),
	}
}

func randomFloridaMessage(s Season) string {
	messages := map[Season][]string{
		SeasonDry: {
			"The sunsets over Honeymoon Island are pure magic.",
			"A gentle breeze carries the scent of salt and citrus.",
			"The sand feels warm beneath your feet.",
			"Pelicans glide over the emerald water.",
		},
		SeasonWet: {
			"The afternoon storm washes the island clean.",
			"Raindrops dance on the palm fronds.",
			"A rainbow stretches across the sky.",
			"Everything smells like wet earth and flowers.",
		},
		SeasonHurricane: {
			"The wind howls but your home stands strong.",
			"Waves crash against the shore with ancient fury.",
			"Storm clouds part to reveal a golden sunset.",
			"The well hums with strange energy during the storm.",
		},
		SeasonTurtleNest: {
			"A sea turtle laid eggs on the beach tonight.",
			"Tiny turtle tracks lead to the water.",
			"The moonlight reflects off a turtle's shell.",
			"New life begins on the shore.",
		},
	}
	msgs := messages[s]
	if len(msgs) == 0 {
		return "Another beautiful day in Florida."
	}
	return msgs[rand.Intn(len(msgs))]
}

// ─── Dungeon Scaling ───────────────────────────────────────────────

// DungeonScale returns the difficulty and loot multipliers based on player count.
func DungeonScale(playerCount int) (hpMult, dmgMult, lootMult float64) {
	switch playerCount {
	case 1:
		return 1.0, 1.0, 1.0
	case 2:
		return 1.5, 1.3, 1.5
	case 3:
		return 2.0, 1.5, 2.0
	case 4:
		return 2.5, 1.8, 3.0
	default:
		return 1.0, 1.0, 1.0
	}
}

// FloridaCreatureSpawn returns creature types that spawn based on season.
func FloridaCreatureSpawn(season Season, depth int) string {
	base := []string{"Glowpup", "Shellback", "Rootling"}
	seasonal := map[Season][]string{
		SeasonDry:     {"Drypup", "Sandcrab"},
		SeasonWet:     {"Rainfrog", "Stormfin"},
		SeasonHurricane: {"Hurricane", "Stormserpent"},
		SeasonTurtleNest: {"Shellback", "Seasparkle"},
	}
	extras := seasonal[season]
	all := append(base, extras...)
	return all[rand.Intn(len(all))]
}

// ─── Whimsical Florida Touches ─────────────────────────────────────

var FloridaFacts = []string{
	"Florida has more than 1,000 miles of coastline!",
	"Alligators can climb trees!",
	"The Sunshine State has the longest coastline in the contiguous US.",
	"Honeymoon Island is known for its shelling beaches!",
	"Florida is the only place with both alligators AND crocodiles.",
	"Key lime pie was invented in Florida!",
	"Florida has over 4,500 islands!",
	"A man once wrestled an alligator to save his dog in Florida.",
	"The Florida panther is the state animal.",
	"Florida gets more lightning than any other US state.",
	"Honeymoon Island was originally named Hog Island!",
	"There are over 500 species of fish in Florida waters.",
}

func RandomFloridaFact() string {
	return FloridaFacts[rand.Intn(len(FloridaFacts))]
}

// ─── Print functions ───────────────────────────────────────────────

func (sd *SeasonData) String() string {
	dayPart := "Day"
	if sd.IsNight {
		dayPart = "Night"
	}
	return fmt.Sprintf("%s %d | %s | %s | %.0f°F %d:%02d",
		dayPart, sd.Day, sd.Current.String(), sd.Weather,
		sd.Temperature, sd.Hour, sd.Minute)
}

// FormatSeasonSummary returns a nice multi-line summary for the UI.
func (sd *SeasonData) FormatSeasonSummary() []string {
	summary := []string{
		fmt.Sprintf("☀ Day %d — %s", sd.Day, sd.Current.String()),
		fmt.Sprintf("🌡 %.0f°F | %s", sd.Temperature, sd.Weather),
	}
	if sd.IsNight {
		summary[0] = fmt.Sprintf("🌙 Night %d — %s", sd.Day, sd.Current.String())
		summary = append(summary, "✨ Bioluminescent creatures are active!")
	}
	return summary
}

// UpdateHourly runs hourly maintenance tasks.
func (sd *SeasonData) UpdateHourly(creatureMgr *CreatureManager) {
	sd.AdvanceTime(60)

	// Hourly creature land production
	landCreatures := creatureMgr.GetLandCreatures()
	for _, c := range landCreatures {
		c.UpdateEvolution(3600) // 1 hour of evolution
	}
}