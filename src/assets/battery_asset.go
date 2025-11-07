package assets

const allowedBatteryModes = OperationModeCapacity | OperationModeForecastingService

type BatteryAsset struct {
	asset
}

var _ Asset = (*BatteryAsset)(nil)

func (a *BatteryAsset) SetMode(mode OperationMode) {
	a.operationMode = a.operationMode | (mode & allowedBatteryModes)
}
func (a *asset) ClearMode()         { a.operationMode = OperationMode(0) }
