package main

import (
	"testing"

	cgame "github.com/WillMorrison/JouleQuestCardGame/compact/game"
	"github.com/WillMorrison/JouleQuestCardGame/core"
)

func TestInitResetAndApplyAction(t *testing.T) {
	if code := Reset(2); code != int32(cgame.CodeOK) {
		t.Fatalf("Reset: %d", code)
	}
	if NumPlayers() != 2 {
		t.Fatalf("NumPlayers: %d", NumPlayers())
	}
	if GameStatus() != int32(core.GameStatusOngoing) {
		t.Fatalf("status: %d", GameStatus())
	}

	mask := PossibleActionsMask(0)
	if mask == 0 {
		t.Fatal("expected legal actions for player 0")
	}
	if mask&(1<<cgame.ActionFinished) == 0 {
		t.Fatal("expected finished in mask")
	}

	if code := ApplyAction(0, cgame.ActionFinished); code != int32(cgame.CodeOK) {
		t.Fatalf("ApplyAction finished: %d", code)
	}
	if PlayerMoney(0) <= 0 {
		t.Fatal("expected positive money after init")
	}
}

func TestApplyActionErrors(t *testing.T) {
	Reset(2)

	if code := ApplyAction(99, 0); code != int32(cgame.CodeInvalidAction) {
		t.Fatalf("bad index: %d", code)
	}
	if code := ApplyAction(0, 99); code != int32(cgame.CodeInvalidAction) {
		t.Fatalf("bad action: %d", code)
	}
}

func TestAssetBucketGetters(t *testing.T) {
	Reset(2)

	if n := PlayerFossilsWholesaleAssets(0); n <= 0 {
		t.Fatalf("expected starting fossils, got %d", n)
	}
	if TakeoverRenewableAssets() != 0 {
		t.Fatal("expected empty takeover pool")
	}
}
