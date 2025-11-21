// package assets defines the grid asset types for JouleQuest.
package assets

import "fmt"

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

func (at Type) MarshalText() ([]byte, error) {
	return []byte(at.String()), nil
}

func (at *Type) UnmarshalText(text []byte) error {
	switch string(text){
	case TypeBattery.String():
		*at = TypeBattery
	case TypeRenewable.String():
		*at = TypeRenewable
	case TypeFossil.String():
		*at = TypeFossil
	default:
		return fmt.Errorf("%q is not a valid asset Type", text)
	}
	return nil
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
	fmt.Stringer
	Type() Type            // Returns the type of the asset
	Mode() OperationMode   // Returns the current operation mode of an asset
	SetMode(OperationMode) // Sets the the given operation mode on the asset, if possible
	ClearMode()            // Resets the asset to its default operation mode
}

// asset implements the Asset interface
type asset struct {
	assetType     Type
	operationMode OperationMode
	allowedModes  OperationMode
}

func (a asset) Type() Type          { return a.assetType }
func (a asset) Mode() OperationMode { return a.operationMode }
func (a *asset) ClearMode()         { a.operationMode = OperationMode(0) }
func (a *asset) SetMode(mode OperationMode) {
	a.operationMode = a.operationMode | (mode & a.allowedModes)
}

func (a asset) String() string {
	if a.operationMode&OperationModeCapacity != 0 {
		return fmt.Sprintf("%sAsset{%s}", a.assetType.String(), OperationModeCapacity.String())
	}
	return fmt.Sprintf("%sAsset{}", a.assetType.String())
}

func New(t Type) Asset {
	switch t {
	case TypeBattery:
		return &asset{assetType: TypeBattery, allowedModes: OperationModeCapacity}
	case TypeFossil:
		return &asset{assetType: TypeFossil, allowedModes: OperationModeCapacity}
	case TypeRenewable:
		return &asset{assetType: TypeRenewable}
	}
	return nil
}
