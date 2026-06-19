---
name: WASM codegen generators
overview: Implement three code generation tools—Python bindings, enum WAT globals, and struct-tagged WASM accessors—sharing a small internal AST/manifest package, with clear developer UX and phased Python API evolution.
todos:
  - id: shared-wasmcodegen
    content: Create src/internal/wasmcodegen (parse wasmexport funcs, //joulequestwasm: annotated funcs and enum consts, struct tags)
    status: pending
  - id: pybindgen-mvp
    content: "Implement cmd/wasm_pybindgen; golden tests; register tool in go.mod"
    status: pending
  - id: build-pybindgen
    content: Add Makefiles: target to generate bindings in client.py; target to run tinygo build → joulequest_core.wasm
    status: pending
  - id: python-package
    content: Add rl_agent/joulequest_wasm/ package, wasmtime dep, smoke test against joulequest.wasm
    status: pending
  - id: enum-wat-gen
    content: Implement cmd/enum_wat_gen, golden tests, register tool in go.mod
    status: pending
  - id: action-type
    content: Introduce type Action int32 in game/actions.go; annotate WASM-relevant enum types with joulequest:enum directives
    status: pending
  - id: wasm-merge-pipeline
    content: "Add Make targets: parse enums.wat, wasm-merge → joulequest.wasm, validate"
    status: pending
  - id: pybindgen-enums
    content: Extend pybindgen to emit enums.py + use Enum types in args and returns; update smoke tests and goldens
    status: pending
  - id: accessor-gen
    content: Implement cmd/wasm_exportgen, register tool in go.mod; 
    status: pending
  - id: struct-tags
    content: Annotate relevant structs (Game and nested + CompactParams); generate exports_game.go and exports_params.go to replace hand-written exports.go
    status: pending
  - id: interface-cleanup
    content: Remove Wazero joulequest_wasm_execute; update Python smoke tests to use new interface
    status: pending
isProject: false
---

# WASM Codegen Generators Plan

**Purpose:** generate Python bindings, exported accessor functions, and WASM enum definitions from metadata embedded in Go source code. Build system with Makefiles.

**We will explicitly not do:**
- Component Model / WIT / WAI bindgen (per prior analysis)
- Auto-extracting interface from `.wasm` binary
- Generating `state.go` game-control exports (stay hand-coded)

## Architecture overview

`go generate` will only be used to generate files in the same directory as the Go source. A Makefile build pipeline will drive the rest of the process.

Once all three phases are complete, the build pipeline will look like the following.

```mermaid
flowchart TB
  subgraph metadata [Go source annotations]
    Exports(["`//go:wasmexport` annotated functions])
    ExportCtl(["`//joulequestwasm:*`" annotated functions])
    Enums(["`//joulequestwasm:enumvalues` annotated const blocks"])
    StructTags(["`joulequestwasm:"..."` struct tags"])
  end
 
  subgraph generate ["go generate ./..."]
    %% artifacts
    ExportsGo["compact/wasm/exports*.go"]
    EnumsWat["**/enums.wat"]

    %% build steps
    ExportGen[["go generate cmd/wasm_exportgen"]]
    EnumWatGen[["go generate cmd/enum_wat_gen"]]

    %% annotated + wasmexported setters and getters to compact/wasm/exports_*.go
    StructTags --> ExportGen
    ExportGen --> ExportsGo

    %% WAT files defining immutable globals per enum value
    Enums --> EnumWatGen
    EnumWatGen --> EnumsWat
  end

  subgraph pybindings [Build Python bindings]
    %% artifacts
    ClientPy["rl_agent/joulequest_wasm/*.py"]

    %% build steps
    PyBindGen[["cmd/wasm_pybindgen"]]

    Exports --> PyBindGen
    ExportCtl --> PyBindGen
    Enums --> PyBindGen
    PyBindGen --> ClientPy
  end

  subgraph buildwasm [Build wasm binary]
    %% artifacts
    EnumsWasm["**/enums.wasm"]
    CoreWasm["joulequest_core.wasm"]
    FinalWasm["joulequest.wasm"]

    %% build steps
    SourceGo["handwritten .go source"]
    Assemble[["wasm-as"]]
    Merge[["wasm-merge"]]
    TinyGo[["tinygo build .compact/wasm"]]

    SourceGo --> TinyGo
    ExportsGo --> TinyGo
    TinyGo --> CoreWasm
    EnumsWat --> Assemble
    Assemble --> EnumsWasm
    CoreWasm --> Merge
    EnumsWasm --> Merge
    Merge --> FinalWasm
  end
```

**Shared library:** [`src/internal/wasmcodegen/`](src/internal/wasmcodegen/) — find `//go:wasmexport` and `//joulequestwasm:*` functions, enum types/values (reusing stringer-style const parsing), and struct-tag expansion. Each `cmd/*` tool is a thin CLI over this package.

**Register tools** in [`src/go.mod`](src/go.mod) `tool` block (like existing `stringer`):

```go
tool (
    golang.org/x/tools/cmd/stringer
    github.com/WillMorrison/JouleQuestCardGame/cmd/wasm_pybindgen
    github.com/WillMorrison/JouleQuestCardGame/cmd/enum_wat_gen
    github.com/WillMorrison/JouleQuestCardGame/cmd/wasm_exportgen
)
```

---

## Phase 1 (MVP): `wasm_pybindgen`

### Purpose

Scan [`src/compact/wasm/*.go`](src/compact/wasm/) for `//go:wasmexport` functions and emit a wasmtime-based Python client exposing them as methods. This replaces the need for a hand-maintained mirror of the WASM interface on the Python side. The Go Wazero glue code in [`src/cmd/joulequest_wasm_execute/wasm_module.go`](src/cmd/joulequest_wasm_execute/wasm_module.go) is out of scope and will not be autogenerated.

### Export classification rules

Any functions marked with `//go:wasmexport` will be converted into a Python method on the client class. No updates to existing Go code needed.

### Python API (phase 1 — raw integers)

- It should be possible to load the wasm module once and create several instances from it.
- Load-time checks that the wasm module exports the expected set of functions with the right signatures.
- Methods take and return raw integers; no enum types until phase 2.
- Methods with multiple parameters enforce keyword arguments with leading `*,`.
- Flat method space; no `@property` for scalar getters, no dotted fields for nested structs, no `collections.abc.Sequence` for players array, etc.

```python
from joulequest_wasm import JouleQuestWasm

with JouleQuestWasm("joulequest.wasm") as joulequest_module:
    game = joulequest_module.instance()
    game.reset(num_players=4)
    game.set_rng_seed(0)

    code = game.apply_action(player_index=0, action_int=14) # Keyword args enforced
    assert code == 0  # ErrCode.OK, still a bare int

    assert game.game_status() == 0
    assert game.round() >= 1
    assert game.num_players() == 4
    assert game.player_money(0) > 0
    assert game.possible_actions_mask(0) & (1 << 14)

    assert game.max_action() == 14
```

**Naming convention:** follow PEP8 and convert to snake_case method names; export `PlayerMoney` + `playerIndex` → method `player_money(self, player_index: int) -> int`.

**Lifecycle:** constructor loads module, `instance()` creates a new instance and calls `_initialize()` once (matching [README host contract](src/compact/wasm/README.md)). Context manager closes store/engine.

### Build Step

Since `go generate ./...` doesn't guarantee execution order (important after phase 3), and we're generating files outside of the source package, create a build pipeline (Makefile target) to run `wasm_pybindgen`.

```sh
# run from ./src
go tool wasm_pybindgen -pkg ./compact/wasm -o ../rl_agent/joulequest_wasm/
```

CLI flags: `-pkg` (package dir), `-o` (output path), `-class JouleQuestWasm` (default).

- Generated `rl_agent/joulequest_wasm/client.py` — client class definition
- Generated `rl_agent/joulequest_wasm/__init__.py` — re-export client class

### Tests

- Go golden tests in `cmd/wasm_pybindgen`: feed fixture `.go` snippets, compare emitted Python
- Smoke test script in `rl_agent/joulequest_wasm/smoke_test.py` that loads a built `joulequest.wasm` and calls `reset` + a few getters

### One Time Updates

- Add `wasmtime` to [`rl_agent/pyproject.toml`](rl_agent/pyproject.toml)
- Document how to use build pipeline to generate the bindings in the top level README.

---

## Phase 2: Enum types from `wasm_pybindgen`; `enum_wat_gen` + Binaryen merge pipeline

### Purpose

Keep Go enum values authoritative; `wasm_pybindgen` generates matching Python `enum.IntEnum` classes for const blocks (no per-constant getter functions).

Add enums as **exported immutable `i32` globals** in the final `.wasm`, and validate name/value matches at wasm module load time as a secondary nice-to-have.

### Enum scope

Types used by WASM-visible compact structs plus action/error codes:

| Package | Type | Used in |
|---|---|---|
| `game` | `Action` (new typed alias — see below) | `ApplyAction`/`CanPerformAction` args |
| `game` | `ErrCode` | `Reset`/`ApplyAction` returns |
| `core` | `GameStatus`, `PlayerStatus`, `LossCondition` | `Game`, `Player` structs |
| `core` | `PriceVolatility`, `GridStability` | `Snapshot` structs |
| `params` | `CapacityRule`, `CarbonTaxRule`, `WinConditionRule`, `GenerationConstraintRule`, `TakeoverRule` | `CompactParams` structs |

**Prerequisite:** introduce `type Action int32` in [`actions.go`](src/compact/game/actions.go) and re-home bare action constants under it (same numeric values). This mirrors `ErrCode` and enables `enum_wat_gen` on the block.

### Developer-facing UX

| Annotation | Annotated code | `wasm_pybindgen` | `enum_wat_gen` |
|---|---|---|---|
| `//joulequestwasm:enumvalues type=GameStatus trimprefix=GameStatus` | `const` block defining values of an enum type | `enum.IntEnum` class definition | `enums.wat` file with globals |
| `//joulequestwasm:enum core.GameStatus` | exported function with `int32` return | getter method returns the corresponding `Enum` type | None |
| `//joulequestwasm:enumarg volatility=core.PriceVolatility` | exported function with the named `int32` argument | setter method argument has the `Enum` type | None |

Handling of `type` and `trimprefix` mirrors [stringer flags](https://pkg.go.dev/golang.org/x/tools/cmd/stringer).

```go
// compact/game/actions.go
type Action int32

//joulequestwasm:enumvalues type=Action trimprefix=Action
const (
    ActionBuildRenewable Action = iota
    ...
    ActionFinished
)
```

These annotations can co-exist with stringer:

```go
// core/core.go
type GameStatus int

//joulequestwasm:enumvalues type=GameStatus trimprefix=GameStatus
//go:generate go tool stringer -type=GameStatus -trimprefix=GameStatus
const ( ... )
```

Function annotations drive `wasm_pybindgen` method generation:

```go
//joulequestwasm:enum core.GameStatus
//go:wasmexport GameStatus
func GameStatus() int32 {
    return int32(gGame.Status)
}

//joulequestwasm:enum core.PlayerStatus
//go:wasmexport PlayerStatus
func PlayerStatus(playerIndex int32) int32 {
    return int32(gGame.Players[playerIndex].Status)
}

//joulequestwasm:enumarg actionInt=game.Action
//joulequestwasm:enum game.ErrCode
//go:wasmexport ApplyAction
func ApplyAction(playerIndex int32, actionInt int32) int32 {
	return int32(gGame.ApplyPlayerAction(playerIndex, actionInt))
}
```

### `enum_wat_gen` outputs WAT

One `//go:generate go tool enum_wat_gen` line per package generates `enums.wat` in the package directory (e.g. `compact/game/enums.wat`, `core/enums.wat`), containing all the enums defined in `//joulequestwasm:enumvalues`-annotated `const` blocks. 

Shared const-parsing logic lives in `internal/wasmcodegen/enum.go`.

```wat
(module
  ;; game.Action
  (global $enum_game_Action_Finished i32 (i32.const 14))
  (export "enum_game_Action_Finished" (global $enum_game_Action_Finished))
  ;; ... one global+export per const
)
```

Export name scheme: `enum_<packageName>_<Type>_<TrimmedConstName>` (package name from import path tail, e.g. `game`, `core`, `params`).

### `wasm_pybindgen` phase-2 additions

- Generate new `rl_agent/joulequest_wasm/enums.py`:
  - One `IntEnum` per Go type (`Action`, `GameStatus`, …)
  - Members named from const names with `trimprefix` applied, then PEP8 upper snake (`ActionBuildRenewable` → `BUILD_RENEWABLE`)
- Updated generation of `rl_agent/joulequest_wasm/client.py`:
  - Getters & setters now use generated enum types from `//joulequestwasm:enum` and `//joulequestwasm:enumarg`
  - Load-time checks also ensure that the wasm module exports the expected set of enum globals with values matching the `IntEnum` classes.

### Python API (phase 2 — typed enums)

```python
from joulequest_wasm import JouleQuestWasm
from joulequest_wasm.enums import Action, ErrCode, GameStatus, PlayerStatus

with JouleQuestWasm("joulequest.wasm") as joulequest_module:
    game = joulequest_module.instance()
    game.reset(4)
    code = game.apply_action(player_index=0, action_int=Action.FINISHED)
    assert code == ErrCode.OK

    assert game.game_status() == GameStatus.ONGOING
    assert game.player_status(0) == PlayerStatus.ACTIVE
```

### Build step

New Makefile targets parse `.wat` to `.wasm`, build the core reactor module, and combine everything into one:

```bash
# Build step - compile reactor
tinygo build -size short -gc=none -no-debug -scheduler=none -panic=trap \
  -target=wasm-unknown -o compact/wasm/joulequest_core.wasm ./compact/wasm

# Build steps - convert enum wat to wasm
wasm-tools parse compact/game/enums.wat -o compact/game/enums.wasm
wasm-tools parse compact/params/enums.wat -o compact/params/enums.wasm
wasm-tools parse core/enums.wat -o core/enums.wasm

# Build step: combine everything into one wasm binary
wasm-merge compact/wasm/joulequest_core.wasm main \
           compact/game/enums.wasm game \
           compact/params/enums.wasm params \
           core/enums.wasm core \
           -o compact/wasm/joulequest.wasm

# Optional post-build checks
wasm-tools validate compact/wasm/joulequest.wasm
wasm-tools print compact/wasm/joulequest.wasm | rg 'export "enum_'
```

### One Time Updates

- Introduce `type Action int32` and update const definitions
- Annotate existing exported functions and const blocks with enum metadata
- Document [Binaryen](https://github.com/WebAssembly/binaryen) and [`wasm-tools`](https://github.com/bytecodealliance/wasm-tools) as a dev dependency in [`compact/wasm/README.md`](src/compact/wasm/README.md).
- Remove redundant `MaxAction()` export once globals exist.
- Update smoke test to use new `Enum` types.

---

## Phase 3: `wasm_exportgen`

### Purpose

Generate accessors for global struct variables from struct field tags so adding a field to `Game` / `Player` / `CompactParams` automatically creates exported WASM getters (and param setters) that `wasm_pybindgen` picks up on the next build. The names can differ from the current accessor names, since the bindings will be regenerated and don't require updates by hand.

Control functions remain hand-coded in `state.go`.

### Struct tag vocabulary

Tags on fields of [`game.Game`](src/compact/game/game.go), [`game.Player`](src/compact/game/player.go), [`params.CompactParams`](src/compact/params/params.go):

| Tag | Meaning | Notes |
|---|---|---|
| `joulequestwasm:"export"` | scalar getter on current receiver | Only valid on types castable to `int32`. Applied after `index=` to handle arrays. |
| `joulequestwasm:"set"` | scalar setter on current receiver | Only valid on types castable to `int32`. Applied after `index=` to handle arrays. |
| `joulequestwasm:"enum"` | add comment so `wasm_pybindgen` wraps getter return or setter arg as enum. | Only valid with `export` or `set`. | 
| `joulequestwasm:"nest"` | exports on nested struct fields have this field's name as a prefix. | Only valid on structs. Applied after `index=` to handle arrays of structs. |
| `joulequestwasm:"index=playerIndex"` | exported getters and setters have an index argument. | Only valid on arrays. Applied first. |
| `joulequestwasm:"indexenum=core.PriceVolatility"` | add comment so `wasm_pybindgen` wraps index arg as enum. | Only valid with `index=`. |

#### Example tagged structs

```go
// compact/game/...
type Game struct {
    Status          core.GameStatus `joulequestwasm:"export,enum"`
    Reason          core.LossCondition `joulequestwasm:"export,enum"`
    Round           int32 `joulequestwasm:"export"`
    CarbonEmissions int32 `joulequestwasm:"export"`
    NumPlayers      int32 `joulequestwasm:"export"`
    Players         [cparams.MaxPlayers]Player `joulequestwasm:"nest,index=playerIndex"`
    TakeoverPool    assets.AssetMix `joulequestwasm:"nest"`
    LastSnapshot    Snapshot `joulequestwasm:"nest"`
    // Params: no tag → not exported
    // phase, pcg: unexported fields inaccessible to reflection → not exported
}

type Player struct {
    Status     core.PlayerStatus `joulequestwasm:"export,enum"`
    Reason     core.LossCondition `joulequestwasm:"export,enum"`
    Money      int32 `joulequestwasm:"export"`
    Mix        assets.AssetMix `joulequestwasm:"nest"`
    IsBuilding bool  // no tag → not exported
}

type Snapshot struct {
	AssetMix        assets.AssetMix `joulequestwasm:"nest"`
	PriceVolatility core.PriceVolatility `joulequestwasm:"export,enum"`
	GridStability   core.GridStability `joulequestwasm:"export,enum"`
}

// compact/params/params.go — setters for training-time param mutation
type CompactParams struct {
    TakeoverRule params.TakeoverRule `joulequestwasm:"export,set,enum"`
    InitialCash  int32               `joulequestwasm:"export,set"`
    RenewablePnL [4]int32            `joulequestwasm:"export,set,index=volatility,indexenum=core.PriceVolatility"`
    // other fields ...
}

// assets/asset_mix.go 
type AssetMix struct {
	Renewables         int `joulequestwasm:"export"`
	BatteriesArbitrage int `joulequestwasm:"export"`
	BatteriesCapacity  int `joulequestwasm:"export"`
	FossilsWholesale   int `joulequestwasm:"export"`
	FossilsCapacity    int `joulequestwasm:"export"`
}
```

### Generator invocation

In `compact/wasm/state.go`:

```go
var (
  //go:generate go tool wasm_exportgen -prefix=Params -o exports_params.go
	gParams params.CompactParams = params.Default

  //go:generate go tool wasm_exportgen -prefix=Game -o exports_game.go
	gGame   game.Game
)
```

Generated files use `// Code generated by wasm_exportgen; DO NOT EDIT.` and are written to the file specified via `-o`. 

#### Example generator output 

Generated Go demonstrating the outcome of the combined struct tags:

```go
//go:wasmexport GameRound
func GameRound() int32 {
    return int32(gGame.Round)
}

//joulequestwasm:enum core.GameStatus
//go:wasmexport GameStatus
func GameStatus() int32 {
    return int32(gGame.Status)
}

//go:wasmexport GamePlayersMoney
func GamePlayersMoney(playerIndex int32) int32 {
    return int32(gGame.Players[playerIndex].Money)
}

//joulequestwasm:enum core.PlayerStatus
//go:wasmexport GamePlayersStatus
func GamePlayersStatus(playerIndex int32) int32 {
    return int32(gGame.Players[playerIndex].Status)
}

//go:wasmexport GamePlayersMixRenewables
func GamePlayersMixRenewables(playerIndex int32) int32 {
  return int32(gGame.Players[playerIndex].Mix.Renewables)
}

//go:wasmexport ParamsInitialCash
func ParamsInitialCash() int32 {
  return int32(gParams.InitialCash)
}

//go:wasmexport SetParamsInitialCash
func SetParamsInitialCash(val int32) {
  gParams.InitialCash = val
}

//joulequestwasm:enum params.TakeoverRule
//go:wasmexport ParamsTakeoverRule
func ParamsTakeoverRule() int32 {
  return int32(gParams.TakeoverRule)
}

//joulequestwasm:enumarg val=params.TakeoverRule
//go:wasmexport SetParamsTakeoverRule
func SetParamsTakeoverRule(val int32) {
  gParams.TakeoverRule = params.TakeoverRule(val)
}

//joulequestwasm:enumarg volatility=core.PriceVolatility
//go:wasmexport ParamsRenewablePnL
func ParamsRenewablePnL(volatility int32) int32 {
  return int32(gParams.RenewablePnL[volatility])
}

//joulequestwasm:enumarg volatility=core.PriceVolatility
//go:wasmexport SetParamsRenewablePnL
func SetParamsRenewablePnL(volatility int32, val int32) {
  gParams.RenewablePnL[volatility] = val
}
```

### One Time Updates

- Annotate struct fields with tags.
- Delete `src/cmd/joulequest_wasm_execute/` as it will be broken by the interface change.
- Update the smoke test to use newly generated setters.
- Hand-written query helpers (`PossibleActionsMask`, `CanPerformAction`) stay in `exports.go`, other exported accessor functions are now obsolete and get deleted.
