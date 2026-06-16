package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// wasmModule wraps a loaded joulequest_wasm reactor instance.
type wasmModule struct {
	r   wazero.Runtime
	mod api.Module
}

func loadWasmModule(ctx context.Context, path string) (*wasmModule, error) {
	wasmBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read wasm: %w", err)
	}

	r := wazero.NewRuntime(ctx)

	// WASI only needed for wasm files created by go build, not tinygo
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("instantiate wasi: %w", err)
	}

	mod, err := r.InstantiateWithConfig(ctx, wasmBytes, wazero.NewModuleConfig().
		WithStartFunctions("_initialize"))
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("instantiate module: %w", err)
	}

	return &wasmModule{r: r, mod: mod}, nil
}

func (m *wasmModule) callI32(ctx context.Context, name string, args ...uint64) (int32, error) {
	fn := m.mod.ExportedFunction(name)
	if fn == nil {
		return 0, fmt.Errorf("export %q not found", name)
	}
	results, err := fn.Call(ctx, args...)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	if len(results) == 0 {
		return 0, nil
	}
	return int32(results[0]), nil
}

func (m *wasmModule) Reset(ctx context.Context, numPlayers int32) (int32, error) {
	return m.callI32(ctx, "Reset", uint64(numPlayers))
}

func (m *wasmModule) SetRNGSeed(ctx context.Context, seed int32) (int32, error) {
	return m.callI32(ctx, "SetRNGSeed", uint64(uint32(seed)))
}

func (m *wasmModule) ApplyAction(ctx context.Context, playerIndex, action int32) (int32, error) {
	return m.callI32(ctx, "ApplyAction", uint64(playerIndex), uint64(action))
}

func (m *wasmModule) NumPlayers(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "NumPlayers")
}

func (m *wasmModule) GameStatus(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "GameStatus")
}

func (m *wasmModule) GameReason(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "GameReason")
}

func (m *wasmModule) Round(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "Round")
}

func (m *wasmModule) CarbonEmissions(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "CarbonEmissions")
}

func (m *wasmModule) PossibleActionsMask(ctx context.Context, playerIndex int32) (uint32, error) {
	v, err := m.callI32(ctx, "PossibleActionsMask", uint64(playerIndex))
	return uint32(v), err
}

func (m *wasmModule) PlayerMoney(ctx context.Context, playerIndex int32) (int32, error) {
	return m.callI32(ctx, "PlayerMoney", uint64(playerIndex))
}

func (m *wasmModule) PlayerStatus(ctx context.Context, playerIndex int32) (int32, error) {
	return m.callI32(ctx, "PlayerStatus", uint64(playerIndex))
}

func (m *wasmModule) PlayerLossReason(ctx context.Context, playerIndex int32) (int32, error) {
	return m.callI32(ctx, "PlayerLossReason", uint64(playerIndex))
}

func (m *wasmModule) PlayerRenewableAssets(ctx context.Context, playerIndex int32) (int32, error) {
	return m.callI32(ctx, "PlayerRenewableAssets", uint64(playerIndex))
}

func (m *wasmModule) PlayerBatteriesArbitrageAssets(ctx context.Context, playerIndex int32) (int32, error) {
	return m.callI32(ctx, "PlayerBatteriesArbitrageAssets", uint64(playerIndex))
}

func (m *wasmModule) PlayerBatteriesCapacityAssets(ctx context.Context, playerIndex int32) (int32, error) {
	return m.callI32(ctx, "PlayerBatteriesCapacityAssets", uint64(playerIndex))
}

func (m *wasmModule) PlayerFossilsWholesaleAssets(ctx context.Context, playerIndex int32) (int32, error) {
	return m.callI32(ctx, "PlayerFossilsWholesaleAssets", uint64(playerIndex))
}

func (m *wasmModule) PlayerFossilsCapacityAssets(ctx context.Context, playerIndex int32) (int32, error) {
	return m.callI32(ctx, "PlayerFossilsCapacityAssets", uint64(playerIndex))
}

func (m *wasmModule) TakeoverRenewableAssets(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "TakeoverRenewableAssets")
}

func (m *wasmModule) TakeoverBatteriesArbitrageAssets(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "TakeoverBatteriesArbitrageAssets")
}

func (m *wasmModule) TakeoverBatteriesCapacityAssets(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "TakeoverBatteriesCapacityAssets")
}

func (m *wasmModule) TakeoverFossilsWholesaleAssets(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "TakeoverFossilsWholesaleAssets")
}

func (m *wasmModule) TakeoverFossilsCapacityAssets(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "TakeoverFossilsCapacityAssets")
}

func (m *wasmModule) LastSnapshotPriceVolatility(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "LastSnapshotPriceVolatility")
}

func (m *wasmModule) LastSnapshotGridStability(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "LastSnapshotGridStability")
}

func (m *wasmModule) LastSnapshotRenewableAssets(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "LastSnapshotRenewableAssets")
}

func (m *wasmModule) LastSnapshotBatteriesArbitrageAssets(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "LastSnapshotBatteriesArbitrageAssets")
}

func (m *wasmModule) LastSnapshotBatteriesCapacityAssets(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "LastSnapshotBatteriesCapacityAssets")
}

func (m *wasmModule) LastSnapshotFossilsWholesaleAssets(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "LastSnapshotFossilsWholesaleAssets")
}

func (m *wasmModule) LastSnapshotFossilsCapacityAssets(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "LastSnapshotFossilsCapacityAssets")
}

func (m *wasmModule) MaxAction(ctx context.Context) (int32, error) {
	return m.callI32(ctx, "MaxAction")
}
