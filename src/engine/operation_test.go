package engine

import (
	"testing"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/eventlog"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

func TestGameState_generationConstraintMet(t *testing.T) {
	tests := []struct {
		name     string // description of this test case
		game     GameState
		assetMix assets.AssetMix
		want     bool
	}{
		{
			name: "minimum not met",
			want: false,
			game: GameState{
				Params: params.BuilderFrom(params.Default).GenerationConstraint(params.GenerationConstraintRuleMinimum, 15).Build(),
			},
			assetMix: assets.AssetMix{BatteriesArbitrage: 50, Renewables: 10, FossilsWholesale: 4}, // 14 generating assets
		},
		{
			name: "minimum met",
			want: true,
			game: GameState{
				Params: params.BuilderFrom(params.Default).GenerationConstraint(params.GenerationConstraintRuleMinimum, 15).Build(),
			},
			assetMix: assets.AssetMix{BatteriesArbitrage: 50, Renewables: 10, FossilsWholesale: 4, FossilsCapacity: 1}, // 15 generating assets
		},
		{
			name: "max decrease not met",
			want: false,
			game: GameState{
				Params:       params.BuilderFrom(params.Default).GenerationConstraint(params.GenerationConstraintRuleMaxDecrease, 15).Build(),
				LastSnapshot: Snapshot{AssetMix: assets.AssetMix{FossilsCapacity: 30}}, // 30 generating assets
			},
			assetMix: assets.AssetMix{BatteriesArbitrage: 50, Renewables: 10, FossilsWholesale: 4}, // 14 generating assets (16 fewer)
		},
		{
			name: "max decrease met",
			want: true,
			game: GameState{
				Params:       params.BuilderFrom(params.Default).GenerationConstraint(params.GenerationConstraintRuleMaxDecrease, 15).Build(),
				LastSnapshot: Snapshot{AssetMix: assets.AssetMix{FossilsCapacity: 30}}, // 30 generating assets
			},
			assetMix: assets.AssetMix{BatteriesArbitrage: 50, Renewables: 10, FossilsWholesale: 4, FossilsCapacity: 1}, // 15 generating assets (15 fewer)
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.game.generationConstraintMet(tt.assetMix)
			if got != tt.want {
				t.Errorf("generationConstraintMet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGameState_winConditionMet(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		game GameState
		want bool
	}{
		{
			name: "renewable penetration not met",
			want: false,
			game: GameState{
				Params:       params.BuilderFrom(params.Default).WinConditionRule(params.WinConditionRuleRenewablePenetrationThreshold, 91).Build(),
				LastSnapshot: Snapshot{AssetMix: assets.AssetMix{FossilsCapacity: 1, Renewables: 9}}, // 90% penetration
			},
		},
		{
			name: "renewable penetration met",
			want: true,
			game: GameState{
				Params:       params.BuilderFrom(params.Default).WinConditionRule(params.WinConditionRuleRenewablePenetrationThreshold, 90).Build(),
				LastSnapshot: Snapshot{AssetMix: assets.AssetMix{FossilsCapacity: 1, Renewables: 9}}, // 90% penetration
			},
		},
		{
			name: "last fossil not met, multiple fossil holders",
			want: false,
			game: GameState{
				Params: params.BuilderFrom(params.Default).WinConditionRule(params.WinConditionRuleLastFossilLoses, 0).Build(),
				Players: []PlayerState{
					{Status: PlayerStatusActive, Assets: []assets.Asset{assets.New(assets.TypeFossil)}},
					{Status: PlayerStatusActive, Assets: []assets.Asset{assets.New(assets.TypeFossil)}},
				},
			},
		},
		{
			name: "last fossil not met, one player with fossil assets, fossil takeover assets",
			want: true,
			game: GameState{
				Params: params.BuilderFrom(params.Default).WinConditionRule(params.WinConditionRuleLastFossilLoses, 0).Build(),
				Players: []PlayerState{
					{Status: PlayerStatusActive, Assets: []assets.Asset{assets.New(assets.TypeRenewable)}},
					{Status: PlayerStatusActive, Assets: []assets.Asset{assets.New(assets.TypeFossil)}},
				},
				TakeoverPool: []assets.Asset{assets.New(assets.TypeFossil)},
			},
		},
		{
			name: "last fossil met, no players with fossil assets, fossil takeover assets",
			want: true,
			game: GameState{
				Params: params.BuilderFrom(params.Default).WinConditionRule(params.WinConditionRuleLastFossilLoses, 0).Build(),
				Players: []PlayerState{
					{Status: PlayerStatusActive, Assets: []assets.Asset{assets.New(assets.TypeRenewable)}},
					{Status: PlayerStatusActive, Assets: []assets.Asset{assets.New(assets.TypeRenewable)}},
				},
				TakeoverPool: []assets.Asset{assets.New(assets.TypeFossil)},
			},
		},
		{
			name: "last fossil met, one player with fossil assets, no takeover assets",
			want: true,
			game: GameState{
				Params: params.BuilderFrom(params.Default).WinConditionRule(params.WinConditionRuleLastFossilLoses, 0).Build(),
				Players: []PlayerState{
					{Status: PlayerStatusActive, Assets: []assets.Asset{assets.New(assets.TypeFossil)}},
					{Status: PlayerStatusActive, Assets: []assets.Asset{assets.New(assets.TypeRenewable)}},
				},
			},
		},
		{
			name: "last fossil met, no players with fossil assets, no takeover assets",
			want: true,
			game: GameState{
				Params: params.BuilderFrom(params.Default).WinConditionRule(params.WinConditionRuleLastFossilLoses, 0).Build(),
				Players: []PlayerState{
					{Status: PlayerStatusActive, Assets: []assets.Asset{assets.New(assets.TypeBattery)}},
					{Status: PlayerStatusActive, Assets: []assets.Asset{assets.New(assets.TypeRenewable)}},
				},
				TakeoverPool: nil,
			},
		},
		{
			name: "last fossil met, no players with fossil assets, no fossil takeover assets",
			want: true,
			game: GameState{
				Params: params.BuilderFrom(params.Default).WinConditionRule(params.WinConditionRuleLastFossilLoses, 0).Build(),
				Players: []PlayerState{
					{Status: PlayerStatusActive, Assets: []assets.Asset{assets.New(assets.TypeRenewable)}},
					{Status: PlayerStatusActive, Assets: []assets.Asset{assets.New(assets.TypeRenewable)}},
				},
				TakeoverPool: []assets.Asset{assets.New(assets.TypeBattery)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.game.winConditionMet()
			if got != tt.want {
				t.Errorf("winConditionMet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGameState_OperatePhase(t *testing.T) {
	tests := []struct {
		name       string // description of this test case
		game       GameState
		wantStatus GameStatus
		wantReason LossCondition
	}{
		{
			name: "climate change loss",
			game: GameState{
				Params:          params.BuilderFrom(params.Default).EmissionsCap(100).Build(),
				CarbonEmissions: 99,
				Players: []PlayerState{
					{Status: PlayerStatusActive, Assets: makeAssets(assets.AssetMix{FossilsWholesale: 20})},
					{Status: PlayerStatusActive, Assets: makeAssets(assets.AssetMix{FossilsWholesale: 20})},
				},
			},
			wantStatus: GameStatusLoss,
			wantReason: LossConditionCarbonEmissionsExceeded,
		},
		{
			name: "minimum generation loss",
			game: GameState{
				Params: params.BuilderFrom(params.Default).GenerationConstraint(params.GenerationConstraintRuleMinimum, 15).Build(),
				Players: []PlayerState{
					{Status: PlayerStatusActive, Assets: makeAssets(assets.AssetMix{Renewables: 5, FossilsCapacity: 2})},
					{Status: PlayerStatusActive, Assets: makeAssets(assets.AssetMix{FossilsWholesale: 5, BatteriesArbitrage: 10})},
				},
			},
			wantStatus: GameStatusLoss,
			wantReason: LossConditionInsufficientGeneration,
		},
		{
			name: "bankrupt one player and continue",
			game: GameState{
				Params: params.BuilderFrom(params.Default).
					PnL(core.PnLTable{}, core.PnLTable{}, core.PnLTable{-50, -50, -50, -50}). // Renewables will lose money
					Build(),
				Players: []PlayerState{
					{Status: PlayerStatusActive, Money: 0, Assets: makeAssets(assets.AssetMix{Renewables: 5, FossilsWholesale: 2})},
					{Status: PlayerStatusActive, Money: 0, Assets: makeAssets(assets.AssetMix{FossilsWholesale: 5, BatteriesArbitrage: 10})},
					{Status: PlayerStatusActive, Money: 0, Assets: makeAssets(assets.AssetMix{FossilsWholesale: 8, BatteriesArbitrage: 10})},
				},
			},
			wantStatus: GameStatusOngoing,
		},
		{
			name: "bankrupt all players",
			game: GameState{
				Params: params.BuilderFrom(params.Default).
					PnL(core.PnLTable{}, core.PnLTable{}, core.PnLTable{-50, -50, -50, -50}). // Renewables will lose money
					Build(),
				Players: []PlayerState{
					{Status: PlayerStatusActive, Money: 0, Assets: makeAssets(assets.AssetMix{Renewables: 5, FossilsWholesale: 2})},
					{Status: PlayerStatusActive, Money: 0, Assets: makeAssets(assets.AssetMix{Renewables: 5, BatteriesArbitrage: 10})},
					{Status: PlayerStatusActive, Money: 0, Assets: makeAssets(assets.AssetMix{Renewables: 8, BatteriesArbitrage: 10})},
				},
			},
			wantStatus: GameStatusLoss,
			wantReason: LossConditionNoActivePlayers,
		},
		{
			name: "last fossil loses, everyone else wins",
			game: GameState{
				Params: params.Default,
				Players: []PlayerState{
					{Status: PlayerStatusActive, Money: 0, Assets: makeAssets(assets.AssetMix{Renewables: 5, BatteriesArbitrage: 2})},
					{Status: PlayerStatusActive, Money: 0, Assets: makeAssets(assets.AssetMix{FossilsWholesale: 10, BatteriesArbitrage: 10})},
				},
			},
			wantStatus: GameStatusWin,
		},
		{
			name: "bankrupt one player, one last fossil loses",
			game: GameState{
				Params: params.BuilderFrom(params.Default).
					PnL(core.PnLTable{}, core.PnLTable{}, core.PnLTable{-50, -50, -50, -50}). // Renewables will lose money
					Build(),
				Players: []PlayerState{
					{Status: PlayerStatusActive, Money: 0, Assets: makeAssets(assets.AssetMix{Renewables: 5, FossilsWholesale: 2})},
					{Status: PlayerStatusActive, Money: 0, Assets: makeAssets(assets.AssetMix{FossilsWholesale: 10, BatteriesArbitrage: 10})},
				},
			},
			wantStatus: GameStatusLoss,
			wantReason: LossConditionNoActivePlayers,
		},
		{
			name: "last fossil already went bankrupt, everyone else wins",
			game: GameState{
				Params: params.BuilderFrom(params.Default).
					PnL(core.PnLTable{}, core.PnLTable{-50, -50, -50, -50}, core.PnLTable{}). // Fossils will lose money
					Build(),
				Players: []PlayerState{
					{Status: PlayerStatusActive, Money: 0, Assets: makeAssets(assets.AssetMix{Renewables: 10, BatteriesArbitrage: 10})},
					{Status: PlayerStatusActive, Money: 0, Assets: makeAssets(assets.AssetMix{FossilsWholesale: 10, Renewables: 10})},
					{Status: PlayerStatusActive, Money: 0, Assets: makeAssets(assets.AssetMix{FossilsWholesale: 10, BatteriesArbitrage: 10})},
				},
			},
			wantStatus: GameStatusWin,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.game.Logger = eventlog.NewJsonLogger(t.Output())
			OperatePhase(&tt.game)
			if tt.game.Status != tt.wantStatus {
				t.Errorf("game.Status = %s, want %s", tt.game.Status, tt.wantStatus)
			}
			if tt.wantStatus == GameStatusLoss && tt.game.Reason != tt.wantReason {
				t.Errorf("game.Reason = %q, want %q", tt.game.Reason, tt.wantReason)
			}
		})
	}
}
