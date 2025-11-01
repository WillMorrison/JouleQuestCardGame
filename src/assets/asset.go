// package assets defines the grid asset types for JouleQuest.
package assets

import "github.com/WillMorrison/JouleQuestCardGame/core"

// Asset represents a generic asset in the game.
type Asset interface {
	Type() core.AssetType  // Returns the type of the asset
	GetPnL() core.PnLTable // Returns the profit and loss table for the asset in the current operation mode
	Reset()                // Resets the asset to its default operation mode
}

// CapacityAsset represents an asset that can participate in capacity markets.
type CapacityAsset interface {
	Asset
	IsCapacity() bool
	PledgeCapacity()
}

type MarketMode int

//go:generate go tool stringer -type=MarketMode -trimprefix=MarketMode
const (
	MarketModeCapacity  MarketMode = iota
	MarketModeWholesale            // Fossil asset default
	MarketModeArbitrage            // Battery asset default
)

func (mm MarketMode) LogKey() string {
	return "market_mode"
}
