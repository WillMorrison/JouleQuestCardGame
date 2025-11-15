package params

import (
	"errors"
	"fmt"

	"github.com/WillMorrison/JouleQuestCardGame/core"
)

func isIncreasing(table core.PnLTable, name string) error {
	var errs []error
	for currentVolatility := core.PriceVolatilityLow; currentVolatility < core.PriceVolatilityExtreme; currentVolatility++ {
		higherVolatility := currentVolatility + 1
		currentVolatilityPnL := table[currentVolatility]
		higherVolatilityPnL := table[higherVolatility]
		if currentVolatilityPnL >= higherVolatilityPnL {
			errs = append(errs, fmt.Errorf("%s[%s] = %d should be less than %s[%s] = %d", name, currentVolatility.String(), currentVolatilityPnL, name, higherVolatility.String(), higherVolatilityPnL))
		}
	}
	return errors.Join(errs...)
}

func isNonDecreasing(table core.PnLTable, name string) error {
	var errs []error
	for currentVolatility := core.PriceVolatilityLow; currentVolatility < core.PriceVolatilityExtreme; currentVolatility++ {
		higherVolatility := currentVolatility + 1
		currentVolatilityPnL := table[currentVolatility]
		higherVolatilityPnL := table[higherVolatility]
		if currentVolatilityPnL > higherVolatilityPnL {
			errs = append(errs, fmt.Errorf("%s[%s] = %d should be less than or equal to %s[%s] = %d", name, currentVolatility.String(), currentVolatilityPnL, name, higherVolatility.String(), higherVolatilityPnL))
		}
	}
	return errors.Join(errs...)
}

func isDecreasing(table core.PnLTable, name string) error {
	var errs []error
	for currentVolatility := core.PriceVolatilityLow; currentVolatility < core.PriceVolatilityExtreme; currentVolatility++ {
		higherVolatility := currentVolatility + 1
		currentVolatilityPnL := table[currentVolatility]
		higherVolatilityPnL := table[higherVolatility]
		if currentVolatilityPnL <= higherVolatilityPnL {
			errs = append(errs, fmt.Errorf("%s[%s] = %d should be more than %s[%s] = %d", name, currentVolatility.String(), currentVolatilityPnL, name, higherVolatility.String(), higherVolatilityPnL))
		}
	}
	return errors.Join(errs...)
}

// Checks that every element of a is greater than the corresponding element of b
func isElementwiseGreater(a, b core.PnLTable, namea, nameb string) error {
	var errs []error
	for i := range a {
		if a[i] <= b[i] {
			errs = append(errs, fmt.Errorf("%s[%s] = %d should be more than %s[%s] = %d", namea, core.PriceVolatility(i).String(), a[i], nameb, core.PriceVolatility(i), b[i]))
		}
	}
	return errors.Join(errs...)
}

// Checks that some elements of a are greater than the corresponding element of b, and some are lesser
func isElementwiseGreaterAndLesser(a, b core.PnLTable, namea, nameb string) error {
	var errs []error
	var foundMore, foundLess bool
	for i := range a {
		if a[i] < b[i] {
			foundLess = true
		}
		if a[i] > b[i] {
			foundMore = true
		}
	}
	if !foundMore {
		errs = append(errs, fmt.Errorf("no elements of %s are greater than those of %s", namea, nameb))
	}
	if !foundLess {
		errs = append(errs, fmt.Errorf("no elements of %s are less than those of %s", namea, nameb))
	}
	return errors.Join(errs...)
}

