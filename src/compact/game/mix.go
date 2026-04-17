package game

import "github.com/WillMorrison/JouleQuestCardGame/assets"

// addMix adds the assets in the src mix to the dst mix.
func addMix(dst *assets.AssetMix, src assets.AssetMix) {
	dst.Renewables += src.Renewables
	dst.BatteriesArbitrage += src.BatteriesArbitrage
	dst.BatteriesCapacity += src.BatteriesCapacity
	dst.FossilsWholesale += src.FossilsWholesale
	dst.FossilsCapacity += src.FossilsCapacity
}

// moveMixTo moves the assets from the src mix to the dst mix, then zeros the src mix.
func moveMixTo(dst *assets.AssetMix, src *assets.AssetMix) {
	addMix(dst, *src)
	*src = assets.AssetMix{}
}
