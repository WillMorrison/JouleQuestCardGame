package game

import "github.com/WillMorrison/JouleQuestCardGame/assets"

// Player is one player's compact state (five-bucket mix, no per-asset slices).
type Player struct {
	Status    PlayerStatus
	Reason    LossCondition
	Money     int32
	Mix       assets.AssetMix
	IsBuilding bool
}

func (p *Player) resetModesForBuild() {
	p.Mix.FossilsWholesale += p.Mix.FossilsCapacity
	p.Mix.FossilsCapacity = 0
	p.Mix.BatteriesArbitrage += p.Mix.BatteriesCapacity
	p.Mix.BatteriesCapacity = 0
}

func (p Player) hasFossilAssets() bool {
	return p.Mix.FossilsWholesale > 0 || p.Mix.FossilsCapacity > 0
}

func (p *Player) setLoss(reason LossCondition) {
	p.Status = PlayerStatusLost
	p.Reason = reason
}
