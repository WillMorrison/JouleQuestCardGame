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

// EventRisk represents the risk level of a random event drawn during the Operate phase.
type EventRisk int

//go:generate go tool stringer -type=EventRisk -trimprefix=EventRisk
const (
	EventRiskLow EventRisk = iota
	EventRiskMedium
	EventRiskHigh
)

func (er EventRisk) LogKey() string {
	return "event_risk"
}

type PlayerStatus int

//go:generate go tool stringer -type=PlayerStatus -trimprefix=PlayerStatus
const (
	PlayerStatusActive PlayerStatus = iota // Player is still active in the game
	PlayerStatusLost                       // Player lost due to an individual loss condition
)

func (ps PlayerStatus) LogKey() string {
	return "player_status"
}

type GameStatus int

//go:generate go tool stringer -type=GameStatus -trimprefix=GameStatus
const (
	GameStatusOngoing GameStatus = iota // Game is not finished
	GameStatusWin                       // Game is finished and the remaining active players won
	GameStatusLoss                      // Game is finished and everyone lost
)

func (o GameStatus) LogKey() string {
	return "game_status"
}

// Reasons for individual or global game losses
type LossCondition int

//go:generate go tool stringer -type=LossCondition -trimprefix=LossCondition
const (
	LossConditionNone LossCondition = iota

	// Player loss reasons
	LossConditionPlayerBankrupt
	LossConditionLastPlayerWithFossilAssets

	// Global loss reasons
	LossConditionGridUnstable
	LossConditionInsufficientGeneration
	LossConditionCarbonEmissionsExceeded

	// Technicality global loss reasons
	LossConditionNoActivePlayers
	LossConditionUnownedTakeoverAssets
)

func (lc LossCondition) LogKey() string {
	return "loss_reason"
}
