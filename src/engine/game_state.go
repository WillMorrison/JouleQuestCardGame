// This file contains definitions and helper methods for game state

package engine

import (
	"encoding/json"
	"fmt"
	"iter"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/eventlog"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

type PlayerState struct {
	Status PlayerStatus   // Whether the player has lost
	Reason LossCondition  // Reason for player loss, if applicable
	Money  int            // Player's current money
	Assets []assets.Asset // Player's owned assets

	isBuilding bool // Internal tracker of whether the player has finished the build round
}

func (ps PlayerState) getAssetMix() assets.AssetMix {
	var am assets.AssetMix
	for _, a := range ps.Assets {
		am.AddAsset(a)
	}
	return am
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
		Assets: ps.getAssetMix(),
	}
	if ps.Status != PlayerStatusActive {
		psj.Reason = ps.Reason.String()
	}
	return json.Marshal(psj)
}

// Returns whether the player owns any fossil assets
func (ps PlayerState) HasFossilAssets() bool {
	for _, a := range ps.Assets {
		if a.Type() == assets.TypeFossil {
			return true
		}
	}
	return false
}

// Resets all of the player's assets to their default operating mode
func (ps *PlayerState) resetAllAssets() {
	for _, a := range ps.Assets {
		a.ClearMode()
	}
}

// SetLossWithReason sets status and reason for loss. Caller is responsible for logging, handling asset takeover, etc.
func (ps *PlayerState) SetLossWithReason(reason LossCondition) {
	ps.Status = PlayerStatusLost
	ps.Reason = reason
}

// Snapshot holds summary statistics from the outcome of an Operate phase
type Snapshot struct {
	AssetMix        assets.AssetMix
	PriceVolatility core.PriceVolatility
	GridStability   core.GridStability
}

type GameState struct {
	Status          GameStatus
	Reason          LossCondition `json:",omitzero"` // Reason for global loss, if applicable
	Round           int
	CarbonEmissions int // Total carbon emissions in the world
	Players         []PlayerState
	TakeoverPool    []assets.Asset // Assets available for takeover

	LastSnapshot Snapshot // Summary of the previous round's Operate phase

	Params          params.Params
	Logger          eventlog.Logger `json:"-"`
	GetPlayerAction GetPlayerAction // callback when the game needs to pick the next player action
}

// allAssets iterates over assets in player portfolios and in the takeover pool
func (gs GameState) allAssets() iter.Seq[assets.Asset] {
	return func(yield func(assets.Asset) bool) {
		for pi := range gs.Players {
			for ai := range gs.Players[pi].Assets {
				if !yield(gs.Players[pi].Assets[ai]) {
					return
				}
			}
		}
		for ai := range gs.TakeoverPool {
			if !yield(gs.TakeoverPool[ai]) {
				return
			}
		}
	}
}

func (gs GameState) getAssetMix() assets.AssetMix {
	var am assets.AssetMix
	for a := range gs.allAssets() {
		am.AddAsset(a)
	}
	return am
}

// activePlayers returns an iterator over the players that have not yet lost
func (gs *GameState) activePlayers() iter.Seq2[int, *PlayerState] {
	return func(yield func(int, *PlayerState) bool) {
		for pi := range gs.Players {
			if gs.Players[pi].Status == PlayerStatusActive {
				if !yield(pi, &gs.Players[pi]) {
					return
				}
			}
		}
	}
}

// SetGlobalLossWithReason sets global game status and reason for loss. Caller is responsible for logging, state transitions, etc.
func (gs *GameState) SetGlobalLossWithReason(reason LossCondition) {
	gs.Status = GameStatusLoss
	gs.Reason = reason
}

// Moves all assets from the specified player to the takeover pool.
func (gs *GameState) movePlayerAssetsToTakeoverPool(pi int) {
	gs.TakeoverPool = append(gs.TakeoverPool, gs.Players[pi].Assets...)
	gs.Players[pi].Assets = nil
}

// NewGame returns a new GameState ready to play
func NewGame(numPlayers int, gameParams params.Params, logger eventlog.Logger, getAction GetPlayerAction) (*GameState, error) {
	initialAssetsPerPlayer, ok := gameParams.StartingFossilAssetsPerPlayer[numPlayers]
	if !ok {
		return nil, fmt.Errorf("invalid number of players: %d", numPlayers)
	}

	var game = GameState{
		Status:          GameStatusOngoing,
		Round:           0,
		CarbonEmissions: 0,
		Params:          gameParams,
		Logger:          logger,
		GetPlayerAction: getAction,
	}

	for range numPlayers {
		p := PlayerState{
			Money:  gameParams.InitialCash,
			Status: PlayerStatusActive,
		}
		for range initialAssetsPerPlayer {
			p.Assets = append(p.Assets, assets.New(assets.TypeFossil))
		}
		game.Players = append(game.Players, p)
	}

	game.LastSnapshot = game.getSnapshot()

	return &game, nil
}
