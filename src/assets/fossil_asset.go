package assets

const allowedFossilModes = OperationModeCapacity | OperationModeApplyCarbonTax
const nonResettableFossilModes = OperationModeApplyCarbonTax

type FossilAsset struct {
	asset
}

var _ Asset = (*FossilAsset)(nil)

func (a *FossilAsset) SetMode(mode OperationMode) {
	a.operationMode = a.operationMode | (mode & allowedFossilModes)
}
func (a *FossilAsset) ClearMode() { a.operationMode = a.operationMode & nonResettableFossilModes }
