package engine

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ─── Hitbox System ─────────────────────────────────────────────────
// AABB-based hitbox/collision system for entities, attacks, and
// environment interactions.

// HitboxType categorizes what a hitbox is used for.
type HitboxType int

const (
	HitboxBody   HitboxType = iota // Entity body collision
	HitboxHurt                     // Receives damage
	HitboxHit                      // Deals damage (attack)
	HitboxTrigger                  // Interaction trigger zone
	HitboxProjectile               // Projectile collision
)

// Hitbox represents an axis-aligned bounding box for collision.
type Hitbox struct {
	Type      HitboxType
	X, Y      float64 // center position
	W, H      float64 // width and height
	OffsetX   float64 // offset from entity position
	OffsetY   float64
	Active    bool
	Damage    float64
	Knockback float64
	OwnerID   string // entity ID that owns this hitbox
	Tags      []string // custom tags for filtering
}

// NewHitbox creates a hitbox centered at an entity's position.
func NewHitbox(hType HitboxType, w, h float64) *Hitbox {
	return &Hitbox{
		Type:   hType,
		W:      w,
		H:      h,
		Active: true,
	}
}

// Center returns the hitbox center position.
func (hb *Hitbox) Center() (float64, float64) {
	return hb.X + hb.OffsetX, hb.Y + hb.OffsetY
}

// Bounds returns the hitbox min/max corners.
func (hb *Hitbox) Bounds() (minX, minY, maxX, maxY float64) {
	cx, cy := hb.Center()
	return cx - hb.W/2, cy - hb.H/2, cx + hb.W/2, cy + hb.H/2
}

// SetPosition updates the hitbox position (called from entity).
func (hb *Hitbox) SetPosition(x, y float64) {
	hb.X = x
	hb.Y = y
}

// ─── Collision Detection ───────────────────────────────────────────

// AABBOverlap checks if two hitboxes overlap.
func AABBOverlap(a, b *Hitbox) bool {
	if !a.Active || !b.Active {
		return false
	}
	ax1, ay1, ax2, ay2 := a.Bounds()
	bx1, by1, bx2, by2 := b.Bounds()
	return ax1 < bx2 && ax2 > bx1 && ay1 < by2 && ay2 > by1
}

// AABBOverlapRaw checks overlap with raw bounds (no hitbox objects).
func AABBOverlapRaw(ax, ay, aw, ah, bx, by, bw, bh float64) bool {
	return ax < bx+bw && ax+aw > bx && ay < by+bh && ay+ah > by
}

// PointInAABB checks if a point is inside a hitbox.
func PointInAABB(px, py float64, hb *Hitbox) bool {
	if !hb.Active {
		return false
	}
	minX, minY, maxX, maxY := hb.Bounds()
	return px >= minX && px <= maxX && py >= minY && py <= maxY
}

// DistanceToAABB returns the shortest distance from a point to a hitbox.
func DistanceToAABB(px, py float64, hb *Hitbox) float64 {
	minX, minY, maxX, maxY := hb.Bounds()
	// Closest point on AABB to the point
	closestX := math.Max(minX, math.Min(px, maxX))
	closestY := math.Max(minY, math.Min(py, maxY))
	dx := px - closestX
	dy := py - closestY
	return math.Sqrt(dx*dx + dy*dy)
}

// ─── Hitbox Manager ────────────────────────────────────────────────
// Manages all hitboxes in a scene and processes collisions.

type HitResult struct {
	AttackerID string
	TargetID   string
	Damage     float64
	KnockbackX float64
	KnockbackY float64
	HitType    HitboxType
}

type HitboxManager struct {
	hitboxes  []*Hitbox
	results   []HitResult
	DebugDraw bool // visualizes hitboxes when true
	OnHit     func(result HitResult)
}

func NewHitboxManager() *HitboxManager {
	return &HitboxManager{}
}

// Register adds a hitbox to the manager.
func (hm *HitboxManager) Register(hb *Hitbox) {
	hm.hitboxes = append(hm.hitboxes, hb)
}

// Remove removes a hitbox by owner ID.
func (hm *HitboxManager) Remove(ownerID string) {
	alive := hm.hitboxes[:0]
	for _, hb := range hm.hitboxes {
		if hb.OwnerID != ownerID {
			alive = append(alive, hb)
		}
	}
	hm.hitboxes = alive
}

