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
	switch string(text) {
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
