package assets

import "github.com/WillMorrison/JouleQuestCardGame/core"

type RenewableAsset struct{}

var _ Asset = (*RenewableAsset)(nil)

func (a RenewableAsset) Type() core.AssetType  { return core.AssetTypeRenewable }
func (a RenewableAsset) GetPnL() core.PnLTable { return core.RenewablePnLTable }
func (a *RenewableAsset) Reset()               {}
