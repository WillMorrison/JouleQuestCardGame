package engine

import (
	"cmp"
	"slices"
	"testing"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

func makeAssets(am assets.AssetMix) (as []assets.Asset) {
	for range am.Renewables {
		as = append(as, assets.New(assets.TypeRenewable))
	}
	for range am.BatteriesArbitrage {
		as = append(as, assets.New(assets.TypeBattery))
	}
	for range am.BatteriesCapacity {
		a := assets.New(assets.TypeBattery)
		a.SetMode(assets.OperationModeCapacity)
		as = append(as, a)
	}
	for range am.FossilsWholesale {
		as = append(as, assets.New(assets.TypeFossil))
	}
	for range am.FossilsCapacity {
		a := assets.New(assets.TypeFossil)
		a.SetMode(assets.OperationModeCapacity)
		as = append(as, a)
	}
	return
}

func cmpPlayerAction(a, b PlayerAction) int {
	return cmp.Or(cmp.Compare(a.Type, b.Type),
		cmp.Compare(a.PlayerIndex, b.PlayerIndex),
		cmp.Compare(a.AssetType, b.AssetType),
		cmp.Compare(a.Cost, b.Cost))
}

func Test_GameState_possibleActions_NoActionsIfNotBuilding(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: false,
			},
		},
		Params:       params.Default,
		TakeoverPool: nil,
	}
	got := gameState.possibleActions()
	if len(got) != 0 {
		t.Errorf("Expected possible actions to be empty, got %+v", got)
	}
}

func Test_GameState_possibleActions_NoActionsIfLost(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusLost,
				isBuilding: true,
			},
		},
		Params:       params.Default,
		TakeoverPool: nil,
	}
	got := gameState.possibleActions()
	if len(got) != 0 {
		t.Errorf("Expected possible actions to be empty, got %+v", got)
	}
}

func Test_GameState_possibleActions_CanFinishIfTakeoverPoolEmpty(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				Money:      0,
				isBuilding: true,
			},
		},
		Params:       params.Default,
		TakeoverPool: nil,
	}
	got := gameState.possibleActions()
	want := PlayerAction{
		Type:        ActionTypeFinished,
		PlayerIndex: 0,
		Cost:        0,
	}
	if !slices.Contains(got, want) {
		t.Errorf("Player should be able to finish the build round for free if the takeover pool is empty, got %+v", got)
	}
}

func Test_GameState_possibleActions_CannotFinishIfTakeoverPoolHasAssetsWithForcedTakeover(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      0,
			},
		},
		Params:       params.BuilderFrom(params.Default).TakeoverRule(params.TakeoverRuleForcedTakeover).Build(),
		TakeoverPool: makeAssets(assets.AssetMix{BatteriesArbitrage: 1}),
	}
	got := gameState.possibleActions()
	if slices.ContainsFunc(got, func(pa PlayerAction) bool { return pa.Type == ActionTypeFinished }) {
		t.Errorf("Player should not be able to finish the build round if there are assets in the takeover pool, got %+v", got)
	}
}

func Test_GameState_possibleActions_CanFinishIfTakeoverPoolHasAssetsWithVirtualOwnerRule(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      0,
			},
		},
		Params:       params.BuilderFrom(params.Default).TakeoverRule(params.TakeoverRuleVirtualOwner).Build(),
		TakeoverPool: makeAssets(assets.AssetMix{BatteriesArbitrage: 1}),
	}
	got := gameState.possibleActions()
	want := PlayerAction{
		Type:        ActionTypeFinished,
		PlayerIndex: 0,
		Cost:        0,
	}
	if !slices.Contains(got, want) {
		t.Errorf("Player should be able to finish the build round for free if the takeover pool is not empty but using TakeoverRuleVirtualOwner, got %+v", got)
	}
}

