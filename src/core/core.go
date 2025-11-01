// package core provides core types and constants for the JouleQuest game.
package core

// AssetType represents the type of an asset.
type AssetType int

//go:generate go tool stringer -type=AssetType -trimprefix=AssetType
const (
	AssetTypeRenewable AssetType = iota
	AssetTypeFossil
	AssetTypeBattery
)

func (at AssetType) LogKey() string {
	return "asset_type"
}

func (at AssetType) BuildCost() int {
	switch at {
	case AssetTypeRenewable:
		return RenewableBuildCost
	case AssetTypeFossil:
		return FossilBuildCost
	case AssetTypeBattery:
		return BatteryBuildCost
	default:
		panic("unknown asset type: " + at.String())
	}
}

func (at AssetType) ScrapCost() int {
	switch at {
	case AssetTypeRenewable:
		return RenewableScrapCost
	case AssetTypeFossil:
		return FossilScrapCost
	case AssetTypeBattery:
		return BatteryScrapCost
	default:
		panic("unknown asset type: " + at.String())
	}
}

func (at AssetType) IsGeneration() bool {
	return at == AssetTypeRenewable || at == AssetTypeFossil
}

func (at AssetType) CanBeCapacity() bool {
	return at == AssetTypeFossil || at == AssetTypeBattery
}

func (at AssetType) TakeoverCost() int {
	return at.ScrapCost()
}

// PnLTable represents profit and loss values for an asset for different volatility levels.
type PnLTable [4]int

// PriceVolatility represents the price volatility levels of the market.
type PriceVolatility int

//go:generate go tool stringer -type=PriceVolatility -trimprefix=PriceVolatility
const (
	PriceVolatilityLow PriceVolatility = iota
	PriceVolatilityMedium
	PriceVolatilityHigh
	PriceVolatilityExtreme
)

func (pv PriceVolatility) LogKey() string {
	return "price_volatility"
}

type GridStability int

//go:generate go tool stringer -type=GridStability -trimprefix=GridStability
const (
	GridStabilityDangerous GridStability = iota
	GridStabilityBad
	GridStabilityOk
	GridStabilityGood
)

func (gs GridStability) LogKey() string {
	return "grid_stability"
}
