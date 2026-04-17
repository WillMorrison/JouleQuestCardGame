package game

import (
	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
)

var priceVolatilityCalculation = assets.RatioCalculation{
	CoefficientsA: assets.AssetMixCoefficients{FossilsWholesale: 1, BatteriesArbitrage: 1},
	CoefficientsB: assets.AssetMixCoefficients{Renewables: 1, BatteriesArbitrage: -1},
	Rollover:      3,
}
var priceVolatilityMap = [4]core.PriceVolatility{
	core.PriceVolatilityLow,
	core.PriceVolatilityMedium,
	core.PriceVolatilityHigh,
	core.PriceVolatilityExtreme,
}

var gridStabilityCalculation = assets.RatioCalculation{
	CoefficientsA: assets.AssetMixCoefficients{
		BatteriesCapacity: 1, BatteriesArbitrage: 1, FossilsCapacity: 1, FossilsWholesale: 1,
	},
	CoefficientsB: assets.AssetMixCoefficients{
		Renewables: 1, FossilsCapacity: -1, BatteriesCapacity: -2, BatteriesArbitrage: -1,
	},
	Rollover: 3,
}
var gridStabilityMap = [4]core.GridStability{
	core.GridStabilityGood,
	core.GridStabilityOk,
	core.GridStabilityBad,
	core.GridStabilityDangerous,
}

// Snapshot matches engine.Snapshot shape for API parity.
type Snapshot struct {
	AssetMix        assets.AssetMix
	PriceVolatility core.PriceVolatility
	GridStability   core.GridStability
}

func snapshotFromGlobalMix(am assets.AssetMix) Snapshot {
	return Snapshot{
		AssetMix:        am,
		PriceVolatility: assets.MapRatioTo(priceVolatilityCalculation, am, priceVolatilityMap),
		GridStability:   assets.MapRatioTo(gridStabilityCalculation, am, gridStabilityMap),
	}
}
