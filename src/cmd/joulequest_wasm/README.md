# joulequest_wasm

WASI **reactor** module exposing the compact game engine to Python (wasmtime) via `//go:wasmexport` functions with **int32-only** signatures.

## Host contract

1. Load the `.wasm` module into a wasmtime (or compatible) instance.
2. Call `_initialize()` once before any other export (Go runtime / package init).
3. Call `Reset(numPlayers)` to (re)start the game.
4. Step with `ApplyAction(playerIndex, actionInt)` (action ints 0–14, same encoding as PettingZoo `PlayerActionToInt`).
5. Read state via scalar getters (`GameStatus`, `PlayerMoney`, `PossibleActionsMask`, etc.).


### Error codes (`ApplyAction`, `Reset`, `InitDefaultParams`)

| Code | Meaning |
|------|---------|
| 0 | OK |
| 1 | Invalid player count |
| 2 | No starting fossils for player count in params |
| 3 | Action only valid in build phase |
| 4 | Action not allowed for player |
| 5 | Params not initialized (`InitDefaultParams` not called) |
| 6 | Internal / unexpected error |

## Build (Go 1.24+ reference / CI)

From `src/`:

```bash
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o joulequest.wasm ./cmd/joulequest_wasm
```

Produces a **reactor** binary: `_initialize` + exports, no automatic `main` run.

## Build (TinyGo training artifact)

Planned shipping path for RL (smaller binary). Requires TinyGo ≥ 0.34 (`//go:wasmexport` support).

From `src/`:

```bash
tinygo build -size short -gc=leaking -no-debug -scheduler=none -panic=trap \
  -target=wasm-unknown \
  -o joulequest.wasm \
  ./cmd/joulequest_wasm
```

`wasm-unknown` is a reactor-style target: the host keeps the module loaded and calls exports repeatedly. No WASI `_initialize` on this path—follow TinyGo/wasmtime embedding notes for your runtime version.

## Tests (native Go)

```bash
go test ./cmd/joulequest_wasm/...
```

Tests exercise the same logic without a WASM build.