func Test_GameState_possibleActions_CanPledgeAssetsForFree(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				Money:      0,
				Assets:     makeAssets(assets.AssetMix{BatteriesArbitrage: 1}),
				isBuilding: true,
			},
		},
		Params:       params.Default,
		TakeoverPool: nil,
	}
	got := gameState.possibleActions()
	want := PlayerAction{
		Type:        ActionTypePledgeCapacity,
		PlayerIndex: 0,
		AssetType:   assets.TypeBattery,
		Cost:        0,
	}
	if !slices.Contains(got, want) {
		t.Errorf("Player should be able to finish the build round for free if the takeover pool is empty, got %+v", got)
	}
}

func Test_GameState_possibleActions_CannotPledgeAssetsAgainstRules(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				Money:      0,
				Assets:     makeAssets(assets.AssetMix{BatteriesArbitrage: 1}),
				isBuilding: true,
			},
		},
		Params:       params.BuilderFrom(params.Default).Capacity(params.CapacityRuleNoCapacityMarket, core.PnLTable{}, core.PnLTable{}, core.PnLTable{}).Build(),
		TakeoverPool: nil,
	}
	got := gameState.possibleActions()
	if slices.ContainsFunc(got, func(pa PlayerAction) bool { return pa.Type == ActionTypePledgeCapacity }) {
		t.Errorf("Player should not be able to pledge capacity assets if there is no capacity market, got %+v", got)
	}
}

func Test_GameState_possibleActions_CannotPledgeRenewables(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				Money:      0,
				Assets:     makeAssets(assets.AssetMix{Renewables: 1}),
				isBuilding: true,
			},
		},
		Params:       params.Default,
		TakeoverPool: nil,
	}
	got := gameState.possibleActions()
	if slices.ContainsFunc(got, func(pa PlayerAction) bool { return pa.Type == ActionTypePledgeCapacity }) {
		t.Errorf("Player should not be able to pledge renewables as capacity assets, got %+v", got)
	}
}

func Test_GameState_possibleActions_CannotPledgeAlreadyPledgedAssets(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				Money:      0,
				Assets:     makeAssets(assets.AssetMix{BatteriesCapacity: 1, FossilsCapacity: 1}),
				isBuilding: true,
			},
		},
		Params:       params.Default,
		TakeoverPool: nil,
	}
	got := gameState.possibleActions()
	if slices.ContainsFunc(got, func(pa PlayerAction) bool { return pa.Type == ActionTypePledgeCapacity }) {
		t.Errorf("Player should not be able to pledge assets that are already pledgd to capacity, got %+v", got)
	}
}

func Test_GameState_possibleActions_CanBuildWithSufficientMoney(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      100,
				Assets:     nil,
			},
		},
		Params:       params.BuilderFrom(params.Default).RenewableCosts(50, 40).FossilCosts(50, 40).BatteryCosts(50, 40).Build(),
		TakeoverPool: nil,
	}
	got := gameState.possibleActions()
	for _, at := range assets.Types {
		want := PlayerAction{
			Type:        ActionTypeBuildAsset,
			PlayerIndex: 0,
			AssetType:   at,
			Cost:        50,
		}
		if !slices.Contains(got, want) {
			t.Errorf("Player should be able to buy %s assets, got %+v", want.AssetType.String(), got)
		}
	}
}

func Test_GameState_possibleActions_CanScrapWithSufficientMoney(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      100,
				Assets:     makeAssets(assets.AssetMix{Renewables: 1, FossilsWholesale: 1, BatteriesCapacity: 1}),
			},
		},
		Params:       params.BuilderFrom(params.Default).RenewableCosts(50, 40).FossilCosts(50, 40).BatteryCosts(50, 40).Build(),
		TakeoverPool: nil,
	}
	got := gameState.possibleActions()
	for _, at := range assets.Types {
		want := PlayerAction{
			Type:        ActionTypeScrapAsset,
			PlayerIndex: 0,
			AssetType:   at,
			Cost:        40,
		}
		if !slices.Contains(got, want) {
			t.Errorf("Player should be able to scrap %s assets, got %+v", want.AssetType.String(), got)
		}
	}
}

