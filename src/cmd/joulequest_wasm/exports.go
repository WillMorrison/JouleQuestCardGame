package main

// Asset bucket indices for mix getters (player, takeover pool, last snapshot).
const (
	AssetBucketRenewables int32 = iota
	AssetBucketBatteriesArbitrage
	AssetBucketBatteriesCapacity
	AssetBucketFossilsWholesale
	AssetBucketFossilsCapacity
)

//go:wasmexport NumPlayers
func NumPlayers() int32 {
	return int32(gGame.NumPlayers)
}

//go:wasmexport GameStatus
func GameStatus() int32 {
	return int32(gGame.Status)
}

//go:wasmexport GameReason
func GameReason() int32 {
	return int32(gGame.Reason)
}

//go:wasmexport Round
func Round() int32 {
	return gGame.Round
}

//go:wasmexport CarbonEmissions
func CarbonEmissions() int32 {
	return gGame.CarbonEmissions
}

//go:wasmexport PlayerMoney
func PlayerMoney(playerIndex int32) int32 {
	return gGame.PlayerMoney(int(playerIndex))
}

//go:wasmexport PlayerStatus
func PlayerStatus(playerIndex int32) int32 {
	return int32(gGame.PlayerStatus(int(playerIndex)))
}

//go:wasmexport PlayerLossReason
func PlayerLossReason(playerIndex int32) int32 {
	return int32(gGame.PlayerLossReason(int(playerIndex)))
}

//go:wasmexport PlayerRenewableAssets
func PlayerRenewableAssets(playerIndex int32) int32 {
	return int32(gGame.PlayerAssetMix(int(playerIndex)).Renewables)
}

//go:wasmexport PlayerBatteriesArbitrageAssets
func PlayerBatteriesArbitrageAssets(playerIndex int32) int32 {
	return int32(gGame.PlayerAssetMix(int(playerIndex)).BatteriesArbitrage)
}

//go:wasmexport PlayerBatteriesCapacityAssets
func PlayerBatteriesCapacityAssets(playerIndex int32) int32 {
	return int32(gGame.PlayerAssetMix(int(playerIndex)).BatteriesCapacity)
}

//go:wasmexport PlayerFossilsWholesaleAssets
func PlayerFossilsWholesaleAssets(playerIndex int32) int32 {
	return int32(gGame.PlayerAssetMix(int(playerIndex)).FossilsWholesale)
}

//go:wasmexport PlayerFossilsCapacityAssets
func PlayerFossilsCapacityAssets(playerIndex int32) int32 {
	return int32(gGame.PlayerAssetMix(int(playerIndex)).FossilsCapacity)
}

//go:wasmexport TakeoverRenewableAssets
func TakeoverRenewableAssets() int32 {
	return int32(gGame.TakeoverPool.Renewables)
}

//go:wasmexport TakeoverBatteriesArbitrageAssets
func TakeoverBatteriesArbitrageAssets() int32 {
	return int32(gGame.TakeoverPool.BatteriesArbitrage)
}

//go:wasmexport TakeoverBatteriesCapacityAssets
func TakeoverBatteriesCapacityAssets() int32 {
	return int32(gGame.TakeoverPool.BatteriesCapacity)
}

//go:wasmexport TakeoverFossilsWholesaleAssets
func TakeoverFossilsWholesaleAssets() int32 {
	return int32(gGame.TakeoverPool.FossilsWholesale)
}

//go:wasmexport TakeoverFossilsCapacityAssets
func TakeoverFossilsCapacityAssets() int32 {
	return int32(gGame.TakeoverPool.FossilsCapacity)
}

//go:wasmexport LastSnapshotPriceVolatility
func LastSnapshotPriceVolatility() int32 {
	return int32(gGame.LastSnapshot.PriceVolatility)
}

//go:wasmexport LastSnapshotGridStability
func LastSnapshotGridStability() int32 {
	return int32(gGame.LastSnapshot.GridStability)
}

//go:wasmexport LastSnapshotRenewableAssets
func LastSnapshotRenewableAssets() int32 {
	return int32(gGame.LastSnapshot.AssetMix.Renewables)
}

//go:wasmexport LastSnapshotBatteriesArbitrageAssets
func LastSnapshotBatteriesArbitrageAssets() int32 {
	return int32(gGame.LastSnapshot.AssetMix.BatteriesArbitrage)
}

//go:wasmexport LastSnapshotBatteriesCapacityAssets
func LastSnapshotBatteriesCapacityAssets() int32 {
	return int32(gGame.LastSnapshot.AssetMix.BatteriesCapacity)
}

//go:wasmexport LastSnapshotFossilsWholesaleAssets
func LastSnapshotFossilsWholesaleAssets() int32 {
	return int32(gGame.LastSnapshot.AssetMix.FossilsWholesale)
}

//go:wasmexport LastSnapshotFossilsCapacityAssets
func LastSnapshotFossilsCapacityAssets() int32 {
	return int32(gGame.LastSnapshot.AssetMix.FossilsCapacity)
}

//go:wasmexport PossibleActionsMask
func PossibleActionsMask(playerIndex int32) int32 {
	return int32(gGame.PossibleActionMask(playerIndex))
}

//go:wasmexport CanPerformAction
func CanPerformAction(playerIndex int32, actionInt int32) int32 {
	if gGame.PossibleActionMask(playerIndex)&(1<<actionInt) != 0 {
		return CodeOK
	} else {
		return CodeInvalidAction
	}
}
