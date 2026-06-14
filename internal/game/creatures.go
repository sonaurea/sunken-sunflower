package game

import (
	"fmt"
	"math/rand"
	"time"
)

// ─── Creature/Pikmin System ────────────────────────────────────────
// "You bought land by the beaches of Honeymoon Island and discovered
//  the Well. The well was your source of all growth..."
//
// Creatures are collected from the well, evolve on your land,
// and accompany you on deeper dives.

type CreatureType int

const (
	CreatureGlowpup  CreatureType = iota // Healer — light affinity
	CreatureShellback                     // Tank — defensive
	CreatureSpikefin                      // Attacker — aggressive
	CreatureMotesprite                    // Scout — fast, finds secrets
	CreatureRootling                      // Farmer — improves land/evolution
	CreatureCrystalisk                    // Rare — powerful, deep well only
)

// CreatureRarity determines how rare a creature is.
type CreatureRarity int

const (
	RarityCommon CreatureRarity = iota
	RarityUncommon
	RarityRare
	RarityLegendary
)

func (r CreatureRarity) String() string {
	switch r {
	case RarityCommon:
		return "Common"
	case RarityUncommon:
		return "Uncommon"
	case RarityRare:
		return "Rare"
	case RarityLegendary:
		return "Legendary"
	}
	return "Unknown"
}

// Creature represents a single creature with growth/evolution.
type Creature struct {
	ID         string        `json:"id"`
	Type       CreatureType  `json:"type"`
	Name       string        `json:"name"`
	Rarity     CreatureRarity `json:"rarity"`
	Level      int           `json:"level"`
	MaxLevel   int           `json:"max_level"`
	XP         float64       `json:"xp"`
	XPToNext   float64       `json:"xp_to_next"`
	Health     float64       `json:"health"`
	MaxHealth  float64       `json:"max_health"`
	Damage     float64       `json:"damage"`
	Speed      float64       `json:"speed"`
	// Evolution
	EvolutionStage int       `json:"evo_stage"` // 0=base, 1=evolved, 2=max
	EvolutionTimer float64   `json:"evo_timer"` // time until next evo (real-time)
	EvoTimeNeeded  float64   `json:"evo_time"`  // total time needed
	// Land/Farm
	OnLand     bool          `json:"on_land"`
	LandSlot   int           `json:"land_slot"`
	// Bond (affects stats)
	BondLevel  int           `json:"bond"` // 0-10, increases with time together
	// Traits
	Traits     []string      `json:"traits"`
	// Meta
	FoundDepth int           `json:"found_depth"`
	FoundTime  time.Time     `json:"found_time"`
	IsActive   bool          `json:"is_active"` // currently in party
}

// CreatureBlueprint defines a creature type's base stats.
type CreatureBlueprint struct {
	Type        CreatureType
	Name        string
	Rarity      CreatureRarity
	BaseHealth  float64
	BaseDamage  float64
	BaseSpeed   float64
	MaxLevel    int
	EvoTimes    []float64 // hours at each evolution stage
	Description string
	Color       string // hex color
}

