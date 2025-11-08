// Build Phase logic

package engine

import (
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

type PlayerAction struct {
	Type        ActionType
	PlayerIndex int         // Index of the player performing the action
	AssetType   assets.Type // Type of asset involved in the action. Not relevant for ActionTypeFinished
	Cost        int         // Cost of performing the action
}

// Apply performs the described action, or returns an error (for example, it was only valid for a previous state)
func (pa PlayerAction) Apply() error {
	return nil
}

func BuildPhase(gs *GameState) StateRunner {
	logger := gs.Logger.Sub().Set(StateMachineStateBuildPhase)
	logger.Event().With(GameLogEventStateMachineTransition).Log()

	// Build phase logic would go here
	numActivePlayers := len(gs.Players)
	for {
		if numActivePlayers == 0{
			break
		}
		actions := gs.possibleActions()
		if len(actions) == 0 {
			//game loss, assets in takeover pool that nobody can afford to take over
		}
		// Get and apply player action from client
	}

	return OperatePhase
}

func (gs *GameState) possibleActions() []PlayerAction {
	var actions []PlayerAction
	for pi, p := range gs.Players {
		if p.Status != PlayerStatusActive {
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
