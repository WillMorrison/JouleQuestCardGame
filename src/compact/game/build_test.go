package game

import (
	"errors"
	"testing"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	cparams "github.com/WillMorrison/JouleQuestCardGame/compact/params"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

func mustNewGame(t *testing.T, numPlayers int, params params.Params) (cparams.CompactParams, *Game) {
	t.Helper()
	cp, err := cparams.FromLegacy(params)
	if err != nil {
		t.Fatal(err)
	}
	g, err := NewGame(numPlayers, cp)
	if err != nil {
		t.Fatal(err)
	}
	return cp, g
}

func TestNewGame_EntersBuildPhaseRoundOneWithActiveBuildingPlayers(t *testing.T) {
	// arrange
	cp, err := cparams.FromLegacy(params.Default)
	if err != nil {
		t.Fatal(err)
	}

	// act
	g, err := NewGame(2, cp)
	if err != nil {
		t.Fatal(err)
	}

	// assert
	if g.phase != phaseBuild {
		t.Errorf("Phase = %v, want build", g.phase)
	}
	if g.Status != core.GameStatusOngoing {
		t.Errorf("Status = %v, want ongoing", g.Status)
	}
	if g.Round() != 1 {
		t.Errorf("Round = %d, want 1", g.Round())
	}
	wantFossils := int(cp.StartingFossils(g.NumPlayers))
	for i := range g.NumPlayers {
		if g.Players[i].Status != core.PlayerStatusActive {
			t.Errorf("player %d status = %v, want active", i, g.Players[i].Status)
		}
		if !g.Players[i].IsBuilding {
			t.Errorf("player %d IsBuilding = false, want true", i)
		}
		if g.PlayerMoney(i) != cp.InitialCash {
			t.Errorf("player %d money = %d, want %d", i, g.PlayerMoney(i), cp.InitialCash)
		}
		m := g.PlayerAssetMix(i)
		if m.FossilsWholesale != wantFossils || m.NumAssets() != wantFossils {
			t.Errorf("player %d mix = %+v, want %d wholesale fossils only", i, m, wantFossils)
		}
	}
}

func TestStartBuildPhase_ResetsCapacityModesForActivePlayers(t *testing.T) {
	// arrange
	_, g := mustNewGame(t, 2, params.Default)
	g.Players[0].Mix = assets.AssetMix{
		Renewables:         2,
		BatteriesArbitrage: 1,
		BatteriesCapacity:  3,
		FossilsWholesale:   4,
		FossilsCapacity:    2,
	}
	roundBefore := g.Round()

	// act
	g.startBuildPhase()

	// assert
	want := assets.AssetMix{
		Renewables:         2,
		BatteriesArbitrage: 4,
		FossilsWholesale:   6,
	}
	if g.Players[0].Mix != want {
		t.Errorf("player 0 mix = %+v, want %+v", g.Players[0].Mix, want)
	}
	if !g.Players[0].IsBuilding {
		t.Error("active player should be building after startBuildPhase")
	}
	if g.Round() != roundBefore+1 {
		t.Errorf("Round = %d, want %d", g.Round(), roundBefore+1)
	}
}

func TestStartBuildPhase_DoesNotReactivateLostPlayers(t *testing.T) {
	// arrange
	_, g := mustNewGame(t, 2, params.Default)
	g.Players[1].Status = core.PlayerStatusLost
	g.Players[1].IsBuilding = false
	g.Players[1].Mix.FossilsCapacity = 3
	lostMixBefore := g.Players[1].Mix

	// act
	g.startBuildPhase()

	// assert
	if g.Players[1].Status != core.PlayerStatusLost {
		t.Errorf("lost player status = %v, want lost", g.Players[1].Status)
	}
	if g.Players[1].IsBuilding {
		t.Error("lost player must not be set to building")
	}
	if g.Players[1].Mix != lostMixBefore {
		t.Errorf("lost player mix changed: got %+v, want %+v", g.Players[1].Mix, lostMixBefore)
	}
}

func TestAfterOperate_NewBuildRoundResetsIsBuildingForActivePlayers(t *testing.T) {
	// arrange
	_, g := mustNewGame(t, 2, params.Default)

	// act — both players finish build, triggering operate then startBuildPhase
	if err := g.ApplyPlayerAction(0, ActionFinished); err != nil {
		t.Fatal(err)
	}
	if err := g.ApplyPlayerAction(1, ActionFinished); err != nil {
		t.Fatal(err)
	}

	// assert
	if g.phase != phaseBuild {
		t.Errorf("Phase = %v, want build after operate", g.phase)
	}
	if g.Round() != 2 {
		t.Errorf("Round = %d, want 2", g.Round())
	}
	for i := range g.NumPlayers {
		if g.Players[i].Status != core.PlayerStatusActive {
			t.Errorf("player %d status = %v, want active", i, g.Players[i].Status)
		}
		if !g.Players[i].IsBuilding {
			t.Errorf("player %d IsBuilding = false after new build round", i)
		}
	}
}

func TestLostPlayer_HasZeroPossibleActionMask(t *testing.T) {
	// arrange
	_, g := mustNewGame(t, 2, params.Default)
	g.Players[0].Status = core.PlayerStatusLost
	g.Players[0].IsBuilding = true

	// act
	mask := g.PossibleActionMask(0)

	// assert
	if mask != 0 {
		t.Errorf("mask = %b, want 0 for lost player", mask)
	}
}

func TestPossibleActionMask_NonBuildingActivePlayer_HasZeroPossibleActionMask(t *testing.T) {
	// arrange
	_, g := mustNewGame(t, 2, params.Default)
	g.Players[0].IsBuilding = false

	// act
	mask := g.PossibleActionMask(0)

	// assert
	if mask != 0 {
		t.Errorf("mask = %b, want 0 for player not building", mask)
	}
}

func TestPossibleActionMask_PlayerIndexOutOfRange_HasZeroPossibleActionMask(t *testing.T) {
	// arrange
	_, g := mustNewGame(t, 2, params.Default)

	// act & assert
	if mask := g.PossibleActionMask(-1); mask != 0 {
		t.Errorf("mask for pi=-1 = %b, want 0", mask)
	}
	if mask := g.PossibleActionMask(2); mask != 0 {
		t.Errorf("mask for pi=2 = %b, want 0", mask)
	}
}

func TestPossibleActionMask_ForcedTakeoverWithNonemptyPool_ExcludesFinishedFromMask(t *testing.T) {
	// arrange
	p := params.BuilderFrom(params.Default).TakeoverRule(params.TakeoverRuleForcedTakeover).Build()
	_, g := mustNewGame(t, 2, p)
	g.TakeoverPool.FossilsWholesale = 1

	// act
	mask := g.PossibleActionMask(0)

	// assert
	if mask&(1<<ActionFinished) != 0 {
		t.Errorf("finished bit set with nonempty pool under forced takeover, mask=%b", mask)
	}
}

func TestApplyPlayerAction_ForcedTakeoverWithNonemptyPool_FinishRejected(t *testing.T) {
	// arrange
	p := params.BuilderFrom(params.Default).TakeoverRule(params.TakeoverRuleForcedTakeover).Build()
	_, g := mustNewGame(t, 2, p)
	g.TakeoverPool.Renewables = 1

	// act
	err := g.ApplyPlayerAction(0, ActionFinished)

	// assert
	if !errors.Is(err, ErrInvalidAction) {
		t.Errorf("err = %v, want ErrInvalidAction", err)
	}
	if !g.Players[0].IsBuilding {
		t.Error("player should still be building after rejected finish")
	}
}

func TestPossibleActionMask_VirtualOwnerWithNonemptyPool_FinishedAllowed(t *testing.T) {
	// arrange
	p := params.BuilderFrom(params.Default).TakeoverRule(params.TakeoverRuleVirtualOwner).Build()
	_, g := mustNewGame(t, 2, p)
	g.TakeoverPool.BatteriesArbitrage = 1

	// act
	mask := g.PossibleActionMask(0)

	// assert
	if mask&(1<<ActionFinished) == 0 {
		t.Errorf("finished bit not set under virtual owner with takeover pool assets, mask=%b", mask)
	}
}

func TestApplyPlayerAction_BuildRenewable_UpdatesMoneyAndMix(t *testing.T) {
	// arrange
	p := params.BuilderFrom(params.Default).RenewableCosts(20, 5).Build()
	_, g := mustNewGame(t, 2, p)
	g.Players[0].Money = 100
	g.Players[0].Mix = assets.AssetMix{FossilsWholesale: 5}

	// act
	if err := g.ApplyPlayerAction(0, ActionBuildRenewable); err != nil {
		t.Fatal(err)
	}

	// assert
	var wantMoney int32 = 80
	if g.PlayerMoney(0) != wantMoney {
		t.Errorf("money = %d, want %d", g.PlayerMoney(0), wantMoney)
	}
	want := assets.AssetMix{Renewables: 1, FossilsWholesale: 5}
	if g.PlayerAssetMix(0) != want {
		t.Errorf("mix = %+v, want %+v", g.PlayerAssetMix(0), want)
	}
}

func TestApplyPlayerAction_BuildBattery_UpdatesMoneyAndMix(t *testing.T) {
	// arrange
	p := params.BuilderFrom(params.Default).BatteryCosts(40, 5).Build()
	_, g := mustNewGame(t, 2, p)
	g.Players[0].Money = 100
	g.Players[0].Mix = assets.AssetMix{FossilsWholesale: 5}

	// act
	if err := g.ApplyPlayerAction(0, ActionBuildBattery); err != nil {
		t.Fatal(err)
	}

	// assert
	var wantMoney int32 = 60
	if g.PlayerMoney(0) != wantMoney {
		t.Errorf("money = %d, want %d", g.PlayerMoney(0), wantMoney)
	}
	want := assets.AssetMix{BatteriesArbitrage: 1, FossilsWholesale: 5}
	if g.PlayerAssetMix(0) != want {
		t.Errorf("mix = %+v, want %+v", g.PlayerAssetMix(0), want)
	}
}

func TestApplyPlayerAction_BuildFossil_UpdatesMoneyAndMix(t *testing.T) {
	// arrange
	p := params.BuilderFrom(params.Default).FossilCosts(40, 20).Build()
	_, g := mustNewGame(t, 2, p)
	g.Players[0].Money = 100
	g.Players[0].Mix = assets.AssetMix{FossilsWholesale: 5}

	// act
	if err := g.ApplyPlayerAction(0, ActionBuildFossil); err != nil {
		t.Fatal(err)
	}

	// assert
	var wantMoney int32 = 60
	if g.PlayerMoney(0) != wantMoney {
		t.Errorf("money = %d, want %d", g.PlayerMoney(0), wantMoney)
	}
	want := assets.AssetMix{FossilsWholesale: 6}
	if g.PlayerAssetMix(0) != want {
		t.Errorf("mix = %+v, want %+v", g.PlayerAssetMix(0), want)
	}
}

func TestApplyPlayerAction_ScrapFossil_UpdatesMoneyAndMix(t *testing.T) {
	// arrange
	p := params.BuilderFrom(params.Default).FossilCosts(40, 20).Build()
	_, g := mustNewGame(t, 2, p)
	g.Players[0].Money = 100
	g.Players[0].Mix = assets.AssetMix{FossilsWholesale: 5}

	// act
	if err := g.ApplyPlayerAction(0, ActionScrapFossil); err != nil {
		t.Fatal(err)
	}

	// assert
	var wantMoney int32 = 80
	if g.PlayerMoney(0) != wantMoney {
		t.Errorf("money = %d, want %d", g.PlayerMoney(0), wantMoney)
	}
	want := assets.AssetMix{FossilsWholesale: 4}
	if g.PlayerAssetMix(0) != want {
		t.Errorf("mix = %+v, want %+v", g.PlayerAssetMix(0), want)
	}
}

func TestApplyPlayerAction_Takeover_UpdatesPlayerPoolAndMoney(t *testing.T) {
	// arrange
	p := params.BuilderFrom(params.Default).
		TakeoverRule(params.TakeoverRuleVirtualOwner).
		RenewableCosts(20, 5).
		Build()
	_, g := mustNewGame(t, 2, p)
	g.TakeoverPool = assets.AssetMix{Renewables: 2}
	g.Players[0].Money = 100
	g.Players[0].Mix = assets.AssetMix{FossilsWholesale: 5}

	// act
	if err := g.ApplyPlayerAction(0, ActionTakeoverRenewable); err != nil {
		t.Fatal(err)
	}

	// assert
	var wantMoney int32 = 95
	if g.PlayerMoney(0) != wantMoney {
		t.Errorf("money = %d, want %d", g.PlayerMoney(0), wantMoney)
	}
	wantMix := assets.AssetMix{Renewables: 1, FossilsWholesale: 5}
	if g.PlayerAssetMix(0) != wantMix {
		t.Errorf("player mix = %+v, want %+v", g.PlayerAssetMix(0), wantMix)
	}
	wantPool := assets.AssetMix{Renewables: 1}
	if g.TakeoverPoolMix() != wantPool {
		t.Errorf("pool = %+v, want %+v", g.TakeoverPoolMix(), wantPool)
	}
}

func TestApplyPlayerAction_TakeoverScrap_UpdatesPoolAndMoneyOnly(t *testing.T) {
	// arrange
	p := params.BuilderFrom(params.Default).
		TakeoverRule(params.TakeoverRuleVirtualOwner).
		RenewableCosts(20, 5).
		Build()
	_, g := mustNewGame(t, 2, p)
	g.TakeoverPool = assets.AssetMix{Renewables: 2}
	g.Players[0].Money = 100
	g.Players[0].Mix = assets.AssetMix{FossilsWholesale: 5}

	// act
	if err := g.ApplyPlayerAction(0, ActionTakeoverScrapRenewable); err != nil {
		t.Fatal(err)
	}

	// assert
	var wantMoney int32 = 95
	if g.PlayerMoney(0) != wantMoney {
		t.Errorf("money = %d, want %d", g.PlayerMoney(0), wantMoney)
	}
	wantMix := assets.AssetMix{FossilsWholesale: 5}
	if g.PlayerAssetMix(0) != wantMix {
		t.Errorf("player mix = %+v, want %+v", g.PlayerAssetMix(0), wantMix)
	}
	wantPool := assets.AssetMix{Renewables: 1}
	if g.TakeoverPoolMix() != wantPool {
		t.Errorf("pool = %+v, want %+v", g.TakeoverPoolMix(), wantPool)
	}
}

func TestApplyPlayerAction_PledgeBattery_MovesArbitrageToCapacity(t *testing.T) {
	// arrange
	_, g := mustNewGame(t, 2, params.Default)
	g.Players[0].Mix = assets.AssetMix{FossilsWholesale: 5, BatteriesArbitrage: 3}

	// act
	if err := g.ApplyPlayerAction(0, ActionPledgeBattery); err != nil {
		t.Fatal(err)
	}

	// assert
	want := assets.AssetMix{FossilsWholesale: 5, BatteriesArbitrage: 2, BatteriesCapacity: 1}
	if g.PlayerAssetMix(0) != want {
		t.Errorf("mix = %+v, want %+v", g.PlayerAssetMix(0), want)
	}
}

func TestApplyPlayerAction_PledgeFossil_MovesWholesaleToCapacity(t *testing.T) {
	// arrange
	_, g := mustNewGame(t, 2, params.Default)
	g.Players[0].Mix = assets.AssetMix{FossilsWholesale: 5}

	// act
	if err := g.ApplyPlayerAction(0, ActionPledgeFossil); err != nil {
		t.Fatal(err)
	}

	// assert
	want := assets.AssetMix{FossilsWholesale: 4, FossilsCapacity: 1}
	if g.PlayerAssetMix(0) != want {
		t.Errorf("mix = %+v, want %+v", g.PlayerAssetMix(0), want)
	}
}
