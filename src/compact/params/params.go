// Package params holds a fixed-layout, map-free copy of game parameters for the compact / WASM engine.
package params

import (
	"errors"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

// MaxPlayers is the maximum number of players supported by the compact engine.
const MaxPlayers = 10

// MaxPlayerCount is the inclusive upper bound for indexing StartingFossilAssetsPerPlayerCount by player count.
const MaxPlayerCount = MaxPlayers

const defaultCost int32 = 1 << 30

// CompactParams mirrors legacy params.Params with only fixed-size fields (no maps or slices).
type CompactParams struct {
	CapacityRule             params.CapacityRule
	CarbonTaxRule            params.CarbonTaxRule
	WinConditionRule         params.WinConditionRule
	GenerationConstraintRule params.GenerationConstraintRule
	TakeoverRule             params.TakeoverRule

	InitialCash int32
	// StartingFossilAssetsPerPlayerCount is indexed by player count (1..MaxPlayerCount); index 0 unused.
	StartingFossilAssetsPerPlayerCount [MaxPlayerCount + 1]int32

	BatteryBuildCost   int32
	BatteryScrapCost   int32
	RenewableBuildCost int32
	RenewableScrapCost int32
	FossilBuildCost    int32
	FossilScrapCost    int32

	EmissionsCap         int32
	GenerationConstraint int32
	CarbonTaxThreshold   int32
	CarbonTaxCost        int32
	RenewablePenetration int32

	RenewablePnL        [4]int32
	BatteryArbitragePnL [4]int32
	BatteryCapacityPnL  [4]int32
	FossilWholesalePnL  [4]int32
	FossilCapacityPnL   [4]int32
	CapacityPoolPnL     [4]int32
}

func int32FromPnL(t core.PnLTable) [4]int32 {
	return [4]int32{int32(t[0]), int32(t[1]), int32(t[2]), int32(t[3])}
}

var NegativeAssetsError = errors.New("negative asset count")

// FromLegacy builds CompactParams from the canonical params package value.
func FromLegacy(p params.Params) (CompactParams, error) {
	var c CompactParams
	c.CapacityRule = p.CapacityRule
	c.CarbonTaxRule = p.CarbonTaxRule
	c.WinConditionRule = p.WinConditionRule
	c.GenerationConstraintRule = p.GenerationConstraintRule
	c.TakeoverRule = p.TakeoverRule

	c.InitialCash = int32(p.InitialCash)
	for n := 1; n <= MaxPlayerCount; n++ {
		v, ok := p.StartingFossilAssetsPerPlayer[n]
		if !ok {
			c.StartingFossilAssetsPerPlayerCount[n] = 0
			continue
		}
		if v < 0 {
			return CompactParams{}, NegativeAssetsError
		}
		c.StartingFossilAssetsPerPlayerCount[n] = int32(v)
	}

	c.BatteryBuildCost = int32(p.BatteryBuildCost)
	c.BatteryScrapCost = int32(p.BatteryScrapCost)
	c.RenewableBuildCost = int32(p.RenewableBuildCost)
	c.RenewableScrapCost = int32(p.RenewableScrapCost)
	c.FossilBuildCost = int32(p.FossilBuildCost)
	c.FossilScrapCost = int32(p.FossilScrapCost)

	c.EmissionsCap = int32(p.EmissionsCap)
	c.GenerationConstraint = int32(p.GenerationConstraint)
	c.CarbonTaxThreshold = int32(p.CarbonTaxThreshold)
	c.CarbonTaxCost = int32(p.CarbonTaxCost)
	c.RenewablePenetration = int32(p.RenewablePenetration)

	c.RenewablePnL = int32FromPnL(p.RenewablePnL)
	c.BatteryArbitragePnL = int32FromPnL(p.BatteryArbitragePnL)
	c.BatteryCapacityPnL = int32FromPnL(p.BatteryCapacityPnL)
	c.FossilWholesalePnL = int32FromPnL(p.FossilWholesalePnL)
	c.FossilCapacityPnL = int32FromPnL(p.FossilCapacityPnL)
	c.CapacityPoolPnL = int32FromPnL(p.CapacityPoolPnL)

	return c, nil
}

// StartingFossils returns the starting fossil count per player for n players.
func (c CompactParams) StartingFossils(numPlayers int32) int32 {
	if numPlayers < 2 || numPlayers > MaxPlayerCount {
		return 0
	}
	return c.StartingFossilAssetsPerPlayerCount[numPlayers]
}

// BuildCost returns the cost to build one asset of the given type.
func (c CompactParams) BuildCost(at assets.Type) int32 {
	switch at {
	case assets.TypeBattery:
		return c.BatteryBuildCost
	case assets.TypeRenewable:
		return c.RenewableBuildCost
	case assets.TypeFossil:
		return c.FossilBuildCost
	default:
		return defaultCost
	}
}

// ScrapCost returns the cost to scrap one asset of the given type.
func (c CompactParams) ScrapCost(at assets.Type) int32 {
	switch at {
	case assets.TypeBattery:
		return c.BatteryScrapCost
	case assets.TypeRenewable:
		return c.RenewableScrapCost
	case assets.TypeFossil:
		return c.FossilScrapCost
	default:
		return defaultCost
	}
}

// TakeoverCost matches legacy params.Params.TakeoverCost (same numeric values as scrap costs).
func (c CompactParams) TakeoverCost(at assets.Type) int32 {
	return c.ScrapCost(at)
}

// OperatePnLForPlayerMix returns total market PnL for one player's asset mix for the operate phase.
// volIdx is core.PriceVolatility (0..3). worldCapacityAssets is global capacity asset count from the grid snapshot.
func (c CompactParams) OperatePnLForPlayerMix(m assets.AssetMix, volIdx int32, globalEmissions, worldCapacityAssets int32) int32 {
	if volIdx < 0 || volIdx > 3 {
		return -defaultCost
	}
	v := volIdx
	numCap := worldCapacityAssets
	if numCap < 1 {
		numCap = 1
	}

	tax := int32(0)
	if c.CarbonTaxRule == params.CarbonTaxRuleApplyCarbonTax && globalEmissions > c.CarbonTaxThreshold {
		tax = c.CarbonTaxCost
	}

	var sum int32
	sum += int32(m.Renewables) * c.RenewablePnL[v]
	sum += int32(m.BatteriesArbitrage) * c.BatteryArbitragePnL[v]

	switch c.CapacityRule {
	case params.CapacityRulePaymentPerAsset:
		sum += int32(m.BatteriesCapacity) * c.BatteryCapacityPnL[v]
	case params.CapacityRuleSharedCapacityPaymentPool:
		sum += int32(m.BatteriesCapacity) * (c.CapacityPoolPnL[v] / numCap)
	case params.CapacityRuleNoCapacityMarket:
		if m.BatteriesCapacity > 0 {
			sum += int32(m.BatteriesCapacity) * (-defaultCost)
		}
	}

	sum += int32(m.FossilsWholesale) * (c.FossilWholesalePnL[v] - tax)

	switch c.CapacityRule {
	case params.CapacityRulePaymentPerAsset:
		sum += int32(m.FossilsCapacity) * (c.FossilCapacityPnL[v] - tax)
	case params.CapacityRuleSharedCapacityPaymentPool:
		sum += int32(m.FossilsCapacity) * (c.CapacityPoolPnL[v]/numCap - tax)
	case params.CapacityRuleNoCapacityMarket:
		if m.FossilsCapacity > 0 {
			sum += int32(m.FossilsCapacity) * (-defaultCost)
		}
	}

	return sum
}
