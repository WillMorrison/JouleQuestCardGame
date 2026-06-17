package game_test

import (
	"math/rand/v2"
	"testing"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/compact/game"
	cparams "github.com/WillMorrison/JouleQuestCardGame/compact/params"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/engine"
	"github.com/WillMorrison/JouleQuestCardGame/eventlog"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

type actionStep struct {
	playerIndex int
	actionCode  int32
}

func actionCodeToLegacy(pi int, actionCode int32, p params.Params) engine.PlayerAction {
	switch actionCode {
	case game.ActionBuildRenewable:
		return engine.PlayerAction{Type: engine.ActionTypeBuildAsset, PlayerIndex: pi, AssetType: assets.TypeRenewable, Cost: p.BuildCost(assets.TypeRenewable)}
	case game.ActionBuildBattery:
		return engine.PlayerAction{Type: engine.ActionTypeBuildAsset, PlayerIndex: pi, AssetType: assets.TypeBattery, Cost: p.BuildCost(assets.TypeBattery)}
	case game.ActionBuildFossil:
		return engine.PlayerAction{Type: engine.ActionTypeBuildAsset, PlayerIndex: pi, AssetType: assets.TypeFossil, Cost: p.BuildCost(assets.TypeFossil)}
	case game.ActionScrapRenewable:
		return engine.PlayerAction{Type: engine.ActionTypeScrapAsset, PlayerIndex: pi, AssetType: assets.TypeRenewable, Cost: p.ScrapCost(assets.TypeRenewable)}
	case game.ActionScrapBattery:
		return engine.PlayerAction{Type: engine.ActionTypeScrapAsset, PlayerIndex: pi, AssetType: assets.TypeBattery, Cost: p.ScrapCost(assets.TypeBattery)}
	case game.ActionScrapFossil:
		return engine.PlayerAction{Type: engine.ActionTypeScrapAsset, PlayerIndex: pi, AssetType: assets.TypeFossil, Cost: p.ScrapCost(assets.TypeFossil)}
	case game.ActionTakeoverRenewable:
		return engine.PlayerAction{Type: engine.ActionTypeTakeoverAsset, PlayerIndex: pi, AssetType: assets.TypeRenewable, Cost: p.TakeoverCost(assets.TypeRenewable)}
	case game.ActionTakeoverBattery:
		return engine.PlayerAction{Type: engine.ActionTypeTakeoverAsset, PlayerIndex: pi, AssetType: assets.TypeBattery, Cost: p.TakeoverCost(assets.TypeBattery)}
	case game.ActionTakeoverFossil:
		return engine.PlayerAction{Type: engine.ActionTypeTakeoverAsset, PlayerIndex: pi, AssetType: assets.TypeFossil, Cost: p.TakeoverCost(assets.TypeFossil)}
	case game.ActionTakeoverScrapRenewable:
		return engine.PlayerAction{Type: engine.ActionTypeTakeoverScrapAsset, PlayerIndex: pi, AssetType: assets.TypeRenewable, Cost: p.TakeoverCost(assets.TypeRenewable)}
	case game.ActionTakeoverScrapBattery:
		return engine.PlayerAction{Type: engine.ActionTypeTakeoverScrapAsset, PlayerIndex: pi, AssetType: assets.TypeBattery, Cost: p.TakeoverCost(assets.TypeBattery)}
	case game.ActionTakeoverScrapFossil:
		return engine.PlayerAction{Type: engine.ActionTypeTakeoverScrapAsset, PlayerIndex: pi, AssetType: assets.TypeFossil, Cost: p.TakeoverCost(assets.TypeFossil)}
	case game.ActionPledgeBattery:
		return engine.PlayerAction{Type: engine.ActionTypePledgeCapacity, PlayerIndex: pi, AssetType: assets.TypeBattery, Cost: 0}
	case game.ActionPledgeFossil:
		return engine.PlayerAction{Type: engine.ActionTypePledgeCapacity, PlayerIndex: pi, AssetType: assets.TypeFossil, Cost: 0}
	case game.ActionFinished:
		return engine.PlayerAction{Type: engine.ActionTypeFinished, PlayerIndex: pi, Cost: 0}
	default:
		panic("invalid action code")
	}
}

func checkParity(t *testing.T, step int, pgs *engine.ProceduralGameState, cg *game.Game) {
	t.Helper()
	legacyGame := pgs.Game()

	if legacyGame.Status != cg.Status {
		t.Errorf("step %d: Status mismatch: legacy=%v, compact=%v", step, legacyGame.Status, cg.Status)
	}
	if legacyGame.Reason != cg.Reason {
		t.Errorf("step %d: Reason mismatch: legacy=%v, compact=%v", step, legacyGame.Reason, cg.Reason)
	}
	if int32(legacyGame.Round) != cg.Round {
		t.Errorf("step %d: Round mismatch: legacy=%v, compact=%v", step, legacyGame.Round, cg.Round)
	}
	if int32(legacyGame.CarbonEmissions) != cg.CarbonEmissions {
		t.Errorf("step %d: Emissions mismatch: legacy=%v, compact=%v", step, legacyGame.CarbonEmissions, cg.CarbonEmissions)
	}

	if legacyGame.LastSnapshot.AssetMix != cg.LastSnapshot.AssetMix {
		t.Errorf("step %d: LastSnapshot.AssetMix mismatch: legacy=%+v, compact=%+v", step, legacyGame.LastSnapshot.AssetMix, cg.LastSnapshot.AssetMix)
	}
	if legacyGame.LastSnapshot.PriceVolatility != cg.LastSnapshot.PriceVolatility {
		t.Errorf("step %d: LastSnapshot.PriceVolatility mismatch: legacy=%v, compact=%v", step, legacyGame.LastSnapshot.PriceVolatility, cg.LastSnapshot.PriceVolatility)
	}
	if legacyGame.LastSnapshot.GridStability != cg.LastSnapshot.GridStability {
		t.Errorf("step %d: LastSnapshot.GridStability mismatch: legacy=%v, compact=%v", step, legacyGame.LastSnapshot.GridStability, cg.LastSnapshot.GridStability)
	}

	for i := int32(0); i < cg.NumPlayers; i++ {
		pStatus := cg.PlayerStatus(i)
		pMoney := cg.PlayerMoney(i)
		pMix := cg.PlayerAssetMix(i)

		var legacyStatus core.PlayerStatus
		var legacyMoney int
		var legacyMix assets.AssetMix

		if int(i) < len(legacyGame.Players) {
			legacyStatus = legacyGame.Players[i].Status
			legacyMoney = legacyGame.Players[i].Money
			legacyMix = legacyGame.Players[i].Assets
		} else {
			legacyStatus = core.PlayerStatusLost
		}

		if legacyStatus != pStatus {
			t.Errorf("step %d: player %d Status mismatch: legacy=%v, compact=%v", step, i, legacyStatus, pStatus)
		}
		if legacyStatus == core.PlayerStatusActive {
			if int32(legacyMoney) != pMoney {
				t.Errorf("step %d: player %d Money mismatch: legacy=%v, compact=%v", step, i, legacyMoney, pMoney)
			}
			if legacyMix != pMix {
				t.Errorf("step %d: player %d Mix mismatch: legacy=%+v, compact=%+v", step, i, legacyMix, pMix)
			}
		}

		// Check possible actions mask
		legacyMask := uint32(0)
		for _, la := range pgs.PossibleActions() {
			if la.PlayerIndex == int(i) {
				for code := int32(0); code <= game.ActionFinished; code++ {
					if actionCodeToLegacy(int(i), code, legacyGame.Params) == la {
						legacyMask |= (1 << code)
					}
				}
			}
		}
		compactMask := cg.PossibleActionMask(int32(i))
		if legacyMask != compactMask {
			t.Errorf("step %d: player %d PossibleActionMask mismatch: legacy=%b, compact=%b", step, i, legacyMask, compactMask)
		}
	}

	// Takeover pool mix
	if legacyGame.TakeoverPool != cg.TakeoverPool {
		t.Errorf("step %d: TakeoverPoolMix mismatch: legacy=%+v, compact=%+v", step, legacyGame.TakeoverPool, cg.TakeoverPool)
	}
}

func runParityScenario(t *testing.T, numPlayers int, legacyParams params.Params, seed uint64, steps []actionStep) {
	t.Helper()
	compactParams, err := cparams.FromLegacy(legacyParams)
	if err != nil {
		t.Fatal(err)
	}

	logger := eventlog.NullLogger{}
	pgs, err := engine.NewProceduralGame(numPlayers, legacyParams, logger)
	if err != nil {
		t.Fatal(err)
	}

	cg, err := game.NewGame(int32(numPlayers), compactParams)
	if err != nil {
		t.Fatal(err)
	}

	checkParity(t, -1, pgs, cg) // initial state

	for i, step := range steps {
		// Set seed before every apply just in case an operate phase triggers
		pgs.SetRNGSeed(seed)
		cg.SetRNGSeed(seed)

		legacyAction := actionCodeToLegacy(step.playerIndex, step.actionCode, legacyParams)
		pgs.ApplyPlayerAction(legacyAction)

		err := cg.ApplyPlayerAction(int32(step.playerIndex), step.actionCode)
		if err != game.CodeOK {
			t.Fatalf("step %d: %v", i, err.Error())
		}

		checkParity(t, i, pgs, cg)
	}
}

func TestParity_Minimal(t *testing.T) {
	runParityScenario(t, 2, params.Default, 42, []actionStep{
		{0, game.ActionFinished},
		{1, game.ActionFinished},
	})
}

func TestParity_BuildScrapPledge(t *testing.T) {
	b := params.BuilderFrom(params.Default)
	b.TakeoverRule(params.TakeoverRuleVirtualOwner)
	b.InitialCash(100)

	runParityScenario(t, 2, b.Build(), 100, []actionStep{
		{0, game.ActionBuildBattery},
		{0, game.ActionPledgeBattery},
		{1, game.ActionScrapFossil}, // sells their fossil
		{0, game.ActionFinished},
		{1, game.ActionFinished},
	})
}

func TestParity_TakeoverForcedRule(t *testing.T) {
	b := params.BuilderFrom(params.Default)
	b.TakeoverRule(params.TakeoverRuleForcedTakeover)
	b.InitialCash(0) // forced bankruptcy

	runParityScenario(t, 2, b.Build(), 42, []actionStep{
		{0, game.ActionFinished},
		{1, game.ActionFinished},
		// After operate, players go bankrupt -> pool gets assets. Then loss triggers if nobody can afford them.
	})
}

func TestParity_OperateOutcomes(t *testing.T) {
	b := params.BuilderFrom(params.Default)
	b.EmissionsCap(5)
	b.CarbonTax(params.CarbonTaxRuleApplyCarbonTax, 2, 50)
	b.WinConditionRule(params.WinConditionRuleLastFossilLoses, 0)
	b.InitialCash(50)

	runParityScenario(t, 2, b.Build(), 123, []actionStep{
		{0, game.ActionFinished},
		{1, game.ActionFinished}, // should trigger operate and emissions cap or bankruptcy
	})
}

func TestParity_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	b := params.BuilderFrom(params.Default)
	b.Capacity(params.CapacityRuleNoCapacityMarket, core.PnLTable{}, core.PnLTable{}, core.PnLTable{})
	legacyParams := b.Build()
	compactParams, _ := cparams.FromLegacy(legacyParams)

	logger := eventlog.NullLogger{}
	pgs, err := engine.NewProceduralGame(3, legacyParams, logger)
	if err != nil {
		t.Fatal(err)
	}

	cg, err := game.NewGame(3, compactParams)
	if err != nil {
		t.Fatal(err)
	}

	rng := rand.New(rand.NewPCG(42, 0))
	var seed uint64 = 42

	for step := range 100 {
		if cg.Status != core.GameStatusOngoing {
			break
		}

		pgs.SetRNGSeed(seed + uint64(step))
		cg.SetRNGSeed(seed + uint64(step))

		// Find a random valid action across all players
		var valid []actionStep
		for pi := 0; pi < 3; pi++ {
			mask := cg.PossibleActionMask(int32(pi))
			if mask == 0 {
				continue
			}
			for code := int32(0); code <= game.ActionFinished; code++ {
				if mask&(1<<code) != 0 {
					valid = append(valid, actionStep{pi, code})
				}
			}
		}

		if len(valid) == 0 {
			t.Fatalf("No valid actions but game is ongoing at step %d", step)
		}

		chosen := valid[rng.IntN(len(valid))]

		legacyAction := actionCodeToLegacy(chosen.playerIndex, chosen.actionCode, legacyParams)
		pgs.ApplyPlayerAction(legacyAction)
		err := cg.ApplyPlayerAction(int32(chosen.playerIndex), chosen.actionCode)
		if err != game.CodeOK {
			t.Fatalf("step %d: %v", step, err.Error())
		}

		checkParity(t, step, pgs, cg)
	}
}
