package geocty

import (
	"fmt"
	"time"

	"github.com/nathan-osman/go-sunrise"
	timecty "github.com/tsarna/time-cty-funcs"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// maxPolarSearch is the maximum number of days to search forward when the sun
// does not rise or set on a given day (polar regions).
const maxPolarSearch = 400

// SunriseFunc returns the next sunrise (+ optional offset) after time t.
var SunriseFunc = makeSolarFunc(
	"Returns the next sunrise (+ optional offset) after a reference time.",
	solarSunrise,
)

// SunsetFunc returns the next sunset (+ optional offset) after time t.
var SunsetFunc = makeSolarFunc(
	"Returns the next sunset (+ optional offset) after a reference time.",
	solarSunset,
)

// SolarNoonFunc returns the next solar noon (+ optional offset) after time t.
var SolarNoonFunc = makeSolarFunc(
	"Returns the next solar noon (+ optional offset) after a reference time.",
	solarNoon,
)

// SolarMidnightFunc returns the next solar midnight (+ optional offset) after time t.
var SolarMidnightFunc = makeSolarFunc(
	"Returns the next solar midnight (+ optional offset) after a reference time.",
	solarMidnight,
)

// eventFunc computes a solar event for a given location and local date.
// Returns the event time and a bool indicating success. A false return means
// the event does not occur on that date (polar regions).
type eventFunc func(lat, lon float64, d time.Time) (time.Time, bool)

func solarSunrise(lat, lon float64, d time.Time) (time.Time, bool) {
	rise, _ := sunrise.SunriseSunset(lat, lon, d.Year(), d.Month(), d.Day())
	return rise, !rise.IsZero()
}

func solarSunset(lat, lon float64, d time.Time) (time.Time, bool) {
	_, set := sunrise.SunriseSunset(lat, lon, d.Year(), d.Month(), d.Day())
	return set, !set.IsZero()
}

func solarNoon(lat, lon float64, d time.Time) (time.Time, bool) {
	rise, set := sunrise.SunriseSunset(lat, lon, d.Year(), d.Month(), d.Day())
	if rise.IsZero() || set.IsZero() {
		return time.Time{}, false
	}
	mid := rise.Add(set.Sub(rise) / 2)
	return mid, true
}

func solarMidnight(lat, lon float64, d time.Time) (time.Time, bool) {
	// Midpoint between today's sunset and tomorrow's sunrise.
	_, set := sunrise.SunriseSunset(lat, lon, d.Year(), d.Month(), d.Day())
	if set.IsZero() {
		return time.Time{}, false
	}
	nextDay := d.AddDate(0, 0, 1)
	nextRise, _ := sunrise.SunriseSunset(lat, lon, nextDay.Year(), nextDay.Month(), nextDay.Day())
	if nextRise.IsZero() {
		return time.Time{}, false
	}
	mid := set.Add(nextRise.Sub(set) / 2)
	return mid, true
}

// nextOccurrence implements the "next occurrence" algorithm.
// It finds the next time after t where (event + offset) is in the future,
// advancing day-by-day through polar regions up to maxPolarSearch days.
// All computation is done in UTC. The search starts one day before the
// current UTC date to catch events like solar midnight that span two
// calendar days.
func nextOccurrence(ef eventFunc, lat, lon float64, offset time.Duration, t time.Time) (time.Time, error) {
	utc := t.UTC()
	d := time.Date(utc.Year(), utc.Month(), utc.Day()-1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < maxPolarSearch; i++ {
		ev, ok := ef(lat, lon, d)
		if ok {
			target := ev.Add(offset)
			if target.After(t) {
				return target, nil
			}
		}
		d = d.AddDate(0, 0, 1)
	}

	return time.Time{}, fmt.Errorf("no solar event found within %d days (polar region?)", maxPolarSearch)
}

// makeSolarFunc builds a cty function for a solar event. All four solar
// functions share the same signature:
//
//	f(point)
//	f(point, offset)       — offset is a duration capsule
//	f(point, t)            — t is a time capsule
//	f(point, offset, t)
func makeSolarFunc(desc string, ef eventFunc) function.Function {
	return function.New(&function.Spec{
		Description: desc,
		Params: []function.Parameter{
			{Name: "point", Type: cty.DynamicPseudoType},
		},
		VarParam: &function.Parameter{
			Name: "args",
			Type: cty.DynamicPseudoType,
		},
		Type: function.StaticReturnType(timecty.TimeCapsuleType),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			if len(args) > 3 {
				return cty.NilVal, fmt.Errorf("expected 1-3 arguments, got %d", len(args))
			}

			lat, lon, err := extractLatLon(args[0])
			if err != nil {
				return cty.NilVal, err
			}

			var offset time.Duration
			t := time.Now()

			for _, arg := range args[1:] {
				switch arg.Type() {
				case timecty.DurationCapsuleType:
					d, err := timecty.GetDuration(arg)
					if err != nil {
						return cty.NilVal, err
					}
					offset = d
				case timecty.TimeCapsuleType:
					tv, err := timecty.GetTime(arg)
					if err != nil {
						return cty.NilVal, err
					}
					t = tv
				default:
					return cty.NilVal, fmt.Errorf(
						"arguments after point must be a duration offset or a time, got %s",
						arg.Type().FriendlyName(),
					)
				}
			}

			result, err := nextOccurrence(ef, lat, lon, offset, t)
			if err != nil {
				return cty.NilVal, err
			}
			return timecty.NewTimeCapsule(result), nil
		},
	})
}

// extractLatLon reads lat and lon number attributes from a cty object.
func extractLatLon(pt cty.Value) (float64, float64, error) {
	if !pt.Type().IsObjectType() {
		return 0, 0, fmt.Errorf("point must be an object, got %s", pt.Type().FriendlyName())
	}
	attrs := pt.AsValueMap()
	lat, err := pointCoord(attrs, "lat", axisLat)
	if err != nil {
		return 0, 0, err
	}
	lon, err := pointCoord(attrs, "lon", axisLon)
	if err != nil {
		return 0, 0, err
	}
	return lat, lon, nil
}
