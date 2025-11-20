// Operation phase logic

package engine

import (
	"math/rand"
	"slices"

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

// getSnapshot calculates asset mix, price volatility, and grid stability
func (gs GameState) getSnapshot() Snapshot {
	am := gs.getAssetMix()
	return Snapshot{
		AssetMix:        am,
		PriceVolatility: assets.MapRatioTo(priceVolatilityCalculation, am, priceVolatilityMap),
		GridStability:   assets.MapRatioTo(gridStabilityCalculation, am, gridStabilityMap),
	}
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

func (gs GameState) winConditionMet() bool {
	switch gs.Params.WinConditionRule {
	case params.WinConditionRuleRenewablePenetrationThreshold:
		return gs.LastSnapshot.AssetMix.RenewablePenetration() >= gs.Params.RenewablePenetration
	case params.WinConditionRuleLastFossilLoses:
		// If there are fossil assets in the takeover pool, the game cannot end
		for _, a := range gs.TakeoverPool {
			if a.Type() == assets.TypeFossil {
				return false
			}
		}
		// Check how many active players have fossil assets
		var numFossilHolders int
		for _, p := range gs.Players {
			if p.Status == PlayerStatusActive && p.HasFossilAssets() {
				numFossilHolders++
			}
		}
		return numFossilHolders <= 1
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
	gridOutcome := gs.getSnapshot()
	logger.Event().
		WithKey("grid_outcome", gridOutcome).
		WithKey("new_emissions", gridOutcome.AssetMix.Emissions()).
		With(GameLogEventGridOutcome).Log()

	// Check global loss conditions
	if !gs.generationConstraintMet(gridOutcome.AssetMix) {
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
	var numActivePlayers int
	for pi, p := range gs.activePlayers() {
		pLogger := logger.Sub().SetKey("player_index", pi)
		numActivePlayers++
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
			numActivePlayers--
		}
	}

	gs.LastSnapshot = gridOutcome

	// If all players are out (e.g. due to bankruptcy), the game is a loss
	if numActivePlayers == 0 {
		gs.SetGlobalLossWithReason(LossConditionNoActivePlayers)
		logger.Event().With(GameLogEventEveryoneLoses, LossConditionNoActivePlayers).Log()
		return GameEnd
	}

	// If the win condition is not met, start another round
	if !gs.winConditionMet() {
		return BuildPhase
	}

	if gs.Params.WinConditionRule == params.WinConditionRuleLastFossilLoses {
		// The last player with fossil assets left loses
		lastFossilPlayerIndex := slices.IndexFunc(gs.Players, PlayerState.HasFossilAssets)
		if lastFossilPlayerIndex != -1 {
			gs.Players[lastFossilPlayerIndex].SetLossWithReason(LossConditionLastPlayerWithFossilAssets)
			logger.Event().WithKey("player_index", lastFossilPlayerIndex).With(GameLogEventPlayerLoses, LossConditionLastPlayerWithFossilAssets).Log()

			// Check if we just eliminated the last player. If so, the game is a loss.
			numActivePlayers--
			if numActivePlayers == 0 {
				gs.SetGlobalLossWithReason(LossConditionNoActivePlayers)
				logger.Event().With(GameLogEventEveryoneLoses, LossConditionNoActivePlayers).Log()
				return GameEnd
			}
		}

	}

	// There are active players left, they win!
	gs.Status = GameStatusWin
	logger.Event().With(GameLogEventGlobalWin).Log()
	return GameEnd
}