// Valid returns an error if the parameters aren't sensible
func (p Params) Valid() error {
	var errs []error
	var zeroPnL core.PnLTable

	// Check that rule enums are valid
	switch p.CapacityRule {
	case CapacityRuleNoCapacityMarket, CapacityRulePaymentPerAsset, CapacityRuleSharedCapacityPaymentPool:
		break
	default:
		errs = append(errs, fmt.Errorf("capacity rule is not valid"))
	}
	switch p.CarbonTaxRule {
	case CarbonTaxRuleNoCarbonTax, CarbonTaxRuleApplyCarbonTax:
		break
	default:
		errs = append(errs, fmt.Errorf("carbon tax rule is not valid"))
	}
	switch p.WinConditionRule {
	case WinConditionRuleLastFossilLoses, WinConditionRuleRenewablePenetrationThreshold:
		break
	default:
		errs = append(errs, fmt.Errorf("win condition rule is not valid"))
	}
	switch p.GenerationConstraintRule {
	case GenerationConstraintRuleMaxDecrease, GenerationConstraintRuleMinimum:
		break
	default:
		errs = append(errs, fmt.Errorf("generation constraint rule is not valid"))
	}

	// Check that PnL does the right thing based on volatility
	errs = append(errs, isDecreasing(p.RenewablePnL, "RenewablePnL"))
	errs = append(errs, isIncreasing(p.BatteryArbitragePnL, "BatteryArbitragePnL"))
	errs = append(errs, isDecreasing(p.FossilWholesalePnL, "FossilWholesalePnL"))
	switch p.CapacityRule {
	case CapacityRulePaymentPerAsset:
		errs = append(errs, isNonDecreasing(p.BatteryCapacityPnL, "BatteryCapacityPnL"))
		errs = append(errs, isNonDecreasing(p.FossilCapacityPnL, "FossilCapacityPnL"))
	case CapacityRuleSharedCapacityPaymentPool:
		errs = append(errs, isNonDecreasing(p.CapacityPoolPnL, "CapacityPoolPnL"))
		errs = append(errs, isElementwiseGreater(p.CapacityPoolPnL, zeroPnL, "CapacityPoolPnL", "zeroPnL"))
	}

	// Check that putting assets in capacity mode is a meaningful decision
	if p.CapacityRule == CapacityRulePaymentPerAsset {
		errs = append(errs, isElementwiseGreaterAndLesser(p.BatteryArbitragePnL, p.BatteryCapacityPnL, "BatteryArbitragePnL", "BatteryCapacityPnL"))
		errs = append(errs, isElementwiseGreaterAndLesser(p.FossilWholesalePnL, p.FossilCapacityPnL, "FossilWholesalePnL", "FossilCapacityPnL"))
	}

	// Check that assets can have losses
	errs = append(errs, isElementwiseGreaterAndLesser(p.RenewablePnL, zeroPnL, "RenewablePnL", "zeroPnL"))
	errs = append(errs, isElementwiseGreaterAndLesser(p.BatteryArbitragePnL, zeroPnL, "BatteryArbitragePnL", "zeroPnL"))
	errs = append(errs, isElementwiseGreaterAndLesser(p.FossilWholesalePnL, zeroPnL, "FossilWholesalePnL", "zeroPnL"))

	if p.InitialCash <= 0 {
		errs = append(errs, fmt.Errorf("initial money (%d) should be greater than 0", p.InitialCash))
	}

	// Check that build cost is more than scrap cost
	if p.BatteryBuildCost <= p.BatteryScrapCost {
		errs = append(errs, fmt.Errorf("battery build cost (%d) should be greater than scrap cost (%d)", p.BatteryBuildCost, p.BatteryScrapCost))
	}
	if p.RenewableBuildCost <= p.RenewableScrapCost {
		errs = append(errs, fmt.Errorf("renewable build cost (%d) should be greater than scrap cost (%d)", p.RenewableBuildCost, p.RenewableScrapCost))
	}
	if p.FossilBuildCost <= p.FossilScrapCost {
		errs = append(errs, fmt.Errorf("fossil build cost (%d) should be greater than scrap cost (%d)", p.FossilBuildCost, p.FossilScrapCost))
	}

	// Check that build costs are less than starting money
	if p.BatteryBuildCost > p.InitialCash {
		errs = append(errs, fmt.Errorf("battery build cost (%d) should be less than initial money (%d)", p.BatteryBuildCost, p.InitialCash))
	}
	if p.RenewableBuildCost > p.InitialCash {
		errs = append(errs, fmt.Errorf("renewable build cost (%d) should be less than initial money (%d)", p.RenewableBuildCost, p.InitialCash))
	}
	if p.FossilBuildCost > p.InitialCash {
		errs = append(errs, fmt.Errorf("fossil build cost (%d) should be less than initial money (%d)", p.FossilBuildCost, p.InitialCash))
	}

	// Check that generation can be kept constant in round 1
	if p.FossilScrapCost+p.RenewableBuildCost > p.InitialCash {
		errs = append(errs, fmt.Errorf("player cannot keep generation constant in round one by scrapping a fossil and building a renewable: scrap cost (%d) + build cost (%d) > starting money (%d)", p.FossilScrapCost, p.RenewableBuildCost, p.InitialCash))
	}

	// Check that starting assets meet minimum generation
	for numPlayers, numFossil := range p.StartingFossilAssetsPerPlayer {
		if numFossil*numPlayers <= p.GenerationConstraint {
			errs = append(errs, fmt.Errorf("starting fossil assets (%d assets * %d players = %d) should exceed minimum generation assets (%d)", numFossil, numPlayers, numFossil*numPlayers, p.GenerationConstraint))
		}
	}

	// Check if carbon tax parameters make sense if it's used
	if p.CarbonTaxRule == CarbonTaxRuleApplyCarbonTax {
		if p.CarbonTaxThreshold <= 0 {
			errs = append(errs, fmt.Errorf("carbon tax threshold (%d) should be greater than 0", p.CarbonTaxThreshold))
		}
		if p.CarbonTaxCost <= 0 {
			errs = append(errs, fmt.Errorf("carbon tax cost (%d) should be greater than 0", p.CarbonTaxCost))
		}

		if p.EmissionsCap <= p.CarbonTaxThreshold {
			errs = append(errs, fmt.Errorf("emissions cap (%d) should be greater than carbon tax threshold (%d)", p.EmissionsCap, p.CarbonTaxThreshold))
		}
	}

	// Check that the emissions cap is reasonable
	for numPlayers, numFossil := range p.StartingFossilAssetsPerPlayer {
		if numFossil*(numFossil+1)/2*numPlayers >= p.EmissionsCap {
			errs = append(errs, fmt.Errorf("emissions cap (%d) would be exceeded by %d players starting with %d fossil assets and scrapping one per round. Raise the cap", p.EmissionsCap, numPlayers, numFossil))
		}

		if p.EmissionsCap/(numFossil*numPlayers) > 20 {
			errs = append(errs, fmt.Errorf("emissions cap (%d) would allow %d players starting with %d fossil assets to do nothing for 20 rounds. Lower the cap", p.EmissionsCap, numPlayers, numFossil))
		}
	}

	// Check that renewable penetration goal is a percentage
	if p.WinConditionRule == WinConditionRuleRenewablePenetrationThreshold {
		if p.RenewablePenetration <= 0 || p.RenewablePenetration > 100 {
			errs = append(errs, fmt.Errorf("renewable penetration (%d) should be between 1 and 100", p.RenewablePenetration))
		}
	}

	return errors.Join(errs...)
}