func Test_GameState_possibleActions_CanTakeoverWithSufficientMoney(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      100,
				Assets:     nil,
			},
		},
		Params:       params.BuilderFrom(params.Default).RenewableCosts(50, 40).FossilCosts(50, 40).BatteryCosts(50, 40).Build(),
		TakeoverPool: makeAssets(assets.AssetMix{Renewables: 1, FossilsWholesale: 1, BatteriesArbitrage: 1}),
	}
	got := gameState.possibleActions()
	for _, at := range assets.Types {
		want := PlayerAction{
			Type:        ActionTypeTakeoverAsset,
			PlayerIndex: 0,
			AssetType:   at,
			Cost:        40,
		}
		if !slices.Contains(got, want) {
			t.Errorf("Player should be able to takeover %s assets, got %+v", want.AssetType.String(), got)
		}
		want.Type = ActionTypeTakeoverScrapAsset
		if !slices.Contains(got, want) {
			t.Errorf("Player should be able to takeover and scrap %s assets, got %+v", want.AssetType.String(), got)
		}
	}
}

func Test_GameState_possibleActions_InsufficientMoney(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      10,
				Assets:     makeAssets(assets.AssetMix{Renewables: 1, FossilsWholesale: 1, FossilsCapacity: 1, BatteriesArbitrage: 1, BatteriesCapacity: 1}),
			},
		},
		Params:       params.BuilderFrom(params.Default).RenewableCosts(50, 40).FossilCosts(50, 40).BatteryCosts(50, 40).Build(),
		TakeoverPool: makeAssets(assets.AssetMix{Renewables: 1, FossilsWholesale: 1, BatteriesArbitrage: 1}),
	}
	got := gameState.possibleActions()
	if slices.ContainsFunc(got, func(pa PlayerAction) bool {
		return (pa.Type == ActionTypeBuildAsset ||
			pa.Type == ActionTypeScrapAsset ||
			pa.Type == ActionTypeTakeoverAsset ||
			pa.Type == ActionTypeTakeoverScrapAsset)
	}) {
		t.Errorf("Player should not be able to perform actions with insufficient money, got %+v", got)
	}
}

func Test_GameState_possibleActions_MultiplePlayers(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      0, // Can pledge to capacity
				Assets:     makeAssets(assets.AssetMix{Renewables: 1, FossilsWholesale: 1, FossilsCapacity: 1, BatteriesArbitrage: 1, BatteriesCapacity: 1}),
			},
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      100, // Can buy
				Assets:     nil,
			},
			{
				Status:     PlayerStatusActive,
				isBuilding: false, // Finished, no actions
				Money:      100,
				Assets:     makeAssets(assets.AssetMix{Renewables: 1, FossilsWholesale: 1}),
			},
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      40, // Can scrap
				Assets:     makeAssets(assets.AssetMix{Renewables: 1}),
			},
		},
		Params:       params.BuilderFrom(params.Default).RenewableCosts(50, 40).FossilCosts(50, 40).BatteryCosts(50, 40).Build(),
		TakeoverPool: nil,
	}
	got := gameState.possibleActions()
	want := []PlayerAction{
		{
			Type:        ActionTypePledgeCapacity,
			PlayerIndex: 0,
			AssetType:   assets.TypeBattery,
			Cost:        0,
		},
		{
			Type:        ActionTypePledgeCapacity,
			PlayerIndex: 0,
			AssetType:   assets.TypeFossil,
			Cost:        0,
		},
		{
			Type:        ActionTypeBuildAsset,
			PlayerIndex: 1,
			AssetType:   assets.TypeBattery,
			Cost:        50,
		},
		{
			Type:        ActionTypeBuildAsset,
			PlayerIndex: 1,
			AssetType:   assets.TypeRenewable,
			Cost:        50,
		},
		{
			Type:        ActionTypeBuildAsset,
			PlayerIndex: 1,
			AssetType:   assets.TypeFossil,
			Cost:        50,
		},
		{
			Type:        ActionTypeScrapAsset,
			PlayerIndex: 3,
			AssetType:   assets.TypeRenewable,
			Cost:        40,
		},
	}
	slices.SortFunc(got, cmpPlayerAction)
	slices.SortFunc(want, cmpPlayerAction)
	if slices.Equal(got, want) {
		t.Errorf("possibleActions() = %+v, want %+v", got, want)
	}
}