// CreatureBlueprints defines all creature types in the game.
var CreatureBlueprints = map[CreatureType]CreatureBlueprint{
	CreatureGlowpup: {
		Type: CreatureGlowpup, Name: "Glowpup",
		Rarity: RarityCommon, BaseHealth: 30, BaseDamage: 5, BaseSpeed: 80,
		MaxLevel: 10, EvoTimes: []float64{2, 8}, // 2h → stage 1, 8h → stage 2
		Description: "A bioluminescent pup that heals with light.",
		Color: "#00FFCC",
	},
	CreatureShellback: {
		Type: CreatureShellback, Name: "Shellback",
		Rarity: RarityCommon, BaseHealth: 80, BaseDamage: 8, BaseSpeed: 40,
		MaxLevel: 10, EvoTimes: []float64{3, 12},
		Description: "A sturdy shelled creature that protects allies.",
		Color: "#2EC4B6",
	},
	CreatureSpikefin: {
		Type: CreatureSpikefin, Name: "Spikefin",
		Rarity: RarityUncommon, BaseHealth: 25, BaseDamage: 18, BaseSpeed: 110,
		MaxLevel: 15, EvoTimes: []float64{4, 16},
		Description: "A swift predator with razor-sharp fins.",
		Color: "#FF4444",
	},
	CreatureMotesprite: {
		Type: CreatureMotesprite, Name: "Motesprite",
		Rarity: RarityUncommon, BaseHealth: 20, BaseDamage: 3, BaseSpeed: 140,
		MaxLevel: 12, EvoTimes: []float64{1.5, 6},
		Description: "A wisp of light that reveals hidden paths.",
		Color: "#FFD700",
	},
	CreatureRootling: {
		Type: CreatureRootling, Name: "Rootling",
		Rarity: RarityCommon, BaseHealth: 40, BaseDamage: 2, BaseSpeed: 30,
		MaxLevel: 8, EvoTimes: []float64{1, 4},
		Description: "A gentle gardener that enriches the land.",
		Color: "#8B4513",
	},
	CreatureCrystalisk: {
		Type: CreatureCrystalisk, Name: "Crystalisk",
		Rarity: RarityLegendary, BaseHealth: 60, BaseDamage: 25, BaseSpeed: 70,
		MaxLevel: 20, EvoTimes: []float64{12, 48},
		Description: "A crystalline being from the deepest well. Extremely rare.",
		Color: "#7B2D8E",
	},
}

// NewCreature creates a new creature found at a given depth.
func NewCreature(cType CreatureType, depth int) *Creature {
	bp := CreatureBlueprints[cType]
	lvl := 1
	// Depth affects starting level
	if depth > 5 {
		lvl = 2 + rand.Intn(3)
	}

	statMult := 1.0 + float64(lvl-1)*0.15

	return &Creature{
		ID:           fmt.Sprintf("creature_%s_%d", bp.Name, rand.Intn(99999)),
		Type:         cType,
		Name:         bp.Name,
		Rarity:       bp.Rarity,
		Level:        lvl,
		MaxLevel:     bp.MaxLevel,
		XPToNext:     20 * float64(lvl),
		Health:       bp.BaseHealth * statMult,
		MaxHealth:    bp.BaseHealth * statMult,
		Damage:       bp.BaseDamage * statMult,
		Speed:        bp.BaseSpeed,
		EvolutionStage: 0,
		EvoTimeNeeded:  bp.EvoTimes[0] * 3600, // convert hours to seconds
		FoundDepth:   depth,
		FoundTime:    time.Now(),
		Traits:       []string{},
	}
}

// UpdateEvolution progresses creature evolution based on real time on land.
func (c *Creature) UpdateEvolution(dt float64) {
	if !c.OnLand || c.EvolutionStage >= len(CreatureBlueprints[c.Type].EvoTimes) {
		return
	}
	c.EvolutionTimer += dt
	if c.EvolutionTimer >= c.EvoTimeNeeded {
		c.Evolve()
	}
}

// Evolve advances the creature to its next stage.
func (c *Creature) Evolve() {
	c.EvolutionStage++
	// Stats increase with evolution
	mult := 1.0 + float64(c.EvolutionStage)*0.5
	c.MaxHealth *= mult
	c.Health = c.MaxHealth
	c.Damage *= mult
	c.Speed *= 1.1
	// Set next evolution time
	bp := CreatureBlueprints[c.Type]
	if c.EvolutionStage < len(bp.EvoTimes) {
		c.EvoTimeNeeded = bp.EvoTimes[c.EvolutionStage] * 3600
		c.EvolutionTimer = 0
	}
	// Gain a trait
	traits := c.GetPossibleTraits()
	if len(traits) > 0 {
		trait := traits[rand.Intn(len(traits))]
		c.Traits = append(c.Traits, trait)
	}
}

