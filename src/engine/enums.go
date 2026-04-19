package engine

import "github.com/WillMorrison/JouleQuestCardGame/eventlog"



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
