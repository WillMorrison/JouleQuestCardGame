package game

// Enum numeric values match github.com/WillMorrison/JouleQuestCardGame/engine for the same concepts.
// They are duplicated here to avoid pulling in the engine package (and eventlog, json, etc.) for the WASM engine.

type GameStatus int32

const (
	GameStatusOngoing GameStatus = iota
	GameStatusWin
	GameStatusLoss
)

type PlayerStatus int32

const (
	PlayerStatusActive PlayerStatus = iota
	PlayerStatusLost
)

type LossCondition int32

const (
	LossConditionNone LossCondition = iota
	LossConditionPlayerBankrupt
	LossConditionLastPlayerWithFossilAssets
	LossConditionGridUnstable
	LossConditionInsufficientGeneration
	LossConditionCarbonEmissionsExceeded
	LossConditionNoActivePlayers
	LossConditionUnownedTakeoverAssets
)

type phase int32

const (
	phaseGameStart phase = iota
	phaseBuild
	phaseOperate
	phaseGameEnd
)