// GetPossibleTraits returns traits this creature can gain at evolution.
func (c *Creature) GetPossibleTraits() []string {
	switch c.Type {
	case CreatureGlowpup:
		return []string{"Radiant Aura", "Healing Burst", "Light Shield"}
	case CreatureShellback:
		return []string{"Iron Shell", "Taunt", "Shield Ally"}
	case CreatureSpikefin:
		return []string{"Double Strike", "Bleed", "Swift Rush"}
	case CreatureMotesprite:
		return []string{"Treasure Sense", "Secret Paths", "Dodge"}
	case CreatureRootling:
		return []string{"Fertile Soil", "Fast Growth", "Harvest Boost"}
	case CreatureCrystalisk:
		return []string{"Crystal Armor", "Prismatic Beam", "Time Warp"}
	}
	return nil
}

// GetStatBonus returns the bond-based stat multiplier.
func (c *Creature) GetStatBonus() float64 {
	return 1.0 + float64(c.BondLevel)*0.05
}

// AddXP adds experience and handles leveling up.
func (c *Creature) AddXP(amount float64) bool {
	c.XP += amount
	if c.XP >= c.XPToNext && c.Level < c.MaxLevel {
		c.XP -= c.XPToNext
		c.Level++
		c.XPToNext = 20 * float64(c.Level)
		// Stat increase
		bp := CreatureBlueprints[c.Type]
		c.MaxHealth = bp.BaseHealth * (1.0 + float64(c.Level-1)*0.15)
		c.Health = c.MaxHealth
		c.Damage = bp.BaseDamage * (1.0 + float64(c.Level-1)*0.15)
		return true // leveled up
	}
	return false
}

// ─── Creature Manager (Farm/Land) ──────────────────────────────────

type CreatureManager struct {
	AllCreatures   []*Creature     `json:"all"`
	ActiveParty    []string        `json:"active_party"` // creature IDs
	MaxPartySize   int             `json:"max_party"`
	LandSlots      int             `json:"land_slots"`
	LandUpgradeLevel int           `json:"land_level"`
}

func NewCreatureManager() *CreatureManager {
	return &CreatureManager{
		MaxPartySize:   3,
		LandSlots:      4,
		LandUpgradeLevel: 1,
	}
}

// AddCreature adds a newly found creature.
func (cm *CreatureManager) AddCreature(c *Creature) {
	cm.AllCreatures = append(cm.AllCreatures, c)
}

// AssignToLand places a creature on the land to evolve.
func (cm *CreatureManager) AssignToLand(creatureID string) bool {
	slotsUsed := 0
	for _, c := range cm.AllCreatures {
		if c.OnLand {
			slotsUsed++
		}
	}
	if slotsUsed >= cm.LandSlots {
		return false
	}
	for _, c := range cm.AllCreatures {
		if c.ID == creatureID {
			c.OnLand = true
			c.IsActive = false
			return true
		}
	}
	return false
}

// AssignToParty adds a creature to the active party.
func (cm *CreatureManager) AssignToParty(creatureID string) bool {
	if len(cm.ActiveParty) >= cm.MaxPartySize {
		return false
	}
	for _, c := range cm.AllCreatures {
		if c.ID == creatureID {
			c.IsActive = true
			c.OnLand = false
			cm.ActiveParty = append(cm.ActiveParty, creatureID)
			return true
		}
	}
	return false
}

// RemoveFromParty removes a creature from the party.
func (cm *CreatureManager) RemoveFromParty(creatureID string) {
	for i, id := range cm.ActiveParty {
		if id == creatureID {
			cm.ActiveParty = append(cm.ActiveParty[:i], cm.ActiveParty[i+1:]...)
			for _, c := range cm.AllCreatures {
				if c.ID == creatureID {
					c.IsActive = false
					break
				}
			}
			return
		}
	}
}

