package assets

const allowedBatteryModes = OperationModeCapacity

type BatteryAsset struct {
	asset
}

var _ Asset = (*BatteryAsset)(nil)

func (a *BatteryAsset) SetMode(mode OperationMode) {
	a.operationMode = a.operationMode | (mode & allowedBatteryModes)
}
