// package assets defines the grid asset types for JouleQuest.
package assets

type Type int

//go:generate go tool stringer -type=Type -trimprefix=Type
const (
	TypeRenewable Type = iota
	TypeFossil
	TypeBattery
)

func (at Type) LogKey() string {
	return "asset_type"
}

var Types = [...]Type{TypeBattery, TypeRenewable, TypeFossil}

// Operation mode is a bitfield indicating whether an asset is operating in a certain way.
type OperationMode int

//go:generate go tool stringer -type=OperationMode -trimprefix=OperationMode
const (
	OperationModeCapacity OperationMode = 1 << iota
)

func (mm OperationMode) LogKey() string {
	return "operation_mode"
}

// Asset represents a generic asset in the game.
type Asset interface {
	Type() Type            // Returns the type of the asset
	Mode() OperationMode   // Returns the current operation mode of an asset
	SetMode(OperationMode) // Sets the the given operation mode on the asset, if possible
	ClearMode()            // Resets the asset to its default operation mode
}

// asset provides embeddable implementations of the read-only part of the Asset interface
type asset struct {
	assetType     Type
	operationMode OperationMode
}

func (a asset) Type() Type          { return a.assetType }
func (a asset) Mode() OperationMode { return a.operationMode }
func (a *asset) ClearMode()         { a.operationMode = OperationMode(0) }

func New(t Type) Asset {
	switch t{
	case TypeBattery:
		return &BatteryAsset{asset: asset{assetType: TypeBattery}}
	case TypeFossil:
		return &FossilAsset{asset: asset{assetType: TypeFossil}}
	case TypeRenewable:
		return &RenewableAsset{asset: asset{assetType: TypeRenewable}}
	}
	return nil
}