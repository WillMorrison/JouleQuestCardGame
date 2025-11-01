package assets

import (
	"github.com/WillMorrison/JouleQuestCardGame/core"
)

// Service represents the battery service provider.
type Service int

//go:generate go tool stringer -type=Service -trimprefix=Service
const (
	ServiceNone Service = iota
	ServicePriceForecast
)

type BatteryAsset struct {
	Mode    MarketMode
	Service Service
}

var _ Asset = (*BatteryAsset)(nil)
var _ CapacityAsset = (*BatteryAsset)(nil)

func (a BatteryAsset) Type() core.AssetType { return core.AssetTypeBattery }

func (a BatteryAsset) GetPnL() core.PnLTable {
	switch {
	case a.Mode == MarketModeCapacity:
		return core.CapacityPnLTable
	case a.Service == ServicePriceForecast:
		return core.BatteryPnLTableWithService
	case a.Service == ServiceNone:
		return core.BatteryPnLTable
	default:
		panic("unknown PnL for battery asset with mode " + a.Mode.String() + " and service " + a.Service.String())
	}
}

func (a *BatteryAsset) BuyPriceForecastService() { a.Service = ServicePriceForecast }
func (a *BatteryAsset) PledgeCapacity()          { a.Mode = MarketModeCapacity }
func (a BatteryAsset) IsCapacity() bool          { return a.Mode == MarketModeCapacity }
func (a *BatteryAsset) Reset() {
	a.Mode = MarketModeArbitrage
	a.Service = ServiceNone
}
