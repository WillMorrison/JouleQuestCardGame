# Testing plan: `src/compact/*`

This document is a **plan only** (no test implementations). It covers **parity** tests against the reference engine/OpenAPI-visible surface, and **behavioral / executable-spec** tests for [`compact/game`](game) and [`compact/params`](params). It aligns with the WASM training plan ([`.cursor/plans/wasm_ffi_training_bridge_21345f9c.plan.md`](../../.cursor/plans/wasm_ffi_training_bridge_21345f9c.plan.md)) and [WASM_TRAINING_INTERFACE.md](../../WASM_TRAINING_INTERFACE.md).

---

## 1. Goals

| Track | Purpose |
|--------|---------|
| **Parity** | Confidence that compact output matches what training already observes via REST (and future WASM getters), not slice-internals of `GameState`. |
| **Specification** | Tests as living documentation: build rules, masks, takeover semantics, operate phase, phase machine, RNG. |
| **Safety** | Invalid inputs do not corrupt state; bounds and error paths are explicit. |

Escalation when parity disagrees for non-observable reasons: see [.cursor/rules/joulequest-wasm-training.mdc](../../.cursor/rules/joulequest-wasm-training.mdc).

---

## 2. Layout and conventions

- **Packages**: tests live next to code — `compact/game/*_test.go`, `compact/params/*_test.go`. Optional `compact/game/parity_test.go` (or `package game_test` in `parity_test.go`) if you want parity imports isolated.
- **Patterns**: table-driven tests where many cases differ only on params or actions; helper `mustNewGame(t, players, params)` and `snapshotGame(t, g)` for comparable structs.
- **Naming**: prefer names that read as rules, e.g. `TestInactivePlayersHaveNoPossibleActions`, `TestApplyPlayerAction_InvalidActionDoesNotMutateState`.
- **Determinism**: any test touching operate-phase risk must **fix PCG** via [`SetRNGSeed`](game/operate.go) (or agreed seed) so runs are reproducible.

---

## 3. Parity tests (reference vs compact)

### 3.1 Motivation and ground truth

- **Oracle**: drive [`engine.ProceduralGameState`](../../engine/procedural.go) with a no-op / discarding [`eventlog.Logger`](../../eventlog/eventlog.go) and the same **ordered list of `(playerIndex, actionCode)`** as [`compact/game.Game.ApplyPlayerAction`](game/game.go).
- **Compare**: only fields on the **OpenAPI / RL observable** surface (see [`stateResponse`](../../cmd/rest_api/main.go) and [plan § parity](../../.cursor/plans/wasm_ffi_training_bridge_21345f9c.plan.md)): status, reason, round, emissions, last snapshot mix + `PriceVolatility` + `GridStability`, per-player status, money, **asset mix** (same derivation as JSON: aggregate from holdings, not slice order), **takeover pool `AssetMix`**, and **per-player `PossibleActionMask`** (15 bits, same encoding as PettingZoo / `PlayerActionToInt`).
- **Do not** assert on raw `[]Asset` / takeover slice order.

### 3.2 RNG alignment

- Compact uses [`math/rand/v2.PCG`](game/operate.go) with [`SetRNGSeed`](game/operate.go). Reference uses [`math/rand.Intn`](../../engine/operation.go) on the global source.
- **Plan**: either (a) The owner will refactor the legacy `engine` operate path to draw randomness from a PGC in `GameState` and provide a public `SeedRNG` method so parity tests can inject the same sequence as PCG, or (b) compare branches that **do not** draw risk until a shared story exists. **Document the chosen approach** in test helpers; if blocked, stop and escalate (per project rules).

### 3.3 Scenario ladder