func Test_GameState_applyPlayerAction_Impossible(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      10,
				Assets:     nil,
			},
		},
		Params:       params.BuilderFrom(params.Default).RenewableCosts(50, 40).FossilCosts(50, 40).BatteryCosts(50, 40).Build(),
		TakeoverPool: nil,
	}
	err := gameState.applyPlayerAction(PlayerAction{
		Type:        ActionTypeBuildAsset,
		PlayerIndex: 0,
		AssetType:   assets.TypeBattery,
		Cost:        10,
	})
	if err == nil {
		t.Error("Expected error for impossible action, got nil")
	}
}

func Test_GameState_applyPlayerAction_Finished(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      10,
				Assets:     nil,
			},
		},
		Params:       params.Default,
		TakeoverPool: nil,
	}
	action := PlayerAction{
		Type:        ActionTypeFinished,
		PlayerIndex: 0,
	}

	err := gameState.applyPlayerAction(action)
	if err != nil {
		t.Fatalf("%+v.applyPlayerAction(%+v) = %s, want no error", gameState, action, err)
	}

	if gameState.Players[0].isBuilding {
		t.Errorf("Expected player isBuilding state to be false afer %s action", action.Type.String())
	}
}

func Test_GameState_applyPlayerAction_Build(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      60,
				Assets:     nil,
			},
		},
		Params:       params.BuilderFrom(params.Default).BatteryCosts(50, 40).Build(),
		TakeoverPool: nil,
	}
	action := PlayerAction{
		Type:        ActionTypeBuildAsset,
		PlayerIndex: 0,
		AssetType:   assets.TypeBattery,
		Cost:        50,
	}

	err := gameState.applyPlayerAction(action)
	if err != nil {
		t.Fatalf("%+v.applyPlayerAction(%+v) = %s, want no error", gameState, action, err)
	}

	gotMoney := gameState.Players[0].Money
	wantMoney := 10
	if gotMoney != wantMoney {
		t.Errorf("Money = %d, want %d after %s action", gotMoney, wantMoney, action.Type.String())
	}
	gotMix := gameState.Players[0].getAssetMix()
	wantMix := assets.AssetMix{BatteriesArbitrage: 1}
	if gotMix != wantMix {
		t.Errorf("AssetMix = %+v, want %+v after %s action", gotMix, wantMix, action.Type.String())
	}
}

func Test_GameState_applyPlayerAction_Scrap(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      60,
				Assets:     makeAssets(assets.AssetMix{BatteriesCapacity: 1, FossilsWholesale: 1}),
			},
		},
		Params:       params.BuilderFrom(params.Default).BatteryCosts(50, 40).Build(),
		TakeoverPool: nil,
	}
	action := PlayerAction{
		Type:        ActionTypeScrapAsset,
		PlayerIndex: 0,
		AssetType:   assets.TypeBattery,
		Cost:        40,
	}

	err := gameState.applyPlayerAction(action)
	if err != nil {
		t.Fatalf("%+v.applyPlayerAction(%+v) = %s, want no error", gameState, action, err)
	}

	gotMoney := gameState.Players[0].Money
	wantMoney := 20
	if gotMoney != wantMoney {
		t.Errorf("Money = %d, want %d after %s action", gotMoney, wantMoney, action.Type.String())
	}
	gotMix := gameState.Players[0].getAssetMix()
	wantMix := assets.AssetMix{FossilsWholesale: 1}
	if gotMix != wantMix {
		t.Errorf("AssetMix = %+v, want %+v after %s action", gotMix, wantMix, action.Type.String())
	}
}

