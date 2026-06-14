// This file contains definitions and helper methods for game state

package engine

import (
	"encoding/json"
	"fmt"
	"iter"
	randv2 "math/rand/v2"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/eventlog"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

type PlayerState struct {
	Status core.PlayerStatus  // Whether the player has lost
	Reason core.LossCondition // Reason for player loss, if applicable
	Money  int                // Player's current money
	Assets assets.AssetMix    // Player's owned assets

	isBuilding bool // Internal tracker of whether the player has finished the build round
}

func (ps PlayerState) getAssetMix() assets.AssetMix {
	return ps.Assets
}

type playerStateJSON struct {
	Status string
	Reason string `json:",omitempty"`
	Money  int
	Assets assets.AssetMix
}

func (ps PlayerState) MarshalJSON() ([]byte, error) {
	var psj = playerStateJSON{
		Status: ps.Status.String(),
		Money:  ps.Money,
		Assets: ps.Assets,
	}
	if ps.Status != core.PlayerStatusActive {
		psj.Reason = ps.Reason.String()
	}
	return json.Marshal(psj)
}

// Returns whether the player owns any fossil assets
func (ps PlayerState) HasFossilAssets() bool {
	return ps.Assets.AssetsOfType(assets.TypeFossil) > 0
}

// Resets all of the player's assets to their default operating mode
func (ps *PlayerState) resetAllAssets() {
	ps.Assets.ResetAllCapacityPledges()
}

// SetLossWithReason sets status and reason for loss. Caller is responsible for logging, handling asset takeover, etc.
func (ps *PlayerState) SetLossWithReason(reason core.LossCondition) {
	ps.Status = core.PlayerStatusLost
	ps.Reason = reason
}

// Snapshot holds summary statistics from the outcome of an Operate phase
type Snapshot struct {
	AssetMix        assets.AssetMix
	PriceVolatility core.PriceVolatility
	GridStability   core.GridStability
}

type GameState struct {
	Status          core.GameStatus
	Reason          core.LossCondition `json:",omitzero"` // Reason for global loss, if applicable
	Round           int
	CarbonEmissions int // Total carbon emissions in the world
	Players         []PlayerState
	TakeoverPool    assets.AssetMix // Assets available for takeover

	LastSnapshot Snapshot // Summary of the previous round's Operate phase

	Params          params.Params
	Logger          eventlog.Logger `json:"-"`
	GetPlayerAction GetPlayerAction // callback when the game needs to pick the next player action
	GameOverFunc    func()          // Callback function which is called when the game ends.

	// RNG for operate-phase randomness
	pcg randv2.PCG
}

// getAssetMix returns the total asset mix of all active players and the takeover pool
func (gs GameState) getAssetMix() assets.AssetMix {
	var am assets.AssetMix
	for _, p := range gs.Players {
		am.Add(p.Assets)
	}
	am.Add(gs.TakeoverPool)
	return am
}

// activePlayers returns an iterator over the players that have not yet lost
func (gs *GameState) activePlayers() iter.Seq2[int, *PlayerState] {
	return func(yield func(int, *PlayerState) bool) {
		for pi := range gs.Players {
			if gs.Players[pi].Status == core.PlayerStatusActive {
				if !yield(pi, &gs.Players[pi]) {
					return
				}
			}
		}
	}
}

// SetGlobalLossWithReason sets global game status and reason for loss. Caller is responsible for logging, state transitions, etc.
func (gs *GameState) SetGlobalLossWithReason(reason core.LossCondition) {
	gs.Status = core.GameStatusLoss
	gs.Reason = reason
}

// Moves all assets from the specified player to the takeover pool.
func (gs *GameState) movePlayerAssetsToTakeoverPool(pi int) {
	gs.TakeoverPool.TakeAllAssetsFrom(&(gs.Players[pi].Assets))
}

// SetRNGSeed seeds the operate-phase PCG RNG.
func (gs *GameState) SetRNGSeed(seed uint64) {
	// The seed is used directly, the stream index is fixed to 0.
	gs.pcg.Seed(seed, 0)
}

// NewGame returns a new GameState ready to play
func NewGame(numPlayers int, gameParams params.Params, logger eventlog.Logger, getAction GetPlayerAction, doneCallback func()) (*GameState, error) {
	initialAssetsPerPlayer, ok := gameParams.StartingFossilAssetsPerPlayer[numPlayers]
	if !ok {
		return nil, fmt.Errorf("invalid number of players: %d", numPlayers)
	}

	var game = GameState{
		Status:          core.GameStatusOngoing,
		Round:           0,
		CarbonEmissions: 0,
		Params:          gameParams,
		Logger:          logger,
		GetPlayerAction: getAction,
		GameOverFunc:    doneCallback,
	}

	for range numPlayers {
		p := PlayerState{
			Money:  gameParams.InitialCash,
			Status: core.PlayerStatusActive,
			Assets: assets.AssetMix{FossilsWholesale: initialAssetsPerPlayer},
		}
		game.Players = append(game.Players, p)
	}

	game.LastSnapshot = game.getSnapshot()

	return &game, nil
}