// UpdateLandEvolution progresses all creatures on the land.
func (cm *CreatureManager) UpdateLandEvolution(dt float64) {
	for _, c := range cm.AllCreatures {
		if c.OnLand {
			c.UpdateEvolution(dt)
		}
	}
}

// GetActiveCreatures returns creatures currently in the party.
func (cm *CreatureManager) GetActiveCreatures() []*Creature {
	var active []*Creature
	for _, c := range cm.AllCreatures {
		if c.IsActive {
			active = append(active, c)
		}
	}
	return active
}

// GetLandCreatures returns creatures on the land.
func (cm *CreatureManager) GetLandCreatures() []*Creature {
	var land []*Creature
	for _, c := range cm.AllCreatures {
		if c.OnLand {
			land = append(land, c)
		}
	}
	return land
}

// UpgradeLand increases the number of land slots.
func (cm *CreatureManager) UpgradeLand() {
	cm.LandUpgradeLevel++
	cm.LandSlots = 4 + cm.LandUpgradeLevel*2
	if cm.LandSlots > 20 {
		cm.LandSlots = 20
	}
}

// UpgradePartySize increases max party size.
func (cm *CreatureManager) UpgradePartySize() {
	cm.MaxPartySize++
	if cm.MaxPartySize > 6 {
		cm.MaxPartySize = 6
	}
}

// FindCreatureInWell generates a random creature based on depth.
func FindCreatureInWell(depth int) *Creature {
	roll := rand.Float64()
	var cType CreatureType

	switch {
	case depth >= 15 && roll < 0.05:
		cType = CreatureCrystalisk // 5% at depth 15+
	case depth >= 8 && roll < 0.15:
		cType = []CreatureType{CreatureSpikefin, CreatureMotesprite}[rand.Intn(2)]
	case depth >= 3 && roll < 0.35:
		cType = []CreatureType{CreatureShellback, CreatureGlowpup, CreatureMotesprite}[rand.Intn(3)]
	default:
		cType = []CreatureType{CreatureGlowpup, CreatureShellback, CreatureRootling}[rand.Intn(3)]
	}

	return NewCreature(cType, depth)
}

// ─── Land/Farm on Honeymoon Island ─────────────────────────────────

type LandState struct {
	Level        int     `json:"level"`
	Fertility    float64 `json:"fertility"` // 0-100
	WaterAccess  bool    `json:"water"`
	Sunlight     float64 `json:"sunlight"` // 0-100
	Decorations  []string `json:"decor"`
	TotalHarvest int     `json:"harvest"`
}

func NewLandState() *LandState {
	return &LandState{
		Level:     1,
		Fertility: 30,
		Sunlight:  80, // Florida beach!
		WaterAccess: true,
	}
}

// SimulateGrowth calculates resource production from creatures on land.
func (ls *LandState) SimulateGrowth(creatures []*Creature, hours float64) map[string]int {
	production := make(map[string]int)
	for _, c := range creatures {
		if c.Type == CreatureRootling {
			ls.Fertility += 0.5 * hours
		}
		if ls.Fertility > 100 {
			ls.Fertility = 100
		}
		// Each creature produces resources based on type
		mult := 1.0 + float64(c.EvolutionStage)*0.5
		switch c.Type {
		case CreatureRootling:
			production["moon_sand"] += int(2 * hours * mult)
			production["starfruit"] += int(1 * hours * mult)
		case CreatureGlowpup:
			production["starfruit"] += int(1 * hours * mult)
		case CreatureShellback:
			production["well_shard"] += int(0.5 * hours * mult)
		case CreatureSpikefin:
			production["moon_sand"] += int(3 * hours * mult)
		case CreatureMotesprite:
			production["well_shard"] += int(1 * hours * mult)
		case CreatureCrystalisk:
			production["ancient_seed"] += int(0.2 * hours * mult)
		}
	}
	return production
}