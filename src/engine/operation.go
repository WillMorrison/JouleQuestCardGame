// Operation phase logic

package engine

import (
	"math/rand"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
)

// operationOutcome represents the calculations of the Operate phase.
type operationOutcome struct {
	PriceVolatility core.PriceVolatility
	GridStability   core.GridStability
}

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

func OperatePhase(gs *GameState) StateRunner {
	logger := gs.Logger.Sub().Set(StateMachineStateOperatePhase)
	logger.Event().With(GameLogEventStateMachineTransition).Log()

	// Draw random event
	risk := EventRisk(rand.Intn(3))
	logger.Event().With(GameLogEventEventDrawn, risk).Log()

	// Calculate asset mix, price volatility, grid stability, and new emissions
	am := gs.getAssetMix()
	gridOutcome := operationOutcome{
		PriceVolatility: assets.MapRatioTo(priceVolatilityCalculation, am, priceVolatilityMap),
		GridStability:   assets.MapRatioTo(gridStabilityCalculation, am, gridStabilityMap),
	}
	logger.Event().
		WithKey("asset_mix", am).
		WithKey("grid_outcome", gridOutcome).
		WithKey("new_emissions", am.Emissions()).
		With(GameLogEventGridOutcome).Log()

	// Check global loss conditions
	if am.GenerationAssets() < core.MinimumGenerationAssets {
		gs.SetGlobalLossWithReason(LossConditionInsufficientGeneration)
		logger.Event().With(GameLogEventEveryoneLoses, gs.Reason).WithKey("generation_assets", am.GenerationAssets()).Log()
		return RoundEnd
	}
	if int(gridOutcome.GridStability) < int(risk) {
		gs.SetGlobalLossWithReason(LossConditionGridUnstable)
		logger.Event().With(GameLogEventEveryoneLoses, gs.Reason, gridOutcome.GridStability, risk).Log()
		return RoundEnd
	}
	gs.CarbonEmissions += am.Emissions()
	if gs.CarbonEmissions > core.EmissionsCap {
		gs.SetGlobalLossWithReason(LossConditionCarbonEmissionsExceeded)
		logger.Event().With(GameLogEventEveryoneLoses, gs.Reason).WithKey("total_emissions", gs.CarbonEmissions).WithKey("new_emissions", am.Emissions()).Log()
		return RoundEnd
	}

	// Do market PnL calculations for each player
	for pi, p := range gs.Players {
		pLogger := logger.Sub().SetKey("player_index", pi)
		if p.Status == PlayerStatusActive {
			var playerPnL int
			for _, a := range p.Assets {
				playerPnL += gs.Params.PnL(a, gridOutcome.PriceVolatility, gs.CarbonEmissions, am.CapacityAssets())
			}
			p.Money += playerPnL
			pLogger.Event().WithKey("player_asset_mix", p.getAssetMix()).WithKey("player_PnL", playerPnL).WithKey("player_money", p.Money).With(GameLogEventMarketOutcome).Log()

			// Check player loss conditions
			if p.Money < 0 {
				p.SetLossWithReason(LossConditionPlayerBankrupt)
				gs.movePlayerAssetsToTakeoverPool(pi)
				pLogger.Event().With(GameLogEventPlayerLoses, p.Reason).WithKey("player_money", p.Money).Log()
			}
		}
	}

	return RoundEnd
}
