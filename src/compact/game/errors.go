package game

type ErrCode int32

const (
	CodeOK ErrCode = iota
	CodeInvalidPlayerCount
	CodeInvalidAction
	CodeUnknown
)

func (ec ErrCode) Error() string {
	switch ec {
	case CodeOK:
		return "ok"
	case CodeInvalidPlayerCount:
		return "invalid player num"
	case CodeInvalidAction:
		return "invalid action"
	default:
		return "unknown error"
	}
}
