package client

import (
	"github.com/WillMorrison/JouleQuestCardGame/engine"
	"github.com/WillMorrison/JouleQuestCardGame/eventlog"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

var game engine.ProceduralGameState

// Returns a new game with no log, using default parameters.
func Init(numPlayers uint8) bool {
	if err := game.Reset(int(numPlayers), params.Default, eventlog.NullLogger{}); err != nil {
		return false
	}
	return true
}

func GameStatus() uint8 {
	return uint8(game.Game().Status)
}

func GameReason() uint8 {
	return uint8(game.Game().Reason)
}

func Round() uint32 {
	return uint32(game.Game().Round)
}

func CarbonEmissions() uint32 {
	return uint32(game.Game().CarbonEmissions)
}

func LastSnapshotGridStability() uint8 {
	return uint8(game.Game().LastSnapshot.GridStability)
}

func LastSnapshotPriceVolatility() uint8 {
	return uint8(game.Game().LastSnapshot.PriceVolatility)
}

func LastSnapshotAssetMixRenewables() uint32 {
	return uint32(game.Game().LastSnapshot.AssetMix.Renewables)
}

func LastSnapshotAssetMixFossilsCapacity() uint32 {
	return uint32(game.Game().LastSnapshot.AssetMix.FossilsCapacity)
}

func LastSnapshotAssetMixFossilsWholesale() uint32 {
	return uint32(game.Game().LastSnapshot.AssetMix.FossilsWholesale)
}

func LastSnapshotAssetMixBatteriesArbitrage() uint32 {
	return uint32(game.Game().LastSnapshot.AssetMix.BatteriesArbitrage)
}

func LastSnapshotAssetMixBatteriesCapacity() uint32 {
	return uint32(game.Game().LastSnapshot.AssetMix.BatteriesCapacity)
}

func TakeoverAssetMixRenewables() uint32 {
	return uint32(game.Game().TakeoverAssetMix().Renewables)
}

func TakeoverAssetMixFossilsCapacity() uint32 {
	return uint32(game.Game().TakeoverAssetMix().FossilsCapacity)
}

func TakeoverAssetMixFossilsWholesale() uint32 {
	return uint32(game.Game().TakeoverAssetMix().FossilsWholesale)
}

func TakeoverAssetMixBatteriesArbitrage() uint32 {
	return uint32(game.Game().TakeoverAssetMix().BatteriesArbitrage)
}

func TakeoverAssetMixBatteriesCapacity() uint32 {
	return uint32(game.Game().TakeoverAssetMix().BatteriesCapacity)
}

func NumPlayers() uint8 {
	return uint8(len(game.Game().Players))
}

func PlayerStatus(player uint8) uint8 {
	if player >= uint8(len(game.Game().Players)) {
		return 0
	}
	return uint8(game.Game().Players[player].Status)
}

func PlayerReason(player uint8) uint8 {
	if player >= uint8(len(game.Game().Players)) {
		return 0
	}
	return uint8(game.Game().Players[player].Reason)
}

func PlayerMoney(player uint8) uint32 {
	if player >= uint8(len(game.Game().Players)) {
		return 0
	}
	return uint32(game.Game().Players[player].Money)
}

func PlayerAssetMixRenewables(player uint8) uint32 {
	if player >= uint8(len(game.Game().Players)) {
		return 0
	}
	return uint32(game.Game().Players[player].AssetMix().Renewables)
}

func PlayerAssetMixFossilsCapacity(player uint8) uint32 {
	if player >= uint8(len(game.Game().Players)) {
		return 0
	}
	return uint32(game.Game().Players[player].AssetMix().FossilsCapacity)
}

func PlayerAssetMixFossilsWholesale(player uint8) uint32 {
	if player >= uint8(len(game.Game().Players)) {
		return 0
	}
	return uint32(game.Game().Players[player].AssetMix().FossilsWholesale)
}

func PlayerAssetMixBatteriesArbitrage(player uint8) uint32 {
	if player >= uint8(len(game.Game().Players)) {
		return 0
	}
	return uint32(game.Game().Players[player].AssetMix().BatteriesArbitrage)
}

func PlayerAssetMixBatteriesCapacity(player uint8) uint32 {
	if player >= uint8(len(game.Game().Players)) {
		return 0
	}
	return uint32(game.Game().Players[player].AssetMix().BatteriesCapacity)
}

