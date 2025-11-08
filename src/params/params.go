// params contains logic defining the set of parameters for a game.
package params

import (
	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
)

type CapacityRule int

//go:generate go tool stringer -type=CapacityRule -trimprefix=CapacityRule
const (
	// Capacity payments are looked up from a per-asset table. Default
	CapacityRulePaymentPerAsset CapacityRule = iota

	// Players cannot pledge assets to the capacity market
	CapacityRuleNoCapacityMarket

	// Capacity payments are made from a shared pool.
	CapacityRuleSharedCapacityPaymentPool
)

type CarbonTaxRule int

//go:generate go tool stringer -type=CarbonTaxRule -trimprefix=CarbonTaxRule
const (
	// Carbon tax is not applied. Default.
	CarbonTaxRuleNoCarbonTax CarbonTaxRule = iota

	// Fossil assets are charged a certain amount after emissions pass the carbon tax threshold
	CarbonTaxRuleApplyCarbonTax
)

type WinConditionRule int

//go:generate go tool stringer -type=WinConditionRule -trimprefix=WinConditionRule
const (
	// The win condition is all but one player getting rid of their fossil assets. Default.
	WinConditionRuleLastFossilLoses WinConditionRule = iota

	// The win condition is a certain renewable penetration.
	WinConditionRuleRenewablePenetrationThreshold
)

type GenerationConstraintRule int

//go:generate go tool stringer -type=GenerationConstraintRule -trimprefix=GenerationConstraintRule
const (
	// The generation constraint is a minimum number of generation assets. Default.
	GenerationConstraintRuleMinimum GenerationConstraintRule = iota

	// The generation constraint is a maximum decrease from last round.
	GenerationConstraintRuleMaxDecrease
)

type Params struct {
	CapacityRule CapacityRule
	CarbonTaxRule CarbonTaxRule
	WinConditionRule WinConditionRule
	GenerationConstraintRule GenerationConstraintRule

	InitialCash                   int
	StartingFossilAssetsPerPlayer map[int]int

	BatteryBuildCost   int
	BatteryScrapCost   int
	RenewableBuildCost int
	RenewableScrapCost int
	FossilBuildCost    int
	FossilScrapCost    int

	EmissionsCap         int
	GenerationConstraint int
	CarbonTaxThreshold   int
	CarbonTaxCost        int
	RenewablePenetration int

	RenewablePnL        core.PnLTable
	BatteryArbitragePnL core.PnLTable
	BatteryCapacityPnL  core.PnLTable
	FossilWholesalePnL  core.PnLTable
	FossilCapacityPnL   core.PnLTable
	CapacityPoolPnL     core.PnLTable
}

// defaultCost is a high cost, so that it is unlikely that a player will ever be able to afford it
const defaultCost = 1 << 32

// The cost to build an asset of a given type
func (p Params) BuildCost(at assets.Type) int {
	switch at {
	case assets.TypeBattery:
		return p.BatteryBuildCost
	case assets.TypeRenewable:
		return p.RenewableBuildCost
	case assets.TypeFossil:
		return p.FossilBuildCost
	}
	return defaultCost
}

// The cost to decommission an asset of a given type
func (p Params) ScrapCost(at assets.Type) int {
	switch at {
	case assets.TypeBattery:
		return p.BatteryScrapCost
	case assets.TypeRenewable:
		return p.RenewableScrapCost
	case assets.TypeFossil:
		return p.FossilScrapCost
	}
	return defaultCost
}

// The cost to take over an asset of a given type and add it to the player's portfolio
func (p Params) TakeoverCost(at assets.Type) int {
	switch at {
	case assets.TypeBattery:
		return p.BatteryScrapCost
	case assets.TypeRenewable:
		return p.RenewableScrapCost
	case assets.TypeFossil:
		return p.FossilScrapCost
	}
	return defaultCost
}

// The profit or loss for the given asset. If the asset is in a disallowed state, a large negative number is returned.
func (p Params) PnL(a assets.Asset, pv core.PriceVolatility, globalEmissions int, numCapacityAssets int) int {
	switch a.Type() {
	case assets.TypeRenewable:
		return p.RenewablePnL[pv]
	case assets.TypeBattery:
		if (a.Mode() & assets.OperationModeCapacity) != 0 {
			switch p.CapacityRule{
			case CapacityRulePaymentPerAsset:
				return p.BatteryCapacityPnL[pv]
			case CapacityRuleSharedCapacityPaymentPool:
				return p.CapacityPoolPnL[pv] / numCapacityAssets
			default: // includes NoCapacityMarket
				return -defaultCost
			}
		} else {
			return p.BatteryArbitragePnL[pv]
		}
	case assets.TypeFossil:
		var tax int
		if p.CarbonTaxRule == CarbonTaxRuleApplyCarbonTax && globalEmissions > p.CarbonTaxThreshold {
			tax = p.CarbonTaxCost
		}
		if (a.Mode() & assets.OperationModeCapacity) != 0 {
			switch p.CapacityRule{
			case CapacityRulePaymentPerAsset:
				return p.FossilCapacityPnL[pv] - tax
			case CapacityRuleSharedCapacityPaymentPool:
				return p.CapacityPoolPnL[pv] / numCapacityAssets - tax
			default: // includes NoCapacityMarket
				return -defaultCost
			}
		} else {
			return p.FossilWholesalePnL[pv] - tax
		}
	default:
		return -defaultCost
	}
}
