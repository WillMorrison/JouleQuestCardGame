# WASM training interface

This document describes the **in-process WebAssembly (WASI)** bridge between the JouleQuest game engine (Go) and the RL training stack (Python). It complements the top-level [README.md](README.md). Cursor rules live under [.cursor/rules/](.cursor/rules/) — including [`joulequest-wasm-training.mdc`](.cursor/rules/joulequest-wasm-training.mdc) for this bridge and [`rl-agent-python.mdc`](.cursor/rules/rl-agent-python.mdc) for general Python in `rl_agent/`.

## Problem

Training uses Python in [`rl_agent/`](rl_agent/) and today talks to the Go game through the **OpenAPI REST server** ([`src/cmd/rest_api/`](src/cmd/rest_api/)) over a Unix socket, via a synchronous **httpx** client ([`rl_agent/game_client.py`](rl_agent/game_client.py)). That pattern tends to be **single-threaded on the Python side**, and pays repeated **JSON encode/decode** cost for every action and observation.

## Goal

Provide a **similar conceptual API** for Python as the OpenAPI server (reset game, submit actions, read state and legal actions), but backed by a **WASI module** loaded in-process (e.g. **wasmtime**), with **scalar** host calls instead of REST bodies.

## Architectural approach

```mermaid
flowchart LR
  subgraph today [Current]
    Py1[PettingZoo / training]
    GC1[GameClient + httpx]
    API[rest_api OpenAPI]
    Eng1[engine.GameState]
    Py1 --> GC1 --> API --> Eng1
  end
  subgraph target [Target]
    Py2[PettingZoo / training]
    GC2[GameClientWasm + wasmtime]
    Wasm[JouleQuest WASI module]
    Eng2[Compact alloc-light engine]
    Py2 --> GC2 --> Wasm --> Eng2
  end
```

- **Client-driven control**: mirror [`ProceduralGameState`](src/engine/procedural.go) — the host chooses build-phase actions; the engine advances until the next decision point (same inversion as moving away from `GetPlayerAction` callbacks).
- **One module instance per worker**: no need for goroutines, channels, or concurrency **inside** the WASM library; Python may spawn one instance per process/thread as needed.
- **Singleton state in the module**: a single global **params** struct and a single global **game** struct, with exported functions to reset, apply an action, and read fields.
- **No event log** on this path: training does not rely on [`get_log`](rl_agent/game_client.py); the WASM build can omit logging entirely.

## Contract parity (what RL can observe)

The WASM exports should match **what training already sees through the API**, not every internal detail of [`GameState`](src/engine/game_state.go).

The REST layer defines the relevant shape. For example, [`stateResponse`](src/cmd/rest_api/main.go) includes:

- Game: status, reason, round, emissions counter  
- Players: serialized [`PlayerState`](src/engine/game_state.go) (money, status, holdings as **`assets.AssetMix`**)  
- Last round snapshot  
- **`TakeoverPool` as `assets.AssetMix`**

There is **no ordering** for takeover-pool assets on the wire — only **counts per bucket**. Parity tests and the compact engine should treat that **AssetMix-shaped multiset** as canonical for the pool.

The reference `GameState` / `ProceduralGameState` code paths are a **mechanical oracle** for tests; they grew through exploratory design and may have internal quirks (e.g. slice order). If tests disagree with the compact engine only on **non-observable** internals, **escalate to the maintainer** rather than papering over differences; the reference implementation may be updated so parity stays aligned with the API contract.

## WASM export shape (planned)

- **Parameters and returns**: fixed-width integers (`int32` / `uint32` as needed). Prefer **one `int32` return** per function (data or error code) and **integer-only arguments**, as required for a simple host FFI.
- **Actions**: encode build-phase choices in the same **0–14 integer space** as [`rl_agent/custom_environment/env/joulequest_env.py`](rl_agent/custom_environment/env/joulequest_env.py) (`PlayerActionToInt`).
- **Legal actions**: expose a **bitfield per player** (15 bits) instead of marshaling a list of `PlayerAction` structs.
- **Player-scoped getters** take an explicit **player index**; include **number of players**.

Errors map to **integer codes** (invalid action, wrong phase, bad index, etc.), not JSON error strings.

## Takeover rules (design alignment)

- **[`TakeoverRuleForcedTakeover`](src/params/params.go)**: pool must be **empty** (all mix counts zero) before a player may finish the build round when the rules require it; stale assets in the pool with no affordable takeover lead to the same loss outcomes as today.
- **[`TakeoverRuleVirtualOwner`](src/params/params.go)**: unowned pool assets contribute to **grid / emissions**; the **five-bucket** mix (including wholesale vs capacity) matters for those calculations.
- **Takeover into a portfolio**: assets enter the player’s portfolio in **default mode**, consistent with current `assets.New` behavior after takeover.

## Building the module (Go 1.24+)

The implementation will use Go’s **WASI** port and **`//go:wasmexport`** for exported functions, built as a **reactor** (library-style module) so the host can call exports repeatedly after a single initialization.

Authoritative upstream documentation:

- [Extensible Wasm Applications with Go](https://go.dev/blog/wasmexport) — `//go:wasmexport`, `GOOS=wasip1 GOARCH=wasm`, and **`-buildmode=c-shared`** for a reactor; the host must call **`_initialize`** before invoking exports.

Exact `go build` flags and the `cmd/` entrypoint will be documented next to the WASM command once it exists ([`src/cmd/joulequest_wasm/`](src/cmd/joulequest_wasm/) per plan).

## Python side (planned)

- Use **uv** for **rl_agent** installs and scripts (see [`.cursor/rules/rl-agent-python.mdc`](.cursor/rules/rl-agent-python.mdc)).
- A **GameClientWasm** (or equivalent) accepts a **wasmtime** module/instance and implements the same operations the PettingZoo env needs: reset, step with action int, read state and masks. **`get_log`** may return an empty string or be omitted for training.
- **Enums**: generated or shared definitions kept in sync with Go (see plan: `go generate` + small Go tool emitting Python).

## Tradeoffs and follow-ups

- **Binary size**: the first milestone may use the **official Go** WASI toolchain; if the module is too large, evaluate **TinyGo** with the same compact engine (may require stricter Go subset).
- **Randomness**: [`OperatePhase`](src/engine/operation.go) today uses **`math/rand`** globally; the compact engine and WASM build will use **explicit, seedable RNG state** so tests and RL runs are reproducible.

## Roadmap (remainder of project)

Not implemented in the step that produced this document; tracked in the repo plan *WASM FFI training bridge*:

1. **Compact engine** — fixed max players, `AssetMix`-style holdings and takeover pool, mask-based legal actions, no event logger.  
2. **Go parity tests** — same seeds and action streams vs `ProceduralGameState`, comparing **OpenAPI-visible** fields and masks.  
3. **`cmd/joulequest_wasm`** — WASI reactor + `//go:wasmexport` API.  
4. **Python wasmtime client** + optional **enum codegen** from Go.
