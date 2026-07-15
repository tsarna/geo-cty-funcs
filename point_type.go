package geocty

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// IsGeoPoint reports whether v is a geographic point: an object with numeric `lat` and
// `lon` attributes. Additional attributes (altitude, time, speed, name, …) are allowed,
// and where a function preserves them they are carried through — this is an open
// predicate over the two coordinates, not a fixed shape. It returns nil when v qualifies
// and an error naming what is wrong otherwise.
//
// It is the predicate behind the `geopoint` type a functy host registers, so that a
// `.cty` annotation, a signature, or a host `var` constraint can name the shape every
// geo function speaks in:
//
//	parser.RegisterOpenType("geopoint", geocty.IsGeoPoint)
//
// The requirement is deliberately narrow — lat and lon, both numbers — because that is
// exactly what the functions here read; they are all horizontal (great-circle and
// geodesic on the WGS-84 surface) and none consumes an altitude. It says nothing about
// dimensionality: a value carrying an altitude is a geopoint, and the altitude rides
// along. Coordinates supplied as strings are not a geopoint until geo_point() has parsed
// them into numbers.
func IsGeoPoint(v cty.Value) error {
	ty := v.Type()
	if !ty.IsObjectType() {
		return fmt.Errorf("a geopoint must be an object with lat and lon, got %s", ty.FriendlyName())
	}
	for _, attr := range [...]string{"lat", "lon"} {
		if !ty.HasAttribute(attr) {
			return fmt.Errorf("a geopoint must have a %q attribute", attr)
		}
		if at := ty.AttributeType(attr); at != cty.Number {
			return fmt.Errorf("a geopoint's %q must be a number, got %s", attr, at.FriendlyName())
		}
	}
	return nil
}
