# Sunken Sunflower 🌻

> *A roguelike dungeon-crawler built in Go + Ebitengine. You're a sunflower on Honeymoon Island, Florida, who discovers a mystical well beneath the beach. Descend through procedurally-generated depths, collect creatures, build your land, and uncover the secrets of the well.*

**Created by Sonaurea (Michael Pupo) — Tampa, Florida 🏖️**

**Target Platforms:** Linux (Steam Deck ✓), Windows, macOS, Android, iOS

**License:** All Rights Reserved — see [LICENSE](LICENSE)

---

## 🎮 Quick Start

```bash
git clone <repo> && cd beach-dreams
go run .
```

**Build for Steam:**
```bash
make dist    # Builds all platforms
```

**Run tests:**
```bash
go test ./... -count=1
```

---

## 🏖️ The Story

*You bought a piece of land by the beaches of Honeymoon Island, Florida. While digging the foundation for your new home, you discovered an ancient well hidden beneath the sand. The well is your source of all growth — skills, property, social connection, and spiritual power.*

*Each descent into the well reveals deeper mysteries, stranger creatures, and greater treasures. But beware — the well changes with the Florida seasons, and what you find depends on the weather above.*

---

## 🎮 Gameplay Features

| Feature | Details |
|---------|---------|
| **Above Ground** | Free-roam isometric on Honeymoon Island. WASD or click to move. Talk to NPCs, manage creatures, upgrade your land |
| **The Well (Dungeons)** | Isometric turn-based hex grid with AP system (6 AP/turn). Move, attack, use specials. Flood mechanic when it rains! |
| **Florida Seasons** | Dry Season (Oct-May), Wet Season (Jun-Sep), Hurricane Season, Turtle Nesting Season — each affects creature spawns, loot, and weather |
| **Dynamic Weather** | Florida-style short intense rain showers (15-35 min). Rain buffs Glowpups (+30%), debuffs Spikefins (-25%). Rain causes dungeon flooding! |
| **Sleep System** | All players must sleep to advance the day. Full recovery at 8 hours. Day summary with Florida fact |
| **Creature/Pikmin System** | Collect 6 creature types from the well. They evolve on your land over real-time. Assign to party (up to 3) or land (up to 20 slots) |
| **Multiplayer** | 4-player co-op. Host a game, join via LAN or Steam friends. Dungeons scale per player count. Mod syncing from host |
| **Mod Support** | JSON mod definitions for entities, rooms, items, dialogue. Host mods sync to clients automatically |
| **Procedural Audio** | Music and SFX generated in real-time by a custom synthesizer. Zone-based scales, combat intensity modulation, no audio files needed |
| **Steam Integration** | 13 achievements, Steam Deck optimized, controller support. Achievements lock when mods are active |

---

## 🗺️ Controls

| Input | Above Ground | The Well (Turn-Based) |
|-------|-------------|----------------------|
| WASD / Left stick | Move freely | Select tile / move cursor |
| E / A | Interact | Confirm action |
| ESC / B | Back / Menu | Cancel |
| Shift / LB | Dash | (reserved) |
| F / RB | Companion command | End turn |
| F11 | Toggle fullscreen | Toggle fullscreen |

---

## 🏗️ Architecture

```
Game (Ebitengine)
  └─ SceneManager
      ├─ TitleScene      — Main menu with multiplayer options
      ├─ TownScene       — Free-roam isometric island hub
      ├─ DungeonScene    — Turn-based isometric hex grid
      └─ DreamScene      — Lyric-driven narrative sequences
  ├─ EntityManager       — ECS for player, enemies, companions
  ├─ ParticleManager     — Effect factory (Animal Well-style)
  ├─ TurnManager         — AP system (max 6)
  ├─ FloodSystem         — Dynamic water rising in dungeons
  ├─ AudioManager        — Procedural audio synthesis
  └─ NetworkManager      — 4-player peer-to-peer
```

### Mod Structure
```
mods/
  └─ your_mod.json      # JSON mod — entities, rooms, items, dialogue
```

---

## 📜 License

**All Rights Reserved** — Copyright (c) 2026 Michael Pupo (Sonaurea)

This software is protected by copyright law. You may NOT reproduce, distribute, or create derivative works without explicit written permission from the copyright holder. See the [LICENSE](LICENSE) file for complete terms.

**TL;DR:** I'm making this game to sell on Steam and consoles. Please don't copy it. Contributions are welcome and you keep ownership of your code, but grant me a license to include it in the game.

---

## 🛠️ Tech Stack

| Technology | Purpose |
|------------|---------|
| Go 1.26 | Language |
| Ebitengine v2.9 | Game engine (2D + isometric) |
| GitHub Actions | CI/CD (free for public repos) |
| Docker | Cross-platform builds (all SKUs) |
| golangci-lint | Static analysis |
| Steamworks SDK | Achievements, multiplayer, Steam Deck |

---

## 🏖️ Florida Love

*"The sunset feels like it's about to fade away... but you and me, we're free to explore the possibilities."*
— Sonaurea, *Posal Piece*

Made with love in Tampa, Florida. Sand between my toes, code on my screen. 🏖️