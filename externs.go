package geocty

import (
	"embed"
	"path"
)

//go:embed externs/*.cty
var externsFS embed.FS

// Externs returns the functy `//functy:extern` declarations for the geo functions,
// keyed by a filename to report in diagnostics. This package does not itself parse them;
// the bytes are opaque to it.
//
// Every geo function is declared, for one of two reasons. cty can only make a function's
// *trailing* parameters optional, and only by making them variadic — which erases their
// names, types, and defaults — so an optional coordinate string, format, offset, or time
// reads as `...args`; and cty has one signature per function, so those that dispatch on an
// argument's type (geo::destination's third argument; the solar functions' trailing
// offset-or-time) or vary their result shape (geo::point merging a base) cannot be
// described at all.
//
// The rest have a fixed arity and a fixed return shape, but they are declared too: their
// point arguments are objects with arbitrary extra attributes, so their cty parameter type
// is `dynamic` — and a dynamic argument poisons cty's return type to `dynamic`, hiding the
// whole return payload (a distance and bearing, each waypoint's lat/lon/track). Only the
// declaration can show it, and the `geopoint` shape of the inputs.
//
// There is one file per namespace, because a functy source declares at most one: `geo`
// (geographic, geodesic, geometric) and `sky` (solar events, celestial position). The
// declarations lean on the `geopoint` open type (see IsGeoPoint). A host registers both:
//
//	parser.RegisterOpenType("geopoint", geocty.IsGeoPoint)
//	for name, src := range geocty.Externs() {
//	    parser.RegisterExterns(src, name)
//	}
func Externs() map[string][]byte {
	entries, err := externsFS.ReadDir("externs")
	if err != nil {
		// Unreachable: the directory is embedded at build time.
		panic("geocty: embedded externs unreadable: " + err.Error())
	}

	out := make(map[string][]byte, len(entries))
	for _, e := range entries {
		name := path.Join("externs", e.Name())
		src, err := externsFS.ReadFile(name)
		if err != nil {
			panic("geocty: embedded extern unreadable: " + err.Error())
		}
		out[path.Join("geo-cty-funcs", name)] = src
	}
	return out
}
