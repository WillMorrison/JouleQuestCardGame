package game

import (
	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
)

// Player is one player's compact state
type Player struct {
	Status     core.PlayerStatus
	Reason     core.LossCondition
	Money      int32
	Mix        assets.AssetMix
	IsBuilding bool
}

func (p *Player) setLoss(reason core.LossCondition) {
	p.Status = core.PlayerStatusLost
	p.Reason = reason
}
