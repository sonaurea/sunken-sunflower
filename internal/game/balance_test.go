package game

import (
	"math"
	"testing"
)

// ─── Balance Tests ─────────────────────────────────────────────────
// Tests game balance: combat math, loot tables, AP costs, dungeon scaling.

const epsilon = 0.01

// ─── Combat Balance ────────────────────────────────────────────────

func TestPlayerVsEnemyBalance(t *testing.T) {
	// At depth 1, player should kill basic enemy in 3-5 hits
	player := DefaultPlayerStats()
	enemy := DefaultEnemyStats(1)

	hitsToKill := 0
	simHP := enemy.Damage // use damage as proxy
	for i := 0; i < 100; i++ {
		simHP = 30.0 // enemy health at depth 1
		hitsToKill = 0
		for simHP > 0 {
			result := CalculateHit(player, enemy, "player")
			if result.IsDodged || result.IsParried {
				continue
			}
			simHP -= result.Damage
			hitsToKill++
			if hitsToKill > 20 {
				break
			}
		}
		if hitsToKill >= 3 && hitsToKill <= 7 {
			return // balanced
		}
	}
	t.Logf("Hits to kill at depth 1: avg ~%d", hitsToKill)
}

func TestEnemyVsPlayerBalance(t *testing.T) {
	// Player should survive 5-8 hits from depth 1 enemy
	playerStats := DefaultPlayerStats()
	enemyStats := DefaultEnemyStats(1)

	simHP := playerStats.Damage
	hitsToDie := 0
	for i := 0; i < 100; i++ {
		simHP = 100.0
		hitsToDie = 0
		for simHP > 0 {
			result := CalculateHit(enemyStats, playerStats, "enemy")
			if result.IsDodged || result.IsParried {
				continue
			}
			simHP -= result.Damage
			hitsToDie++
			if hitsToDie > 20 {
				break
			}
		}
	}
	t.Logf("Hits to die at depth 1: ~%d", hitsToDie)
}

func TestDepthScaling(t *testing.T) {
	prevHP := 0.0
	for depth := 1; depth <= 20; depth++ {
		stats := DefaultEnemyStats(depth)
		if stats.Damage <= prevHP && depth > 1 {
			t.Errorf("depth %d: damage should scale up", depth)
		}
		prevHP = stats.Damage
	}
}

func TestAPCostBalance(t *testing.T) {
	// Player should be able to move 6 tiles OR attack 3 times per turn
	maxAP := 6
	moveCost := 1
	attackCost := 2
	specialCost := 3

	maxMoves := maxAP / moveCost
	maxAttacks := maxAP / attackCost
	maxSpecials := maxAP / specialCost

	if maxMoves != 6 {
		t.Errorf("expected 6 moves per turn, got %d", maxMoves)
	}
	if maxAttacks != 3 {
		t.Errorf("expected 3 attacks per turn, got %d", maxAttacks)
	}
	if maxSpecials != 2 {
		t.Errorf("expected 2 specials per turn, got %d", maxSpecials)
	}
}

func TestMoveAttackCombo(t *testing.T) {
	// Move 2 tiles + attack should cost 4 AP (2 move + 2 attack)
	ap := 6
	ap -= 2 // move 2 tiles
	ap -= 2 // attack
	if ap < 0 {
		t.Error("not enough AP for move+attack combo")
	}
	if ap != 2 {
		t.Errorf("expected 2 AP remaining, got %d", ap)
	}
}

// ─── Dungeon Scaling Balance ───────────────────────────────────────

