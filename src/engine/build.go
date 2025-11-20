// Build Phase logic

package engine

import (
	"fmt"
	"slices"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

type ActionType int

//go:generate go tool stringer -type=ActionType -trimprefix=ActionType
const (
	ActionTypeBuildAsset         ActionType = iota // Build a new asset and add it to player's portfolio
	ActionTypeScrapAsset                           // Scrap an existing asset from player's portfolio
	ActionTypeTakeoverAsset                        // Take over an existing asset from a bankrupt player and add it to player's portfolio
	ActionTypeTakeoverScrapAsset                   // Scrap an asset from a bankrupt player's portfolio
	ActionTypePledgeCapacity                       // Pledge an asset in the player's portfolio to the capacity market
	ActionTypeFinished                             // Indicate that the player is done with the build phase
)

func (at ActionType) LogKey() string {
	return "action_type"
}

func (at ActionType) MarshalText() ([]byte, error) {
	return []byte(at.String()), nil
}

type PlayerAction struct {
	Type        ActionType
	PlayerIndex int         // Index of the player performing the action
	AssetType   assets.Type // Type of asset involved in the action. Not relevant for ActionTypeFinished
	Cost        int         // Cost of performing the action
}

// GetPlayerAction is a type that clients must implement to input player actions to the state machine
type GetPlayerAction func([]PlayerAction) PlayerAction

// BuildPhase implements the build phase using the GetPlayerAction callback to get the next player action.
func BuildPhase(gs *GameState) StateRunner {
	gs.Round++
	gs.Logger = gs.Logger.SetKey("round", gs.Round) // Always add round info to game event logs
	logger := gs.Logger.Sub().Set(StateMachineStateBuildPhase)
	logger.Event().With(GameLogEventStateMachineTransition).Log()

	var numBuildingPlayers int
	for _, p := range gs.activePlayers() {
		p.isBuilding = true
		numBuildingPlayers++
		p.resetAllAssets()

	}
	for numBuildingPlayers > 0 {
		actions := gs.possibleActions()
		if len(actions) == 0 {
			// game loss, assets in takeover pool that nobody can afford to take over
			gs.SetGlobalLossWithReason(LossConditionUnownedTakeoverAssets)
			takeoverMix := assets.AssetMixFrom(slices.Values(gs.TakeoverPool))
			var money []int
			for _, p := range gs.Players {
				money = append(money, p.Money)
			}
			logger.Event().With(GameLogEventEveryoneLoses, gs.Reason).WithKey("takeover_pool", takeoverMix).WithKey("player_funds", money).Log()
			return GameEnd
		}
		// Get and apply player action from client
		chosenAction := gs.GetPlayerAction(actions)
		err := gs.applyPlayerAction(chosenAction)
		if err != nil {
			logger.Event().With(GameLogEventPlayerActionInvalid).WithKey("invalid_action", chosenAction).WithKey("error", err.Error()).Log()
			continue
		} else {
			logger.Event().With(GameLogEventPlayerAction).WithKey("action", chosenAction).Log()
		}
		if chosenAction.Type == ActionTypeFinished {
			numBuildingPlayers -= 1
		}
	}

	return OperatePhase
}

// possibleActions returns a slice of build phase player actions that are possible
func (gs *GameState) possibleActions() []PlayerAction {
	var actions []PlayerAction
	for pi, p := range gs.activePlayers() {
		if !p.isBuilding {
			continue
		}
		playerAssetMix := p.getAssetMix()
		takeoverAssetMix := assets.AssetMixFrom(slices.Values(gs.TakeoverPool))
		for _, at := range assets.Types {
			if cost := gs.Params.BuildCost(at); cost <= p.Money {
				actions = append(actions, PlayerAction{Type: ActionTypeBuildAsset, PlayerIndex: pi, AssetType: at, Cost: cost})
			}
			if cost := gs.Params.ScrapCost(at); cost <= p.Money && playerAssetMix.AssetsOfType(at) > 0 {
				actions = append(actions, PlayerAction{Type: ActionTypeScrapAsset, PlayerIndex: pi, AssetType: at, Cost: cost})
			}
			if cost := gs.Params.TakeoverCost(at); cost <= p.Money && takeoverAssetMix.AssetsOfType(at) > 0 {
				actions = append(
					actions,
					PlayerAction{Type: ActionTypeTakeoverAsset, PlayerIndex: pi, AssetType: at, Cost: cost},
					PlayerAction{Type: ActionTypeTakeoverScrapAsset, PlayerIndex: pi, AssetType: at, Cost: cost},
				)
			}
		}
		if gs.Params.CapacityRule != params.CapacityRuleNoCapacityMarket {
			if playerAssetMix.BatteriesArbitrage > 0 {
				actions = append(actions, PlayerAction{Type: ActionTypePledgeCapacity, PlayerIndex: pi, AssetType: assets.TypeBattery})
			}
			if playerAssetMix.FossilsWholesale > 0 {
				actions = append(actions, PlayerAction{Type: ActionTypePledgeCapacity, PlayerIndex: pi, AssetType: assets.TypeFossil})
			}
		}
		if takeoverAssetMix.NumAssets() == 0 {
			actions = append(actions, PlayerAction{Type: ActionTypeFinished, PlayerIndex: pi})
		}
	}
	return actions
}

// applyPlayerAction performs the described action, or returns an error
func (gs *GameState) applyPlayerAction(pa PlayerAction) error {
	if !slices.Contains(gs.possibleActions(), pa) {
		return fmt.Errorf("%+v is not on the list of possible actions", pa)
	}

	var player *PlayerState = &(gs.Players[pa.PlayerIndex])
	firstAssetOfActionType := func(a assets.Asset) bool { return a.Type() == pa.AssetType }
	switch pa.Type {
	case ActionTypeFinished:
		player.isBuilding = false
	case ActionTypeBuildAsset:
		player.Assets = append(player.Assets, assets.New(pa.AssetType))
	case ActionTypeScrapAsset:
		ai := slices.IndexFunc(player.Assets, firstAssetOfActionType)
		if ai == -1 {
			return fmt.Errorf("PlayerIndex %d has no assets of type %s to scrap", pa.PlayerIndex, pa.AssetType.String())
		}
		player.Assets = slices.Delete(player.Assets, ai, ai+1)
	case ActionTypeTakeoverAsset:
		ai := slices.IndexFunc(gs.TakeoverPool, firstAssetOfActionType)
		if ai == -1 {
			return fmt.Errorf("takeover pool has no assets of type %s", pa.AssetType.String())
		}
		gs.TakeoverPool = slices.Delete(gs.TakeoverPool, ai, ai+1)
		player.Assets = append(player.Assets, assets.New(pa.AssetType))
	case ActionTypeTakeoverScrapAsset:
		ai := slices.IndexFunc(gs.TakeoverPool, firstAssetOfActionType)
		if ai == -1 {
			return fmt.Errorf("takeover pool has no assets of type %s", pa.AssetType.String())
		}
		gs.TakeoverPool = slices.Delete(gs.TakeoverPool, ai, ai+1)
	case ActionTypePledgeCapacity:
		ai := slices.IndexFunc(player.Assets, func(a assets.Asset) bool {
			return a.Type() == pa.AssetType && (a.Mode()&assets.OperationModeCapacity == 0)
		})
		if ai == -1 {
			return fmt.Errorf("PlayerIndex %d has no assets of type %s to pledge", pa.PlayerIndex, pa.AssetType.String())
		}
		player.Assets[ai].SetMode(assets.OperationModeCapacity)
	}
	player.Money -= pa.Cost
	return nil
}
