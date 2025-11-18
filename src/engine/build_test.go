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

func Test_GameState_possibleActions_CannotFinishIfTakeoverPoolHasAssets(t *testing.T) {
	gameState := GameState{
		Players: []PlayerState{
			{
				Status:     PlayerStatusActive,
				isBuilding: true,
				Money:      0,
			},
		},
		Params:       params.Default,
		TakeoverPool: makeAssets(assets.AssetMix{BatteriesArbitrage: 1}),
	}
	got := gameState.possibleActions()
	if slices.ContainsFunc(got, func(pa PlayerAction) bool { return pa.Type == ActionTypeFinished }) {
		t.Errorf("Player should not be able to finish the build round if there are assets in the takeover pool, got %+v", got)
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
