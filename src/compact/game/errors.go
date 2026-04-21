package game

import "errors"

var (
	ErrInvalidPlayerCount = errors.New("compact/game: invalid player count")
	ErrNoStartingFossils  = errors.New("compact/game: no starting fossil count for player count in params")
	ErrNotBuildPhase      = errors.New("compact/game: action only valid in build phase")
	ErrInvalidAction      = errors.New("compact/game: action not allowed for player")
)
