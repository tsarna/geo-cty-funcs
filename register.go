// Package geocty provides geographic, solar/astronomical, geodesic, and
// geometric functions for use in go-cty / HCL2 evaluation contexts.
//
// See GetGeoFunctions for registration.
package geocty

import "github.com/zclconf/go-cty/cty/function"

// GetGeoFunctions returns all geographic cty functions for registration in an
// eval context.
func GetGeoFunctions() map[string]function.Function {
	return map[string]function.Function{
		"geo_point":       GeoPointFunc,
		"geo_format":      GeoFormatFunc,
		"sunrise":         SunriseFunc,
		"sunset":          SunsetFunc,
		"solar_noon":      SolarNoonFunc,
		"solar_midnight":  SolarMidnightFunc,
		"sun_position":    SunPositionFunc,
		"moon_position":   MoonPositionFunc,
		"moon_phase":      MoonPhaseFunc,
		"geo_inverse":     GeoInverseFunc,
		"geo_destination": GeoDestinationFunc,
		"geo_waypoints":       GeoWaypointsFunc,
		"geo_area":            GeoAreaFunc,
		"geo_contains":        GeoContainsFunc,
		"geo_nearest":         GeoNearestFunc,
		"geo_line_intersect":  GeoLineIntersectFunc,
	}
}
