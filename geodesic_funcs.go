package geocty

import (
	"fmt"
	"math"
	"time"

	geodesic "github.com/natemcintosh/geographiclib-go"
	timecty "github.com/tsarna/time-cty-funcs"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"
)

var wgs84 = geodesic.Wgs84()

var geoInverseResultType = cty.Object(map[string]cty.Type{
	"distance":     cty.Number,
	"bearing":      cty.Number,
	"back_bearing": cty.Number,
})

// GeoInverseFunc solves the inverse geodesic problem: given two points, returns
// the distance and bearings between them on the WGS-84 ellipsoid.
var GeoInverseFunc = function.New(&function.Spec{
	Description: "Returns distance (meters), bearing, and back_bearing between two points.",
	Params: []function.Parameter{
		{Name: "point_a", Type: cty.DynamicPseudoType},
		{Name: "point_b", Type: cty.DynamicPseudoType},
	},
	Type: function.StaticReturnType(geoInverseResultType),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		lat1, lon1, err := extractLatLon(args[0])
		if err != nil {
			return cty.NilVal, fmt.Errorf("point_a: %w", err)
		}
		lat2, lon2, err := extractLatLon(args[1])
		if err != nil {
			return cty.NilVal, fmt.Errorf("point_b: %w", err)
		}
		r := wgs84.InverseCalcDistanceAzimuths(lat1, lon1, lat2, lon2)
		return cty.ObjectVal(map[string]cty.Value{
			"distance":     cty.NumberFloatVal(r.DistanceM),
			"bearing":      cty.NumberFloatVal(normalizeAz(r.Azimuth1Deg)),
			"back_bearing": cty.NumberFloatVal(normalizeAz(r.Azimuth2Deg + 180)),
		}), nil
	},
})

// GeoDestinationFunc returns the point reached by travelling from origin on
// the given bearing. The third argument selects how far to travel, dispatched
// on its type: number (meters), duration capsule, or time capsule.
var GeoDestinationFunc = function.New(&function.Spec{
	Description: "Returns the point reached by travelling from origin on a bearing for a distance, duration, or until a target time.",
	Params: []function.Parameter{
		{Name: "origin", Type: cty.DynamicPseudoType},
		{Name: "bearing", Type: cty.Number},
		{Name: "distance_or_duration_or_time", Type: cty.DynamicPseudoType},
	},
	VarParam: &function.Parameter{
		Name: "extras",
		Type: cty.DynamicPseudoType,
	},
	Type: function.StaticReturnType(cty.DynamicPseudoType),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		if len(args) > 4 {
			return cty.NilVal, fmt.Errorf("geo_destination takes 3 or 4 arguments, got %d", len(args))
		}

		origin := args[0]
		if !origin.Type().IsObjectType() {
			return cty.NilVal, fmt.Errorf("origin must be an object, got %s", origin.Type().FriendlyName())
		}

		var bearing float64
		if err := gocty.FromCtyValue(args[1], &bearing); err != nil {
			return cty.NilVal, fmt.Errorf("bearing: %w", err)
		}

		effective := mergeObjects(origin, args[3:]...)
		effMap := effective.AsValueMap()

		lat, err := pointCoord(effMap, "lat", axisLat)
		if err != nil {
			return cty.NilVal, err
		}
		lon, err := pointCoord(effMap, "lon", axisLon)
		if err != nil {
			return cty.NilVal, err
		}

		third := args[2]
		var distM float64
		var updateTime func(map[string]cty.Value)

		switch third.Type() {
		case cty.Number:
			if err := gocty.FromCtyValue(third, &distM); err != nil {
				return cty.NilVal, fmt.Errorf("distance: %w", err)
			}

		case timecty.DurationCapsuleType:
			dur, err := timecty.GetDuration(third)
			if err != nil {
				return cty.NilVal, err
			}
			speed, err := getFloatAttr(effMap, "speed")
			if err != nil {
				return cty.NilVal, fmt.Errorf("duration form requires speed: %w", err)
			}
			distM = speed * dur.Seconds()
			updateTime = func(result map[string]cty.Value) {
				if _, ok := effMap["time"]; ok {
					origTime, err := timecty.GetTime(effMap["time"])
					if err == nil {
						result["time"] = timecty.NewTimeCapsule(origTime.Add(dur))
					}
				}
			}

		case timecty.TimeCapsuleType:
			targetTime, err := timecty.GetTime(third)
			if err != nil {
				return cty.NilVal, err
			}
			speed, err := getFloatAttr(effMap, "speed")
			if err != nil {
				return cty.NilVal, fmt.Errorf("time form requires speed: %w", err)
			}
			origTimeVal, ok := effMap["time"]
			if !ok {
				return cty.NilVal, fmt.Errorf("time form requires origin to have a time field")
			}
			origTime, err := timecty.GetTime(origTimeVal)
			if err != nil {
				return cty.NilVal, err
			}
			dur := targetTime.Sub(origTime)
			if dur < 0 {
				return cty.NilVal, fmt.Errorf("target_time %v is before origin time %v", targetTime, origTime)
			}
			distM = speed * dur.Seconds()
			updateTime = func(result map[string]cty.Value) {
				result["time"] = timecty.NewTimeCapsule(targetTime)
			}

		default:
			return cty.NilVal, fmt.Errorf(
				"third argument must be a number (meters), duration, or time; got %s",
				third.Type().FriendlyName(),
			)
		}

		dest := wgs84.DirectCalcLatLon(lat, lon, bearing, distM)
		result := make(map[string]cty.Value, len(effMap))
		for k, v := range effMap {
			result[k] = v
		}
		result["lat"] = cty.NumberFloatVal(dest.LatDeg)
		result["lon"] = cty.NumberFloatVal(dest.LonDeg)
		if updateTime != nil {
			updateTime(result)
		}
		return cty.ObjectVal(result), nil
	},
})