// RemoveAll clears all hitboxes.
func (hm *HitboxManager) RemoveAll() {
	hm.hitboxes = nil
	hm.results = nil
}

// Update processes all hitbox collisions for this frame.
func (hm *HitboxManager) Update() {
	hm.results = nil

	// Check all hit vs hurt pairs
	for _, hitter := range hm.hitboxes {
		if hitter.Type != HitboxHit && hitter.Type != HitboxProjectile {
			continue
		}
		for _, receiver := range hm.hitboxes {
			if receiver.Type != HitboxHurt && receiver.Type != HitboxBody {
				continue
			}
			if hitter.OwnerID == receiver.OwnerID {
				continue // don't hit self
			}
			if AABBOverlap(hitter, receiver) {
				result := HitResult{
					AttackerID: hitter.OwnerID,
					TargetID:   receiver.OwnerID,
					Damage:     hitter.Damage,
					HitType:    hitter.Type,
				}
				hm.results = append(hm.results, result)
				if hm.OnHit != nil {
					hm.OnHit(result)
				}
				// Projectiles get consumed on hit
				if hitter.Type == HitboxProjectile {
					hitter.Active = false
				}
			}
		}
	}
}

// Results returns all hits from this frame.
func (hm *HitboxManager) Results() []HitResult {
	return hm.results
}

// GetHitboxes returns all hitboxes for an owner.
func (hm *HitboxManager) GetHitboxes(ownerID string) []*Hitbox {
	var result []*Hitbox
	for _, hb := range hm.hitboxes {
		if hb.OwnerID == ownerID {
			result = append(result, hb)
		}
	}
	return result
}

// CheckOverlap checks if a specific hitbox overlaps any hitboxes
// of a given type from other owners.
func (hm *HitboxManager) CheckOverlap(check *Hitbox, targetType HitboxType) []HitResult {
	var results []HitResult
	for _, hb := range hm.hitboxes {
		if hb.Type != targetType || hb.OwnerID == check.OwnerID {
			continue
		}
		if AABBOverlap(check, hb) {
			results = append(results, HitResult{
				TargetID: hb.OwnerID,
				Damage:   hb.Damage,
			})
		}
	}
	return results
}

// Draw visualizes all hitboxes (debug mode).
func (hm *HitboxManager) Draw(screen *ebiten.Image) {
	if !hm.DebugDraw {
		return
	}
	for _, hb := range hm.hitboxes {
		if !hb.Active {
			continue
		}
		minX, minY, maxX, maxY := hb.Bounds()
		clr := color.RGBA{0, 255, 0, 100}
		switch hb.Type {
		case HitboxHit:
			clr = color.RGBA{255, 0, 0, 100}
		case HitboxHurt:
			clr = color.RGBA{255, 255, 0, 100}
		case HitboxTrigger:
			clr = color.RGBA{0, 255, 255, 100}
		case HitboxProjectile:
			clr = color.RGBA{255, 0, 255, 100}
		}
		DrawRect(screen, minX, minY, maxX-minX, maxY-minY, clr)
	}
}

// ─── Convenience Hitbox Creators ───────────────────────────────────

// BodyHitbox creates a standard body hitbox for an entity.
func BodyHitbox(ownerID string, x, y, w, h float64) *Hitbox {
	return &Hitbox{
		Type:    HitboxBody,
		X:       x,
		Y:       y,
		W:       w,
		H:       h,
		Active:  true,
		OwnerID: ownerID,
	}
}

// AttackHitbox creates a hitbox for an attack (deals damage).
func AttackHitbox(ownerID string, x, y, w, h, damage, knockback float64) *Hitbox {
	return &Hitbox{
		Type:      HitboxHit,
		X:         x,
		Y:         y,
		W:         w,
		H:         h,
		Damage:    damage,
		Knockback: knockback,
		Active:    true,
		OwnerID:   ownerID,
	}
}

// TriggerHitbox creates an interaction trigger zone.
func TriggerHitbox(ownerID string, x, y, w, h float64) *Hitbox {
	return &Hitbox{
		Type:    HitboxTrigger,
		X:       x,
		Y:       y,
		W:       w,
		H:       h,
		Active:  true,
		OwnerID: ownerID,
	}
}