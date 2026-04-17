package geocty

import (
	"math"
	"testing"
	"time"

	timecty "github.com/tsarna/time-cty-funcs"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// Known pair: SF to NYC.
var (
	sfGeo  = sfPoint(37.7749, -122.4194)
	nycGeo = sfPoint(40.7128, -74.0060)
)

func TestGeoInverse_SFtoNYC(t *testing.T) {
	result, err := GeoInverseFunc.Call([]cty.Value{sfGeo, nycGeo})
	if err != nil {
		t.Fatalf("geo_inverse: %v", err)
	}

	dist := mustObjFloat(t, result, "distance")
	// SF to NYC ≈ 4,139.1 km.
	if math.Abs(dist-4139145.5) > 1000 {
		t.Errorf("distance = %.0f m, expected ~4,139,146", dist)
	}

	bearing := mustObjFloat(t, result, "bearing")
	// Bearing from SF to NYC ≈ 69.92°.
	if math.Abs(bearing-69.92) > 0.5 {
		t.Errorf("bearing = %.2f°, expected ~69.92°", bearing)
	}

	backBearing := mustObjFloat(t, result, "back_bearing")
	// Back-bearing ≈ 281.69°.
	if math.Abs(backBearing-281.69) > 0.5 {
		t.Errorf("back_bearing = %.2f°, expected ~281.69°", backBearing)
	}
}

func TestGeoInverse_SamePoint(t *testing.T) {
	result, err := GeoInverseFunc.Call([]cty.Value{sfGeo, sfGeo})
	if err != nil {
		t.Fatalf("geo_inverse: %v", err)
	}
	dist := mustObjFloat(t, result, "distance")
	if dist > 1e-6 {
		t.Errorf("distance = %v, expected ~0", dist)
	}
}

func TestGeoDestination_DistanceForm(t *testing.T) {
	// 1 km due east from SF.
	result, err := GeoDestinationFunc.Call([]cty.Value{
		sfGeo,
		cty.NumberFloatVal(90.0),
		cty.NumberFloatVal(1000.0),
	})
	if err != nil {
		t.Fatalf("geo_destination: %v", err)
	}

	attrs := result.AsValueMap()
	lat := mustFloat(t, attrs["lat"])
	lon := mustFloat(t, attrs["lon"])

	// Latitude should be nearly the same (very slight change due to geodesic).
	if math.Abs(lat-37.7749) > 0.001 {
		t.Errorf("lat = %v, expected ~37.7749", lat)
	}
	// Longitude should increase (eastward). 1 km at 37.7° ≈ 0.0113°.
	lonDiff := lon - (-122.4194)
	if lonDiff < 0.008 || lonDiff > 0.015 {
		t.Errorf("lon diff = %v, expected ~0.0113", lonDiff)
	}
}

func TestGeoDestination_FieldPreservation(t *testing.T) {
	origin := cty.ObjectVal(map[string]cty.Value{
		"lat":   cty.NumberFloatVal(37.7749),
		"lon":   cty.NumberFloatVal(-122.4194),
		"alt":   cty.NumberFloatVal(11.0),
		"track": cty.NumberFloatVal(90.0),
		"speed": cty.NumberFloatVal(10.0),
	})
	result, err := GeoDestinationFunc.Call([]cty.Value{
		origin,
		cty.NumberFloatVal(90.0),
		cty.NumberFloatVal(1000.0),
	})
	if err != nil {
		t.Fatalf("geo_destination: %v", err)
	}
	attrs := result.AsValueMap()
	// alt, track, speed should be preserved.
	if mustFloat(t, attrs["alt"]) != 11.0 {
		t.Error("alt not preserved")
	}
	if mustFloat(t, attrs["track"]) != 90.0 {
		t.Error("track not preserved")
	}
	if mustFloat(t, attrs["speed"]) != 10.0 {
		t.Error("speed not preserved")
	}
}

func TestGeoDestination_DurationForm(t *testing.T) {
	// 5 minutes at 20 m/s = 6000 m.
	now := time.Date(2025, 3, 20, 12, 0, 0, 0, time.UTC)
	origin := cty.ObjectVal(map[string]cty.Value{
		"lat":   cty.NumberFloatVal(37.7749),
		"lon":   cty.NumberFloatVal(-122.4194),
		"speed": cty.NumberFloatVal(20.0),
		"time":  timecty.NewTimeCapsule(now),
	})
	dur := timecty.NewDurationCapsule(5 * time.Minute)
	result, err := GeoDestinationFunc.Call([]cty.Value{
		origin,
		cty.NumberFloatVal(90.0),
		dur,
	})
	if err != nil {
		t.Fatalf("geo_destination duration: %v", err)
	}
	attrs := result.AsValueMap()

	// time should be advanced by 5 minutes.
	gotTime, err := timecty.GetTime(attrs["time"])
	if err != nil {
		t.Fatalf("extract time: %v", err)
	}
	wantTime := now.Add(5 * time.Minute)
	if !gotTime.Equal(wantTime) {
		t.Errorf("time = %v, want %v", gotTime, wantTime)
	}

	// Distance travelled: 20 m/s × 300 s = 6000 m.
	inv := mustObjFloat(t, mustCallInverse(t, sfGeo, result), "distance")
	if math.Abs(inv-6000) > 10 {
		t.Errorf("distance = %.0f, expected ~6000", inv)
	}
}

func TestGeoDestination_DurationFormNoTime(t *testing.T) {
	// Duration form with speed but no time — should work, result has no time.
	origin := cty.ObjectVal(map[string]cty.Value{
		"lat":   cty.NumberFloatVal(37.7749),
		"lon":   cty.NumberFloatVal(-122.4194),
		"speed": cty.NumberFloatVal(20.0),
	})
	dur := timecty.NewDurationCapsule(5 * time.Minute)
	result, err := GeoDestinationFunc.Call([]cty.Value{
		origin,
		cty.NumberFloatVal(90.0),
		dur,
	})
	if err != nil {
		t.Fatalf("geo_destination: %v", err)
	}
	attrs := result.AsValueMap()
	if _, ok := attrs["time"]; ok {
		t.Error("result should not have time when origin had no time")
	}
}

func TestGeoDestination_TimeForm(t *testing.T) {
	now := time.Date(2025, 3, 20, 12, 0, 0, 0, time.UTC)
	target := now.Add(5 * time.Minute)
	origin := cty.ObjectVal(map[string]cty.Value{
		"lat":   cty.NumberFloatVal(37.7749),
		"lon":   cty.NumberFloatVal(-122.4194),
		"speed": cty.NumberFloatVal(20.0),
		"time":  timecty.NewTimeCapsule(now),
	})
	result, err := GeoDestinationFunc.Call([]cty.Value{
		origin,
		cty.NumberFloatVal(90.0),
		timecty.NewTimeCapsule(target),
	})
	if err != nil {
		t.Fatalf("geo_destination time: %v", err)
	}
	attrs := result.AsValueMap()
	gotTime, err := timecty.GetTime(attrs["time"])
	if err != nil {
		t.Fatalf("extract time: %v", err)
	}
	if !gotTime.Equal(target) {
		t.Errorf("time = %v, want %v", gotTime, target)
	}
}

func TestGeoDestination_TimeFormPastError(t *testing.T) {
	now := time.Date(2025, 3, 20, 12, 0, 0, 0, time.UTC)
	past := now.Add(-5 * time.Minute)
	origin := cty.ObjectVal(map[string]cty.Value{
		"lat":   cty.NumberFloatVal(37.7749),
		"lon":   cty.NumberFloatVal(-122.4194),
		"speed": cty.NumberFloatVal(20.0),
		"time":  timecty.NewTimeCapsule(now),
	})
	if _, err := GeoDestinationFunc.Call([]cty.Value{
		origin,
		cty.NumberFloatVal(90.0),
		timecty.NewTimeCapsule(past),
	}); err == nil {
		t.Error("expected error for target_time in the past")
	}
}

func TestGeoDestination_WithExtras(t *testing.T) {
	bare := cty.ObjectVal(map[string]cty.Value{
		"lat": cty.NumberFloatVal(37.7749),
		"lon": cty.NumberFloatVal(-122.4194),
	})
	extras := cty.ObjectVal(map[string]cty.Value{
		"speed": cty.NumberFloatVal(15.0),
		"time":  timecty.NewTimeCapsule(time.Date(2025, 3, 20, 12, 0, 0, 0, time.UTC)),
	})
	dur := timecty.NewDurationCapsule(10 * time.Minute)
	result, err := GeoDestinationFunc.Call([]cty.Value{
		bare,
		cty.NumberFloatVal(90.0),
		dur,
		extras,
	})
	if err != nil {
		t.Fatalf("geo_destination with extras: %v", err)
	}
	attrs := result.AsValueMap()
	if mustFloat(t, attrs["speed"]) != 15.0 {
		t.Error("speed from extras not preserved")
	}
}

func TestGeoDestination_DurationMissingSpeed(t *testing.T) {
	if _, err := GeoDestinationFunc.Call([]cty.Value{
		sfGeo,
		cty.NumberFloatVal(90.0),
		timecty.NewDurationCapsule(5 * time.Minute),
	}); err == nil {
		t.Error("expected error when speed is missing for duration form")
	}
}

func TestGeoDestination_RoundTrip(t *testing.T) {
	// Go 1000m east, then inverse should give ~1000m.
	result, err := GeoDestinationFunc.Call([]cty.Value{
		sfGeo,
		cty.NumberFloatVal(90.0),
		cty.NumberFloatVal(1000.0),
	})
	if err != nil {
		t.Fatalf("geo_destination: %v", err)
	}
	inv := mustCallInverse(t, sfGeo, result)
	dist := mustObjFloat(t, inv, "distance")
	if math.Abs(dist-1000) > 0.1 {
		t.Errorf("round-trip distance = %v, expected 1000", dist)
	}
}

func TestGeoWaypoints_Basic(t *testing.T) {
	result, err := GeoWaypointsFunc.Call([]cty.Value{
		sfGeo, nycGeo, cty.NumberIntVal(5),
	})
	if err != nil {
		t.Fatalf("geo_waypoints: %v", err)
	}
	if !result.Type().IsListType() {
		t.Fatalf("expected list, got %s", result.Type().FriendlyName())
	}
	points := result.AsValueSlice()
	if len(points) != 5 {
		t.Fatalf("expected 5 waypoints, got %d", len(points))
	}

	// First point should be SF.
	first := points[0].AsValueMap()
	if math.Abs(mustFloat(t, first["lat"])-37.7749) > 0.0001 {
		t.Errorf("first lat = %v, expected 37.7749", mustFloat(t, first["lat"]))
	}
	if math.Abs(mustFloat(t, first["lon"])-(-122.4194)) > 0.0001 {
		t.Errorf("first lon = %v, expected -122.4194", mustFloat(t, first["lon"]))
	}

	// Last point should be NYC.
	last := points[4].AsValueMap()
	if math.Abs(mustFloat(t, last["lat"])-40.7128) > 0.0001 {
		t.Errorf("last lat = %v, expected 40.7128", mustFloat(t, last["lat"]))
	}
	if math.Abs(mustFloat(t, last["lon"])-(-74.0060)) > 0.0001 {
		t.Errorf("last lon = %v, expected -74.0060", mustFloat(t, last["lon"]))
	}

	// All tracks should be in [0, 360).
	for i, p := range points {
		track := mustFloat(t, p.AsValueMap()["track"])
		if track < 0 || track >= 360 {
			t.Errorf("waypoint[%d] track = %v, expected [0, 360)", i, track)
		}
	}
}

func TestGeoWaypoints_MinN(t *testing.T) {
	result, err := GeoWaypointsFunc.Call([]cty.Value{
		sfGeo, nycGeo, cty.NumberIntVal(2),
	})
	if err != nil {
		t.Fatalf("geo_waypoints: %v", err)
	}
	if len(result.AsValueSlice()) != 2 {
		t.Errorf("expected 2 waypoints")
	}
}

func TestGeoWaypoints_NTooSmall(t *testing.T) {
	if _, err := GeoWaypointsFunc.Call([]cty.Value{
		sfGeo, nycGeo, cty.NumberIntVal(1),
	}); err == nil {
		t.Error("expected error for n < 2")
	}
}

func TestGeoWaypoints_EvenSpacing(t *testing.T) {
	result, err := GeoWaypointsFunc.Call([]cty.Value{
		sfGeo, nycGeo, cty.NumberIntVal(5),
	})
	if err != nil {
		t.Fatalf("geo_waypoints: %v", err)
	}
	points := result.AsValueSlice()

	// Check that consecutive segments are roughly equal in distance.
	var distances [4]float64
	for i := 0; i < 4; i++ {
		inv := mustCallInverse(t, points[i], points[i+1])
		distances[i] = mustObjFloat(t, inv, "distance")
	}
	for i := 1; i < 4; i++ {
		ratio := distances[i] / distances[0]
		if math.Abs(ratio-1) > 0.01 {
			t.Errorf("segment %d distance = %.0f, segment 0 = %.0f, ratio = %.4f",
				i, distances[i], distances[0], ratio)
		}
	}
}

func TestGeoInverse_Antimeridian(t *testing.T) {
	// Points on opposite sides of the antimeridian.
	tokyo := sfPoint(35.6762, 139.6503)
	anchorage := sfPoint(61.2181, -149.9003)
	result, err := GeoInverseFunc.Call([]cty.Value{tokyo, anchorage})
	if err != nil {
		t.Fatalf("geo_inverse: %v", err)
	}
	dist := mustObjFloat(t, result, "distance")
	// Tokyo to Anchorage ≈ 5,558 km.
	if math.Abs(dist-5.558e6) > 100e3 {
		t.Errorf("distance = %.0f m, expected ~5,558,000", dist)
	}
}

func mustCallInverse(t *testing.T, a, b cty.Value) cty.Value {
	t.Helper()
	r, err := GeoInverseFunc.Call([]cty.Value{a, b})
	if err != nil {
		t.Fatalf("geo_inverse: %v", err)
	}
	return r
}

func mustObjFloatFromSlice(t *testing.T, v cty.Value, key string) float64 {
	t.Helper()
	attrs := v.AsValueMap()
	var f float64
	if err := gocty.FromCtyValue(attrs[key], &f); err != nil {
		t.Fatalf("extract %s: %v", key, err)
	}
	return f
}