// GeoWaypointsFunc returns n evenly spaced points along the geodesic from
// point_a to point_b, including both endpoints.
var GeoWaypointsFunc = function.New(&function.Spec{
	Description: "Returns n evenly spaced points along the geodesic between two points.",
	Params: []function.Parameter{
		{Name: "point_a", Type: cty.DynamicPseudoType},
		{Name: "point_b", Type: cty.DynamicPseudoType},
		{Name: "n", Type: cty.Number},
	},
	Type: function.StaticReturnType(cty.List(cty.Object(map[string]cty.Type{
		"lat":   cty.Number,
		"lon":   cty.Number,
		"track": cty.Number,
	}))),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		lat1, lon1, err := extractLatLon(args[0])
		if err != nil {
			return cty.NilVal, fmt.Errorf("point_a: %w", err)
		}
		lat2, lon2, err := extractLatLon(args[1])
		if err != nil {
			return cty.NilVal, fmt.Errorf("point_b: %w", err)
		}
		var n int
		if err := gocty.FromCtyValue(args[2], &n); err != nil {
			return cty.NilVal, fmt.Errorf("n: %w", err)
		}
		if n < 2 {
			return cty.NilVal, fmt.Errorf("n must be >= 2, got %d", n)
		}

		inv := wgs84.InverseCalcDistanceAzimuths(lat1, lon1, lat2, lon2)
		totalDist := inv.DistanceM
		azi1 := inv.Azimuth1Deg

		waypoints := make([]cty.Value, n)
		for i := 0; i < n; i++ {
			frac := float64(i) / float64(n-1)
			dist := totalDist * frac
			r := wgs84.DirectCalcLatLonAzi(lat1, lon1, azi1, dist)
			waypoints[i] = cty.ObjectVal(map[string]cty.Value{
				"lat":   cty.NumberFloatVal(r.LatDeg),
				"lon":   cty.NumberFloatVal(r.LonDeg),
				"track": cty.NumberFloatVal(normalizeAz(r.AziDeg)),
			})
		}
		return cty.ListVal(waypoints), nil
	},
})

// normalizeAz normalizes a degree value to [0, 360).
func normalizeAz(deg float64) float64 {
	deg = math.Mod(deg, 360)
	if deg < 0 {
		deg += 360
	}
	return deg
}

// mergeObjects copies all fields from base, then overwrites with fields from
// each extras object in order. extras values win on key conflicts.
func mergeObjects(base cty.Value, extras ...cty.Value) cty.Value {
	if len(extras) == 0 {
		return base
	}
	merged := make(map[string]cty.Value)
	for k, v := range base.AsValueMap() {
		merged[k] = v
	}
	for _, ext := range extras {
		if !ext.Type().IsObjectType() {
			continue
		}
		for k, v := range ext.AsValueMap() {
			merged[k] = v
		}
	}
	return cty.ObjectVal(merged)
}

func getFloatAttr(attrs map[string]cty.Value, key string) (float64, error) {
	v, ok := attrs[key]
	if !ok {
		return 0, fmt.Errorf("missing %q attribute", key)
	}
	var f float64
	if err := gocty.FromCtyValue(v, &f); err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return f, nil
}

// getOptionalTime extracts a time.Time from an optional "time" attribute,
// returning (zero, false, nil) if not present.
func getOptionalTime(attrs map[string]cty.Value) (time.Time, bool, error) {
	v, ok := attrs["time"]
	if !ok {
		return time.Time{}, false, nil
	}
	t, err := timecty.GetTime(v)
	if err != nil {
		return time.Time{}, false, err
	}
	return t, true, nil
}
