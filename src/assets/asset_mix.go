package assets

import (
	"fmt"
	"strings"
)

// AssetMix for calculating the price volatility and grid stability
type AssetMix struct {
	Renewables         int
	BatteriesArbitrage int
	BatteriesCapacity  int
	FossilsWholesale   int
	FossilsCapacity    int
}

func (am *AssetMix) AddAsset(a Asset) {
	switch asset := a.(type) {
	case *RenewableAsset:
		am.Renewables++
	case *BatteryAsset:
		switch asset.Mode() & OperationModeCapacity {
		case OperationModeCapacity:
			am.BatteriesCapacity++
		default:
			am.BatteriesArbitrage++
		}
	case *FossilAsset:
		switch asset.Mode() & OperationModeCapacity {
		case OperationModeCapacity:
			am.FossilsCapacity++
		default:
			am.FossilsWholesale++
		}
	}
}

func (am AssetMix) GenerationAssets() int {
	return am.Renewables + am.FossilsWholesale + am.FossilsCapacity
}

func (am AssetMix) CapacityAssets() int {
	return am.BatteriesCapacity + am.FossilsCapacity
}

func (am AssetMix) Emissions() int {
	return am.FossilsWholesale + am.FossilsCapacity
}

// RenewablePenetration returns the percentage of generation assets that are renewable.
func (am AssetMix) RenewablePenetration() int {
	totalGen := am.GenerationAssets()
	if totalGen == 0 {
		return 0
	}
	return (am.Renewables * 100) / totalGen
}

// Coefficients for summing (multiples of) asset types in calculations.
type AssetMixCoefficients AssetMix

func (amc AssetMixCoefficients) String() string {
	var parts []string
	appendIfNonZero := func(name string, value int) {
		switch value {
		case 0:
			// Do nothing
		case 1:
			parts = append(parts, name)
		case -1:
			parts = append(parts, fmt.Sprintf("-%s", name))
		default:
			parts = append(parts, fmt.Sprintf("%d*%s", value, name))
		}
	}
	appendIfNonZero("Renewables", amc.Renewables)
	appendIfNonZero("BatteriesArbitrage", amc.BatteriesArbitrage)
	appendIfNonZero("BatteriesCapacity", amc.BatteriesCapacity)
	appendIfNonZero("FossilsWholesale", amc.FossilsWholesale)
	appendIfNonZero("FossilsCapacity", amc.FossilsCapacity)
	if len(parts) == 0 {
		return "0"
	}
	return strings.Join(parts, " + ")
}

// RatioCalculation performs a ratio-based calculation on an AssetMix
type RatioCalculation struct {
	CoefficientsA AssetMixCoefficients
	CoefficientsB AssetMixCoefficients
	Rollover      int // How much greater one side must be to beat the other by "lots". Should be at least 2.
}

func (rc RatioCalculation) String() string {
	return fmt.Sprintf("(%s : %s)[%d]", rc.CoefficientsA.String(), rc.CoefficientsB.String(), rc.Rollover)
}

func (rc RatioCalculation) Do(am AssetMix) int {
	var sideA int = max(0,
		rc.CoefficientsA.Renewables*am.Renewables+
			rc.CoefficientsA.BatteriesArbitrage*am.BatteriesArbitrage+
			rc.CoefficientsA.BatteriesCapacity*am.BatteriesCapacity+
			rc.CoefficientsA.FossilsWholesale*am.FossilsWholesale+
			rc.CoefficientsA.FossilsCapacity*am.FossilsCapacity,
	)

	var sideB int = max(0,
		rc.CoefficientsB.Renewables*am.Renewables+
			rc.CoefficientsB.BatteriesArbitrage*am.BatteriesArbitrage+
			rc.CoefficientsB.BatteriesCapacity*am.BatteriesCapacity+
			rc.CoefficientsB.FossilsWholesale*am.FossilsWholesale+
			rc.CoefficientsB.FossilsCapacity*am.FossilsCapacity,
	)

	switch {
	case sideA == sideB:
		return 1 // A >= B
	case sideA*rc.Rollover <= sideB: // A << B
		return 3
	case sideA >= sideB*rc.Rollover: // A >> B
		return 0
	case sideA < sideB: // A < B
		return 2
	default: // A >= B
		return 1
	}
}

func MapRatioTo[T any](rc RatioCalculation, am AssetMix, results [4]T) T {
	return results[rc.Do(am)]
}
