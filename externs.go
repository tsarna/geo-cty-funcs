package geocty

import _ "embed"

//go:embed externs.cty
var externsCty []byte

// ExternsFilename is the name reported for the embedded declarations in
// diagnostics.
const ExternsFilename = "geo-cty-funcs/externs.cty"

// Externs returns the functy `//functy:extern` declarations for the geo functions whose
// real signature their cty metadata cannot express: their bytes, which this package does
// not itself parse.
//
// Every geo function is declared, for one of two reasons. cty can only make a function's
// *trailing* parameters optional, and only by making them variadic — which erases their
// names, types, and defaults — so an optional coordinate string, format, offset, or time
// reads as `...args`; and cty has one signature per function, so those that dispatch on an
// argument's type (geo_destination's third argument; the solar functions' trailing
// offset-or-time) or vary their result shape (geo_point merging a base) cannot be
// described at all.
//
// The rest have a fixed arity and a fixed return shape, but they are declared too: their
// point arguments are objects with arbitrary extra attributes, so their cty parameter type
// is `dynamic` — and a dynamic argument poisons cty's return type to `dynamic`, hiding the
// whole return payload (a distance and bearing, each waypoint's lat/lon/track). Only the
// declaration can show it, and the `geopoint` shape of the inputs.
//
// The declarations lean on the `geopoint` open type (see IsGeoPoint): register it
// alongside them.
//
//	parser.RegisterOpenType("geopoint", geocty.IsGeoPoint)
//	parser.RegisterExterns(geocty.Externs(), geocty.ExternsFilename)
func Externs() []byte { return externsCty }
