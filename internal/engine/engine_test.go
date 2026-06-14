package engine

import (
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// ─── Palette ───────────────────────────────────────────────────────

func TestPaletteConstants(t *testing.T) {
	tests := []string{
		"ColSunflower", "ColSunset", "ColBeachPink", "ColDreamPurple",
		"ColOcean", "ColMidnight", "ColBio", "ColWhite", "ColBlack",
		"ColRed", "ColGreen", "ColBrown", "ColSand",
	}
	colors := []color.Color{
		ColSunflower, ColSunset, ColBeachPink, ColDreamPurple,
		ColOcean, ColMidnight, ColBio, ColWhite, ColBlack,
		ColRed, ColGreen, ColBrown, ColSand,
	}
	for i, c := range colors {
		if _, ok := c.(color.RGBA); !ok {
			t.Errorf("%s is not RGBA", tests[i])
		}
	}
}

func TestColorRGBA(t *testing.T) {
	c := ColorRGBA(255, 128, 64, 255)
	if c.R != 255 || c.G != 128 || c.B != 64 || c.A != 255 {
		t.Errorf("ColorRGBA mismatch: got %v", c)
	}
}

// ─── Scene Manager ─────────────────────────────────────────────────

type testScene struct{ BaseScene; e, x, u, d bool }

func (s *testScene) Enter(g *Game) { s.e = true }
func (s *testScene) Exit(g *Game)  { s.x = true }
func (s *testScene) Update(g *Game) { s.u = true }
func (s *testScene) Draw(screen *ebiten.Image, g *Game) { s.d = true }

func TestSceneManager(t *testing.T) {
	sm := NewSceneManager()
	ts := &testScene{}

	sm.Register("test", ts)
	if sm.Active() != nil {
		t.Error("expected no active scene initially")
	}

	g := &Game{}
	sm.SwitchTo("test", g)
	if !ts.e {
		t.Error("scene Enter not called")
	}
	if sm.ActiveID() != "test" {
		t.Errorf("expected activeID 'test', got %s", sm.ActiveID())
	}

	ts2 := &testScene{}
	sm.Register("test2", ts2)
	sm.SwitchTo("test2", g)
	if !ts.x {
		t.Error("scene Exit not called on switch")
	}
	if !ts2.e {
		t.Error("new scene Enter not called")
	}

	sm.SwitchTo("nonexistent", g)
	if sm.ActiveID() != "test2" {
		t.Error("should not switch to unknown scene")
	}
}

// ─── Entity Manager ────────────────────────────────────────────────

type testEntity struct{ BaseEntity }

func TestEntityManager(t *testing.T) {
	em := NewEntityManager()
	e1 := &testEntity{}
	e1.IDValue = "entity1"
	e1.Active = true

	em.Add(e1)
	if em.Get("entity1") != e1 {
		t.Error("entity not found after Add")
	}

	em.Add(e1) // duplicate
	em.Remove("entity1")
	if em.Get("entity1") != nil {
		t.Error("entity should be nil after Remove")
	}

	e2 := &testEntity{}
	e2.IDValue = "entity2"
	em.Add(e2)
	em.Clear()
	if len(em.All()) != 0 {
		t.Error("Clear should empty all entities")
	}

	// Empty ID
	e3 := &testEntity{}
	em.Add(e3)
}

func TestBaseEntity(t *testing.T) {
	e := &BaseEntity{IDValue: "test", X: 100, Y: 200, Active: true}
	if e.ID() != "test" {
		t.Error("ID() mismatch")
	}
	x, y := e.Position()
	if x != 100 || y != 200 {
		t.Error("Position() mismatch")
	}
	e.SetPosition(50, 75)
	x, y = e.Position()
	if x != 50 || y != 75 {
		t.Error("SetPosition() failed")
	}
	if !e.IsActive() {
		t.Error("should be active")
	}
	e.SetActive(false)
	if e.IsActive() {
		t.Error("should be inactive")
	}
}

// ─── Particles ─────────────────────────────────────────────────────

func TestParticleEmitter(t *testing.T) {
	pe := NewParticleEmitter()
	if !pe.Active {
		t.Error("emitter should be active")
	}
	pe.Emit(100, 100, 8, ColSunflower, 50, 1.0)
	if len(pe.Particles) != 8 {
		t.Errorf("expected 8 particles, got %d", len(pe.Particles))
	}
	pe.Update(0.5)
	pe.Active = false
	pe.Emit(100, 100, 5, ColRed, 50, 1.0)
}

func TestParticleVariety(t *testing.T) {
	pe := NewParticleEmitter()
	pe.EmitExplosion(100, 100, 12, ColSunset, 80, 0.8)
	pe.EmitTrail(100, 100, ColOcean, 60)
	pe.EmitSparkle(100, 100, ColBio)
	pe.EmitDustMotes(100, 100, 20, ColWhite)
}

func TestParticleManager(t *testing.T) {
	pm := NewParticleManager()
	pe := NewParticleEmitter()
	pm.AddEmitter(pe)
	pe.Emit(100, 100, 4, ColWhite, 30, 0.5)
	pm.Update(0.1)
	pm2 := NewParticleManager()
	pm2.Update(0.1)
}

// ─── Input ─────────────────────────────────────────────────────────

func TestInputReset(t *testing.T) {
	i := &Input{}
	i.Update()
	if i.LeftPressed || i.RightPressed {
		t.Error("no keys should be pressed after initial Update")
	}
}

// ─── Graphics ──────────────────────────────────────────────────────

func TestDrawPrimitives(t *testing.T) {
	img := ebiten.NewImage(100, 100)
	DrawCircle(img, 50, 50, 20, ColSunflower)
	DrawCircle(img, 5, 5, 0, ColWhite)
	DrawRect(img, 10, 10, 80, 80, ColOcean)
	DrawLine(img, 10, 10, 90, 90, ColWhite)
}

func TestDrawGradient(t *testing.T) { DrawGradient(ebiten.NewImage(100, 100), ColSunset, ColMidnight) }
func TestDrawGlow(t *testing.T)     { DrawGlow(ebiten.NewImage(100, 100), 50, 50, 30, ColDreamPurple) }
func TestOceanWaves(t *testing.T)   { DrawOceanWaves(ebiten.NewImage(100, 100), 1.5) }

func TestStarField(t *testing.T) {
	img := ebiten.NewImage(100, 100)
	stars := GenerateStarField(10, 100, 100)
	if len(stars) != 10 {
		t.Errorf("expected 10 stars, got %d", len(stars))
	}
	DrawStarField(img, stars, ColWhite)
}

func TestBeachSand(t *testing.T) { GenerateBeachSand(ebiten.NewImage(50, 50), ColSand) }

func TestSpriteGenerators(t *testing.T) {
	if sf := GenerateSunflowerSprite(32); sf == nil || sf.Bounds().Dx() != 32 {
		t.Error("sunflower sprite invalid")
	}
	if enemy := GenerateEnemySprite(24, ColRed); enemy == nil {
		t.Error("enemy sprite nil")
	}
	if comp := GenerateCompanionSprite(20); comp == nil {
		t.Error("companion sprite nil")
	}
	if tile := GenerateDungeonTile(32); tile == nil {
		t.Error("dungeon tile nil")
	}
}

// ─── Easing ────────────────────────────────────────────────────────

func TestEasingFunctions(t *testing.T) {
	const eps = 1e-6
	easings := []struct {
		name string
		fn   EasingFunc
	}{
		{"EaseOutBounce", EaseOutBounce},
		{"EaseInOutQuad", EaseInOutQuad},
		{"EaseOutElastic", EaseOutElastic},
		{"EaseInBack", EaseInBack},
	}
	for _, e := range easings {
		t.Run(e.name, func(t *testing.T) {
			v0 := e.fn(0)
			v1 := e.fn(1)
			if v0 < -eps || v0 > eps {
				t.Errorf("easing(0) = %f, expected ~0", v0)
			}
			if v1 < 1-eps || v1 > 1+eps {
				t.Errorf("easing(1) = %f, expected ~1", v1)
			}
		})
	}
}

// ─── Tween ─────────────────────────────────────────────────────────

func TestTween(t *testing.T) {
	var val float64
	tw := NewTween(1.0, 0, 100, EaseInOutQuad, func(v float64) { val = v }, nil)
	tw.Update(0.5)
	if val <= 0 || val >= 100 {
		t.Errorf("expected mid-value, got %f", val)
	}
	tw.Update(0.5)
	if !tw.Done || val != 100 {
		t.Errorf("expected done with value 100, got done=%v val=%f", tw.Done, val)
	}
	val = 0
	tw.Update(0.5)
	if val != 0 {
		t.Error("done tween should not call OnUpdate")
	}
}

func TestTweenWithOnDone(t *testing.T) {
	done := false
	tw := NewTween(0.5, 0, 1, EaseOutBounce, nil, func() { done = true })
	tw.Update(0.6)
	if !done {
		t.Error("OnDone should have been called")
	}
}

func TestTweenManager(t *testing.T) {
	tm := NewTweenManager()
	var val float64
	t1 := NewTween(0.3, 0, 10, EaseInOutQuad, func(v float64) { val = v }, nil)
	tm.Add(t1)
	tm.Update(0.4)
	if val != 10 {
		t.Errorf("expected 10, got %f", val)
	}
	tm.Clear()
	if len(tm.tweens) != 0 {
		t.Error("Clear should remove all tweens")
	}
}

// ─── Screen Shake ──────────────────────────────────────────────────

func TestScreenShake(t *testing.T) {
	ss := NewScreenShake(10, 0.5)
	if !ss.IsActive() {
		t.Error("shake should be active")
	}
	ss.Update(0.1)
	if ss.OffsetX == 0 && ss.OffsetY == 0 {
		t.Error("shake should have some offset")
	}
	for i := 0; i < 10; i++ {
		ss.Update(0.1)
	}
	if ss.IsActive() {
		t.Error("shake should be done after duration")
	}
}

// ─── Damage Numbers ────────────────────────────────────────────────

func TestDamageNumberManager(t *testing.T) {
	dnm := NewDamageNumberManager()
	dnm.Add(100, 100, "10", ColRed)
	dnm.Add(200, 200, "CRIT!", ColSunflower)
	if len(dnm.Numbers) != 2 {
		t.Errorf("expected 2 numbers, got %d", len(dnm.Numbers))
	}
	dnm.Update(1.5)
	if len(dnm.Numbers) != 0 {
		t.Errorf("expected 0 numbers after expiry, got %d", len(dnm.Numbers))
	}
}

// ─── Flash Overlay ─────────────────────────────────────────────────

func TestFlashOverlay(t *testing.T) {
	f := NewFlashOverlay(ColRed, 1.0, 2.0)
	if !f.Active {
		t.Error("flash should be active")
	}
	f.Update(0.6)
	if f.Active {
		t.Error("flash should be done after decay")
	}
}

// ─── Status Effect ─────────────────────────────────────────────────

func TestStatusEffectManager(t *testing.T) {
	sem := NewStatusEffectManager()
	sem.Add("poison", 3.0)
	sem.Add("burn", 2.0)
	if !sem.Has("poison") {
		t.Error("should have poison")
	}
	sem.Update(2.5)
	if sem.Has("burn") {
		t.Error("burn should have expired")
	}
	if !sem.Has("poison") {
		t.Error("poison should still be active")
	}
	sem.Add("poison", 5.0) // refresh
	sem.Update(3.0)
	if !sem.Has("poison") {
		t.Error("poison should have been refreshed")
	}
	sem.Update(3.0)
	if sem.Has("poison") {
		t.Error("all effects should have expired")
	}
}

// ─── Transitions ───────────────────────────────────────────────────

func TestTransition(t *testing.T) {
	halfCalled := false
	tr := NewTransition(TransitionFade, 1.0, func() { halfCalled = true })
	tr.Update(0.6)
	if !halfCalled {
		t.Error("OnHalf should have been called at 0.5")
	}
	for i := 0; i < 10; i++ {
		tr.Update(0.2)
	}
	if tr.Active {
		t.Error("transition should be done after duration")
	}
}

func TestTransitionDirections(t *testing.T) {
	for _, dir := range []TransitionDirection{
		TransitionSlideUp, TransitionSlideDown, TransitionZoom,
	} {
		tr := NewTransition(dir, 0.5, nil)
		for i := 0; i < 10; i++ {
			tr.Update(0.1)
		}
		if tr.Active {
			t.Errorf("transition %d should be done", dir)
		}
	}
}

// ─── Ambient Dust ──────────────────────────────────────────────────

func TestAmbientDust(t *testing.T) {
	ad := NewAmbientDust(10, 100, 100)
	if len(ad.Motes) != 10 {
		t.Errorf("expected 10 dust motes, got %d", len(ad.Motes))
	}
	ad.Update(0.5)
	ad.Update(0.5)
	ad.Draw(ebiten.NewImage(100, 100))
}

// ─── Floating Text ─────────────────────────────────────────────────

func TestFloatingTextManager(t *testing.T) {
	ftm := NewFloatingTextManager()
	ftm.Add("Hello", 100, 100, ColWhite)
	ftm.Add("World", 200, 200, ColSunflower)
	ftm.Update(2.0)
	if len(ftm.texts) != 0 {
		t.Errorf("all texts should have expired, got %d", len(ftm.texts))
	}
}

// ─── Spawn Animation ───────────────────────────────────────────────

func TestSpawnAnim(t *testing.T) {
	sa := NewSpawnAnim(100, 100, ColRed)
	if sa.Done {
		t.Error("spawn anim should not be done initially")
	}
	sa.Update(0.6)
	if !sa.Done {
		t.Error("spawn anim should be done after duration")
	}
}

// ─── Misc Visual Effects ───────────────────────────────────────────

func TestVisualEffects(t *testing.T) {
	img := ebiten.NewImage(100, 100)
	DrawGlowRing(img, 50, 50, 20, 1.0, ColSunflower)
	DrawPulsingBorder(img, 10, 10, 80, 80, 1.0, ColSunflower, 2)
	DrawOrbitIndicator(img, 50, 50, 15, 1.0, ColBio)
	DrawPlayerLight(img, 50, 50, 30)
	DrawDarkness(img, 50, 50, 20)
}

func TestLightRays(t *testing.T) {
	rays := GenerateLightRays(5, 100, 100)
	if len(rays) != 5 {
		t.Errorf("expected 5 rays, got %d", len(rays))
	}
	DrawLightRays(ebiten.NewImage(100, 100), rays, 1.0)
}

func TestSortParticlesByY(t *testing.T) {
	ps := []*Particle{{Y: 100}, {Y: 50}, {Y: 200}}
	SortParticlesByY(ps)
	if ps[0].Y != 50 || ps[1].Y != 100 || ps[2].Y != 200 {
		t.Error("particles not sorted by Y")
	}
}

// ─── Game ──────────────────────────────────────────────────────────

func TestNewGame(t *testing.T) {
	g := NewGame(DefaultSettings())
	if g.SceneManager() == nil || g.EntityManager() == nil ||
		g.ParticleManager() == nil || g.Input() == nil {
		t.Error("game components should not be nil")
	}
	if g.Settings().Title != "Sunken Sunflower" {
		t.Errorf("expected title 'Sunken Sunflower', got %s", g.Settings().Title)
	}
	if g.GameTime() != 0 || g.DeltaTime() != 0 {
		t.Error("initial time should be 0")
	}
}

// ─── Font ──────────────────────────────────────────────────────────

func TestTextFunctions(t *testing.T) {
	if w := TextWidth("Hello"); w <= 0 {
		t.Errorf("TextWidth should be positive, got %d", w)
	}
	if h := TextHeight("Hello"); h <= 0 {
		t.Errorf("TextHeight should be positive, got %d", h)
	}
	if cx := CenterX("Hello", 100); cx < 0 || cx > 100 {
		t.Errorf("CenterX out of bounds: %d", cx)
	}
	DrawText(ebiten.NewImage(100, 100), "Test", 10, 10, ColWhite)
}

// ─── Isometric ─────────────────────────────────────────────────────

func TestIsoConfig(t *testing.T) {
	cfg := DefaultIsoConfig()
	if cfg.TileWidth != 64 || cfg.TileHeight != 32 {
		t.Error("default iso config mismatch")
	}
}

func TestWorldToIso(t *testing.T) {
	cfg := DefaultIsoConfig()
	sx, sy := WorldToIso(0, 0, cfg)
	if sx != 0 || sy != 0 {
		t.Errorf("WorldToIso(0,0) should be (0,0), got (%f,%f)", sx, sy)
	}
	sx, sy = WorldToIso(1, 0, cfg)
	if sx <= 0 || sy <= 0 {
		t.Errorf("WorldToIso(1,0) should be positive, got (%f,%f)", sx, sy)
	}
}

func TestIsoToWorld(t *testing.T) {
	cfg := DefaultIsoConfig()
	wx, wy := IsoToWorld(0, 0, cfg)
	if wx != 0 || wy != 0 {
		t.Errorf("IsoToWorld(0,0) should be (0,0), got (%f,%f)", wx, wy)
	}
}

func TestRoundTripCoords(t *testing.T) {
	cfg := DefaultIsoConfig()
	ox, oy := 5.0, 3.0
	sx, sy := WorldToIso(ox, oy, cfg)
	bx, by := IsoToWorld(sx, sy, cfg)
	if math.Abs(bx-ox) > 0.01 || math.Abs(by-oy) > 0.01 {
		t.Errorf("round-trip failed: (%f,%f)->(%f,%f)->(%f,%f)", ox, oy, sx, sy, bx, by)
	}
}

func TestWorldToScreen(t *testing.T) {
	cfg := DefaultIsoConfig()
	_, sy := WorldToScreen(0, 0, 0, cfg, 0, 0)
	if sy < 0 {
		t.Errorf("WorldToScreen base Y should be positive, got %f", sy)
	}
	_, sy2 := WorldToScreen(0, 0, 2, cfg, 0, 0)
	if sy2 >= sy {
		t.Errorf("elevation should make Y lower, got %f vs %f", sy2, sy)
	}
}

func TestIsoMap(t *testing.T) {
	m := NewIsoMap(10, 10, DefaultIsoConfig())
	if m.Width != 10 || m.Height != 10 {
		t.Errorf("map size should be 10x10, got %dx%d", m.Width, m.Height)
	}
	if tile := m.TileAt(5, 5); tile == nil || tile.WorldX != 5 || tile.WorldY != 5 {
		t.Error("tile at (5,5) invalid")
	}
	if m.TileAt(-1, 0) != nil || m.TileAt(100, 0) != nil {
		t.Error("out of bounds should be nil")
	}
}

func TestIsoMapSetElevation(t *testing.T) {
	m := NewIsoMap(10, 10, DefaultIsoConfig())
	m.SetElevation(5, 5, 3, 2)
	if m.TileAt(5, 5).Elevation <= 0 {
		t.Error("center tile should have elevation")
	}
	if m.TileAt(0, 0).Elevation != 0 {
		t.Error("far tile should have zero elevation")
	}
}

func TestIsoCamera(t *testing.T) {
	cfg := DefaultIsoConfig()
	cam := NewIsoCamera(cfg)
	if cam.WorldX != 0 || cam.WorldY != 0 {
		t.Error("camera should start at origin")
	}
	cfg2 := DefaultIsoConfig()
	cam2 := NewIsoCamera(cfg2)
	cam2.Smoothing = 1.0 // instant follow
	cam2.Follow(5, 3)
	cam2.Update()
	if cam2.WorldX != cam2.TargetX || cam2.WorldY != cam2.TargetY {
		t.Errorf("camera with smoothing=1 should reach target instantly, got (%f,%f)",
			cam2.WorldX, cam2.WorldY)
	}
	ox, oy := cam2.ScreenOffset()
	if ox == 0 && oy == 0 {
		t.Errorf("screen offset should be non-zero after following with target (%f,%f)",
			cam2.TargetX, cam2.TargetY)
	}
}

func TestIsoDrawFunctions(t *testing.T) {
	img := ebiten.NewImage(100, 100)
	cfg := DefaultIsoConfig()
	drawIsoDiamond(img, 50, 50, cfg.TileWidth, cfg.TileHeight, ColWhite)
	drawIsoTriangle(img, 10, 10, 20, 30, 30, 10, ColWhite)
	drawIsoQuad(img, 10, 10, 20, 20, 30, 10, 20, 0, ColWhite)
	drawIsoWall(img, 50, 50, cfg.TileWidth, cfg.TileHeight, 20, ColBrown)
	drawIsoWallLeft(img, 50, 50, cfg.TileWidth, cfg.TileHeight, 20, ColBrown)
	drawIsoWallRight(img, 50, 50, cfg.TileWidth, cfg.TileHeight, 20, ColBrown)
	drawIsoChest(img, 50, 50, cfg)
	drawIsoExit(img, 50, 50, cfg)
	drawIsoWell(img, 50, 50, cfg)
	DrawIsoHighlight(img, 3, 4, cfg, 50, 50, ColSunflower)
	DrawIsoMinimap(img, NewIsoMap(10, 10, cfg), 5, 5)
}

func TestIsoEntityDrawer(t *testing.T) {
	ied := NewIsoEntityDrawer()
	ied.Add(100, 100, 10, func(screen *ebiten.Image) {})
	ied.Add(200, 200, 5, func(screen *ebiten.Image) {})
	img := ebiten.NewImage(300, 300)
	ied.Draw(img)
	ied.Draw(img) // empty draw
}

// ─── Animation System ──────────────────────────────────────────────

func TestFrameAnim(t *testing.T) {
	fa := NewFrameAnim([]int{0, 1, 2}, 0.15, true, false)
	if fa.CurrentFrame() != 0 {
		t.Errorf("expected frame 0, got %d", fa.CurrentFrame())
	}
	fa.Update(0.1)
	if fa.CurrentFrame() != 0 {
		t.Errorf("expected frame 0, got %d", fa.CurrentFrame())
	}
	fa.Update(0.1)
	if fa.CurrentFrame() != 1 {
		t.Errorf("expected frame 1, got %d", fa.CurrentFrame())
	}
	fa.Update(0.1)
	if fa.CurrentFrame() != 2 {
		t.Errorf("expected frame 2, got %d", fa.CurrentFrame())
	}
	fa.Reset()
	if fa.CurrentFrame() != 0 {
		t.Errorf("after reset expected frame 0, got %d", fa.CurrentFrame())
	}
}

func TestFrameAnimNonLooping(t *testing.T) {
	fa := NewFrameAnim([]int{0, 1}, 0.1, false, false)
	fa.Update(0.25)
	if !fa.IsDone() {
		t.Error("non-looping anim should be done")
	}
}

func TestFrameAnimPingPong(t *testing.T) {
	fa := NewFrameAnim([]int{0, 1, 2}, 0.12, false, true)
	// Advance frame by frame: 0 -> 1 -> 2 -> ping to 1
	updates := []struct {
		dt     float64
		expect int
	}{
		{0.09, 0}, // not enough time
		{0.09, 1}, // 0.18 >= 0.12, advance to 1
		{0.09, 2}, // 0.24 >= 0.12, advance to 2
		{0.09, 1}, // 0.30 >= 0.12, ping back to 1
	}
	for i, u := range updates {
		fa.Update(u.dt)
		if got := fa.CurrentFrame(); got != u.expect {
			t.Errorf("update %d: expected frame %d, got %d (timer=%f, current=%d)",
				i, u.expect, got, fa.timer, fa.current)
		}
	}
}

func TestPropAnim(t *testing.T) {
	vals := make(map[string]float64)
	target := &mockPropAccessor{vals}
	pa := NewPropAnim("scale", 1.0, 2.0, 1.0, EaseInOutQuad)
	pa.Update(0.5, target)
	if vals["scale"] <= 1.0 || vals["scale"] >= 2.0 {
		t.Errorf("expected mid scale, got %f", vals["scale"])
	}
	pa.Update(0.5, target)
	if vals["scale"] != 2.0 {
		t.Errorf("expected final scale 2.0, got %f", vals["scale"])
	}
}

type mockPropAccessor struct{ vals map[string]float64 }

func (m *mockPropAccessor) GetProp(name string) float64 { return m.vals[name] }
func (m *mockPropAccessor) SetProp(name string, val float64) { m.vals[name] = val }

func TestPropAnimNilTarget(t *testing.T) {
	pa := NewPropAnim("x", 0, 100, 0.5, EaseOutBounce)
	called := false
	pa.OnUpdate = func(val float64) { called = true }
	pa.Update(0.6, nil)
	if !called {
		t.Error("OnUpdate should be called even with nil target")
	}
}

func TestAnimationClip(t *testing.T) {
	clip := NewAnimationClip("test")
	clip.Duration = 0.5 // explicit duration
	clip.AddProp("x", 0, 100, 0.5, EaseInOutQuad)
	vals := make(map[string]float64)
	clip.SetTarget(&mockPropAccessor{vals})
	clip.Update(0.3)
	if vals["x"] <= 0 {
		t.Error("x should have changed")
	}
	clip.Update(0.3) // elapsed=0.6 >= 0.5
	if !clip.IsDone() {
		clip.Update(0.1)
		if !clip.IsDone() {
			t.Error("clip should be done after full duration")
		}
	}
}

func TestAnimationClipLoop(t *testing.T) {
	clip := NewAnimationClip("loop_test")
	clip.AddProp("x", 0, 10, 0.3, EaseInOutQuad)
	clip.Loop = true
	clip.SetTarget(&mockPropAccessor{make(map[string]float64)})
	clip.Update(0.4)
	if clip.IsDone() {
		t.Error("looping clip should never be done")
	}
}

func TestAnimationClipReset(t *testing.T) {
	clip := NewAnimationClip("reset_test")
	clip.AddProp("x", 0, 100, 0.3, EaseInOutQuad)
	clip.Update(0.4)
	clip.Reset()
	if clip.IsDone() {
		t.Error("after reset, clip should not be done")
	}
}

func TestAnimationClipEvents(t *testing.T) {
	eventFired := false
	clip := NewAnimationClip("event_test")
	clip.AddEvent(0.3, func() { eventFired = true })
	clip.AddProp("x", 0, 10, 1.0, EaseInOutQuad)
	clip.Update(0.4)
	if !eventFired {
		t.Error("event at 0.3 should have fired")
	}
}

func TestAnimator(t *testing.T) {
	anim := NewAnimator()
	clip := NewAnimationClip("idle")
	clip.AddProp("bob", 0, 5, 0.5, EaseInOutQuad)
	clip.Loop = true
	anim.AddClip(clip)

	stateChanged := false
	anim.OnStateChange = func(from, to string) { stateChanged = true }

	anim.Play("idle", false)
	if anim.CurrentClip() != "idle" {
		t.Errorf("expected 'idle', got %s", anim.CurrentClip())
	}
	if !stateChanged {
		t.Error("OnStateChange should be called")
	}

	stateChanged = false
	anim.Play("idle", false)
	if stateChanged {
		t.Error("same clip should not re-trigger OnStateChange")
	}

	anim.Play("unknown", false)
	if anim.CurrentClip() != "idle" {
		t.Error("should stay on current for unknown clip")
	}
}

func TestAnimatorUpdate(t *testing.T) {
	anim := NewAnimator()
	clip := NewAnimationClip("walk")
	clip.AddProp("x", 0, 10, 0.3, EaseInOutQuad)
	clip.Loop = true
	anim.AddClip(clip)
	anim.Play("walk", true)
	anim.Update(0.1)
}

func TestAnimationPresets(t *testing.T) {
	for _, p := range []*AnimationClip{
		PresetIdle(), PresetWalk(), PresetAttack(),
		PresetHurt(), PresetDeath(),
	} {
		if p.Name == "" || p == nil {
			t.Error("preset clip invalid")
		}
	}
}

func TestAnimationBuilder(t *testing.T) {
	clip := BeginAnim("test_build").
		Frames([]int{0, 1, 2}, 0.1, true, false).
		Prop("x", 0, 100, 0.5, EaseInOutQuad).
		Event(0.5, func() {}).
		Duration(1.0).
		Loop(true).
		NextClip("idle").
		Build()
	if clip.Name != "test_build" || clip.FrameAnim == nil ||
		len(clip.PropAnims) != 1 || clip.NextClip != "idle" {
		t.Error("builder output invalid")
	}
}

func TestJSONAnimation(t *testing.T) {
	clip, err := LoadAnimationFromJSON([]byte(`{
		"name":"test_anim","frame_indices":[0,1,2],
		"frame_rate":10,"loop":true,
		"props":[{"property":"x","from":0,"to":100,"duration":0.5,"easing":"quad"}]
	}`))
	if err != nil {
		t.Fatalf("JSON animation error: %v", err)
	}
	if clip.Name != "test_anim" || clip.FrameAnim == nil {
		t.Error("JSON animation invalid")
	}
	// Invalid JSON
	_, err = LoadAnimationFromJSON([]byte(`{invalid}`))
	if err == nil {
		t.Error("should error on invalid JSON")
	}
}

func TestAnimatedEntity(t *testing.T) {
	ae := NewAnimatedEntity()
	if ae.Alpha != 1.0 || ae.Scale != 1.0 {
		t.Error("default values incorrect")
	}
	if ae.CurrentFrameImage() != nil {
		t.Error("frame should be nil with no frames")
	}
}

func TestAnimatedEntitySpriteSheet(t *testing.T) {
	ae := NewAnimatedEntity()
	ae.SetSpriteSheet(ebiten.NewImage(64, 32), 32, 32)
	if len(ae.Frames) != 2 {
		t.Errorf("expected 2 frames, got %d", len(ae.Frames))
	}
	ae.SetSpriteSheet(nil, 32, 32)
	if len(ae.Frames) != 0 {
		t.Error("nil sheet should clear frames")
	}
}

func TestAnimatedEntityUpdate(t *testing.T) {
	ae := NewAnimatedEntity()
	clip := NewAnimationClip("idle")
	clip.AddProp("bob", 0, 5, 0.5, EaseInOutQuad)
	clip.Loop = true
	ae.Animator.AddClip(clip)
	ae.Animator.Play("idle", true)
	ae.Update(0.1)
	ae.DrawOpts(100, 100)
}

func TestGenerateFrameFunctions(t *testing.T) {
	if len(GenerateIdleFrames(32, ColSunflower)) != 4 {
		t.Error("expected 4 idle frames")
	}
	if len(GenerateWalkFrames(32, ColSunflower)) != 6 {
		t.Error("expected 6 walk frames")
	}
	if len(GenerateAttackFrames(32, ColSunflower)) != 4 {
		t.Error("expected 4 attack frames")
	}
	if len(GenerateHurtFrames(32, ColSunflower)) != 2 {
		t.Error("expected 2 hurt frames")
	}
}

func TestEasingFromName(t *testing.T) {
	for _, name := range []string{"bounce", "quad", "elastic", "back", "unknown"} {
		if fn := easingFromName(name); fn == nil {
			t.Errorf("easingFromName(%q) should not be nil", name)
		}
	}
}

func TestDrawableTileAccessible(t *testing.T) {
	dt := drawableTile{screenX: 100, screenY: 200, depth: 8}
	if dt.depth != 8 {
		t.Error("drawableTile depth mismatch")
	}
}