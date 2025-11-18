package engine

import "github.com/WillMorrison/JouleQuestCardGame/eventlog"

type StateMachineState int

var _ eventlog.Loggable = StateMachineState(0)

//go:generate go tool stringer -type=StateMachineState -trimprefix=GameState
const (
	StateMachineStateGameStart StateMachineState = iota
	StateMachineStateRoundStart
	StateMachineStateBuildPhase
	StateMachineStateOperatePhase
	StateMachineStateRoundEnd
	StateMachineStateGameEnd
)

func (sms StateMachineState) LogKey() string {
	return "state"
}

type GameLogEvent int

var _ eventlog.Loggable = GameLogEvent(0)

//go:generate go tool stringer -type=GameLogEvent -trimprefix=GameLogEvent
const (
	// Game lifecycle events
	GameLogEventStateMachineTransition GameLogEvent = iota

	// Build Phase player actions
	GameLogEventPlayerAction
	GameLogEventPlayerActionInvalid

	// Operate Phase events
	GameLogEventEventDrawn
	GameLogEventGridOutcome
	GameLogEventMarketOutcome
	GameLogEventCarbonTaxApplied

	// Win/Loss events
	GameLogEventPlayerLoses
	GameLogEventEveryoneLoses
	GameLogEventGlobalWin
)

func (gle GameLogEvent) LogKey() string {
	return "game_event"
}

// EventRisk represents the risk level of a random event drawn during the Operate phase.
type EventRisk int

var _ eventlog.Loggable = EventRisk(0)

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

var _ eventlog.Loggable = PlayerStatus(0)

//go:generate go tool stringer -type=PlayerStatus -trimprefix=PlayerStatus
const (
	PlayerStatusActive PlayerStatus = iota // Player is still active in the game
	PlayerStatusLost                       // Player lost due to an individual loss condition
)

func (ps PlayerStatus) LogKey() string {
	return "player_status"
}

type GameStatus int

var _ eventlog.Loggable = GameStatus(0)

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

var _ eventlog.Loggable = LossCondition(0)

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
