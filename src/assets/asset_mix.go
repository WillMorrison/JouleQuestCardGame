package assets

import (
	"fmt"
	"iter"
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
	switch a.Type() {
	case TypeRenewable:
		am.Renewables++
	case TypeBattery:
		switch a.Mode() & OperationModeCapacity {
		case OperationModeCapacity:
			am.BatteriesCapacity++
		default:
			am.BatteriesArbitrage++
		}
	case TypeFossil:
		switch a.Mode() & OperationModeCapacity {
		case OperationModeCapacity:
			am.FossilsCapacity++
		default:
			am.FossilsWholesale++
		}
	}
}

// AddOneAsset adds an asset of the given type to the AssetMix, using the default operation mode.
func (am *AssetMix) AddOneAsset(at Type) {
	switch at {
	case TypeRenewable:
		am.Renewables++
	case TypeBattery:
		am.BatteriesArbitrage++
	case TypeFossil:
		am.FossilsWholesale++
	}
}

// RemoveOneAsset removes one asset of the given type from the AssetMix, starting with capacity mode assets.
// If there are no assets of the given type, this function does nothing.
func (am *AssetMix) RemoveOneAsset(at Type) {
	switch at {
	case TypeRenewable:
		if am.Renewables > 0 {
			am.Renewables--
		}
	case TypeBattery:
		if am.BatteriesCapacity > 0 {
			am.BatteriesCapacity--
		} else if am.BatteriesArbitrage > 0 {
			am.BatteriesArbitrage--
		}
	case TypeFossil:
		if am.FossilsCapacity > 0 {
			am.FossilsCapacity--
		} else if am.FossilsWholesale > 0 {
			am.FossilsWholesale--
		}
	}
}

// Clear resets the AssetMix to an empty state.
func (am *AssetMix) Clear() {
	am.Renewables = 0
	am.BatteriesArbitrage = 0
	am.BatteriesCapacity = 0
	am.FossilsWholesale = 0
	am.FossilsCapacity = 0
}

// PledgeOneAsset moves an asset of the given type from its default operation mode to its capacity operation mode.
// If there are no assets of the given type in the default operation mode, or if the asset type cannot be pledged, this function does nothing.
func (am *AssetMix) PledgeOneAsset(at Type) {
	switch {
	case at == TypeBattery && am.BatteriesArbitrage > 0:
		am.BatteriesArbitrage--
		am.BatteriesCapacity++
	case at == TypeFossil && am.FossilsWholesale > 0:
		am.FossilsWholesale--
		am.FossilsCapacity++
	}
}

// CanPledgeOneAsset returns whether there is at least one asset of the given type that can be pledged to capacity operation mode.
func (am AssetMix) CanPledgeOneAsset(at Type) bool {
	switch {
	case at == TypeBattery && am.BatteriesArbitrage > 0:
		return true
	case at == TypeFossil && am.FossilsWholesale > 0:
		return true
	default:
		return false
	}
}

// ResetAllCapacityPledges moves all assets in capacity operation mode back to their default operation mode.
func (am *AssetMix) ResetAllCapacityPledges() {
	am.BatteriesArbitrage += am.BatteriesCapacity
	am.BatteriesCapacity = 0
	am.FossilsWholesale += am.FossilsCapacity
	am.FossilsCapacity = 0
}

// TakeOneAssetFrom takes one asset of the given type from another AssetMix and adds it to this AssetMix, if possible.
// If the other AssetMix has no assets of the given type, this function does nothing.
// Assets taken from the other AssetMix will revert to their default operation modes.
func (am *AssetMix) TakeOneAssetFrom(at Type, other *AssetMix) {
	switch at {
	case TypeRenewable:
		if other.Renewables > 0 {
			other.Renewables--
			am.Renewables++
		}
	case TypeBattery:
		if other.BatteriesCapacity > 0 {
			other.BatteriesCapacity--
			am.BatteriesArbitrage++
		} else if other.BatteriesArbitrage > 0 {
			other.BatteriesArbitrage--
			am.BatteriesArbitrage++
		}
	case TypeFossil:
		if other.FossilsCapacity > 0 {
			other.FossilsCapacity--
			am.FossilsWholesale++
		} else if other.FossilsWholesale > 0 {
			other.FossilsWholesale--
			am.FossilsWholesale++
		}
	}
}

// TakeAllAssetsFrom adds the assets from another AssetMix to this one, leaving the other AssetMix empty.
// Assets added to this AssetMix will revert to their default operation modes.
func (am *AssetMix) TakeAllAssetsFrom(other *AssetMix) {
	am.Renewables += other.AssetsOfType(TypeRenewable)
	am.BatteriesArbitrage += other.AssetsOfType(TypeBattery)
	am.FossilsWholesale += other.AssetsOfType(TypeFossil)
	other.Clear()
}

// Add adds the assets from another AssetMix to this one, preserving operation modes.
func (am *AssetMix) Add(other AssetMix) {
	am.Renewables += other.Renewables
	am.BatteriesArbitrage += other.BatteriesArbitrage
	am.BatteriesCapacity += other.BatteriesCapacity
	am.FossilsWholesale += other.FossilsWholesale
	am.FossilsCapacity += other.FossilsCapacity
}

func (am AssetMix) AssetsOfType(at Type) int {
	switch at {
	case TypeBattery:
		return am.BatteriesArbitrage + am.BatteriesCapacity
	case TypeRenewable:
		return am.Renewables
	case TypeFossil:
		return am.FossilsWholesale + am.FossilsCapacity
	default:
		return 0
	}
}

func (am AssetMix) NumAssets() int {
	return am.Renewables + am.BatteriesArbitrage + am.BatteriesCapacity + am.FossilsWholesale + am.FossilsCapacity
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

// AssetMixFrom creates a new AssetMix from an Asset iterator
func AssetMixFrom(it iter.Seq[Asset]) (am AssetMix) {
	for a := range it {
		am.AddAsset(a)
	}
	return
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
