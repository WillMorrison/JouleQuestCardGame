package core

import (
	"testing"
)

func Test_Capacity_PnL_is_not_volatility_sensitive(t *testing.T) {
	if CapacityPnLTable[PriceVolatilityLow] != CapacityPnLTable[PriceVolatilityMedium] ||
		CapacityPnLTable[PriceVolatilityMedium] != CapacityPnLTable[PriceVolatilityHigh] ||
		CapacityPnLTable[PriceVolatilityHigh] != CapacityPnLTable[PriceVolatilityExtreme] {
		t.Errorf("Capacity PnL table should not vary with price volatility")
	}

	if FossilCapacityPnLTableWithCarbonTax[PriceVolatilityLow] != FossilCapacityPnLTableWithCarbonTax[PriceVolatilityMedium] ||
		FossilCapacityPnLTableWithCarbonTax[PriceVolatilityMedium] != FossilCapacityPnLTableWithCarbonTax[PriceVolatilityHigh] ||
		FossilCapacityPnLTableWithCarbonTax[PriceVolatilityHigh] != FossilCapacityPnLTableWithCarbonTax[PriceVolatilityExtreme] {
		t.Errorf("Fossil Capacity PnL table with carbon tax should not vary with price volatility")
	}
}

func Test_Renewable_PnL_decreases_with_volatility(t *testing.T) {
	for i := PriceVolatilityLow; i < PriceVolatilityExtreme; i++ {
		higherVolatility := i + 1
		currentVolatility := i
		higherVolatilityPnL := RenewablePnLTable[higherVolatility]
		currentVolatilityPnL := RenewablePnLTable[currentVolatility]
		if currentVolatilityPnL <= higherVolatilityPnL {
			t.Errorf("Renewable PnL[%s] = %d, want > to Renewable PnL[%s] = %d", currentVolatility.String(), currentVolatilityPnL, higherVolatility.String(), higherVolatilityPnL)
		}
	}
}

func Test_fossil_PnL_decreases_with_volatility(t *testing.T) {
	for i := PriceVolatilityLow; i < PriceVolatilityExtreme; i++ {
		higherVolatility := i + 1
		currentVolatility := i
		higherVolatilityPnL := FossilPnLTable[higherVolatility]
		currentVolatilityPnL := FossilPnLTable[currentVolatility]
		if currentVolatilityPnL <= higherVolatilityPnL {
			t.Errorf("Fossil PnL[%s] = %d, want > to Fossil PnL[%s] = %d", currentVolatility.String(), currentVolatilityPnL, higherVolatility.String(), higherVolatilityPnL)
		}
	}
}

func Test_fossil_with_carbon_tax_PnL_decreases_with_volatility(t *testing.T) {
	for i := PriceVolatilityLow; i < PriceVolatilityExtreme; i++ {
		higherVolatility := i + 1
		currentVolatility := i
		higherVolatilityPnL := FossilPnLTableWithCarbonTax[higherVolatility]
		currentVolatilityPnL := FossilPnLTableWithCarbonTax[currentVolatility]
		if currentVolatilityPnL <= higherVolatilityPnL {
			t.Errorf("Fossil with carbon tax PnL[%s] = %d, want > to Fossil with carbon tax PnL[%s] = %d", currentVolatility.String(), currentVolatilityPnL, higherVolatility.String(), higherVolatilityPnL)
		}
	}
}

func Test_Fossil_with_carbon_tax_is_less_profitable(t *testing.T) {
	for volatility := range FossilPnLTable {
		if FossilPnLTableWithCarbonTax[volatility] >= FossilPnLTable[volatility] {
			t.Errorf("Fossil PnL with carbon tax (%d) should be less than without (%d) for %s volatility", FossilPnLTableWithCarbonTax[volatility], FossilPnLTable[volatility], PriceVolatility(volatility).String())
		}
	}
}

func Test_Fossil_as_capacity_with_carbon_tax_is_less_profitable(t *testing.T) {
	for volatility := range FossilCapacityPnLTableWithCarbonTax {
		if FossilCapacityPnLTableWithCarbonTax[volatility] >= CapacityPnLTable[volatility] {
			t.Errorf("Fossil as capacity PnL with carbon tax (%d) should be less than without (%d) for %s volatility", FossilCapacityPnLTableWithCarbonTax[volatility], CapacityPnLTable[volatility], PriceVolatility(volatility).String())
		}
	}
}

