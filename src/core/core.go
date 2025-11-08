// package core provides core types and constants for the JouleQuest game.
package core

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
