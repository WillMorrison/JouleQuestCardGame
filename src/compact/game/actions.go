package game

import (
	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

// Action code layout matches rl_agent/custom_environment/env/joulequest_env.py PlayerActionToInt.
const (
	ActionBuildRenewable = iota
	ActionBuildBattery
	ActionBuildFossil
	ActionScrapRenewable
	ActionScrapBattery
	ActionScrapFossil
	ActionTakeoverRenewable
	ActionTakeoverBattery
	ActionTakeoverFossil
	ActionTakeoverScrapRenewable
	ActionTakeoverScrapBattery
	ActionTakeoverScrapFossil
	ActionPledgeBattery
	ActionPledgeFossil
	ActionFinished
)

// assetTypeForAction maps build / scrap / takeover / takeover-scrap / pledge codes to assets.Type.
// ActionFinished (and any invalid code) use default — this must not be used for asset ops with those codes.
func assetTypeForAction(actionCode int32) assets.Type {
	switch actionCode {
	case ActionBuildRenewable:
		return assets.TypeRenewable
	case ActionBuildBattery:
		return assets.TypeBattery
	case ActionBuildFossil:
		return assets.TypeFossil
	case ActionScrapRenewable:
		return assets.TypeRenewable
	case ActionScrapBattery:
		return assets.TypeBattery
	case ActionScrapFossil:
		return assets.TypeFossil
	case ActionTakeoverRenewable:
		return assets.TypeRenewable
	case ActionTakeoverBattery:
		return assets.TypeBattery
	case ActionTakeoverFossil:
		return assets.TypeFossil
	case ActionTakeoverScrapRenewable:
		return assets.TypeRenewable
	case ActionTakeoverScrapBattery:
		return assets.TypeBattery
	case ActionTakeoverScrapFossil:
		return assets.TypeFossil
	case ActionPledgeBattery:
		return assets.TypeBattery
	case ActionPledgeFossil:
		return assets.TypeFossil
	default:
		// ActionFinished or unknown — not meaningful; applyActionCode handles Finished separately.
		return assets.TypeRenewable
	}
}

func (g *Game) PossibleActionMask(pi int32) uint32 {
	if g.Status != core.GameStatusOngoing || g.phase != phaseBuild {
		return 0
	}
	if pi < 0 || pi >= g.NumPlayers {
		return 0
	}
	p := &g.Players[pi]
	if p.Status != core.PlayerStatusActive || !p.IsBuilding {
		return 0
	}
	var mask uint32
	// Order matches PlayerActionToInt: renewable=0/3/6/9, battery=1/4/7/10, fossil=2/5/8/11
	if cost := g.Params.BuildCost(assets.TypeRenewable); cost <= p.Money {
		mask |= 1 << ActionBuildRenewable
	}
	if cost := g.Params.BuildCost(assets.TypeBattery); cost <= p.Money {
		mask |= 1 << ActionBuildBattery
	}
	if cost := g.Params.BuildCost(assets.TypeFossil); cost <= p.Money {
		mask |= 1 << ActionBuildFossil
	}
	if cost := g.Params.ScrapCost(assets.TypeRenewable); cost <= p.Money && p.Mix.AssetsOfType(assets.TypeRenewable) > 0 {
		mask |= 1 << ActionScrapRenewable
	}
	if cost := g.Params.ScrapCost(assets.TypeBattery); cost <= p.Money && p.Mix.AssetsOfType(assets.TypeBattery) > 0 {
		mask |= 1 << ActionScrapBattery
	}
	if cost := g.Params.ScrapCost(assets.TypeFossil); cost <= p.Money && p.Mix.AssetsOfType(assets.TypeFossil) > 0 {
		mask |= 1 << ActionScrapFossil
	}
	if cost := g.Params.TakeoverCost(assets.TypeRenewable); cost <= p.Money && g.TakeoverPool.AssetsOfType(assets.TypeRenewable) > 0 {
		mask |= 1 << ActionTakeoverRenewable
		mask |= 1 << ActionTakeoverScrapRenewable
	}
	if cost := g.Params.TakeoverCost(assets.TypeBattery); cost <= p.Money && g.TakeoverPool.AssetsOfType(assets.TypeBattery) > 0 {
		mask |= 1 << ActionTakeoverBattery
		mask |= 1 << ActionTakeoverScrapBattery
	}
	if cost := g.Params.TakeoverCost(assets.TypeFossil); cost <= p.Money && g.TakeoverPool.AssetsOfType(assets.TypeFossil) > 0 {
		mask |= 1 << ActionTakeoverFossil
		mask |= 1 << ActionTakeoverScrapFossil
	}
	if g.Params.CapacityRule != params.CapacityRuleNoCapacityMarket {
		if p.Mix.BatteriesArbitrage > 0 {
			mask |= 1 << ActionPledgeBattery
		}
		if p.Mix.FossilsWholesale > 0 {
			mask |= 1 << ActionPledgeFossil
		}
	}
	switch g.Params.TakeoverRule {
	case params.TakeoverRuleVirtualOwner:
		mask |= 1 << ActionFinished
	case params.TakeoverRuleForcedTakeover:
		if g.TakeoverPool.NumAssets() == 0 {
			mask |= 1 << ActionFinished
		}
	}
	return mask
}

func (g *Game) applyActionCode(pi int32, actionCode int32) {
	p := &g.Players[pi]
	var cost int32
	switch actionCode {
	case ActionFinished:
		p.IsBuilding = false
	case ActionBuildRenewable, ActionBuildBattery, ActionBuildFossil:
		at := assetTypeForAction(actionCode)
		cost = g.Params.BuildCost(at)
		p.Money -= cost
		p.Mix.AddOneAsset(at)
	case ActionScrapRenewable, ActionScrapBattery, ActionScrapFossil:
		at := assetTypeForAction(actionCode)
		cost = g.Params.ScrapCost(at)
		p.Money -= cost
		p.Mix.RemoveOneAsset(at)
	case ActionTakeoverRenewable, ActionTakeoverBattery, ActionTakeoverFossil:
		at := assetTypeForAction(actionCode)
		cost = g.Params.TakeoverCost(at)
		p.Money -= cost
		p.Mix.TakeOneAssetFrom(at, &g.TakeoverPool)
	case ActionTakeoverScrapRenewable, ActionTakeoverScrapBattery, ActionTakeoverScrapFossil:
		at := assetTypeForAction(actionCode)
		cost = g.Params.TakeoverCost(at)
		p.Money -= cost
		g.TakeoverPool.RemoveOneAsset(at)
	case ActionPledgeBattery, ActionPledgeFossil:
		at := assetTypeForAction(actionCode)
		p.Mix.PledgeOneAsset(at)
	}
}

func actionCodeAllowed(mask uint32, code int32) bool {
	if code < 0 || code > ActionFinished {
		return false
	}
	return mask&(1<<code) != 0
}
