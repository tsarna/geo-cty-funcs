package geocty

import (
	"math"
	"testing"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

func mustCallPoint(t *testing.T, args ...cty.Value) cty.Value {
	t.Helper()
	v, err := GeoPointFunc.Call(args)
	if err != nil {
		t.Fatalf("geo_point(%v): %v", args, err)
	}
	return v
}

func mustFloat(t *testing.T, v cty.Value) float64 {
	t.Helper()
	var f float64
	if err := gocty.FromCtyValue(v, &f); err != nil {
		t.Fatalf("extract float: %v", err)
	}
	return f
}

func TestGeoPoint_NumbersOnly(t *testing.T) {
	p := mustCallPoint(t, cty.NumberFloatVal(37.7749), cty.NumberFloatVal(-122.4194))
	if !p.Type().IsObjectType() {
		t.Fatalf("expected object, got %s", p.Type().FriendlyName())
	}
	attrs := p.AsValueMap()
	if mustFloat(t, attrs["lat"]) != 37.7749 {
		t.Errorf("lat = %v, want 37.7749", mustFloat(t, attrs["lat"]))
	}
	if mustFloat(t, attrs["lon"]) != -122.4194 {
		t.Errorf("lon = %v, want -122.4194", mustFloat(t, attrs["lon"]))
	}
	if len(attrs) != 2 {
		t.Errorf("expected only lat and lon, got keys %v", attrs)
	}
}

func TestGeoPoint_StringParsing(t *testing.T) {
	p := mustCallPoint(t, cty.StringVal(`37°46'29"N`), cty.StringVal(`122°25'9"W`))
	attrs := p.AsValueMap()
	wantLat := 37 + 46.0/60 + 29.0/3600
	wantLon := -(122 + 25.0/60 + 9.0/3600)
	if math.Abs(mustFloat(t, attrs["lat"])-wantLat) > 1e-9 {
		t.Errorf("lat = %v, want %v", mustFloat(t, attrs["lat"]), wantLat)
	}
	if math.Abs(mustFloat(t, attrs["lon"])-wantLon) > 1e-9 {
		t.Errorf("lon = %v, want %v", mustFloat(t, attrs["lon"]), wantLon)
	}
}

func TestGeoPoint_MixedNumericAndString(t *testing.T) {
	p := mustCallPoint(t, cty.NumberFloatVal(37.7749), cty.StringVal(`122°25'9.8"W`))
	attrs := p.AsValueMap()
	if mustFloat(t, attrs["lat"]) != 37.7749 {
		t.Errorf("lat = %v", mustFloat(t, attrs["lat"]))
	}
	wantLon := -(122 + 25.0/60 + 9.8/3600)
	if math.Abs(mustFloat(t, attrs["lon"])-wantLon) > 1e-9 {
		t.Errorf("lon = %v, want %v", mustFloat(t, attrs["lon"]), wantLon)
	}
}

func TestGeoPoint_WithBase(t *testing.T) {
	base := cty.ObjectVal(map[string]cty.Value{
		"alt":   cty.NumberFloatVal(11.0),
		"track": cty.NumberFloatVal(90.0),
		"speed": cty.NumberFloatVal(0.0),
	})
	p := mustCallPoint(t, cty.NumberFloatVal(37.7749), cty.NumberFloatVal(-122.4194), base)
	attrs := p.AsValueMap()
	if mustFloat(t, attrs["lat"]) != 37.7749 {
		t.Errorf("lat wrong")
	}
	if mustFloat(t, attrs["alt"]) != 11.0 {
		t.Errorf("alt not preserved")
	}
	if mustFloat(t, attrs["track"]) != 90.0 {
		t.Errorf("track not preserved")
	}
	if mustFloat(t, attrs["speed"]) != 0.0 {
		t.Errorf("speed not preserved")
	}
}

func TestGeoPoint_BaseLatLonOverwritten(t *testing.T) {
	base := cty.ObjectVal(map[string]cty.Value{
		"lat": cty.NumberFloatVal(99.0),
		"lon": cty.NumberFloatVal(99.0),
		"alt": cty.NumberFloatVal(5.0),
	})
	p := mustCallPoint(t, cty.NumberFloatVal(10.0), cty.NumberFloatVal(20.0), base)
	attrs := p.AsValueMap()
	if mustFloat(t, attrs["lat"]) != 10.0 {
		t.Errorf("lat should be overwritten to 10, got %v", mustFloat(t, attrs["lat"]))
	}
	if mustFloat(t, attrs["lon"]) != 20.0 {
		t.Errorf("lon should be overwritten to 20, got %v", mustFloat(t, attrs["lon"]))
	}
	if mustFloat(t, attrs["alt"]) != 5.0 {
		t.Errorf("alt should be preserved")
	}
}

func TestGeoPoint_OutOfRange(t *testing.T) {
	if _, err := GeoPointFunc.Call([]cty.Value{cty.NumberFloatVal(91), cty.NumberFloatVal(0)}); err == nil {
		t.Error("expected error for lat 91")
	}
	if _, err := GeoPointFunc.Call([]cty.Value{cty.NumberFloatVal(0), cty.NumberFloatVal(181)}); err == nil {
		t.Error("expected error for lon 181")
	}
}

func mustFormat(t *testing.T, p cty.Value, args ...cty.Value) string {
	t.Helper()
	call := append([]cty.Value{p}, args...)
	v, err := GeoFormatFunc.Call(call)
	if err != nil {
		t.Fatalf("geo_format: %v", err)
	}
	return v.AsString()
}

func TestGeoFormat_Decimal(t *testing.T) {
	p := cty.ObjectVal(map[string]cty.Value{
		"lat": cty.NumberFloatVal(37.7749),
		"lon": cty.NumberFloatVal(-122.4194),
	})
	if got := mustFormat(t, p); got != "37.7749,-122.4194" {
		t.Errorf("default format = %q", got)
	}
	if got := mustFormat(t, p, cty.StringVal("decimal")); got != "37.7749,-122.4194" {
		t.Errorf("decimal = %q", got)
	}
}

func TestGeoFormat_DecimalAlt(t *testing.T) {
	p := cty.ObjectVal(map[string]cty.Value{
		"lat": cty.NumberFloatVal(37.7749),
		"lon": cty.NumberFloatVal(-122.4194),
		"alt": cty.NumberFloatVal(11.0),
	})
	if got := mustFormat(t, p, cty.StringVal("decimal_alt")); got != "37.7749,-122.4194,11" {
		t.Errorf("decimal_alt = %q", got)
	}

	// No alt: falls back to lat,lon
	pNoAlt := cty.ObjectVal(map[string]cty.Value{
		"lat": cty.NumberFloatVal(37.7749),
		"lon": cty.NumberFloatVal(-122.4194),
	})
	if got := mustFormat(t, pNoAlt, cty.StringVal("decimal_alt")); got != "37.7749,-122.4194" {
		t.Errorf("decimal_alt without alt = %q", got)
	}
}

func TestGeoFormat_DMS(t *testing.T) {
	// Pick inputs that round cleanly at 0.1" resolution.
	p := cty.ObjectVal(map[string]cty.Value{
		"lat": cty.NumberFloatVal(37 + 46.0/60 + 29.6/3600),
		"lon": cty.NumberFloatVal(-(122 + 25.0/60 + 9.8/3600)),
	})
	if got := mustFormat(t, p, cty.StringVal("dms")); got != `37°46'29.6"N 122°25'9.8"W` {
		t.Errorf("dms = %q", got)
	}
	if got := mustFormat(t, p, cty.StringVal("dms_ascii")); got != `37d46'29.6"N 122d25'9.8"W` {
		t.Errorf("dms_ascii = %q", got)
	}
	if got := mustFormat(t, p, cty.StringVal("dms_signed")); got != `37°46'29.6" -122°25'9.8"` {
		t.Errorf("dms_signed = %q", got)
	}
}

func TestGeoFormat_RoundTrip(t *testing.T) {
	orig := mustCallPoint(t, cty.StringVal(`37°46'29"N`), cty.StringVal(`122°25'9"W`))
	formatted := mustFormat(t, orig, cty.StringVal("dms"))
	// Re-parse and compare
	parts := splitDMSPair(formatted)
	back := mustCallPoint(t, cty.StringVal(parts[0]), cty.StringVal(parts[1]))
	origAttrs := orig.AsValueMap()
	backAttrs := back.AsValueMap()
	if math.Abs(mustFloat(t, origAttrs["lat"])-mustFloat(t, backAttrs["lat"])) > 0.001 {
		t.Errorf("lat drifted after round trip: %v vs %v",
			mustFloat(t, origAttrs["lat"]), mustFloat(t, backAttrs["lat"]))
	}
	if math.Abs(mustFloat(t, origAttrs["lon"])-mustFloat(t, backAttrs["lon"])) > 0.001 {
		t.Errorf("lon drifted after round trip")
	}
}

// Splits `"37°46'29.6\"N 122°25'9.8\"W"` into its two halves. DMS format always
// has exactly one space between lat and lon.
func splitDMSPair(s string) [2]string {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			return [2]string{s[:i], s[i+1:]}
		}
	}
	return [2]string{s, ""}
}

func TestGeoFormat_UnknownFormat(t *testing.T) {
	p := cty.ObjectVal(map[string]cty.Value{
		"lat": cty.NumberFloatVal(0),
		"lon": cty.NumberFloatVal(0),
	})
	if _, err := GeoFormatFunc.Call([]cty.Value{p, cty.StringVal("bogus")}); err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestGeoFormat_MissingLat(t *testing.T) {
	p := cty.ObjectVal(map[string]cty.Value{
		"lon": cty.NumberFloatVal(0),
	})
	if _, err := GeoFormatFunc.Call([]cty.Value{p}); err == nil {
		t.Error("expected error for missing lat")
	}
}
