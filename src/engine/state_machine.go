// This file defines the state machine for the game

package engine

/*
State machine state diagram.

	┌─────────┐    ┌──────────┬───►┌────────────┐    ┌───────┐
	│GameStart├───►│BuildPhase│    │OperatePhase├───►│GameEnd│
	└─────────┘    └──────────┘◄───┴────────────┘    └───────┘
*/
type StateMachineState int

//go:generate go tool stringer -type=StateMachineState -trimprefix=GameState
const (
	StateMachineStateGameStart StateMachineState = iota
	StateMachineStateBuildPhase
	StateMachineStateOperatePhase
	StateMachineStateGameEnd
)

func (sms StateMachineState) LogKey() string {
	return "state"
}

// StateRunner is a function that executes one step of the state machine, then transitions to the next
type StateRunner func(gs *GameState) StateRunner

// Run executes the state machine. It is a no-op to Run a game which has finished already
func (gs *GameState) Run() {
	if gs.Status != GameStatusOngoing {
		return
	}
	current := GameStart
	for current != nil {
		current = current(gs)
	}
}

// GameStart logs some things, then transitions to the initial build phase
func GameStart(gs *GameState) StateRunner {
	gs.Logger.Event().
		With(GameLogEventStateMachineTransition, StateMachineStateGameStart).
		WithKey("game_parameters", gs.Params).
		WithKey("num_players", len(gs.Players)).
		Log()
	return BuildPhase
}

// GameEnd logs some stats, then exits the state machine
func GameEnd(gs *GameState) StateRunner {
	gs.Logger.Event().
		With(GameLogEventStateMachineTransition, StateMachineStateGameEnd, gs.Status, gs.Reason).
		WithKey("total_emissions", gs.CarbonEmissions).
		WithKey("players", gs.Players).
		Log()
	return nil
}