1. **Minimal**: default [`params.Default`](../../params/params_default.go), 2 players, short scripted sequence (e.g. double-`Finished` through one operate) — smoke parity after RNG story is settled.
2. **Takeover / forced rule**: pool non-empty, affordable vs unaffordable takeovers, global loss `UnownedTakeoverAssets` vs `VirtualOwner` finish behavior.
3. **Build / pledge / scrap**: costs, capacity market on/off via [`params.Builder`](../../params/builder.go) into `CompactParams` via [`FromLegacy`](params/params.go).
4. **Operate outcomes**: bankruptcy → pool mix; generation cap; grid instability; emissions cap; win / last-fossil branch — each paired with **fixed seed** once RNG is aligned.
5. **Stress**: long pseudo-random legal action stream (fuzz-style driver) comparing observables every step (nightly or `-short` skip).

### 3.4 Failure triage

| Symptom | Action |
|---------|--------|
| Mixes and masks match; only slice order differs | Treat as reference bug or document; do not weaken compact tests. |
| Global `rand` vs PCG mismatch | Fix reference hook or narrow test until aligned. |
| Ambiguous rule intent | Ask maintainer; capture answer in test name or comment. |

### 3.5 Specific scenarios

- **Mask vs OpenAPI set**: for small hand-built states, expand mask to a set of `(playerIndex, actionCode)` and compare to `ProceduralGameState.PossibleActions()` **as sets** (order irrelevant), for the same params when parity harness exists. Alternatively, write a helper to convert `ProceduralGameState.PossibleActions()` to a set of `(playerIndex, actionMask)` and compare those with the corresponding pairs from the compact implementation.
- **Takeover pool removals**: The new implementation of scrap and takeover prefer **arbitrage before capacity** per type. The old slice implementation took the first element with a matching type from the pool, which made it dependent on the order of addition to the pool. These parity tests should just check that the combined number of arbitrage+capacity battery and fossil assets remaining in the takeover pool matches the legacy implementation, so that they are not sensitive to order-related mode differences.

---

## 4. Behavioral / specification tests (`compact/game`)

These document **rules of the game** independent of the reference implementation.

### 4.1 Phase machine and build entry

- After [`NewGame`](game/game.go): `Phase` is build (post-`startBuildPhase`), `Round()` is `1`, `GameStatus` ongoing, all in-range players active and `IsBuilding` true with correct starting fossil **wholesale** counts and money from params.
- **`startBuildPhase`**: only **active** players get `IsBuilding = true` and [`resetModesForBuild`](game/player.go) (wholesale absorbs capacity for battery/fossil; renewables unchanged). **Lost** players must not be flipped back to building.
- **Operate → next build**: after a full operate when game still ongoing, round increments and build flags reset as today.

### 4.2 Possible actions and masks

- **Inactive / lost players**: [`PossibleActionMask`](game/actions.go) is **0** for indices in `[0, NumPlayers)` when `PlayerStatusLost` or not building as required.
- **Out-of-range index** (document expected behavior): `PossibleActionMask` for `pi < 0` or `pi >= NumPlayers` returns **0** (current behavior); test documents contract for future WASM.


### 4.3 Takeover rules (build phase)

- **`TakeoverRuleForcedTakeover`**: with **non-empty** pool, **no** `ActionFinished` bit in mask for building players until pool empty; attempting finish when not in mask must be rejected (see §5).
- **`TakeoverRuleVirtualOwner`**: `ActionFinished` allowed with non-empty pool per rules.
- **Global deadlock**: when no player has any legal action and last applied action was not `Finished`, expect loss reason per [`ApplyPlayerAction`](game/game.go) (`UnownedTakeoverAssets` vs `NoActivePlayers`).

### 4.4 Apply / money / assets

- **Build / scrap / takeover / takeover-scrap / pledge**: each legal transition updates **money** and **five-bucket mixes** (player + pool) consistently with costs from [`CompactParams`](params/params.go).
- **Deterministic multiset removals**: scrap and pool removal prefer **arbitrage before capacity** per type (document with examples so future readers know why two orders of scrap can differ from legacy slices but match `AssetMix`).

### 4.5 Operate phase (compact-only until RNG parity)