func Test_Fossil_as_capacity_is_sometimes_more_profitable(t *testing.T) {
	foundMoreProfitable := false
	foundLessProfitable := false
	for i := range FossilPnLTable {
		capacityPnL := CapacityPnLTable[i]
		normalPnL := FossilPnLTable[i]
		if capacityPnL > normalPnL {
			foundMoreProfitable = true
		}
		if capacityPnL < normalPnL {
			foundLessProfitable = true
		}
	}
	if !foundMoreProfitable {
		t.Errorf("Fossil as capacity should be more profitable than normal mode for some volatility levels")
	}
	if !foundLessProfitable {
		t.Errorf("Fossil as capacity should be less profitable than normal mode for some volatility levels")
	}
}

func Test_Fossil_as_capacity_is_sometimes_more_profitable_with_carbon_tax(t *testing.T) {
	foundMoreProfitable := false
	foundLessProfitable := false
	for i := range FossilPnLTable {
		capacityPnL := FossilCapacityPnLTableWithCarbonTax[i]
		normalPnL := FossilPnLTableWithCarbonTax[i]
		if capacityPnL > normalPnL {
			foundMoreProfitable = true
		}
		if capacityPnL < normalPnL {
			foundLessProfitable = true
		}
	}
	if !foundMoreProfitable {
		t.Errorf("Fossil as capacity should be more profitable than normal mode for some volatility levels")
	}
	if !foundLessProfitable {
		t.Errorf("Fossil as capacity should be less profitable than normal mode for some volatility levels")
	}
}

func Test_Battery_PnL_increases_with_volatility(t *testing.T) {
	for i := PriceVolatilityLow; i < PriceVolatilityExtreme; i++ {
		higherVolatility := i + 1
		currentVolatility := i
		higherVolatilityPnL := BatteryPnLTable[higherVolatility]
		currentVolatilityPnL := BatteryPnLTable[currentVolatility]
		if currentVolatilityPnL >= higherVolatilityPnL {
			t.Errorf("Battery PnL[%s] = %d, want < to Battery PnL[%s] = %d", currentVolatility.String(), currentVolatilityPnL, higherVolatility.String(), higherVolatilityPnL)
		}
	}
}

func Test_Battery_with_service_PnL_increases_with_volatility(t *testing.T) {
	for i := PriceVolatilityLow; i < PriceVolatilityExtreme; i++ {
		higherVolatility := i + 1
		currentVolatility := i
		higherVolatilityPnL := BatteryPnLTableWithService[higherVolatility]
		currentVolatilityPnL := BatteryPnLTableWithService[currentVolatility]
		if currentVolatilityPnL >= higherVolatilityPnL {
			t.Errorf("Battery with service PnL[%s] = %d, want < to Battery with service PnL[%s] = %d", currentVolatility.String(), currentVolatilityPnL, higherVolatility.String(), higherVolatilityPnL)
		}
	}
}

func Test_Battery_as_capacity_is_sometimes_more_profitable_than_arbitrage(t *testing.T) {
	foundMoreProfitable := false
	foundLessProfitable := false
	for i := range BatteryPnLTable {
		capacityPnL := CapacityPnLTable[i]
		arbitragePnL := BatteryPnLTable[i]
		if capacityPnL > arbitragePnL {
			foundMoreProfitable = true
		}
		if capacityPnL < arbitragePnL {
			foundLessProfitable = true
		}
	}
	if !foundMoreProfitable {
		t.Errorf("Battery as capacity should be more profitable than arbitrage for some volatility levels")
	}
	if !foundLessProfitable {
		t.Errorf("Battery as capacity should be less profitable than arbitrage for some volatility levels")
	}
}

func Test_Battery_with_service_sometimes_more_profitable_than_without(t *testing.T) {
	foundMoreProfitable := false
	foundLessProfitable := false
	for i := range BatteryPnLTable {
		withServicePnL := BatteryPnLTableWithService[i]
		withoutServicePnL := BatteryPnLTable[i]
		if withServicePnL > withoutServicePnL {
			foundMoreProfitable = true
		}
		if withServicePnL < withoutServicePnL {
			foundLessProfitable = true
		}
	}
	if !foundMoreProfitable {
		t.Errorf("Battery with service should be more profitable than without service for some volatility levels")
	}
	if !foundLessProfitable {
		t.Errorf("Battery with service should be less profitable than without service for some volatility levels")
	}
}

