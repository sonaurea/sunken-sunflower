package game

import (
	"math"
	"math/rand"
)

// ─── Combat System ──────────────────────────────────────────────────
// Inspired by Animal Farm's deep but accessible combat design.
// Supports: damage types, resistances, critical hits, dodge, parry, combos.

// DamageType categorises attacks for resistance/weakness calculations.
type DamageType int

const (
	DamagePhysical DamageType = iota
	DamageMagic
	DamageFire
	DamagePoison
	DamageHoly    // sunflower radiance
	DamageShadow  // deep well corruption
)

// String returns the display name of the damage type.
func (dt DamageType) String() string {
	switch dt {
	case DamagePhysical:
		return "physical"
	case DamageMagic:
		return "magic"
	case DamageFire:
		return "fire"
	case DamagePoison:
		return "poison"
	case DamageHoly:
		return "holy"
	case DamageShadow:
		return "shadow"
	}
	return "unknown"
}

// ─── Hit Result ─────────────────────────────────────────────────────

type HitResult struct {
	Damage       float64
	IsCrit       bool
	IsDodged     bool
	IsParried    bool
	IsBlocked    bool
	DamageType   DamageType
	StatusEffect string // e.g. "burn", "poison", "stun"
	StatusDuration float64
	KnockbackX   float64
	KnockbackY   float64
}

// CombatStats defines an entity's combat capabilities.
type CombatStats struct {
	Damage        float64
	DamageType    DamageType
	CritChance    float64 // 0.0 - 1.0
	CritMultiplier float64
	DodgeChance   float64
	ParryChance   float64
	BlockChance   float64
	BlockReduction float64 // damage multiplier when blocked (0.5 = 50% reduction)
	Armor         float64
	Resistances   map[DamageType]float64 // 0.0 = immune, 1.0 = normal, 2.0 = double damage
	StatusChance  float64
	StatusType    string
	StatusDuration float64
	KnockbackForce float64
}

// DefaultPlayerStats returns the default combat stats for the player.
func DefaultPlayerStats() CombatStats {
	return CombatStats{
		Damage:         15,
		DamageType:     DamageHoly,
		CritChance:     0.10,
		CritMultiplier: 2.0,
		DodgeChance:    0.05,
		ParryChance:    0.10,
		BlockChance:    0.0,
		BlockReduction: 0.5,
		Armor:          5,
		Resistances: map[DamageType]float64{
			DamagePhysical: 1.0,
			DamageMagic:    1.0,
			DamageFire:     0.8,
			DamagePoison:   0.6,
			DamageHoly:     1.0,
			DamageShadow:   1.2,
		},
		StatusChance:   0.05,
		StatusType:     "burn",
		StatusDuration: 3.0,
		KnockbackForce: 80,
	}
}

// DefaultEnemyStats returns combat stats scaled by depth.
func DefaultEnemyStats(depth int) CombatStats {
	return CombatStats{
		Damage:          10 + float64(depth)*3,
		DamageType:      DamagePhysical,
		CritChance:      0.05 + float64(depth)*0.01,
		CritMultiplier:  1.5,
		DodgeChance:     float64(depth) * 0.01,
		Armor:           float64(depth) * 2,
		Resistances: map[DamageType]float64{
			DamagePhysical: 1.0,
			DamageMagic:    1.2,
			DamageFire:     0.8 + float64(depth)*0.02,
			DamagePoison:   0.5,
			DamageHoly:     1.5,
			DamageShadow:   0.3,
		},
		KnockbackForce: 60,
	}
}

// ─── Damage Calculation ────────────────────────────────────────────

// CalculateHit computes the full hit result from attacker stats vs defender stats.
func CalculateHit(attacker, defender CombatStats, attackerType string) HitResult {
	result := HitResult{
		DamageType: attacker.DamageType,
	}

	// 1. Dodge check
	if rand.Float64() < defender.DodgeChance {
		result.IsDodged = true
		return result
	}

	// 2. Parry check (melee only)
	if rand.Float64() < defender.ParryChance {
		result.IsParried = true
		result.Damage = attacker.Damage * 0.3 // reduced damage through parry
		return result
	}

	// 3. Block check
	if rand.Float64() < defender.BlockChance {
		result.IsBlocked = true
	}

	// 4. Base damage calculation
	baseDamage := attacker.Damage

	// 5. Apply resistance
	resistance, ok := defender.Resistances[attacker.DamageType]
	if !ok {
		resistance = 1.0
	}
	baseDamage *= resistance

	// 6. Apply armor mitigation (diminishing returns)
	armorMitigation := 1.0 - (defender.Armor / (defender.Armor + 50))
	baseDamage *= armorMitigation

	// 7. Block reduction
	if result.IsBlocked {
		baseDamage *= defender.BlockReduction
	}

	// 8. Crit check
	if rand.Float64() < attacker.CritChance {
		result.IsCrit = true
		baseDamage *= attacker.CritMultiplier
	}

	// 9. Random variance ±15%
	variance := 0.85 + rand.Float64()*0.30
	baseDamage *= variance

	// 10. Minimum damage
	if baseDamage < 1 {
		baseDamage = 1
	}

	result.Damage = math.Round(baseDamage)

	// 11. Status effect
	if rand.Float64() < attacker.StatusChance && attacker.StatusType != "" {
		result.StatusEffect = attacker.StatusType
		result.StatusDuration = attacker.StatusDuration
	}

	// 12. Knockback
	if attacker.KnockbackForce > 0 {
		angle := rand.Float64() * math.Pi * 2
		result.KnockbackX = math.Cos(angle) * attacker.KnockbackForce
		result.KnockbackY = math.Sin(angle) * attacker.KnockbackForce
	}

	return result
}

