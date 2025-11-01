package engine

import (
	"fmt"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/eventlog"
)

func NewGame(numPlayers int, logger eventlog.Logger) (*GameState, error) {
	initialAssetsPerPlayer, ok := core.StartingFossilAssetsPerPlayer[numPlayers]
	if !ok {
		return nil, fmt.Errorf("invalid number of players: %d", numPlayers)
	}

	var game = GameState{
		Status:          GameStatusOngoing,
		Round:           1,
		CarbonEmissions: 0,
		Logger:          logger,
	}

	for range numPlayers {
		p := PlayerState{
			Money:  core.InitialCash,
			Status: PlayerStatusActive,
		}
		for range initialAssetsPerPlayer {
			p.Assets = append(p.Assets, new(assets.FossilAsset))
		}
		game.Players = append(game.Players, p)
	}
	return &game, nil
}

type StateRunner func(gs *GameState) StateRunner

func (gs *GameState) Run() {
	current := GameStart
	for current != nil {
		current = current(gs)
	}
}

func GameStart(gs *GameState) StateRunner {
	params := map[string]any{
		"initial_cash":                      core.InitialCash,
		"starting_fossil_assets_per_player": core.StartingFossilAssetsPerPlayer,
		"minimum_generation_assets":         core.MinimumGenerationAssets,
		"emissions_cap":                     core.EmissionsCap,
		"fossil_build_cost":                 core.FossilBuildCost,
		"fossil_scrap_cost":                 core.FossilScrapCost,
		"renewable_build_cost":              core.RenewableBuildCost,
		"renewable_scrap_cost":              core.RenewableScrapCost,
		"battery_build_cost":                core.BatteryBuildCost,
		"battery_scrap_cost":                core.BatteryScrapCost,
		"pnl_tables": map[string]core.PnLTable{
			"fossil_wholesale":                 core.FossilPnLTable,
			"fossil_capacity":                  core.CapacityPnLTable,
			"fossil_wholesale_with_carbon_tax": core.FossilPnLTableWithCarbonTax,
			"fossil_capacity_with_carbon_tax":  core.FossilCapacityPnLTableWithCarbonTax,
			"renewable":                        core.RenewablePnLTable,
			"battery_arbitrage_with_service":   core.BatteryPnLTableWithService,
			"battery_arbitrage":                core.BatteryPnLTable,
			"battery_capacity":                 core.CapacityPnLTable,
		},
		"price_volatility_calculation": priceVolatilityCalculation.String(),
		"grid_stability_calculation":   gridStabilityCalculation.String(),
	}
	gs.Logger.Event().
		With(GameLogEventStateMachineTransition, StateMachineStateGameStart).
		WithKey("game_parameters", params).
		WithKey("num_players", len(gs.Players)).
		Log()
	return RoundStart
}

func RoundStart(gs *GameState) StateRunner {
	gs.Logger = gs.Logger.SetKey("round", gs.Round) // Always add round info to game event logs
	logger := gs.Logger.Sub().Set(StateMachineStateRoundStart)
	logger.Event().With(GameLogEventStateMachineTransition).Log()
	for pi, p := range gs.Players {
		p.resetAllAssets()

		// Check for players who have lost and put their assets into the takeover pool
		if p.Status != PlayerStatusActive && len(p.Assets) > 0 {
			gs.TakeoverPool = append(gs.TakeoverPool, p.Assets...)
			gs.Players[pi].Assets = nil
		}
	}
	// Todo: Log game state snapshot?
	return BuildPhase
}

func RoundEnd(gs *GameState) StateRunner {
	logger := gs.Logger.Sub().Set(StateMachineStateRoundEnd)
	logger.Event().With(GameLogEventStateMachineTransition).Log()

	// Game ended in some earlier state, pass through
	if gs.Status != GameStatusOngoing {
		return GameEnd
	}

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
		gs.Round++
		return RoundStart
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

func GameEnd(gs *GameState) StateRunner {
	gs.Logger.Event().
		With(GameLogEventStateMachineTransition, StateMachineStateGameEnd, gs.Status, gs.Reason).
		WithKey("total_emissions", gs.CarbonEmissions).
		WithKey("players", gs.Players).
		Log()
	return nil
}
