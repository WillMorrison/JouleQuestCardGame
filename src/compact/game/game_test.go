package game

import (
	"testing"

	cparams "github.com/WillMorrison/JouleQuestCardGame/compact/params"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

func TestNewGameFourPlayersStartingMix(t *testing.T) {
	cp, err := cparams.FromLegacy(params.Default)
	if err != nil {
		t.Fatal(err)
	}
	g, err := NewGame(4, cp)
	if err != nil {
		t.Fatal(err)
	}
	if g.Round() != 1 {
		t.Fatalf("Round = %d, want 1", g.Round())
	}
	if g.PlayerCount() != 4 {
		t.Fatalf("PlayerCount = %d", g.PlayerCount())
	}
	wantFossil := int32(5) // params.Default map for 4 players
	for i := 0; i < 4; i++ {
		m := g.PlayerAssetMix(i)
		if m.FossilsWholesale != int(wantFossil) || m.NumAssets() != int(wantFossil) {
			t.Fatalf("player %d mix %+v, want %d wholesale fossils", i, m, wantFossil)
		}
		if g.PlayerMoney(i) != cp.InitialCash {
			t.Fatalf("player %d money %d", i, g.PlayerMoney(i))
		}
	}
}

func TestPossibleActionMaskIncludesBuild(t *testing.T) {
	cp, _ := cparams.FromLegacy(params.Default)
	g, _ := NewGame(2, cp)
	mask := g.PossibleActionMask(0)
	if mask&(1<<ActionBuildRenewable) == 0 {
		t.Fatalf("expected renewable build bit, mask=%b", mask)
	}
	if mask&(1<<ActionFinished) == 0 {
		t.Fatalf("expected finished bit for forced takeover with empty pool, mask=%b", mask)
	}
}

func TestApplyFinishedAdvancesWhenAllDone(t *testing.T) {
	cp, _ := cparams.FromLegacy(params.Default)
	g, _ := NewGame(2, cp)
	// Both players immediately finish build (empty takeover pool with forced rule).
	for _, pi := range []int{0, 1} {
		if err := g.ApplyPlayerAction(pi, ActionFinished); err != nil {
			t.Fatal(err)
		}
	}
	// runUntilBuildPhase runs one operate then startBuildPhase when still ongoing.
	if g.Status != core.GameStatusOngoing {
		t.Fatalf("Status = %v, want ongoing", g.Status)
	}
	if g.phase != phaseBuild {
		t.Fatalf("Phase = %d, want build after operate+startBuildPhase", g.phase)
	}
	if g.Round() != 2 {
		t.Fatalf("Round = %d, want 2", g.Round())
	}
}
