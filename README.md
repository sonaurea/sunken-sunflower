# Sunken Sunflower 🌻

> *A roguelike dungeon-crawler built in Go + Ebitengine. You're a sunflower descending a mystical well beneath a Florida beach town, fighting through procedurally-generated depths, collecting resources, and building up a village.*

**Target Platforms:** Linux (Steam Deck ✓), Windows, macOS

---

## Table of Contents
- [Quick Start](#quick-start)
- [Architecture Overview](#-architecture-overview)
- [Engine Core](#-engine-core)
- [Entity System & Modding API](#-entity-system--modding-api)
- [Co-op / Multiplayer Design](#-co-op--multiplayer-design)
- [Procedural Graphics Pipeline](#-procedural-graphics-pipeline)
- [Project Structure](#-project-structure)
- [Contributing / Extending](#-contributing--extending)
- [Adding Mods](#-adding-mods)
- [Steam Deck Tuning](#-steam-deck-tuning)

---

## Quick Start

```bash
# Prerequisites: Go 1.21+
git clone <repo> && cd beach-dreams
go mod tidy
go run .
```

**Build for Steam Deck:**
```bash
go build -o sunken-sunflower .
```

**Cross-compile:**
```bash
GOOS=windows GOARCH=amd64 go build -o sunken-sunflower.exe .
GOOS=darwin GOARCH=amd64 go build -o sunken-sunflower-macos .
```

---

## 🏗 Architecture Overview

The engine follows a **Scene → Entity → Component** hierarchy designed for extensibility from day one:

```
Game (main loop)
  └─ SceneManager
      └─ Scene (active)
          ├─ EntityManager
          │   ├─ Entity (Player, Enemy, NPC)
          │   │   └─ Components (Position, Health, Sprite, AI)
          │   └─ PlayerController (input mapping)
          ├─ ParticleManager
          │   └─ ParticleEmitter[]
          └─ DungeonGenerator (in dungeon scene)
```

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Go + Ebitengine** | Pure Go, zero native deps, single binary output, excellent Steam Deck performance |
| **Scene-based state machine** | Clean separation between town, dungeon, title, dream sequences |
| **Entity + optional Components** | Lightweight ECS without interface overhead; entities compose behaviour |
| **All graphics procedural** | Zero asset dependencies — game generates everything at runtime |
| **JSON mod definitions** | Mods define entities, items, rooms, dialogs in JSON — no recompile needed |

---

## ⚙ Engine Core

### `internal/engine/engine.go` — Game Loop

The top-level [`Game`](internal/engine/engine.go:25) struct implements [`ebiten.Game`](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2#Game):

```go
type Game struct {
    settings       GameSettings
    sceneManager   *SceneManager
    entityManager  *EntityManager
    particleMgr    *ParticleManager
    input          *Input
    gameTime       float64    // total elapsed seconds
    deltaTime      float64    // seconds since last tick
}
```

- `Update()` → scene → entities → particles → input
- `Draw(screen)` → scene → entities → particles (layered)
- `Layout()` returns fixed internal resolution (1280×720) — Ebitengine scales to display

### Scene Lifecycle

Every scene implements [`Scene`](internal/engine/engine.go:97):

```go
type Scene interface {
    Enter(g *Game)
    Exit(g *Game)
    Update(g *Game)
    Draw(screen *ebiten.Image, g *Game)
}
```

Switch scenes with `g.SceneManager().SwitchTo("scene_name", g)` — the manager calls `Exit()` on the old scene and `Enter()` on the new one.

### Adding a New Scene

```go
type MyCustomScene struct {
    engine.BaseScene
    timer float64
}

func (s *MyCustomScene) Enter(g *engine.Game) {
    s.timer = 0
}
func (s *MyCustomScene) Update(g *engine.Game) {
    s.timer += g.DeltaTime()
}
func (s *MyCustomScene) Draw(screen *ebiten.Image, g *engine.Game) {
    engine.DrawCircle(screen, 100, 100, 50, engine.ColorRGBA(255,215,0,255))
}

// Register it:
g.SceneManager().Register("my_scene", &MyCustomScene{})
```

---

## 🧩 Entity System & Modding API

### Entity Interface

```go
type Entity interface {
    ID() string
    Update(g *Game)
    Draw(screen *ebiten.Image, g *Game)
    Position() (x, y float64)
    SetPosition(x, y float64)
    IsActive() bool
    SetActive(active bool)
}
```

Extend [`BaseEntity`](internal/engine/engine.go:125) and override `Update()` / `Draw()`:

```go
type MyEnemy struct {
    engine.BaseEntity
    health int
}
func (e *MyEnemy) Update(g *engine.Game) {
    e.X += 50 * g.DeltaTime() // move right
}
```

### JSON Mod Definitions

Mods live in a `mods/` directory and are loaded at startup. Each mod is a JSON file that the engine parses into entities, rooms, items, and dialogue.

**Example mod (`mods/crab_enemy.json`):**

```json
{
  "name": "Crab Enemy",
  "version": "1.0",
  "entities": [
    {
      "type": "enemy",
      "id": "crab",
      "sprite": "procedural:circle",
      "color": "#FF4444",
      "size": [24, 24],
      "health": 30,
      "speed": 120,
      "damage": 10,
      "drop_table": [
        {"item": "moon_sand", "chance": 0.6, "count": [1, 3]},
        {"item": "starfruit", "chance": 0.3, "count": [1, 1]}
      ],
      "behaviour": {
        "type": "charge",
        "cooldown": 2.0,
        "range": 200
      }
    }
  ],
  "rooms": [
    {
      "id": "crab_arena",
      "enemies": ["crab", "crab"],
      "layout": "open",
      "size": [400, 300]
    }
  ]
}
```

The mod loader at [`internal/mods/loader.go`](internal/mods/loader.go) registers these with the entity factory, room generator, and drop tables — no code changes needed.

### Modding API Hooks

The engine fires events that mods can subscribe to via the event bus:

| Event | Fired When | Use Case |
|-------|-----------|----------|
| `OnEnemyKilled(enemyID)` | An enemy is defeated | Custom drops, quest tracking |
| `OnRoomEntered(roomType)` | Player enters a room | Environment effects, dialogue |
| `OnResourceCollected(resourceID, count)` | Player picks up loot | Resource multipliers, trackers |
| `OnSceneChanged(sceneName)` | Scene transition occurs | UI overlays, custom scenes |
| `OnPlayerDamaged(amount, source)` | Player takes damage | Custom damage reactions |
| `OnCompanionCommand(type, action)` | Player directs a companion | Companion behaviour overrides |

Subscribe in your mod's init:

```go
func init() {
    engine.Subscribe("OnEnemyKilled", func(data interface{}) {
        enemyID := data.(string)
        if enemyID == "crab" {
            // Custom behaviour
        }
    })
}
```

---

## 👥 Co-op / Multiplayer Design

The engine is designed for **drop-in local co-op and future online co-op** from the start. Here's how:

### Architecture for Co-op

```
Game Instance
  └─ PlayerManager
      ├─ Player (P1) — keyboard + gamepad 1
      ├─ Player (P2) — gamepad 2 (local co-op)
      └─ [Future] NetworkPlayer — synced via UDP
```

### Deterministic Update Loop

All game state updates are **deterministic** — they depend only on:
1. `deltaTime` (capped at 100ms)
2. Player input vectors
3. RNG seed (shared in multiplayer)

This means the same inputs produce the same outcomes across network peers.

### Splitscreen / Shared Screen

- **Local co-op**: Shared screen with camera that follows the midpoint between players
- **If players separate too far** (> 600px): Camera splits or screen pushes together
- **Each player** has their own companion squad (max 10 each)

### Netcode Layer (Future)

The [`internal/net`](internal/net/) package (stubbed) provides:

```go
type NetworkManager struct {
    peerID    string
    roomID    string
    state     *GameState
    syncRate  time.Duration  // 60hz for combat, 10hz for village
}
```

- State sync via delta-compressed snapshots
- Input prediction with server reconciliation
- Peer-to-peer for LAN, relay for online

---

## 🎨 Procedural Graphics Pipeline

All rendering is code-generated. No sprites, no textures, no art pipeline.

```
internal/engine/graphics.go
  ├── DrawGradient()      — 2-color gradient fills (sunset skies)
  ├── DrawCircle()        — Filled circles (characters, projectiles)
  ├── DrawGlow()          — Multi-layer glow with falloff (neon, magic)
  ├── DrawOceanWaves()    — Animated sine-wave ocean
  ├── ParticleEmitter     — Configurable particle system
  │   ├── Update(dt)      — Physics, life, interpolation
  │   └── Draw(target)    — Batch render
  ├── GenerateStarField() — Random star distribution
  └── GenerateBeachSand() — Noise-like grain texture
```

### Creating Custom Procedural Sprites

```go
func GenerateSunflowerSprite(size int) *ebiten.Image {
    img := ebiten.NewImage(size, size)
    // Petals (yellow circles around center)
    for i := 0; i < 8; i++ {
        angle := float64(i) * math.Pi / 4
        px := float64(size)/2 + math.Cos(angle)*float64(size)*0.3
        py := float64(size)/2 + math.Sin(angle)*float64(size)*0.3
        DrawCircle(img, px, py, float64(size)*0.12, ColorRGBA(255,215,0,255))
    }
    // Center (brown circle)
    DrawCircle(img, float64(size)/2, float64(size)/2, float64(size)*0.15, ColorRGBA(139,69,19,255))
    return img
}
```

### Color Palette

These constants live in [`internal/engine/palette.go`](internal/engine/palette.go) and are exposed to mods:

| Name | Hex | Go Constant | Usage |
|------|-----|-------------|-------|
| Sunflower Gold | `#FFD700` | `ColSunflower` | Player, seeds, positive effects |
| Sunset Orange | `#FF6B35` | `ColSunset` | Sky gradients, warnings |
| Beach Pink | `#FF8C94` | `ColBeachPink` | Sunset transitions, UI accents |
| Dream Purple | `#7B2D8E` | `ColDreamPurple` | Dream sequences, magic |
| Ocean Teal | `#2EC4B6` | `ColOcean` | Water, bioluminescence |
| Midnight Blue | `#1A1A3E` | `ColMidnight` | Night sky, deep dungeon |
| Bioluminescent | `#00FFCC` | `ColBio` | Glowing effects, companions |

---

## 📁 Project Structure

```
sunken-sunflower/
├── main.go                    # Entry point — window config, scene registration
├── README.md
├── go.mod / go.sum
├── mods/                      # External mods (JSON files, loaded at runtime)
│   └── example_mod.json
├── internal/
│   ├── engine/                # Core engine (no game-specific logic)
│   │   ├── engine.go          # Game loop, SceneManager, EntityManager
│   │   ├── graphics.go        # Procedural graphics: shapes, glow, particles, waves
│   │   ├── input.go           # Input abstraction: KB + gamepad (Steam Deck)
│   │   └── palette.go         # Named color constants
│   ├── scenes/                # Game scenes (game-specific scene implementations)
│   │   ├── scene.go           # Scene base types
│   │   ├── titlescene.go      # Title screen with beach sunset
│   │   ├── townscene.go       # Florida beach town hub
│   │   ├── dungeon.go         # Well dungeon with procedural rooms
│   │   └── dreamscene.go      # Dream sequences
│   ├── entities/              # Game entities
│   │   ├── sunflower.go       # Player character
│   │   ├── companion.go       # Creature companion AI
│   │   └── enemy.go           # Enemy AI behaviours
│   ├── dungeon/               # Dungeon generation
│   │   ├── generation.go      # Procedural room layout & connections
│   │   ├── rooms.go           # Room type definitions & templates
│   │   └── combat.go          # Combat system: damage, dodge, parry
│   ├── village/               # Village management
│   │   ├── buildings.go       # Building definitions & construction
│   │   ├── resources.go       # Resource types & inventory
│   │   └── upgrades.go        # Permanent upgrade tree
│   ├── game/                  # Game state & persistence
│   │   ├── state.go           # Global game state
│   │   ├── story.go           # Story flags & progression
│   │   └── save.go            # Save/load serialization
│   ├── mods/                  # Mod loading system
│   │   ├── loader.go          # JSON mod parser & registry
│   │   └── events.go          # Event bus for mod hooks
│   └── net/                   # Multiplayer (stubbed for future)
│       ├── network.go         # Network manager interface
│       └── sync.go            # State sync protocol
└── steamdeck.txt              # Steam Deck specific tuning
```

---

## 🤝 Contributing / Extending

### Building New Content (No Recompile Needed)

1. Create a `.json` file in `mods/`
2. Define entities, rooms, items, or dialogue
3. Run the game — mods load automatically

See [`mods/example_mod.json`](mods/example_mod.json) for a complete template.

### Building New Code

1. **New entity**: Create a struct embedding `engine.BaseEntity`, implement `Update()` and `Draw()`, register in entity factory
2. **New scene**: Implement the `Scene` interface, register in `main.go`
3. **New room type**: Add a template to [`dungeon/rooms.go`](internal/dungeon/rooms.go) — the generator picks from available templates
4. **New companion type**: Add to [`entities/companion.go`](internal/entities/companion.go) with its AI behaviour
5. **New upgrade**: Add to [`village/upgrades.go`](internal/village/upgrades.go) upgrade tree

### Coding Conventions

| Convention | Rule |
|------------|------|
| **Imports** | Standard lib first, third-party second, internal last |
| **Errors** | Return errors from `Update()`, log in scene |
| **Naming** | `camelCase` for private, `PascalCase` for exported |
| **Tests** | `*_test.go` beside implementation — run with `go test ./...` |
| **Threading** | No goroutines in game loop (Ebitengine's single-threaded model) |

---

## 🧩 Adding Mods

Mods are JSON files placed in the `mods/` directory. The game scans this directory at startup.

### Mod File Reference

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Human-readable mod name |
| `version` | `string` | Semver |
| `entities` | `[]ModEntity` | New entity definitions |
| `rooms` | `[]ModRoom` | New room templates |
| `items` | `[]ModItem` | New item/resource types |
| `dialogue` | `[]ModDialogue` | Dialogue trees / story events |
| `colors` | `map[string]string` | Custom color palette entries |

### Mod-Provided Behaviours

For complex behaviours, mods can provide a `.go` file in `mods/` that gets loaded via Go's `plugin` package (future). Simple mods use only JSON.

---

## 🔧 Steam Deck Tuning

| Tuning | Value | Rationale |
|--------|-------|-----------|
| Internal resolution | 1280×720 | Scales cleanly to Deck's 1280×800 |
| TPS cap | 60 | Smooth gameplay, < 5W TDP |
| VSync | Enabled | Prevents screen tearing on Deck LCD |
| Runnable on unfocused | Enabled | Steam overlay, quick resume |
| Max TPS | 60 | No turbo mode needed |
| Resizing mode | Enabled | Docked/undocked transitions |
| Fullscreen toggle | F11 | Quick switch between windowed/full |

The game uses **no GPU-intensive features** — no post-processing, no shaders, no heavy texture atlases. This keeps framerate stable and battery life long on Steam Deck.

---

> *"The sunflower follows the sun across the sky, but some flowers grow best in the dark."*
> — Inscription above the well