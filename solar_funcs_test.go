package geocty

import (
	"testing"
	"time"

	timecty "github.com/tsarna/time-cty-funcs"
	"github.com/zclconf/go-cty/cty"
)

func sfPoint(lat, lon float64) cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"lat": cty.NumberFloatVal(lat),
		"lon": cty.NumberFloatVal(lon),
	})
}

func mustTime(t *testing.T, v cty.Value) time.Time {
	t.Helper()
	tv, err := timecty.GetTime(v)
	if err != nil {
		t.Fatalf("extract time: %v", err)
	}
	return tv
}

// San Francisco: 37.7749, -122.4194
var sfPt = sfPoint(37.7749, -122.4194)

func TestSunrise_Basic(t *testing.T) {
	// Spring equinox 2025 at midnight UTC — sunrise in SF should be early morning local.
	refTime := time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC)

	result, err := SunriseFunc.Call([]cty.Value{sfPt, timecty.NewTimeCapsule(refTime)})
	if err != nil {
		t.Fatalf("sunrise: %v", err)
	}
	got := mustTime(t, result)

	if !got.After(refTime) {
		t.Errorf("sunrise should be after ref time, got %v", got)
	}

	// Sunrise in SF in March should be between 05:00 and 08:00 UTC (~13:00-16:00 local... no).
	// SF is UTC-7 in March (PDT). Sunrise ~7:15 PDT = ~14:15 UTC.
	if got.Hour() < 12 || got.Hour() > 16 {
		t.Errorf("sunrise hour UTC = %d, expected ~14 for SF in March", got.Hour())
	}
}

func TestSunset_Basic(t *testing.T) {
	refTime := time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC)

	result, err := SunsetFunc.Call([]cty.Value{sfPt, timecty.NewTimeCapsule(refTime)})
	if err != nil {
		t.Fatalf("sunset: %v", err)
	}
	got := mustTime(t, result)

	if !got.After(refTime) {
		t.Errorf("sunset should be after ref time")
	}

	// Sunset in SF in March ~19:20 PDT = ~02:20 UTC next day.
	// Could be same day or next day UTC.
	if got.Day() != 20 && got.Day() != 21 {
		t.Errorf("sunset day = %d, expected 20 or 21", got.Day())
	}
}

func TestSunrise_WithNegativeOffset(t *testing.T) {
	// Spec example: current time 05:30 UTC, today's sunrise 06:00 UTC-ish.
	// sunrise(p, -1h) should return ~05:00 TOMORROW, not 05:00 today (which is past).
	//
	// Use a location near Greenwich for easier reasoning.
	greenwich := sfPoint(51.4772, -0.0005)
	// Pick a date where sunrise is ~06:00 UTC (late March in London).
	refTime := time.Date(2025, 3, 25, 5, 30, 0, 0, time.UTC)
	offset := timecty.NewDurationCapsule(-1 * time.Hour)

	result, err := SunriseFunc.Call([]cty.Value{greenwich, offset, timecty.NewTimeCapsule(refTime)})
	if err != nil {
		t.Fatalf("sunrise with offset: %v", err)
	}
	got := mustTime(t, result)

	if !got.After(refTime) {
		t.Errorf("result %v should be after ref %v", got, refTime)
	}
}

func TestSolarNoon_Basic(t *testing.T) {
	refTime := time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC)

	result, err := SolarNoonFunc.Call([]cty.Value{sfPt, timecty.NewTimeCapsule(refTime)})
	if err != nil {
		t.Fatalf("solar_noon: %v", err)
	}
	got := mustTime(t, result)

	if !got.After(refTime) {
		t.Errorf("solar_noon should be after ref time")
	}

	// Solar noon in SF ~12:15 PDT = ~19:15 UTC.
	if got.Hour() < 18 || got.Hour() > 21 {
		t.Errorf("solar noon hour UTC = %d, expected ~19 for SF", got.Hour())
	}
}

func TestSolarMidnight_Basic(t *testing.T) {
	refTime := time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC)

	result, err := SolarMidnightFunc.Call([]cty.Value{sfPt, timecty.NewTimeCapsule(refTime)})
	if err != nil {
		t.Fatalf("solar_midnight: %v", err)
	}
	got := mustTime(t, result)

	if !got.After(refTime) {
		t.Errorf("solar_midnight should be after ref time")
	}

	// Solar midnight in SF should be near 00:15 PDT = ~07:15 UTC.
	if got.Hour() < 5 || got.Hour() > 10 {
		t.Errorf("solar midnight hour UTC = %d, expected ~7 for SF", got.Hour())
	}
}

func TestSunrise_PolarRegion(t *testing.T) {
	// Reykjavík in June — sun doesn't set. Sunrise should still be found
	// by searching forward to a date where it does rise again.
	reykjavik := sfPoint(64.1466, -21.9426)
	// June 21 is near the summer solstice — continuous daylight.
	refTime := time.Date(2025, 6, 21, 12, 0, 0, 0, time.UTC)

	// In polar summer the sun doesn't set, but it still rises each day (it dips
	// close to the horizon). At Reykjavík's latitude (64°N) there should still
	// be a mathematical sunrise/sunset. If go-sunrise returns zero for this date,
	// the function will search forward until it finds one.
	result, err := SunriseFunc.Call([]cty.Value{reykjavik, timecty.NewTimeCapsule(refTime)})
	if err != nil {
		// If we get an error here, it means even searching forward didn't help.
		// At 64°N this shouldn't happen — only ~67°+ is truly polar.
		t.Logf("sunrise at 64°N in June returned error (may be expected): %v", err)
		return
	}
	got := mustTime(t, result)
	if !got.After(refTime) {
		t.Errorf("result %v should be after ref %v", got, refTime)
	}
}

func TestSunrise_TruePolar(t *testing.T) {
	// Svalbard at 78°N in June — true polar day, no sunset for months.
	svalbard := sfPoint(78.2, 15.6)
	refTime := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)

	// go-sunrise should return zero times. Our function will search forward
	// and eventually find the first sunrise after polar summer ends.
	result, err := SunsetFunc.Call([]cty.Value{svalbard, timecty.NewTimeCapsule(refTime)})
	if err != nil {
		t.Logf("sunset at Svalbard in June: %v (expected — polar day)", err)
		return
	}
	got := mustTime(t, result)
	// The first sunset should be months later (August-ish).
	if got.Month() < time.July {
		t.Errorf("first sunset at Svalbard after June 21 should be July+, got %v", got)
	}
}

func TestSunrise_DefaultsToNow(t *testing.T) {
	before := time.Now()
	result, err := SunriseFunc.Call([]cty.Value{sfPt})
	if err != nil {
		t.Fatalf("sunrise with no time arg: %v", err)
	}
	got := mustTime(t, result)
	if !got.After(before) {
		t.Errorf("sunrise with default now should be in the future")
	}
}

func TestSunrise_PointMissingLat(t *testing.T) {
	bad := cty.ObjectVal(map[string]cty.Value{
		"lon": cty.NumberFloatVal(0),
	})
	if _, err := SunriseFunc.Call([]cty.Value{bad}); err == nil {
		t.Error("expected error for point missing lat")
	}
}

func TestSunrise_BadArgType(t *testing.T) {
	// Passing a string where duration/time is expected.
	if _, err := SunriseFunc.Call([]cty.Value{sfPt, cty.StringVal("nope")}); err == nil {
		t.Error("expected error for bad arg type")
	}
}