func Test_GameState_applyPlayerAction_Takeover(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      60,
				Assets:     nil,
			},
		},
		Params:       params.BuilderFrom(params.Default).BatteryCosts(50, 40).Build(),
		TakeoverPool: makeAssets(assets.AssetMix{BatteriesCapacity: 1, FossilsWholesale: 1}),
	}
	action := PlayerAction{
		Type:        ActionTypeTakeoverAsset,
		PlayerIndex: 0,
		AssetType:   assets.TypeBattery,
		Cost:        40,
	}

	err := gameState.applyPlayerAction(action)
	if err != nil {
		t.Fatalf("%+v.applyPlayerAction(%+v) = %s, want no error", gameState, action, err)
	}

	gotMoney := gameState.Players[0].Money
	wantMoney := 20
	if gotMoney != wantMoney {
		t.Errorf("Money = %d, want %d after %s action", gotMoney, wantMoney, action.Type.String())
	}
	gotMix := gameState.Players[0].getAssetMix()
	wantMix := assets.AssetMix{BatteriesArbitrage: 1}
	if gotMix != wantMix {
		t.Errorf("Player AssetMix = %+v, want %+v after %s action", gotMix, wantMix, action.Type.String())
	}
	gotMix = assets.AssetMixFrom(slices.Values(gameState.TakeoverPool))
	wantMix = assets.AssetMix{FossilsWholesale: 1}
	if gotMix != wantMix {
		t.Errorf("Takeover AssetMix = %+v, want %+v after %s action", gotMix, wantMix, action.Type.String())
	}
}

func Test_GameState_applyPlayerAction_TakeoverScrap(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      60,
				Assets:     nil,
			},
		},
		Params:       params.BuilderFrom(params.Default).FossilCosts(50, 40).Build(),
		TakeoverPool: makeAssets(assets.AssetMix{BatteriesCapacity: 1, FossilsWholesale: 2}),
	}
	action := PlayerAction{
		Type:        ActionTypeTakeoverScrapAsset,
		PlayerIndex: 0,
		AssetType:   assets.TypeFossil,
		Cost:        40,
	}

	err := gameState.applyPlayerAction(action)
	if err != nil {
		t.Fatalf("%+v.applyPlayerAction(%+v) = %s, want no error", gameState, action, err)
	}

	gotMoney := gameState.Players[0].Money
	wantMoney := 20
	if gotMoney != wantMoney {
		t.Errorf("Money = %d, want %d after %s action", gotMoney, wantMoney, action.Type.String())
	}
	if len(gameState.Players[0].Assets) != 0 {
		t.Errorf("Player had %d assets, want 0 after %s action", len(gameState.Players[0].Assets), action.Type.String())
	}
	gotMix := assets.AssetMixFrom(slices.Values(gameState.TakeoverPool))
	wantMix := assets.AssetMix{BatteriesCapacity: 1, FossilsWholesale: 1}
	if gotMix != wantMix {
		t.Errorf("Takeover AssetMix = %+v, want %+v after %s action", gotMix, wantMix, action.Type.String())
	}
}

func Test_GameState_applyPlayerAction_Pledge(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      60,
				Assets:     makeAssets(assets.AssetMix{BatteriesCapacity: 1, BatteriesArbitrage: 2}),
			},
		},
		Params:       params.Default,
		TakeoverPool: nil,
	}
	action := PlayerAction{
		Type:        ActionTypePledgeCapacity,
		PlayerIndex: 0,
		AssetType:   assets.TypeBattery,
		Cost:        0,
	}

	err := gameState.applyPlayerAction(action)
	if err != nil {
		t.Fatalf("%+v.applyPlayerAction(%+v) = %s, want no error", gameState, action, err)
	}

	gotMoney := gameState.Players[0].Money
	wantMoney := 60
	if gotMoney != wantMoney {
		t.Errorf("Money = %d, want %d after %s action", gotMoney, wantMoney, action.Type.String())
	}
	gotMix := gameState.Players[0].getAssetMix()
	wantMix := assets.AssetMix{BatteriesArbitrage: 1, BatteriesCapacity: 2}
	if gotMix != wantMix {
		t.Errorf("AssetMix = %+v, want %+v after %s action", gotMix, wantMix, action.Type.String())
	}
}
