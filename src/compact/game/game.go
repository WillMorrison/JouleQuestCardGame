package game

import (
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
	Round           int32
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
	var g Game
	err := g.Reset(numPlayers, p)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// Reset resets the game to its initial state. The RNG is not reset
func (g *Game) Reset(numPlayers int, p cparams.CompactParams) error {
	if numPlayers < 2 || numPlayers > cparams.MaxPlayers {
		return ErrInvalidPlayerCount
	}
	startingFossils := int(p.StartingFossils(numPlayers))
	if startingFossils <= 0 {
		return ErrNoStartingFossils
	}

	g.Params = p

	g.phase = phaseGameStart
	g.Status = core.GameStatusOngoing
	g.Reason = core.LossConditionNone
	g.Round = 0
	g.CarbonEmissions = 0
	g.TakeoverPool = assets.AssetMix{}

	g.NumPlayers = numPlayers
	for i := range g.Players {
		if i < numPlayers {
			g.Players[i].Money = p.InitialCash
			g.Players[i].Status = core.PlayerStatusActive
			g.Players[i].Reason = core.LossConditionNone
			g.Players[i].IsBuilding = true
			g.Players[i].Mix = assets.AssetMix{FossilsWholesale: startingFossils}
		} else {
			// Reset unused players to default values
			g.Players[i].Money = 0
			g.Players[i].Status = core.PlayerStatusLost
			g.Players[i].Reason = core.LossConditionNone
			g.Players[i].IsBuilding = false
			g.Players[i].Mix = assets.AssetMix{}
		}
	}
	g.LastSnapshot = snapshotFromGlobalMix(g.globalAssetMix())
	g.startBuildPhase()
	return nil
}

func (g *Game) startBuildPhase() {
	g.phase = phaseBuild
	for i := 0; i < g.NumPlayers; i++ {
		if g.Players[i].Status == core.PlayerStatusActive {
			g.Players[i].IsBuilding = true
			g.Players[i].Mix.ResetAllCapacityPledges()
		}
	}
	g.Round++
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

// --- index-checked accessors ---

func (g *Game) PlayerMoney(pi int) int32 {
	if pi < 0 || pi >= g.NumPlayers {
		return 0
	}
	return g.Players[pi].Money
}

func (g *Game) PlayerStatus(pi int) core.PlayerStatus {
	if pi < 0 || pi >= g.NumPlayers {
		return core.PlayerStatusLost
	}
	return g.Players[pi].Status
}

func (g *Game) PlayerLossReason(pi int) core.LossCondition {
	if pi < 0 || pi >= g.NumPlayers {
		return core.LossConditionNone
	}
	return g.Players[pi].Reason
}

func (g *Game) PlayerAssetMix(pi int) assets.AssetMix {
	if pi < 0 || pi >= g.NumPlayers {
		return assets.AssetMix{}
	}
	return g.Players[pi].Mix
}
