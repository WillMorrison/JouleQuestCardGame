package assets

type RenewableAsset struct {
	asset
}

var _ Asset = (*RenewableAsset)(nil)

func (a *RenewableAsset) SetMode(mode OperationMode) {}
