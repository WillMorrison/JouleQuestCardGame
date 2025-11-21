package engine

import (
	"slices"
	"testing"

	"github.com/WillMorrison/JouleQuestCardGame/eventlog"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

func Test_Run_E2E_NoopPlayers(t *testing.T) {
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
	game, err := NewGame(4, params.Default, testLogger, getActionJustFinish, nil)
	if err != nil {
		t.Fatalf("Couldn't create new game: %s", err)
	}

	// Act
	game.Run()

	// Assert
	if game.Status != GameStatusLoss || game.Reason != LossConditionCarbonEmissionsExceeded {
		t.Errorf("Game status was %s: %q, want %s: %q", game.Status.String(), game.Reason.String(), GameStatusLoss.String(), LossConditionCarbonEmissionsExceeded.String())
	}
}
