// Build Phase logic

package engine

import (
	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
)

type ActionType int

//go:generate go tool stringer -type=ActionType -trimprefix=ActionType
const (
	ActionTypeBuildAsset         ActionType = iota // Build a new asset and add it to player's portfolio
	ActionTypeScrapAsset                           // Scrap an existing asset from player's portfolio
	ActionTypeTakeoverAsset                        // Take over an existing asset from a bankrupt player and add it to player's portfolio
	ActionTypeTakeoverScrapAsset                   // Scrap an asset from a bankrupt player's portfolio
	ActionTypePledgeCapacity                       // Pledge an asset in the player's portfolio to the capacity market
	ActionTypeBuyService                           // Buy a forecasting service for all of the player's battery assets
	ActionTypeFinished                             // Indicate that the player is done with the build phase
)

func (at ActionType) LogKey() string {
	return "action_type"
}

type PlayerAction struct {
	Type        ActionType
	PlayerIndex int            // Index of the player performing the action
	AssetType   core.AssetType // Type of asset involved in the action. Not relevant for ActionTypeFinished
	Asset       assets.Asset   // Specific asset involved in the action. Nil for Build, BuyService, and Finished actions
	Cost        int            // Cost of performing the action
	// Todo: Figure out how to detect actions that are no longer valid
}

// Apply performs the described action, or returns an error (for example, it was only valid for a previous state)
func (pa PlayerAction) Apply() error {
	return nil
}

func BuildPhase(gs *GameState) StateRunner {
	logger := gs.Logger.Sub().Set(StateMachineStateBuildPhase)
	logger.Event().With(GameLogEventStateMachineTransition).Log()

	// Build phase logic would go here

	return OperatePhase
}

func (gs *GameState) possibleActions() []PlayerAction {
	var actions []PlayerAction
	for pi, p := range gs.Players {
		if p.Status != PlayerStatusActive {
			continue
		}
		am := p.getAssetMix()
		if p.Money >= core.AssetTypeBattery.BuildCost() {
			actions = append(actions, PlayerAction{Type: ActionTypeBuildAsset, PlayerIndex: pi, AssetType: core.AssetTypeBattery, Cost: core.AssetTypeBattery.BuildCost()})
		}
		if p.Money >= core.AssetTypeRenewable.BuildCost() {
			actions = append(actions, PlayerAction{Type: ActionTypeBuildAsset, PlayerIndex: pi, AssetType: core.AssetTypeRenewable, Cost: core.AssetTypeRenewable.BuildCost()})
		}
		if p.Money >= core.AssetTypeFossil.BuildCost() {
			actions = append(actions, PlayerAction{Type: ActionTypeBuildAsset, PlayerIndex: pi, AssetType: core.AssetTypeFossil, Cost: core.AssetTypeFossil.BuildCost()})
		}
		if am.BatteriesArbitrage+am.BatteriesCapacity > 0 && p.Money >= core.AssetTypeBattery.ScrapCost() {
			actions = append(actions, PlayerAction{Type: ActionTypeScrapAsset, PlayerIndex: pi, AssetType: core.AssetTypeBattery, Cost: core.AssetTypeBattery.ScrapCost()})
		}
		if am.FossilsCapacity+am.FossilsWholesale > 0 && p.Money >= core.AssetTypeFossil.ScrapCost() {
			actions = append(actions, PlayerAction{Type: ActionTypeScrapAsset, PlayerIndex: pi, AssetType: core.AssetTypeFossil, Cost: core.AssetTypeFossil.ScrapCost()})
		}
		if am.Renewables > 0 && p.Money >= core.AssetTypeRenewable.ScrapCost() {
			actions = append(actions, PlayerAction{Type: ActionTypeScrapAsset, PlayerIndex: pi, AssetType: core.AssetTypeRenewable, Cost: core.AssetTypeRenewable.ScrapCost()})
		}
		// Todo: Takeover actions, capacity pledges, and service purchases, and Finished action if takeover pool is empty
	}
	return actions
}