- **Generation constraint** (min / max-decrease), **grid vs risk** loss, **emissions cap**, **per-player PnL** and bankruptcy → pool, **all-bankrupt loss**, **win / last fossil** branches — one focused test each with minimal fixtures.
- **Carbon tax / capacity rule variants**: table over [`CompactParams`](params/params.go) / `FromLegacy` builder combinations mirroring [`params.Params.PnL`](../../params/params.go) branches.

---

## 5. Error handling, bounds, and invariants (`compact/game` + `compact/params`)

### 5.1 `ApplyPlayerAction` must not mutate on failure

For each failure path, **capture state** (deep enough: status, reason, phase, round, emissions, all player mixes/money/status/building flags, pool mix, PCG could be checked via two draws if needed) **before** call, invoke `ApplyPlayerAction`, expect **error** and **bytes.Equal / cmp.Diff** unchanged state:

| Condition | Expected error (current) |
|-----------|---------------------------|
| Not build phase | `ErrNotBuildPhase` |
| `playerIndex < 0` or `>= NumPlayers` | `ErrInvalidAction` |
| `actionCode` not in mask (including **not** in `0..14`, e.g. `200`) | `ErrInvalidAction` |
| `applyActionCode` internal false (should not occur if mask correct) | `ErrInvalidAction` |

- **`actionCode(200)`** (and `-1`, `15`): must return `ErrInvalidAction` and **no** change to game fields and **no** advance via [`runUntilBuildPhase`](game/game.go).
- **Illegal but “in mask” inconsistency**: if tests ever construct impossible state, document; otherwise rely on mask+apply consistency.

### 5.2 `NewGame` / params

- **Player count**: `< 2`, `> MaxPlayers`, **missing starting fossil table** for `n` → `ErrInvalidPlayerCount` / `ErrNoStartingFossils`.
- **`FromLegacy`**: negative map values error; zero/omitted entries for player counts behave as today for `StartingFossils`.

### 5.3 Read-only accessors

- [`PlayerMoney`](game/game.go) / [`PlayerStatusI`](game/game.go) / [`PlayerAssetMix`](game/game.go): out-of-range indices — document **safe sentinel** behavior (`0` money, `Lost` status, empty mix) and test so WASM getters stay consistent.

---

## 6. `compact/params` tests

- **`FromLegacy`**: identity on `params.Default`; spot-check each PnL table int32; rule enum passthrough.
- **`OperatePnLForPlayerMix`**: tables for each **capacity rule** + **carbon tax** + **fossil wholesale vs capacity** counts vs legacy `Params.PnL` on synthetic `assets.Asset` slices (helper builds legacy state) for one volatility index and emissions threshold — ensures compact arithmetic matches reference **PnL** without running full engine.

---

## 7. Tooling and CI

- **`go test ./compact/...`**: must pass locally and in CI.
- **`-short`**: long random parity streams skip under `-short`.
- **Coverage**: prioritize `game` package branches over line %; optional coverage report in CI for regression diffs.
- **Allocation**: benchmark or other alloc-tracer which runs actions and resets the game to ensure that the implementation doesn't allocate memory to avoid WASM problems.

---

## 8. Out of scope (for this plan)

- Python / wasmtime integration tests (separate track).
- Fuzzing `ApplyPlayerAction` against invariants only (optional extension of §3.3).

---

## 9. Checklist summary

- [ ] Parity harness + observable field comparator + RNG story decided  
- [ ] Masks vs reference (set equality) on small cases  
- [ ] Inactive / lost → no actions  
- [ ] Forced takeover → cannot finish with pool non-empty  
- [ ] Build start → modes reset for active players  
- [ ] Invalid action / bad index / `actionCode(200)` → no state change  
- [ ] `NewGame` / params errors  
- [ ] Accessor out-of-range contract  
- [ ] Operate phase scenarios with fixed PCG seed  
- [ ] `OperatePnLForPlayerMix` vs legacy `PnL` tables
- [ ] Allocation check

When implementing, keep test names and comments aligned with this list so the suite reads as an **executable specification** of JouleQuest compact behavior.
