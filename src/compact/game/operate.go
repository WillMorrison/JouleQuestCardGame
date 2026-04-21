package game

import (
	"math/bits"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	legacy "github.com/WillMorrison/JouleQuestCardGame/params"
)

func (g *Game) generationConstraintMet(am assets.AssetMix) bool {
	switch g.Params.GenerationConstraintRule {
	case legacy.GenerationConstraintRuleMinimum:
		return am.GenerationAssets() >= int(g.Params.GenerationConstraint)
	case legacy.GenerationConstraintRuleMaxDecrease:
		prev := g.LastSnapshot.AssetMix.GenerationAssets()
		return (prev - am.GenerationAssets()) <= int(g.Params.GenerationConstraint)
	default:
		return false
	}
}

func (g *Game) winConditionMet() bool {
	switch g.Params.WinConditionRule {
	case legacy.WinConditionRuleRenewablePenetrationThreshold:
		return g.LastSnapshot.AssetMix.RenewablePenetration() >= int(g.Params.RenewablePenetration)
	case legacy.WinConditionRuleLastFossilLoses:
		var n int
		for i := 0; i < g.NumPlayers; i++ {
			if g.Players[i].Status == core.PlayerStatusActive && g.Players[i].hasFossilAssets() {
				n++
			}
		}
		return n <= 1
	default:
		return false
	}
}

func (g *Game) globalAssetMix() assets.AssetMix {
	var m assets.AssetMix
	for i := 0; i < g.NumPlayers; i++ {
		addMix(&m, g.Players[i].Mix)
	}
	addMix(&m, g.TakeoverPool)
	return m
}

func (g *Game) nextRisk() int32 {
	// EventRisk equivalent: uniform in {0,1,2} (see engine.OperatePhase).
	return int32(g.pcg.Uint64() % 3)
}

// runOperatePhase runs one operate round (mirrors engine.OperatePhase side effects).
func (g *Game) runOperatePhase() {
	risk := g.nextRisk()
	gridOutcome := snapshotFromGlobalMix(g.globalAssetMix())

	if !g.generationConstraintMet(gridOutcome.AssetMix) {
		g.Status = core.GameStatusLoss
		g.Reason = core.LossConditionInsufficientGeneration
		return
	}
	if int32(gridOutcome.GridStability) < risk {
		g.Status = core.GameStatusLoss
		g.Reason = core.LossConditionGridUnstable
		return
	}

	g.CarbonEmissions += int32(gridOutcome.AssetMix.Emissions())
	if g.CarbonEmissions > g.Params.EmissionsCap {
		g.Status = core.GameStatusLoss
		g.Reason = core.LossConditionCarbonEmissionsExceeded
		return
	}

	volIdx := int32(gridOutcome.PriceVolatility)
	worldCap := int32(gridOutcome.AssetMix.CapacityAssets())
	if worldCap < 1 {
		worldCap = 1
	}

	var numActive int
	for i := 0; i < g.NumPlayers; i++ {
		p := &g.Players[i]
		if p.Status != core.PlayerStatusActive {
			continue
		}
		numActive++
		pnl := g.Params.OperatePnLForPlayerMix(p.Mix, volIdx, g.CarbonEmissions, worldCap)
		p.Money += pnl
		if p.Money < 0 {
			p.setLoss(core.LossConditionPlayerBankrupt)
			moveMixTo(&g.TakeoverPool, &p.Mix)
			numActive--
		}
	}

	g.LastSnapshot = gridOutcome

	if numActive == 0 {
		g.Status = core.GameStatusLoss
		g.Reason = core.LossConditionNoActivePlayers
		return
	}

	if !g.winConditionMet() {
		return
	}

	if g.Params.WinConditionRule == legacy.WinConditionRuleLastFossilLoses {
		idx := g.firstPlayerIndexWithFossil()
		if idx >= 0 {
			g.Players[idx].setLoss(core.LossConditionLastPlayerWithFossilAssets)
			numActive--
			if numActive == 0 {
				g.Status = core.GameStatusLoss
				g.Reason = core.LossConditionNoActivePlayers
				return
			}
		}
	}

	g.Status = core.GameStatusWin
	g.Reason = core.LossConditionNone
}

func (g *Game) firstPlayerIndexWithFossil() int {
	for i := 0; i < g.NumPlayers; i++ {
		if g.Players[i].hasFossilAssets() {
			return i
		}
	}
	return -1
}

// SetRNGSeed seeds the operate-phase PCG RNG.
func (g *Game) SetRNGSeed(seed uint64) {
	// Two 64-bit words; second derived so a single-seed API still spreads state.
	g.pcg.Seed(seed, bits.ReverseBytes64(seed)^0xdeadbeefcafebabe)
}

// LastPriceVolatility exposes core enum for API parity.
func (g *Game) LastPriceVolatility() core.PriceVolatility { return g.LastSnapshot.PriceVolatility }

// LastGridStability exposes core enum for API parity.
func (g *Game) LastGridStability() core.GridStability { return g.LastSnapshot.GridStability }
