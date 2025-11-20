// Operation phase logic

package engine

import (
	"math/rand"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/params"
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

// generationConstraintMet returns whether the number of generating assets meet the requirements for the rule
func (gs GameState) generationConstraintMet(am assets.AssetMix) bool {
	switch gs.Params.GenerationConstraintRule {
	case params.GenerationConstraintRuleMinimum:
		return am.GenerationAssets() >= gs.Params.GenerationConstraint
	case params.GenerationConstraintRuleMaxDecrease:
		return (gs.LastSnapshot.AssetMix.GenerationAssets() - am.GenerationAssets()) >= -gs.Params.GenerationConstraint
	}
	return false
}

// OperatePhase handles calculations
func OperatePhase(gs *GameState) StateRunner {
	logger := gs.Logger.Sub().Set(StateMachineStateOperatePhase)
	logger.Event().With(GameLogEventStateMachineTransition).Log()

	// Draw random event
	risk := EventRisk(rand.Intn(3))
	logger.Event().With(GameLogEventEventDrawn, risk).Log()

	// Calculate asset mix, price volatility, grid stability, and new emissions
	am := gs.getAssetMix()
	gridOutcome := Snapshot{
		AssetMix:        am,
		PriceVolatility: assets.MapRatioTo(priceVolatilityCalculation, am, priceVolatilityMap),
		GridStability:   assets.MapRatioTo(gridStabilityCalculation, am, gridStabilityMap),
	}
	logger.Event().
		WithKey("grid_outcome", gridOutcome).
		WithKey("new_emissions", gridOutcome.AssetMix.Emissions()).
		With(GameLogEventGridOutcome).Log()

	// Check global loss conditions
	if gs.generationConstraintMet(gridOutcome.AssetMix) {
		gs.SetGlobalLossWithReason(LossConditionInsufficientGeneration)
		logger.Event().With(GameLogEventEveryoneLoses, gs.Reason).WithKey("generation_assets", gridOutcome.AssetMix.GenerationAssets()).Log()
		return GameEnd
	}
	if int(gridOutcome.GridStability) < int(risk) {
		gs.SetGlobalLossWithReason(LossConditionGridUnstable)
		logger.Event().With(GameLogEventEveryoneLoses, gs.Reason, gridOutcome.GridStability, risk).Log()
		return GameEnd
	}
	gs.CarbonEmissions += gridOutcome.AssetMix.Emissions()
	if gs.CarbonEmissions > gs.Params.EmissionsCap {
		gs.SetGlobalLossWithReason(LossConditionCarbonEmissionsExceeded)
		logger.Event().With(GameLogEventEveryoneLoses, gs.Reason).WithKey("total_emissions", gs.CarbonEmissions).WithKey("new_emissions", gridOutcome.AssetMix.Emissions()).Log()
		return GameEnd
	}

	// Do market PnL calculations for each player
	for pi, p := range gs.Players {
		pLogger := logger.Sub().SetKey("player_index", pi)
		if p.Status == PlayerStatusActive {
			var playerPnL int
			for _, a := range p.Assets {
				playerPnL += gs.Params.PnL(a, gridOutcome.PriceVolatility, gs.CarbonEmissions, gridOutcome.AssetMix.CapacityAssets())
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

	gs.LastSnapshot = gridOutcome

	return RoundEnd
}


// RoundEnd checks whether the win condition is met after an Operate round
func RoundEnd(gs *GameState) StateRunner {
	logger := gs.Logger.Sub().Set(StateMachineStateRoundEnd)
	logger.Event().With(GameLogEventStateMachineTransition).Log()

	// Check for loss condition: last player with fossil assets, or all players lost
	var lastFossilPlayerIndex int = -1
	var fossilPlayerCount int
	var activePlayerCount int
	for pi, p := range gs.Players {
		if p.Status == PlayerStatusActive {
			activePlayerCount++
			if p.HasFossilAssets() {
				lastFossilPlayerIndex = pi
				fossilPlayerCount++
			}
		}
	}

	// If the end-game condition is not satisfied yet, start another round
	if fossilPlayerCount > 1 && activePlayerCount > 0 {
		return BuildPhase
	}

	// Handle the game end trigger: only one player left has fossil assets
	if fossilPlayerCount == 1 {
		gs.Players[lastFossilPlayerIndex].SetLossWithReason(LossConditionLastPlayerWithFossilAssets)
		activePlayerCount--
		logger.Event().WithKey("player_index", lastFossilPlayerIndex).With(GameLogEventPlayerLoses, LossConditionLastPlayerWithFossilAssets).Log()
		// Normally we would add the player's assets to the takeover pool when they lose, but having only
		// one player left with fossil assets means the game is over, won by the other players.
	}

	// If the last player left was just eliminated due to holding fossil assets, all players are out
	if activePlayerCount == 0 {
		gs.SetGlobalLossWithReason(LossConditionNoActivePlayers)
		logger.Event().With(GameLogEventEveryoneLoses, LossConditionNoActivePlayers).Log()
		return GameEnd
	}

	// There are active players left after eliminating the last fossil holder (if any), they win!
	gs.Status = GameStatusWin
	logger.Event().With(GameLogEventGlobalWin).Log()
	return GameEnd
}