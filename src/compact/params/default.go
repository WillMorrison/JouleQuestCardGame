package params

import (
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

var Default = CompactParams{CapacityRule: params.CapacityRulePaymentPerAsset,
	CarbonTaxRule:            params.CarbonTaxRuleNoCarbonTax,
	WinConditionRule:         params.WinConditionRuleLastFossilLoses,
	GenerationConstraintRule: params.GenerationConstraintRuleMinimum,
	TakeoverRule:             params.TakeoverRuleForcedTakeover,

	InitialCash: 50,
	StartingFossilAssetsPerPlayerCount: [MaxPlayerCount + 1]int32{
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

	RenewablePnL: [4]int32{
		10, // PriceVolatilityLow
		5,  // PriceVolatilityMedium
		0,  // PriceVolatilityHigh
		-5, // PriceVolatilityExtreme
	},
	BatteryArbitragePnL: [4]int32{
		-1, // PriceVolatilityLow
		2,  // PriceVolatilityMedium
		5,  // PriceVolatilityHigh
		8,  // PriceVolatilityExtreme
	},
	BatteryCapacityPnL: [4]int32{
		1, // PriceVolatilityLow
		2, // PriceVolatilityMedium
		3, // PriceVolatilityHigh
		4, // PriceVolatilityExtreme
	},
	FossilWholesalePnL: [4]int32{
		5,  // PriceVolatilityLow
		3,  // PriceVolatilityMedium
		1,  // PriceVolatilityHigh
		-1, // PriceVolatilityExtreme
	},
	FossilCapacityPnL: [4]int32{
		1, // PriceVolatilityLow
		1, // PriceVolatilityMedium
		2, // PriceVolatilityHigh
		3, // PriceVolatilityExtreme
	},
}