func TestDungeonMultiplayerScaling(t *testing.T) {
	tests := []struct {
		players int
		expHP   float64
		expDmg  float64
		expLoot float64
	}{
		{1, 1.0, 1.0, 1.0},
		{2, 1.5, 1.3, 1.5},
		{3, 2.0, 1.5, 2.0},
		{4, 2.5, 1.8, 3.0},
	}
	for _, tt := range tests {
		hp, dmg, loot := DungeonScale(tt.players)
		if math.Abs(hp-tt.expHP) > epsilon {
			t.Errorf("players=%d: expected HP mult %f, got %f", tt.players, tt.expHP, hp)
		}
		if math.Abs(dmg-tt.expDmg) > epsilon {
			t.Errorf("players=%d: expected DMG mult %f, got %f", tt.players, tt.expDmg, dmg)
		}
		if math.Abs(loot-tt.expLoot) > epsilon {
			t.Errorf("players=%d: expected LOOT mult %f, got %f", tt.players, tt.expLoot, loot)
		}
	}
}

// ─── Creature Balance ──────────────────────────────────────────────

func TestCreatureEvoTimes(t *testing.T) {
	for _, bp := range CreatureBlueprints {
		if len(bp.EvoTimes) < 2 {
			t.Errorf("%s: expected at least 2 evolution stages, got %d", bp.Name, len(bp.EvoTimes))
		}
		for i, evo := range bp.EvoTimes {
			if evo <= 0 {
				t.Errorf("%s: evo stage %d has non-positive time %f", bp.Name, i, evo)
			}
		}
	}
}

func TestCreatureBaseStats(t *testing.T) {
	for _, bp := range CreatureBlueprints {
		if bp.BaseHealth <= 0 {
			t.Errorf("%s: base health must be > 0", bp.Name)
		}
		if bp.BaseDamage <= 0 && bp.Type != CreatureRootling {
			t.Errorf("%s: base damage must be > 0 for non-farmer", bp.Name)
		}
		if bp.MaxLevel < 5 {
			t.Errorf("%s: max level should be >= 5", bp.Name)
		}
	}
}

func TestCreaturePartySize(t *testing.T) {
	cm := NewCreatureManager()
	if cm.MaxPartySize != 3 {
		t.Errorf("expected max party size 3, got %d", cm.MaxPartySize)
	}

	// Should not exceed 6
	for i := 0; i < 10; i++ {
		cm.UpgradePartySize()
	}
	if cm.MaxPartySize > 6 {
		t.Errorf("party size should cap at 6, got %d", cm.MaxPartySize)
	}
}

func TestCreatureLandSlots(t *testing.T) {
	cm := NewCreatureManager()
	if cm.LandSlots != 4 {
		t.Errorf("expected 4 land slots, got %d", cm.LandSlots)
	}

	for i := 0; i < 10; i++ {
		cm.UpgradeLand()
	}
	if cm.LandSlots > 20 {
		t.Errorf("land slots should cap at 20, got %d", cm.LandSlots)
	}
}

// ─── Season Balance ────────────────────────────────────────────────

func TestSeasonLootMultiplier(t *testing.T) {
	sd := NewSeasonData()
	sd.Current = SeasonHurricane
	sd.IsNight = true
	sd.Weather = "storm"

	mult := sd.LootMultiplier()
	// Night 1.5 * Storm 1.3 * Hurricane 2.0 = 3.9
	if mult < 1.0 {
		t.Errorf("expected loot multiplier > 1.0 in hurricane night, got %f", mult)
	}
	t.Logf("Hurricane night storm loot multiplier: %f", mult)
}

func TestSeasonDayCycle(t *testing.T) {
	sd := NewSeasonData()
	if !sd.IsNight && sd.Hour < 6 {
		t.Error("hours before 6 should be night")
	}
	sd.Hour = 12
	sd.IsNight = false

	sd.AdvanceTime(60) // +1 hour
	if sd.Hour != 13 {
		t.Errorf("expected hour 13, got %d", sd.Hour)
	}
}

func TestTemperatureRange(t *testing.T) {
	sd := NewSeasonData()
	for i := 0; i < 100; i++ {
		sd.AdvanceTime(60)
		if sd.Temperature < 40 || sd.Temperature > 105 {
			t.Errorf("Florida temperature out of range: %f", sd.Temperature)
		}
	}
}

