package geocty

import (
	"math"
	"testing"
	"time"

	timecty "github.com/tsarna/time-cty-funcs"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

func mustObjFloat(t *testing.T, obj cty.Value, key string) float64 {
	t.Helper()
	v := obj.GetAttr(key)
	var f float64
	if err := gocty.FromCtyValue(v, &f); err != nil {
		t.Fatalf("extract %s: %v", key, err)
	}
	return f
}

func TestSunPosition_NoonEquinoxEquator(t *testing.T) {
	// At the equator on the spring equinox at solar noon (~12:00 UTC at lon 0),
	// the sun should be nearly directly overhead (altitude ~90°).
	equator := sfPoint(0, 0)
	noon := time.Date(2025, 3, 20, 12, 0, 0, 0, time.UTC)

	result, err := SunPositionFunc.Call([]cty.Value{equator, timecty.NewTimeCapsule(noon)})
	if err != nil {
		t.Fatalf("sun_position: %v", err)
	}

	alt := mustObjFloat(t, result, "altitude")
	// Should be very close to overhead (~88°).
	if alt < 85 || alt > 91 {
		t.Errorf("altitude at equator noon on equinox = %.1f°, expected ~88°", alt)
	}
}

func TestSunPosition_NightTime(t *testing.T) {
	// SF at midnight UTC in March — sun should be below the horizon.
	midnight := time.Date(2025, 3, 20, 8, 0, 0, 0, time.UTC) // midnight PDT = 08:00 UTC
	result, err := SunPositionFunc.Call([]cty.Value{sfPt, timecty.NewTimeCapsule(midnight)})
	if err != nil {
		t.Fatalf("sun_position: %v", err)
	}
	alt := mustObjFloat(t, result, "altitude")
	if alt > 0 {
		t.Errorf("altitude at SF midnight = %.1f°, expected negative", alt)
	}
}

func TestSunPosition_AzimuthRange(t *testing.T) {
	noon := time.Date(2025, 6, 21, 12, 0, 0, 0, time.UTC)
	result, err := SunPositionFunc.Call([]cty.Value{sfPt, timecty.NewTimeCapsule(noon)})
	if err != nil {
		t.Fatalf("sun_position: %v", err)
	}
	az := mustObjFloat(t, result, "azimuth")
	if az < 0 || az >= 360 {
		t.Errorf("azimuth = %.1f, expected [0, 360)", az)
	}
}

func TestSunPosition_DefaultsToNow(t *testing.T) {
	result, err := SunPositionFunc.Call([]cty.Value{sfPt})
	if err != nil {
		t.Fatalf("sun_position with no time: %v", err)
	}
	// Just verify we get valid numbers.
	alt := mustObjFloat(t, result, "altitude")
	if math.IsNaN(alt) {
		t.Error("altitude is NaN")
	}
}

func TestMoonPosition_Basic(t *testing.T) {
	noon := time.Date(2025, 3, 20, 12, 0, 0, 0, time.UTC)
	result, err := MoonPositionFunc.Call([]cty.Value{sfPt, timecty.NewTimeCapsule(noon)})
	if err != nil {
		t.Fatalf("moon_position: %v", err)
	}

	az := mustObjFloat(t, result, "azimuth")
	if az < 0 || az >= 360 {
		t.Errorf("azimuth = %.1f, expected [0, 360)", az)
	}

	dist := mustObjFloat(t, result, "distance")
	// Moon distance: 356,000–406,000 km → 356e6–406e6 meters.
	if dist < 356e6 || dist > 406e6 {
		t.Errorf("distance = %.0f m, expected 356e6–406e6", dist)
	}
}

func TestMoonPosition_DefaultsToNow(t *testing.T) {
	result, err := MoonPositionFunc.Call([]cty.Value{sfPt})
	if err != nil {
		t.Fatalf("moon_position with no time: %v", err)
	}
	dist := mustObjFloat(t, result, "distance")
	if dist < 356e6 || dist > 406e6 {
		t.Errorf("distance = %.0f m, expected 356e6–406e6", dist)
	}
}

func TestMoonPhase_KnownFullMoon(t *testing.T) {
	// March 14, 2025 was a full moon.
	fullMoon := time.Date(2025, 3, 14, 6, 55, 0, 0, time.UTC)
	result, err := MoonPhaseFunc.Call([]cty.Value{timecty.NewTimeCapsule(fullMoon)})
	if err != nil {
		t.Fatalf("moon_phase: %v", err)
	}

	phase := mustObjFloat(t, result, "phase")
	// Full moon → phase ≈ 0.5.
	if math.Abs(phase-0.5) > 0.05 {
		t.Errorf("phase at known full moon = %.3f, expected ~0.5", phase)
	}

	frac := mustObjFloat(t, result, "fraction")
	// Full moon → fraction ≈ 1.0.
	if frac < 0.95 {
		t.Errorf("fraction at known full moon = %.3f, expected ~1.0", frac)
	}
}

func TestMoonPhase_KnownNewMoon(t *testing.T) {
	// March 29, 2025 was a new moon.
	newMoon := time.Date(2025, 3, 29, 10, 58, 0, 0, time.UTC)
	result, err := MoonPhaseFunc.Call([]cty.Value{timecty.NewTimeCapsule(newMoon)})
	if err != nil {
		t.Fatalf("moon_phase: %v", err)
	}

	phase := mustObjFloat(t, result, "phase")
	// New moon → phase ≈ 0.0 or ≈ 1.0 (wraps around).
	if phase > 0.05 && phase < 0.95 {
		t.Errorf("phase at known new moon = %.3f, expected ~0.0 or ~1.0", phase)
	}

	frac := mustObjFloat(t, result, "fraction")
	// New moon → fraction ≈ 0.0.
	if frac > 0.05 {
		t.Errorf("fraction at known new moon = %.3f, expected ~0.0", frac)
	}
}

func TestMoonPhase_DefaultsToNow(t *testing.T) {
	result, err := MoonPhaseFunc.Call([]cty.Value{})
	if err != nil {
		t.Fatalf("moon_phase with no args: %v", err)
	}
	frac := mustObjFloat(t, result, "fraction")
	if frac < 0 || frac > 1 {
		t.Errorf("fraction = %.3f, expected [0, 1]", frac)
	}
	phase := mustObjFloat(t, result, "phase")
	if phase < 0 || phase > 1 {
		t.Errorf("phase = %.3f, expected [0, 1]", phase)
	}
}

func TestMoonPhase_AngleInDegrees(t *testing.T) {
	noon := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	result, err := MoonPhaseFunc.Call([]cty.Value{timecty.NewTimeCapsule(noon)})
	if err != nil {
		t.Fatalf("moon_phase: %v", err)
	}
	angle := mustObjFloat(t, result, "angle")
	// Angle should be in degrees, roughly [-180, 180].
	if angle < -180 || angle > 180 {
		t.Errorf("angle = %.1f°, expected [-180, 180]", angle)
	}
}

func TestSouthAzToNorthDeg(t *testing.T) {
	cases := []struct {
		inRad float64
		want  float64
	}{
		{0, 180},             // south in suncalc → 180° from north
		{math.Pi / 2, 270},   // west in suncalc → 270° from north
		{math.Pi, 0},         // north in suncalc → 0° (or 360°)
		{-math.Pi / 2, 90},   // east in suncalc → 90° from north
	}
	for _, c := range cases {
		got := southAzToNorthDeg(c.inRad)
		// Allow the 0/360 wrap.
		diff := math.Abs(got - c.want)
		if diff > 0.01 && math.Abs(diff-360) > 0.01 {
			t.Errorf("southAzToNorthDeg(%.4f) = %.1f, want %.1f", c.inRad, got, c.want)
		}
	}
}
