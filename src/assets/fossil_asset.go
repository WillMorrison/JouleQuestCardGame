package assets

const allowedFossilModes = OperationModeCapacity

type FossilAsset struct {
	asset
}

var _ Asset = (*FossilAsset)(nil)

func (a *FossilAsset) SetMode(mode OperationMode) {
	a.operationMode = a.operationMode | (mode & allowedFossilModes)
}