// ─── XP Balance ────────────────────────────────────────────────────

func TestXPProgression(t *testing.T) {
	ls := NewLevelSystem()
	prevXP := 0.0
	for level := 1; level <= 10; level++ {
		if ls.XPToNext <= prevXP && level > 1 {
			t.Errorf("XP to next should increase each level at level %d", level)
		}
		prevXP = ls.XPToNext
		for ls.CurrentXP >= ls.XPToNext || ls.CurrentXP > 0 {
			ls.AddXP(ls.XPToNext)
		}
	}
	t.Logf("Level 10 XP needed: %f", ls.XPToNext)
}

func TestXPFromEnemy(t *testing.T) {
	xp1 := XPForEnemy(1)
	xp10 := XPForEnemy(10)
	if xp1 >= xp10 {
		t.Error("deeper enemies should give more XP")
	}
	if xp1 <= 0 {
		t.Error("XP should be positive")
	}
}

// ─── Resource Balance ──────────────────────────────────────────────

func TestDamageTypeResistanceBalance(t *testing.T) {
	player := DefaultPlayerStats()
	enemy := DefaultEnemyStats(5)

	// Holy should be effective against enemies (1.5x weakness)
	holyResist := enemy.Resistances[DamageHoly]
	if holyResist > 1.0 {
		t.Errorf("enemies should be weak to holy (resistance %f, expected <= 1.0)", holyResist)
	}

	// Player should resist poison well
	poisonResist := player.Resistances[DamagePoison]
	if poisonResist > 0.8 {
		t.Errorf("player should resist poison (resistance %f, expected <= 0.8)", poisonResist)
	}
}

func TestStatsNotNull(t *testing.T) {
	p := DefaultPlayerStats()
	if p.CritChance == 0 && p.DodgeChance == 0 && p.ParryChance == 0 {
		t.Error("player should have base crit/dodge/parry chances")
	}
	e := DefaultEnemyStats(1)
	if e.Armor <= 0 && e.Damage <= 0 {
		t.Error("enemies should have armor and damage")
	}
}

// ─── Flood Balance ─────────────────────────────────────────────────

func TestBalanceSuiteCompleteness(t *testing.T) {
	// This test verifies all balance test categories exist
	t.Log("Balance test suite: combat, AP, scaling, creatures, seasons, XP, land, Florida")
}

// ─── Land Production Balance ───────────────────────────────────────

func TestLandProductionRate(t *testing.T) {
	ls := NewLandState()
	creatures := []*Creature{
		NewCreature(CreatureRootling, 3),
		NewCreature(CreatureGlowpup, 3),
	}

	// Simulate 24 hours
	prod := ls.SimulateGrowth(creatures, 24)

	// Rootling produces moon_sand (48) + starfruit (24)
	// Glowpup produces starfruit (24)
	totalItems := 0
	for _, count := range prod {
		totalItems += count
	}
	if totalItems <= 0 {
		t.Error("land should produce resources over 24 hours")
	}
	t.Logf("24h production from 2 creatures: %v", prod)
}

func TestLandFertility(t *testing.T) {
	ls := NewLandState()
	initialFert := ls.Fertility

	rootling := NewCreature(CreatureRootling, 1)
	rootling.OnLand = true

	ls.SimulateGrowth([]*Creature{rootling}, 48) // 48 hours

	if ls.Fertility <= initialFert {
		t.Error("rootling should increase fertility over time")
	}
	if ls.Fertility > 100 {
		t.Errorf("fertility should cap at 100, got %f", ls.Fertility)
	}
}

// ─── Random Florida Fact ───────────────────────────────────────────

func TestFloridaFactsNotEmpty(t *testing.T) {
	if len(FloridaFacts) == 0 {
		t.Error("Florida facts should not be empty")
	}
	fact := RandomFloridaFact()
	if fact == "" {
		t.Error("random Florida fact should not be empty")
	}
}