package geocty

import (
	"fmt"
	"math"
	"time"

	"github.com/kixorz/suncalc"
	timecty "github.com/tsarna/time-cty-funcs"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var sunPositionResultType = cty.Object(map[string]cty.Type{
	"azimuth":  cty.Number,
	"altitude": cty.Number,
})

var moonPositionResultType = cty.Object(map[string]cty.Type{
	"azimuth":  cty.Number,
	"altitude": cty.Number,
	"distance": cty.Number,
})

var moonPhaseResultType = cty.Object(map[string]cty.Type{
	"fraction": cty.Number,
	"phase":    cty.Number,
	"angle":    cty.Number,
})

// SunPositionFunc returns the sun's azimuth and altitude as seen from a point.
var SunPositionFunc = function.New(&function.Spec{
	Description: "Returns the sun's azimuth (degrees, clockwise from north) and altitude (degrees above horizon).",
	Params: []function.Parameter{
		{Name: "point", Type: cty.DynamicPseudoType},
	},
	VarParam: &function.Parameter{
		Name: "t",
		Type: timecty.TimeCapsuleType,
	},
	Type: function.StaticReturnType(sunPositionResultType),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		if len(args) > 2 {
			return cty.NilVal, fmt.Errorf("sun_position takes 1 or 2 arguments, got %d", len(args))
		}
		lat, lon, err := extractLatLon(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		t := time.Now()
		if len(args) == 2 {
			t, err = timecty.GetTime(args[1])
			if err != nil {
				return cty.NilVal, err
			}
		}
		pos := suncalc.GetPosition(t, lat, lon)
		return cty.ObjectVal(map[string]cty.Value{
			"azimuth":  cty.NumberFloatVal(southAzToNorthDeg(pos.Azimuth)),
			"altitude": cty.NumberFloatVal(radToDeg(pos.Altitude)),
		}), nil
	},
})

// MoonPositionFunc returns the moon's azimuth, altitude, and distance as seen
// from a point.
var MoonPositionFunc = function.New(&function.Spec{
	Description: "Returns the moon's azimuth (degrees), altitude (degrees), and distance (meters).",
	Params: []function.Parameter{
		{Name: "point", Type: cty.DynamicPseudoType},
	},
	VarParam: &function.Parameter{
		Name: "t",
		Type: timecty.TimeCapsuleType,
	},
	Type: function.StaticReturnType(moonPositionResultType),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		if len(args) > 2 {
			return cty.NilVal, fmt.Errorf("moon_position takes 1 or 2 arguments, got %d", len(args))
		}
		lat, lon, err := extractLatLon(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		t := time.Now()
		if len(args) == 2 {
			t, err = timecty.GetTime(args[1])
			if err != nil {
				return cty.NilVal, err
			}
		}
		pos := suncalc.GetMoonPosition(t, lat, lon)
		return cty.ObjectVal(map[string]cty.Value{
			"azimuth":  cty.NumberFloatVal(southAzToNorthDeg(pos.Azimuth)),
			"altitude": cty.NumberFloatVal(radToDeg(pos.Altitude)),
			"distance": cty.NumberFloatVal(pos.Distance * 1000),
		}), nil
	},
})

// MoonPhaseFunc returns the moon's illumination fraction, phase, and angle.
var MoonPhaseFunc = function.New(&function.Spec{
	Description: "Returns the moon's illuminated fraction, phase (0=new, 0.5=full), and bright-limb angle (degrees).",
	Params:      []function.Parameter{},
	VarParam: &function.Parameter{
		Name: "t",
		Type: timecty.TimeCapsuleType,
	},
	Type: function.StaticReturnType(moonPhaseResultType),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		if len(args) > 1 {
			return cty.NilVal, fmt.Errorf("moon_phase takes 0 or 1 arguments, got %d", len(args))
		}
		t := time.Now()
		if len(args) == 1 {
			var err error
			t, err = timecty.GetTime(args[0])
			if err != nil {
				return cty.NilVal, err
			}
		}
		ill := suncalc.GetMoonIllumination(t)
		return cty.ObjectVal(map[string]cty.Value{
			"fraction": cty.NumberFloatVal(ill.Fraction),
			"phase":    cty.NumberFloatVal(ill.Phase),
			"angle":    cty.NumberFloatVal(radToDeg(ill.Angle)),
		}), nil
	},
})

// southAzToNorthDeg converts an azimuth in radians measured clockwise from
// south (suncalc convention) to degrees measured clockwise from north (spec
// convention: 0=N, 90=E, 180=S, 270=W).
func southAzToNorthDeg(azRad float64) float64 {
	deg := math.Mod((azRad+math.Pi)*180/math.Pi, 360)
	if deg < 0 {
		deg += 360
	}
	return deg
}

func radToDeg(r float64) float64 {
	return r * 180 / math.Pi
}
