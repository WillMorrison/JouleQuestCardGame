package game

import (
	"fmt"
	randv2 "math/rand/v2"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	cparams "github.com/WillMorrison/JouleQuestCardGame/compact/params"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

// phase is internal to the compact procedural state machine (not exported to REST/WASM).
type phase int32

const (
	phaseGameStart phase = iota
	phaseBuild
	phaseOperate
	phaseGameEnd
)

// Game is the compact procedural controller (build / operate loop) with fixed-size state.
type Game struct {
	phase           phase
	Status          core.GameStatus
	Reason          core.LossCondition
	round           int32
	CarbonEmissions int32
	NumPlayers      int
	Players         [cparams.MaxPlayers]Player
	TakeoverPool    assets.AssetMix
	LastSnapshot    Snapshot
	Params          cparams.CompactParams
	// PCG RNG for operate-phase randomness
	pcg randv2.PCG
}

// NewGame constructs a game in the first build phase (same entry behavior as engine.NewProceduralGame).
func NewGame(numPlayers int, p cparams.CompactParams) (*Game, error) {
	if numPlayers < 2 || numPlayers > cparams.MaxPlayers {
		return nil, fmt.Errorf("%w: %d", ErrInvalidPlayerCount, numPlayers)
	}
	n := int(p.StartingFossils(numPlayers))
	if n <= 0 {
		return nil, ErrNoStartingFossils
	}
	var g Game
	g.Params = p
	g.NumPlayers = numPlayers
	g.Status = core.GameStatusOngoing
	g.phase = phaseGameStart
	for i := 0; i < numPlayers; i++ {
		g.Players[i].Money = p.InitialCash
		g.Players[i].Status = core.PlayerStatusActive
		g.Players[i].IsBuilding = true
		g.Players[i].Mix.FossilsWholesale = n
	}
	g.refreshLastSnapshot()
	g.startBuildPhase()
	return &g, nil
}

func (g *Game) refreshLastSnapshot() {
	g.LastSnapshot = snapshotFromGlobalMix(g.globalAssetMix())
}

func (g *Game) startBuildPhase() {
	g.phase = phaseBuild
	for i := 0; i < g.NumPlayers; i++ {
		if g.Players[i].Status == core.PlayerStatusActive {
			g.Players[i].IsBuilding = true
			g.Players[i].resetModesForBuild()
		}
	}
	g.round++
}

func (g *Game) haveBuildingPlayers() bool {
	for i := 0; i < g.NumPlayers; i++ {
		if g.Players[i].Status == core.PlayerStatusActive && g.Players[i].IsBuilding {
			return true
		}
	}
	return false
}

func (g *Game) anyPlayerHasPossibleActions() bool {
	for i := 0; i < g.NumPlayers; i++ {
		if g.possibleActionMask(i) != 0 {
			return true
		}
	}
	return false
}

func (g *Game) runUntilBuildPhase() {
	switch g.phase {
	case phaseGameStart:
		g.startBuildPhase()
	case phaseBuild:
		// stay in build until an action advances state
	case phaseOperate:
		g.runOperatePhase()
		if g.Status == core.GameStatusOngoing {
			g.startBuildPhase()
		} else {
			g.phase = phaseGameEnd
		}
	case phaseGameEnd:
	}
}

// ApplyPlayerAction applies an action for the given player index.
func (g *Game) ApplyPlayerAction(playerIndex int, actionCode int32) error {
	if g.phase != phaseBuild {
		return ErrNotBuildPhase
	}
	if playerIndex < 0 || playerIndex >= g.NumPlayers {
		return ErrInvalidAction
	}
	mask := g.possibleActionMask(playerIndex)
	if !actionCodeAllowed(mask, actionCode) {
		return ErrInvalidAction
	}
	if !g.applyActionCode(playerIndex, actionCode) {
		return ErrInvalidAction
	}

	if !g.anyPlayerHasPossibleActions() {
		if actionCode == ActionFinished {
			if !g.haveBuildingPlayers() {
				g.phase = phaseOperate
			}
		} else {
			if g.Params.TakeoverRule == params.TakeoverRuleForcedTakeover {
				g.Status = core.GameStatusLoss
				g.Reason = core.LossConditionUnownedTakeoverAssets
			} else {
				g.Status = core.GameStatusLoss
				g.Reason = core.LossConditionNoActivePlayers
			}
			g.phase = phaseGameEnd
		}
	}
	g.runUntilBuildPhase()
	return nil
}

// --- accessors (int32-friendly for future WASM) ---

func (g *Game) GameStatus() core.GameStatus    { return g.Status }
func (g *Game) LossReason() core.LossCondition { return g.Reason }
func (g *Game) Round() int32                   { return g.round }
func (g *Game) EmissionsCounter() int32        { return g.CarbonEmissions }
func (g *Game) PlayerCount() int32             { return int32(g.NumPlayers) }

func (g *Game) TakeoverPoolMix() assets.AssetMix { return g.TakeoverPool }

func (g *Game) LastRoundAssetMix() assets.AssetMix { return g.LastSnapshot.AssetMix }

func (g *Game) PlayerMoney(pi int) int32 {
	if pi < 0 || pi >= g.NumPlayers {
		return 0
	}
	return g.Players[pi].Money
}

func (g *Game) PlayerStatusI(pi int) core.PlayerStatus {
	if pi < 0 || pi >= g.NumPlayers {
		return core.PlayerStatusLost
	}
	return g.Players[pi].Status
}

func (g *Game) PlayerAssetMix(pi int) assets.AssetMix {
	if pi < 0 || pi >= g.NumPlayers {
		return assets.AssetMix{}
	}
	return g.Players[pi].Mix
}
