// Package geocty provides geographic, solar/astronomical, geodesic, and
// geometric functions for use in go-cty / HCL2 evaluation contexts.
//
// See GetGeoFunctions for registration.
package geocty

import "github.com/zclconf/go-cty/cty/function"

// GetGeoFunctions returns all geographic cty functions for registration in an
// eval context.
//
// The names are namespaced. The geographic, geodesic, and geometric functions —
// whose bare names (point, area, contains, nearest) would be too generic to expose
// globally — live under `geo::`. The solar-event and celestial-position functions
// live under `sky::`. HCL parses `a::b(x)` natively as a single flat map key, so this
// is a naming choice, not a structural one, and the leaf names drop the redundant
// prefix the flat names carried.
func GetGeoFunctions() map[string]function.Function {
	return map[string]function.Function{
		// Geographic, geodesic, geometric.
		"geo::point":          GeoPointFunc,
		"geo::format":         GeoFormatFunc,
		"geo::destination":    GeoDestinationFunc,
		"geo::inverse":        GeoInverseFunc,
		"geo::waypoints":      GeoWaypointsFunc,
		"geo::area":           GeoAreaFunc,
		"geo::contains":       GeoContainsFunc,
		"geo::nearest":        GeoNearestFunc,
		"geo::line_intersect": GeoLineIntersectFunc,

		// Solar events and celestial position.
		"sky::sunrise":        SunriseFunc,
		"sky::sunset":         SunsetFunc,
		"sky::solar_noon":     SolarNoonFunc,
		"sky::solar_midnight": SolarMidnightFunc,
		"sky::sun_position":   SunPositionFunc,
		"sky::moon_position":  MoonPositionFunc,
		"sky::moon_phase":     MoonPhaseFunc,
	}
}
