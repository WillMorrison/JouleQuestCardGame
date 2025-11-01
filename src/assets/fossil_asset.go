package assets

import (
	"fmt"

	"github.com/WillMorrison/JouleQuestCardGame/core"
)

type Policy int

//go:generate go tool stringer -type=Policy -trimprefix=Policy
const (
	PolicyNone Policy = iota
	PolicyCarbonTax
)

func (p Policy) LogKey() string {
	return "carbon_tax_policy"
}

type FossilAsset struct {
	Mode   MarketMode
	Policy Policy
}

var _ Asset = (*FossilAsset)(nil)
var _ CapacityAsset = (*FossilAsset)(nil)

func (a FossilAsset) Type() core.AssetType { return core.AssetTypeFossil }

func (a FossilAsset) GetPnL() core.PnLTable {
	switch {
	case a.Mode == MarketModeCapacity && a.Policy == PolicyNone:
		return core.CapacityPnLTable
	case a.Mode == MarketModeCapacity && a.Policy == PolicyCarbonTax:
		return core.FossilCapacityPnLTableWithCarbonTax
	case a.Mode == MarketModeWholesale && a.Policy == PolicyNone:
		return core.FossilPnLTable
	case a.Mode == MarketModeWholesale && a.Policy == PolicyCarbonTax:
		return core.FossilPnLTableWithCarbonTax
	default:
		panic(fmt.Sprintf("unknown PnL for fossil asset with mode %s and policy %s", a.Mode.String(), a.Policy.String()))
	}
}

func (a *FossilAsset) PledgeCapacity() { a.Mode = MarketModeCapacity }
func (a FossilAsset) IsCapacity() bool { return a.Mode == MarketModeCapacity }
func (a *FossilAsset) Reset()          { a.Mode = MarketModeWholesale }
func (a *FossilAsset) ApplyCarbonTax() { a.Policy = PolicyCarbonTax }