// ─── Combo System ───────────────────────────────────────────────────

// ComboStage tracks a multi-hit combo.
type ComboStage struct {
	Name      string
	DamageMult float64
	Effect     string // status to apply on this stage
}

// Combo tracks the player's current combo state.
type Combo struct {
	Stages      []ComboStage
	Current     int
	Timer       float64
	MaxTimer    float64 // window between hits before combo resets
	Active      bool
}

// NewPlayerCombo creates the default sunflower combo chain.
func NewPlayerCombo() *Combo {
	return &Combo{
		Stages: []ComboStage{
			{Name: "Petal Slash", DamageMult: 1.0, Effect: ""},
			{Name: "Sunflare", DamageMult: 1.3, Effect: "burn"},
			{Name: "Bloom Burst", DamageMult: 1.8, Effect: "stun"},
		},
		MaxTimer: 1.5,
	}
}

func (c *Combo) Update(dt float64) {
	if !c.Active {
		return
	}
	c.Timer += dt
	if c.Timer >= c.MaxTimer {
		c.Reset()
	}
}

func (c *Combo) Advance() ComboStage {
	c.Timer = 0
	c.Active = true
	stage := c.Stages[c.Current]
	c.Current++
	if c.Current >= len(c.Stages) {
		c.Current = 0
	}
	return stage
}

func (c *Combo) Reset() {
	c.Current = 0
	c.Timer = 0
	c.Active = false
}

// ─── Status Effects (DoT / HoT) ────────────────────────────────────

type ActiveStatus struct {
	Type     string
	Duration float64
	Elapsed  float64
	Damage   float64  // per-second damage/ heal
	SourceID string
}

type StatusManager struct {
	statuses []*ActiveStatus
}

func NewStatusManager() *StatusManager {
	return &StatusManager{}
}

func (sm *StatusManager) Add(status *ActiveStatus) {
	// Refresh if existing
	for _, s := range sm.statuses {
		if s.Type == status.Type && s.SourceID == status.SourceID {
			s.Duration = status.Duration
			s.Elapsed = 0
			return
		}
	}
	sm.statuses = append(sm.statuses, status)
}

func (sm *StatusManager) Update(dt float64) []ActiveStatus {
	var ticks []ActiveStatus
	alive := sm.statuses[:0]
	for _, s := range sm.statuses {
		s.Elapsed += dt
		ticks = append(ticks, *s)
		if s.Elapsed >= s.Duration {
			continue
		}
		alive = append(alive, s)
	}
	sm.statuses = alive
	return ticks
}

func (sm *StatusManager) HasType(statusType string) bool {
	for _, s := range sm.statuses {
		if s.Type == statusType {
			return true
		}
	}
	return false
}

func (sm *StatusManager) Clear() {
	sm.statuses = nil
}

// DoTDamage calculates the damage for a tick of a damage-over-time effect.
func DoTDamage(baseDamage float64, statusType string) float64 {
	switch statusType {
	case "burn":
		return baseDamage * 0.15
	case "poison":
		return baseDamage * 0.10
	case "heal":
		return -baseDamage * 0.20 // negative = healing
	default:
		return 0
	}
}

// ─── XP / Leveling ─────────────────────────────────────────────────

type LevelSystem struct {
	Level       int
	CurrentXP   float64
	XPToNext    float64
	TotalXP     float64
	StatPoints  int
}

func NewLevelSystem() *LevelSystem {
	return &LevelSystem{
		Level:    1,
		XPToNext: 100,
	}
}

func (ls *LevelSystem) AddXP(amount float64) bool {
	ls.CurrentXP += amount
	ls.TotalXP += amount
	if ls.CurrentXP >= ls.XPToNext {
		ls.LevelUp()
		return true
	}
	return false
}

func (ls *LevelSystem) LevelUp() {
	ls.CurrentXP -= ls.XPToNext
	ls.Level++
	ls.XPToNext = 100 * math.Pow(1.15, float64(ls.Level-1))
	ls.StatPoints += 3
}

func (ls *LevelSystem) XPProgress() float64 {
	return ls.CurrentXP / ls.XPToNext
}

// XPForEnemy returns XP gained for killing an enemy at a given depth.
func XPForEnemy(depth int) float64 {
	return 15.0 + float64(depth)*5.0
}

// ─── Stealth ───────────────────────────────────────────────────────

type StealthState struct {
	IsHidden      bool
	Detection     float64 // 0=hidden, 1=detected
	DetectionRate float64
	DecayRate     float64
	BackstabMult  float64
}

func NewStealthState() *StealthState {
	return &StealthState{
		DetectionRate: 0.5,
		DecayRate:     1.5,
		BackstabMult:  3.0,
	}
}

func (ss *StealthState) Update(dt float64, isMoving, nearEnemy bool) {
	if nearEnemy {
		if isMoving {
			ss.Detection += ss.DetectionRate * dt
		} else {
			ss.Detection += ss.DetectionRate * 0.5 * dt
		}
	} else {
		ss.Detection -= ss.DecayRate * dt
	}

	if ss.Detection <= 0 {
		ss.Detection = 0
		ss.IsHidden = true
	} else {
		ss.IsHidden = false
	}
	if ss.Detection > 1 {
		ss.Detection = 1
	}
}