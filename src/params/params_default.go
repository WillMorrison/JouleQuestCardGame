package params

import "github.com/WillMorrison/JouleQuestCardGame/core"

var Default = Params{
	CapacityRule:             CapacityRulePaymentPerAsset,
	CarbonTaxRule:            CarbonTaxRuleNoCarbonTax,
	WinConditionRule:         WinConditionRuleLastFossilLoses,
	GenerationConstraintRule: GenerationConstraintRuleMinimum,

	InitialCash: 50,
	StartingFossilAssetsPerPlayer: map[int]int{
		2: 9,
		3: 7,
		4: 5,
		5: 4,
		6: 3,
		7: 3,
	},

	BatteryBuildCost:   40,
	BatteryScrapCost:   5,
	RenewableBuildCost: 20,
	RenewableScrapCost: 5,
	FossilBuildCost:    40,
	FossilScrapCost:    20,

	EmissionsCap:         100,
	GenerationConstraint: 15,

	RenewablePnL: core.PnLTable{
		10, // PriceVolatilityLow
		5,  // PriceVolatilityMedium
		0,  // PriceVolatilityHigh
		-5, // PriceVolatilityExtreme
	},
	BatteryArbitragePnL: core.PnLTable{
		-1, // PriceVolatilityLow
		2,  // PriceVolatilityMedium
		5,  // PriceVolatilityHigh
		8,  // PriceVolatilityExtreme
	},
	BatteryCapacityPnL: core.PnLTable{
		1, // PriceVolatilityLow
		2, // PriceVolatilityMedium
		3, // PriceVolatilityHigh
		4, // PriceVolatilityExtreme
	},
	FossilWholesalePnL: core.PnLTable{
		5,  // PriceVolatilityLow
		3,  // PriceVolatilityMedium
		1,  // PriceVolatilityHigh
		-1, // PriceVolatilityExtreme
	},
	FossilCapacityPnL: core.PnLTable{
		1, // PriceVolatilityLow
		1, // PriceVolatilityMedium
		2, // PriceVolatilityHigh
		3, // PriceVolatilityExtreme
	},
}
