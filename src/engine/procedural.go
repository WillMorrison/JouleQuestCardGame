package engine

import (
	"errors"
	"slices"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/eventlog"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

type ProceduralGameState struct {
	s  StateMachineState
	gs GameState
}

func NewProceduralGame(numPlayers int, gameParams params.Params, logger eventlog.Logger) (*ProceduralGameState, error) {
	gs, err := NewGame(numPlayers, gameParams, logger, nil, nil)
	if err != nil {
		return nil, err
	}
	GameStart(gs)
	pgs := &ProceduralGameState{
		s:  StateMachineStateGameStart,
		gs: *gs,
	}
	pgs.startBuildPhase()
	return pgs, nil
}

func (pgs ProceduralGameState) logEvent() eventlog.LogEvent {
	return pgs.gs.Logger.Event().With(pgs.s)
}

func (pgs ProceduralGameState) PossibleActions() []PlayerAction {
	if pgs.s != StateMachineStateBuildPhase {
		return nil
	}
	return pgs.gs.possibleActions()
}

func (pgs ProceduralGameState) Game() GameState {
	gs := pgs.gs
	gs.Logger = nil
	gs.GetPlayerAction = nil
	gs.GameOverFunc = nil
	return gs
}

func (pgs ProceduralGameState) haveBuildingPlayers() bool {
	for _, p := range pgs.gs.activePlayers() {
		if p.isBuilding {
			return true
		}
	}
	return false
}

var ErrCannotApplyActionsOutsideBuildPhase = errors.New("cannot apply player actions outside the build phase")

func (pgs *ProceduralGameState) startBuildPhase() {
	pgs.s = StateMachineStateBuildPhase
	pgs.gs.Round++
	pgs.gs.Logger = pgs.gs.Logger.SetKey("round", pgs.gs.Round)
	pgs.logEvent().With(GameLogEventStateMachineTransition).Log()
}

func (pgs *ProceduralGameState) runUntilBuildPhase() {
	switch pgs.s {
	case StateMachineStateGameStart:
		GameStart(&pgs.gs)
		pgs.startBuildPhase()
	case StateMachineStateBuildPhase:
		// do nothing, we're already in the build phase
	case StateMachineStateOperatePhase:
		OperatePhase(&pgs.gs)
		if pgs.gs.Status == GameStatusOngoing {
			pgs.startBuildPhase()
		} else {
			pgs.s = StateMachineStateGameEnd
			GameEnd(&pgs.gs)
		}
	case StateMachineStateGameEnd:
		GameEnd(&pgs.gs)
	}
}

func (pgs *ProceduralGameState) ApplyPlayerAction(chosenAction PlayerAction) {
	if pgs.s != StateMachineStateBuildPhase {
		return
	}

	// Try to apply the player action to the underlying game state
	err := pgs.gs.applyPlayerAction(chosenAction)
	if err != nil {
		pgs.logEvent().With(GameLogEventPlayerActionInvalid).WithKey("invalid_action", chosenAction).WithKey("error", err.Error()).Log()
		return
	}
	pgs.logEvent().With(GameLogEventPlayerAction).WithKey("action", chosenAction).Log()

	// Figure out where the game goes from here.
	actions := pgs.gs.possibleActions()
	if len(actions) == 0 {
		if chosenAction.Type == ActionTypeFinished {
			// If everyone is now done building, run the operate phase.
			if !pgs.haveBuildingPlayers() {
				pgs.s = StateMachineStateOperatePhase
			}
		} else {
			// If there are no possible actions and a player didn't just finish building, then the
			// game cannot continue and there's a loss by technicality.
			if pgs.gs.Params.TakeoverRule == params.TakeoverRuleForcedTakeover {
				// game loss, assets in takeover pool that nobody can afford to take over
				pgs.gs.SetGlobalLossWithReason(LossConditionUnownedTakeoverAssets)
				takeoverMix := assets.AssetMixFrom(slices.Values(pgs.gs.TakeoverPool))
				var money []int
				for _, p := range pgs.gs.Players {
					money = append(money, p.Money)
				}
				pgs.logEvent().With(GameLogEventEveryoneLoses, pgs.gs.Reason).WithKey("takeover_pool", takeoverMix).WithKey("player_funds", money).Log()
			} else {
				pgs.gs.SetGlobalLossWithReason(LossConditionNoActivePlayers) // Should never happen, but if it does, force a game loss
				pgs.logEvent().With(GameLogEventEveryoneLoses, pgs.gs.Reason).Log()
			}
			pgs.s = StateMachineStateGameEnd
		}
	}
	pgs.runUntilBuildPhase()
}
