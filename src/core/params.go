// Game parameters to be tweaked for balancing.

package core

// CapacityPnLTable defines the PnL values for assets in capacity market mode.
var CapacityPnLTable = PnLTable{
	2, // PriceVolatilityLow
	2, // PriceVolatilityMedium
	2, // PriceVolatilityHigh
	2, // PriceVolatilityExtreme
}

// BatteryPnLTable defines the PnL values for battery assets in normal market mode.
var BatteryPnLTable = PnLTable{
	0, // PriceVolatilityLow
	2, // PriceVolatilityMedium
	4, // PriceVolatilityHigh
	6, // PriceVolatilityExtreme
}

// BatteryPnLTableWithService defines the PnL values for battery assets with a service provider in normal market mode.
var BatteryPnLTableWithService = PnLTable{
	-2, // PriceVolatilityLow
	2,  // PriceVolatilityMedium
	6,  // PriceVolatilityHigh
	10, // PriceVolatilityExtreme
}

// FossilPnLTable defines the PnL values for fossil assets in normal market mode.
var FossilPnLTable = PnLTable{
	5,  // PriceVolatilityLow
	3,  // PriceVolatilityMedium
	1,  // PriceVolatilityHigh
	-1, // PriceVolatilityExtreme
}

// FossilPnLTableWithCarbonTax defines the PnL values for fossil assets under a carbon tax in normal market mode.
var FossilPnLTableWithCarbonTax = PnLTable{
	3,  // PriceVolatilityLow
	1,  // PriceVolatilityMedium
	-1, // PriceVolatilityHigh
	-3, // PriceVolatilityExtreme
}

var FossilCapacityPnLTableWithCarbonTax = PnLTable{
	0, // PriceVolatilityLow
	0, // PriceVolatilityMedium
	0, // PriceVolatilityHigh
	0, // PriceVolatilityExtreme
}

// RenewablePnLTable defines the PnL values for renewable assets in normal market mode.
var RenewablePnLTable = PnLTable{
	10, // PriceVolatilityLow
	5,  // PriceVolatilityMedium
	0,  // PriceVolatilityHigh
	-5, // PriceVolatilityExtreme
}

const (
	BatteryBuildCost   = 40
	BatteryScrapCost   = 5
	RenewableBuildCost = 20
	RenewableScrapCost = 5
	FossilBuildCost    = 40
	FossilScrapCost    = 20
)

// InitialCash is the starting money for each player.
const InitialCash = 50

// EmissionsCap is the maximum carbon emissions allowed in the world.
const EmissionsCap = 100

// CarbonTaxThreshold is the carbon emissions level at which carbon tax is applied.
const CarbonTaxThreshold = 50

// MinimumGenerationAssets is the minimum total generation assets required to avoid global loss.
const MinimumGenerationAssets = 15

// StartingFossilAssetsPerPlayer defines the number of Fossil assets each player starts with based on total players.
var StartingFossilAssetsPerPlayer = map[int]int{
	2: 9,
	3: 7,
	4: 5,
	5: 4,
	6: 3,
	7: 3,
}
