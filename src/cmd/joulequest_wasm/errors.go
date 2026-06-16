package main

import (
	"errors"

	cgame "github.com/WillMorrison/JouleQuestCardGame/compact/game"
)

// WASM export error codes (non-zero = failure).
const (
	CodeOK int32 = iota
	CodeInvalidPlayerCount
	CodeNoStartingFossils
	CodeNotBuildPhase
	CodeInvalidAction
	CodeUnknown
)

func errCode(err error) int32 {
	if err == nil {
		return CodeOK
	}
	switch {
	case errors.Is(err, cgame.ErrInvalidPlayerCount):
		return CodeInvalidPlayerCount
	case errors.Is(err, cgame.ErrNoStartingFossils):
		return CodeNoStartingFossils
	case errors.Is(err, cgame.ErrNotBuildPhase):
		return CodeNotBuildPhase
	case errors.Is(err, cgame.ErrInvalidAction):
		return CodeInvalidAction
	default:
		return CodeUnknown
	}
}