func Test_assets_can_have_losses(t *testing.T) {
	if BatteryPnLTableWithService[PriceVolatilityLow] >= 0 {
		t.Errorf("Battery with service PnL[%s] = %d, want < 0", PriceVolatilityLow.String(), BatteryPnLTableWithService[PriceVolatilityLow])
	}
	if FossilPnLTable[PriceVolatilityExtreme] >= 0 {
		t.Errorf("Fossil PnL[%s] = %d, want < 0", PriceVolatilityExtreme.String(), FossilPnLTable[PriceVolatilityExtreme])
	}
	if FossilPnLTableWithCarbonTax[PriceVolatilityExtreme] >= 0 {
		t.Errorf("Fossil with carbon tax PnL[%s] = %d, want < 0", PriceVolatilityExtreme.String(), FossilPnLTableWithCarbonTax[PriceVolatilityExtreme])
	}
	if RenewablePnLTable[PriceVolatilityExtreme] >= 0 {
		t.Errorf("Renewable PnL[%s] = %d, want < 0", PriceVolatilityExtreme.String(), RenewablePnLTable[PriceVolatilityExtreme])
	}
}

func Test_Build_Costs_less_than_scrap(t *testing.T) {
	if BatteryBuildCost <= BatteryScrapCost {
		t.Errorf("Battery build cost (%d) should be greater than scrap cost (%d)", BatteryBuildCost, BatteryScrapCost)
	}
	if RenewableBuildCost <= RenewableScrapCost {
		t.Errorf("Renewable build cost (%d) should be greater than scrap cost (%d)", RenewableBuildCost, RenewableScrapCost)
	}
	if FossilBuildCost <= FossilScrapCost {
		t.Errorf("Fossil build cost (%d) should be greater than scrap cost (%d)", FossilBuildCost, FossilScrapCost)
	}
}

func Test_Build_costs_less_than_starting_money(t *testing.T) {
	if BatteryBuildCost >= InitialCash {
		t.Errorf("Battery build cost (%d) should be less than starting money (%d)", BatteryBuildCost, InitialCash)
	}
	if RenewableBuildCost >= InitialCash {
		t.Errorf("Renewable build cost (%d) should be less than starting money (%d)", RenewableBuildCost, InitialCash)
	}
	if FossilBuildCost >= InitialCash {
		t.Errorf("Fossil build cost (%d) should be less than starting money (%d)", FossilBuildCost, InitialCash)
	}
}

func Test_Starting_assets_meet_minimum_generation(t *testing.T) {
	for numPlayers, numFossil := range StartingFossilAssetsPerPlayer {
		if numFossil*numPlayers <= MinimumGenerationAssets {
			t.Errorf("Starting fossil assets (%d assets * %d players = %d) should exceed minimum generation assets (%d)", numFossil, numPlayers, numFossil*numPlayers, MinimumGenerationAssets)
		}
	}
}

func Test_Emissions_cap_reasonable(t *testing.T) {
	for numPlayers, numFossil := range StartingFossilAssetsPerPlayer {
		if numFossil*(numFossil+1)/2*numPlayers >= EmissionsCap {
			t.Errorf("Emissions cap (%d) would be exceeded by %d players starting with %d fossil assets and scrapping one per round. Raise the cap.", EmissionsCap, numPlayers, numFossil)
		} else {
			t.Logf("%d players starting with %d fossil assets and scrapping one per round would result in %d emissions", numPlayers, numFossil, numFossil*(numFossil+1)/2*numPlayers)
		}

		if EmissionsCap/(numFossil*numPlayers) > 20 {
			t.Errorf("Emissions cap (%d) would allow %d players starting with %d fossil assets to do nothing for 20 rounds. Lower the cap.", EmissionsCap, numPlayers, numFossil)
		} else {
			t.Logf("%d players starting with %d fossil assets could do nothing for %d rounds before exceeding the emissions cap", numPlayers, numFossil, EmissionsCap/(numFossil*numPlayers))
		}
	}
}

func Test_player_can_keep_generation_constant_in_round_one(t *testing.T) {
	if FossilScrapCost+RenewableBuildCost > InitialCash {
		t.Errorf("Player cannot keep generation constant in round one by scrapping a fossil and building a renewable: scrap cost (%d) + build cost (%d) > starting money (%d)", FossilScrapCost, RenewableBuildCost, InitialCash)
	}
}

func Test_emissions_cap_is_greater_than_the_carbon_tax_threshold(t *testing.T) {
	if EmissionsCap <= CarbonTaxThreshold {
		t.Errorf("Emissions cap (%d) should be greater than carbon tax threshold (%d)", EmissionsCap, CarbonTaxThreshold)
	}
}
