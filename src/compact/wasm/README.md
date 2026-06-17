# wasm

WASM **reactor** module exposing the compact game engine to Python (wasmtime) via `//go:wasmexport` functions with **int32-only** signatures.

## Host contract

1. Load the `.wasm` module into a wasmtime (or compatible) instance.
2. Call `_initialize()` once before any other export (Go runtime / package init).
3. Call `Reset(numPlayers)` to (re)start the game.
4. Read state via scalar getters (`GameStatus`, `PlayerMoney`, `PossibleActionsMask`, etc.).
5. Step with `ApplyAction(playerIndex, actionInt)` (action ints 0–14, same encoding as PettingZoo `PlayerActionToInt`).

## Build (TinyGo artifact)

Building the . Requires TinyGo ≥ 0.34 (`//go:wasmexport` support).

From `src/`:

```bash
tinygo build -size short -gc=none -no-debug -scheduler=none -panic=trap -target=wasm-unknown \
  -o joulequest.wasm \
  ./compact/wasm
```

`wasm-unknown` is a reactor-style target without WASI: the host keeps the module loaded and calls exports repeatedly.

## Tests 

Go tests exercise the interface without a WASM build.

```bash
go test ./compact/wasm/...
```

Wazero-based executor exercises the interface on a pre-built WASM module. This should play through a single game, picking random actions.

```bash
go run ./cmd/joulequest_wasm_execute -wasm joulequest.wasm
```
