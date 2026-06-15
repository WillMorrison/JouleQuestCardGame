package game

import (
	"testing"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
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
	if g.Round != 1 {
		t.Fatalf("Round = %d, want 1", g.Round)
	}
	if g.NumPlayers != 4 {
		t.Fatalf("PlayerCount = %d", g.NumPlayers)
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
	if g.Round != 2 {
		t.Fatalf("Round = %d, want 2", g.Round)
	}
}

func TestReset(t *testing.T) {
	// arrange
	g := Game{
		phase:           phaseGameStart,
		Status:          core.GameStatusLoss,
		Reason:          core.LossConditionCarbonEmissionsExceeded,
		Round:           15,
		CarbonEmissions: 100,
		NumPlayers:      3,
		Players: [cparams.MaxPlayers]Player{
			{Money: 123, Status: core.PlayerStatusLost, Reason: core.LossConditionLastPlayerWithFossilAssets, IsBuilding: true, Mix: assets.AssetMix{FossilsWholesale: 8}},
			{Money: 456, Status: core.PlayerStatusLost, Reason: core.LossConditionCarbonEmissionsExceeded, IsBuilding: false, Mix: assets.AssetMix{Renewables: 9}},
			{Money: 789, Status: core.PlayerStatusLost, Reason: core.LossConditionCarbonEmissionsExceeded, IsBuilding: true, Mix: assets.AssetMix{BatteriesArbitrage: 10}},
		},
		TakeoverPool: assets.AssetMix{Renewables: 10, BatteriesCapacity: 3},
		LastSnapshot: Snapshot{
			AssetMix:        assets.AssetMix{Renewables: 19, BatteriesArbitrage: 10, BatteriesCapacity: 3, FossilsWholesale: 8},
			PriceVolatility: core.PriceVolatilityHigh,
			GridStability:   core.GridStabilityOk,
		},
		Params: cparams.CompactParams{
			CapacityRule:  params.CapacityRuleSharedCapacityPaymentPool,
			CarbonTaxRule: params.CarbonTaxRuleApplyCarbonTax,
			TakeoverRule:  params.TakeoverRuleVirtualOwner,
		},
	}

	// act
	p := params.BuilderFrom(params.Default).StartingAssets(map[int]int{2: 10}).InitialCash(100).Build()
	cp, err := cparams.FromLegacy(p)
	if err != nil {
		t.Fatal(err)
	}
	err = g.Reset(2, cp)
	if err != nil {
		t.Fatal(err)
	}

	// assert
	if g.phase != phaseBuild {
		t.Errorf("Phase = %d, want Build", g.phase)
	}
	if g.Status != core.GameStatusOngoing {
		t.Errorf("Status = %v, want ongoing", g.Status)
	}
	if g.Reason != core.LossConditionNone {
		t.Errorf("Reason = %v, want none", g.Reason)
	}
	if g.Round != 1 {
		t.Errorf("Round = %d, want 1", g.Round)
	}
	if g.CarbonEmissions != 0 {
		t.Errorf("CarbonEmissions = %d, want 0", g.CarbonEmissions)
	}
	if g.TakeoverPool != (assets.AssetMix{}) {
		t.Errorf("TakeoverPool = %+v, want empty", g.TakeoverPool)
	}
	if g.LastSnapshot.GridStability != core.GridStabilityGood {
		t.Errorf("LastSnapshot.GridStability = %v, want %s", g.LastSnapshot.GridStability, core.GridStabilityGood.String())
	}
	if g.LastSnapshot.PriceVolatility != core.PriceVolatilityLow {
		t.Errorf("LastSnapshot.PriceVolatility = %v, want %s", g.LastSnapshot.PriceVolatility, core.PriceVolatilityLow.String())
	}
	if g.LastSnapshot.AssetMix != (assets.AssetMix{FossilsWholesale: 20}) {
		t.Errorf("LastSnapshot.AssetMix = %+v, want %+v", g.LastSnapshot.AssetMix, assets.AssetMix{FossilsWholesale: 20})
	}
	if g.NumPlayers != 2 {
		t.Errorf("NumPlayers = %d, want 2", g.NumPlayers)
	}
	wantPlayers := [cparams.MaxPlayers]Player{
		{Money: cp.InitialCash, Status: core.PlayerStatusActive, Reason: core.LossConditionNone, Mix: assets.AssetMix{FossilsWholesale: 10}, IsBuilding: true},
		{Money: cp.InitialCash, Status: core.PlayerStatusActive, Reason: core.LossConditionNone, Mix: assets.AssetMix{FossilsWholesale: 10}, IsBuilding: true},
		{Money: 0, Status: core.PlayerStatusLost, Reason: core.LossConditionNone, Mix: assets.AssetMix{}, IsBuilding: false},
		{Money: 0, Status: core.PlayerStatusLost, Reason: core.LossConditionNone, Mix: assets.AssetMix{}, IsBuilding: false},
		{Money: 0, Status: core.PlayerStatusLost, Reason: core.LossConditionNone, Mix: assets.AssetMix{}, IsBuilding: false},
		{Money: 0, Status: core.PlayerStatusLost, Reason: core.LossConditionNone, Mix: assets.AssetMix{}, IsBuilding: false},
		{Money: 0, Status: core.PlayerStatusLost, Reason: core.LossConditionNone, Mix: assets.AssetMix{}, IsBuilding: false},
		{Money: 0, Status: core.PlayerStatusLost, Reason: core.LossConditionNone, Mix: assets.AssetMix{}, IsBuilding: false},
		{Money: 0, Status: core.PlayerStatusLost, Reason: core.LossConditionNone, Mix: assets.AssetMix{}, IsBuilding: false},
		{Money: 0, Status: core.PlayerStatusLost, Reason: core.LossConditionNone, Mix: assets.AssetMix{}, IsBuilding: false},
	}
	for i := 0; i < cparams.MaxPlayers; i++ {
		if g.Players[i] != wantPlayers[i] {
			t.Errorf("Players[%d] = %+v, want %+v", i, g.Players[i], wantPlayers[i])
		}
	}
}
