package engine

import (
	"slices"
	"testing"

	"github.com/WillMorrison/JouleQuestCardGame/eventlog"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

func Test_Procedural_E2E_NoopPlayers(t *testing.T) {
	// Arrange

	// This causes each player to just finish the build round.
	var getActionJustFinish GetPlayerAction = func(pas []PlayerAction) PlayerAction {
		i := slices.IndexFunc(pas, func(pa PlayerAction) bool { return pa.Type == ActionTypeFinished })
		if i == -1 {
			t.Fatalf("No finished action in %+v", pas)
		}
		return pas[i]
	}
	testLogger := eventlog.NewJsonLogger(t.Output())
	// Create a game with 4 players who do nothing
	game, err := NewProceduralGame(4, params.Default, testLogger)
	if err != nil {
		t.Fatalf("Couldn't create new game: %s", err)
	}

	// Act
	for game.Game().Status == GameStatusOngoing {
		action := getActionJustFinish(game.PossibleActions())
		game.ApplyPlayerAction(action)
	}

	// Assert
	if game.Game().Status != GameStatusLoss || game.Game().Reason != LossConditionCarbonEmissionsExceeded {
		t.Errorf("Game status was %s: %q, want %s: %q", game.Game().Status.String(), game.Game().Reason.String(), GameStatusLoss.String(), LossConditionCarbonEmissionsExceeded.String())
	}
}
