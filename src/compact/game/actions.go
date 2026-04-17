package game

import (
	"github.com/WillMorrison/JouleQuestCardGame/assets"
	legacy "github.com/WillMorrison/JouleQuestCardGame/params"
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

func (g *Game) possibleActionMask(pi int) uint32 {
	if g.Status != GameStatusOngoing {
		return 0
	}
	if pi < 0 || pi >= g.NumPlayers {
		return 0
	}
	p := &g.Players[pi]
	if p.Status != PlayerStatusActive || !p.IsBuilding {
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
	if g.Params.CapacityRule != legacy.CapacityRuleNoCapacityMarket {
		if p.Mix.BatteriesArbitrage > 0 {
			mask |= 1 << ActionPledgeBattery
		}
		if p.Mix.FossilsWholesale > 0 {
			mask |= 1 << ActionPledgeFossil
		}
	}
	switch g.Params.TakeoverRule {
	case legacy.TakeoverRuleVirtualOwner:
		mask |= 1 << ActionFinished
	case legacy.TakeoverRuleForcedTakeover:
		if g.TakeoverPool.NumAssets() == 0 {
			mask |= 1 << ActionFinished
		}
	}
	return mask
}

func (g *Game) PossibleActionMask(pi int32) uint32 {
	return g.possibleActionMask(int(pi))
}

func (g *Game) applyBuild(pi int, at assets.Type) {
	p := &g.Players[pi]
	switch at {
	case assets.TypeRenewable:
		p.Mix.Renewables++
	case assets.TypeBattery:
		p.Mix.BatteriesArbitrage++
	case assets.TypeFossil:
		p.Mix.FossilsWholesale++
	}
}

func scrapOneFromPlayer(p *Player, at assets.Type) bool {
	return scrapOneFromPool(&p.Mix, at)
}

func scrapOneFromPool(m *assets.AssetMix, at assets.Type) bool {
	switch at {
	case assets.TypeRenewable:
		if m.Renewables > 0 {
			m.Renewables--
			return true
		}
	case assets.TypeBattery:
		if m.BatteriesArbitrage > 0 {
			m.BatteriesArbitrage--
			return true
		}
		if m.BatteriesCapacity > 0 {
			m.BatteriesCapacity--
			return true
		}
	case assets.TypeFossil:
		if m.FossilsWholesale > 0 {
			m.FossilsWholesale--
			return true
		}
		if m.FossilsCapacity > 0 {
			m.FossilsCapacity--
			return true
		}
	}
	return false
}

func pledgeOne(p *Player, at assets.Type) bool {
	m := &p.Mix
	switch at {
	case assets.TypeBattery:
		if m.BatteriesArbitrage > 0 {
			m.BatteriesArbitrage--
			m.BatteriesCapacity++
			return true
		}
	case assets.TypeFossil:
		if m.FossilsWholesale > 0 {
			m.FossilsWholesale--
			m.FossilsCapacity++
			return true
		}
	}
	return false
}

func (g *Game) applyActionCode(pi int, actionCode int32) bool {
	p := &g.Players[pi]
	var cost int32
	switch actionCode {
	case ActionFinished:
		p.IsBuilding = false
		return true
	case ActionBuildRenewable, ActionBuildBattery, ActionBuildFossil:
		at := assetTypeForAction(actionCode)
		cost = g.Params.BuildCost(at)
		p.Money -= cost
		g.applyBuild(pi, at)
		return true
	case ActionScrapRenewable, ActionScrapBattery, ActionScrapFossil:
		at := assetTypeForAction(actionCode)
		cost = g.Params.ScrapCost(at)
		if !scrapOneFromPlayer(p, at) {
			return false
		}
		p.Money -= cost
		return true
	case ActionTakeoverRenewable, ActionTakeoverBattery, ActionTakeoverFossil:
		at := assetTypeForAction(actionCode)
		cost = g.Params.TakeoverCost(at)
		if !scrapOneFromPool(&g.TakeoverPool, at) {
			return false
		}
		p.Money -= cost
		g.applyBuild(pi, at)
		return true
	case ActionTakeoverScrapRenewable, ActionTakeoverScrapBattery, ActionTakeoverScrapFossil:
		at := assetTypeForAction(actionCode)
		cost = g.Params.TakeoverCost(at)
		if !scrapOneFromPool(&g.TakeoverPool, at) {
			return false
		}
		p.Money -= cost
		return true
	case ActionPledgeBattery:
		if !pledgeOne(p, assets.TypeBattery) {
			return false
		}
		return true
	case ActionPledgeFossil:
		if !pledgeOne(p, assets.TypeFossil) {
			return false
		}
		return true
	default:
		return false
	}
}

func actionCodeAllowed(mask uint32, code int32) bool {
	if code < 0 || code > ActionFinished {
		return false
	}
	return mask&(1<<code) != 0
}
