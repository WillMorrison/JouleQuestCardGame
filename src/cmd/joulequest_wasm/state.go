package main

import (
	"github.com/WillMorrison/JouleQuestCardGame/compact/game"
	cparams "github.com/WillMorrison/JouleQuestCardGame/compact/params"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

var (
	gParams cparams.CompactParams
	gGame   game.Game
)

func main() {}

// This should be exported as _initialize when building a reactor module.
func init() {
	gParams, _ = cparams.FromLegacy(params.Default)
	gGame.Reset(4, gParams)
	gGame.SetRNGSeed(0)
}

//go:wasmexport Reset
func Reset(numPlayers int32) int32 {
	if err := gGame.Reset(int(numPlayers), gParams); err != nil {
		return errCode(err)
	}
	return CodeOK
}

//go:wasmexport ApplyAction
func ApplyAction(playerIndex int32, actionInt int32) int32 {
	if err := gGame.ApplyPlayerAction(int(playerIndex), actionInt); err != nil {
		return errCode(err)
	}
	return CodeOK
}

//go:wasmexport SetRNGSeed
func SetRNGSeed(seed int32) int32 {
	gGame.SetRNGSeed(uint64(uint32(seed)))
	return CodeOK
}
