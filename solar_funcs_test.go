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

	// Solar noon in SF ≈ 20:17 UTC (midpoint of rise ~14:12 and set ~02:21 next day).
	if got.Hour() < 19 || got.Hour() > 21 {
		t.Errorf("solar noon hour UTC = %d, expected ~20 for SF in March", got.Hour())
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

	// Solar midnight in SF ≈ 08:16 UTC (midpoint of sunset ~02:21 and next sunrise ~14:11).
	if got.Hour() < 7 || got.Hour() > 9 {
		t.Errorf("solar midnight hour UTC = %d, expected ~8 for SF in March", got.Hour())
	}
}

func TestSolarMidnight_CrossDayBoundary(t *testing.T) {
	// At 03:38 UTC, the user is in late evening local time (e.g. US East).
	// Solar midnight for "tonight" is the midpoint of yesterday's sunset and
	// today's sunrise — it should be only a few hours away, not 25h.
	nyc := sfPoint(40.7128, -74.0060)
	refTime := time.Date(2026, 4, 17, 3, 38, 0, 0, time.UTC)

	result, err := SolarMidnightFunc.Call([]cty.Value{nyc, timecty.NewTimeCapsule(refTime)})
	if err != nil {
		t.Fatalf("solar_midnight: %v", err)
	}
	got := mustTime(t, result)

	if !got.After(refTime) {
		t.Errorf("solar_midnight should be after ref time")
	}

	// Solar midnight for NYC on the night of Apr 16/17 should be around
	// Apr 17 05:00 UTC — within ~2 hours, NOT 25 hours away.
	hoursAway := got.Sub(refTime).Hours()
	if hoursAway > 4 {
		t.Errorf("solar midnight is %.1f hours away (got %v), expected <4h", hoursAway, got)
	}
}

func TestSunrise_HighLatitude(t *testing.T) {
	// Reykjavík at 64°N on the summer solstice still has sunrise/sunset per
	// go-sunrise (~02:55 rise, ~00:03 set UTC). Verify we get a reasonable result.
	reykjavik := sfPoint(64.1466, -21.9426)
	refTime := time.Date(2025, 6, 21, 12, 0, 0, 0, time.UTC)

	result, err := SunriseFunc.Call([]cty.Value{reykjavik, timecty.NewTimeCapsule(refTime)})
	if err != nil {
		t.Fatalf("sunrise at Reykjavík: %v", err)
	}
	got := mustTime(t, result)
	if !got.After(refTime) {
		t.Errorf("result %v should be after ref %v", got, refTime)
	}
	// Next sunrise should be June 22 ~02:55 UTC.
	if got.Day() != 22 || got.Hour() > 4 {
		t.Errorf("expected June 22 ~02:55 UTC, got %v", got)
	}
}

func TestSunset_PolarDay(t *testing.T) {
	// Svalbard at 78°N in June — true polar day, no sunset until September.
	svalbard := sfPoint(78.2, 15.6)
	refTime := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)

	result, err := SunsetFunc.Call([]cty.Value{svalbard, timecty.NewTimeCapsule(refTime)})
	if err != nil {
		t.Fatalf("sunset at Svalbard: %v", err)
	}
	got := mustTime(t, result)
	if !got.After(refTime) {
		t.Errorf("result %v should be after ref %v", got, refTime)
	}
	// First sunset at 78°N after June 21 is late August (around Aug 25).
	if got.Month() != time.August && got.Month() != time.September {
		t.Errorf("first sunset at Svalbard after June 21 should be Aug/Sep, got %v", got)
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
