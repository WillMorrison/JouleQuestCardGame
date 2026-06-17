package main

import (
	"github.com/WillMorrison/JouleQuestCardGame/compact/game"
	"github.com/WillMorrison/JouleQuestCardGame/compact/params"
)

var (
	gParams params.CompactParams = params.Default
	gGame   game.Game
)

// This should be exported as _initialize when building a reactor module.
func init() {
	gGame.Reset(4, gParams)
	gGame.SetRNGSeed(0)
}

//go:wasmexport Reset
func Reset(numPlayers int32) int32 {
	return int32(gGame.Reset(numPlayers, gParams))
}

//go:wasmexport ApplyAction
func ApplyAction(playerIndex int32, actionInt int32) int32 {
	return int32(gGame.ApplyPlayerAction(playerIndex, actionInt))
}

//go:wasmexport SetRNGSeed
func SetRNGSeed(seed int32) {
	gGame.SetRNGSeed(uint64(uint32(seed)))
}
