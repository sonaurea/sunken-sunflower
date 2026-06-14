# Contributing to Sunken Sunflower 🌻

Thanks for wanting to contribute! Here's how to keep the codebase clean and safe.

## 🔒 Mandatory Rules

### 1. Tests Required for ALL New Code
Every new function, method, or feature **must** have corresponding tests.
```
new code → new tests → PR
```
The CI will **reject** any PR that reduces coverage below 70%.

### 2. Run These Before Pushing
```bash
go test ./...          # All tests pass
go vet ./...           # No static issues
gofmt -l .             # No formatting issues (should output nothing)
golangci-lint run ./...  # No lint issues (install: see below)
```

### 3. Branch Protection
- `main` is protected — no direct pushes
- Create a feature branch: `feature/your-feature-name`
- Open a Pull Request to `main`
- CI must pass before merge

### 4. Code Style
| Rule | Standard |
|------|----------|
| **Formatting** | `gofmt` (run `gofmt -w .`) |
| **Naming** | `camelCase` for private, `PascalCase` for exported |
| **Imports** | stdlib → third-party → internal (groups separated by blank line) |
| **Errors** | Always check errors. Return errors from `Update()`, log in scene |
| **Comments** | Every exported function needs a doc comment |
| **Line length** | Soft limit 100 chars, hard limit 120 |

### 5. Install Linters Locally
```bash
# golangci-lint (all-in-one linter)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# staticcheck (extra static analysis)
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### 6. PR Checklist
- [ ] Tests added for new code
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes
- [ ] `golangci-lint run ./...` passes
- [ ] `gofmt -l .` produces no output
- [ ] Coverage ≥ 70%

## 🎮 Project Architecture Quick Reference

```
internal/
├── engine/     # Core engine (game loop, ECS, graphics, input, effects, animation, isometric)
├── entities/   # Game entities (player, companion, enemy)
├── game/       # Game state, combat, save system, Steam integration
└── scenes/     # Game scenes (title, town, dungeon, dream)
```

### Adding a New Entity
1. Create file in `internal/entities/`
2. Embed `engine.BaseEntity` + `engine.AnimatedEntity`
3. Implement `Update()` and `Draw()`
4. Add animation clips using the Builder API
5. Register in entity factory / scene
6. Write tests in `*_test.go`

### Adding a New Scene
1. Create file in `internal/scenes/`
2. Implement `engine.Scene` interface
3. Register in `main.go`
4. Add transitions with `engine.NewTransition()`
5. Write tests

### Adding a New Animation
```go
// In code:
clip := engine.BeginAnim("my_anim").
    Frames([]int{0, 1, 2}, 0.1, true, false).
    Prop("scale", 1.0, 1.5, 0.3, engine.EaseOutBounce).
    Build()

// Or in JSON (mods):
// { "name": "my_anim", "frame_indices": [0,1,2], "frame_rate": 10, ... }
```

## 🔄 CI Pipeline (Free on GitHub Actions)

Every push/PR triggers:
1. **Lint** — `go vet`, `golangci-lint`, `staticcheck`, `gofmt`
2. **Test** — All tests + race detector + coverage ≥ 70%
3. **Build** — Cross-compile for Linux, Windows, macOS

All checks must pass before merging.