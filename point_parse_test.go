package geocty

import (
	"math"
	"testing"
)

func nearlyEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}

func TestParseCoord_SignedDecimal(t *testing.T) {
	cases := []struct {
		in   string
		ax   axis
		want float64
	}{
		{"37.7749", axisLat, 37.7749},
		{"-37.7749", axisLat, -37.7749},
		{"+37.7749", axisLat, 37.7749},
		{"0", axisLat, 0},
		{"37", axisLat, 37},
		{"90", axisLat, 90},
		{"-90", axisLat, -90},
		{"-122.4194", axisLon, -122.4194},
		{"180", axisLon, 180},
	}
	for _, c := range cases {
		got, err := parseCoord(c.in, c.ax)
		if err != nil {
			t.Errorf("parseCoord(%q): unexpected error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseCoord(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseCoord_DecimalHemisphereSuffix(t *testing.T) {
	cases := []struct {
		in   string
		ax   axis
		want float64
	}{
		{"37.7749N", axisLat, 37.7749},
		{"37.7749 N", axisLat, 37.7749},
		{"37.7749S", axisLat, -37.7749},
		{"122.4194W", axisLon, -122.4194},
		{"122.4194 E", axisLon, 122.4194},
	}
	for _, c := range cases {
		got, err := parseCoord(c.in, c.ax)
		if err != nil {
			t.Errorf("parseCoord(%q): unexpected error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseCoord(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseCoord_DecimalHemispherePrefix(t *testing.T) {
	cases := []struct {
		in   string
		ax   axis
		want float64
	}{
		{"N 37.7749", axisLat, 37.7749},
		{"N37.7749", axisLat, 37.7749},
		{"S 37.7749", axisLat, -37.7749},
		{"W 122.4194", axisLon, -122.4194},
	}
	for _, c := range cases {
		got, err := parseCoord(c.in, c.ax)
		if err != nil {
			t.Errorf("parseCoord(%q): unexpected error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseCoord(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseCoord_DMSWithDegreeSymbol(t *testing.T) {
	cases := []struct {
		in   string
		ax   axis
		want float64
	}{
		{`37°46'29"N`, axisLat, 37 + 46.0/60 + 29.0/3600},
		{`37°46'29.6"N`, axisLat, 37 + 46.0/60 + 29.6/3600},
		{`37° 46' 29.6" N`, axisLat, 37 + 46.0/60 + 29.6/3600},
		{`122°25'9"W`, axisLon, -(122 + 25.0/60 + 9.0/3600)},
		{`37°46'N`, axisLat, 37 + 46.0/60},
		{`37°N`, axisLat, 37},
		{`-37°46'29.6"`, axisLat, -(37 + 46.0/60 + 29.6/3600)},
	}
	for _, c := range cases {
		got, err := parseCoord(c.in, c.ax)
		if err != nil {
			t.Errorf("parseCoord(%q): unexpected error %v", c.in, err)
			continue
		}
		if !nearlyEqual(got, c.want, 1e-9) {
			t.Errorf("parseCoord(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseCoord_DMSWithDLetter(t *testing.T) {
	got, err := parseCoord(`37d46'29.6"N`, axisLat)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	want := 37 + 46.0/60 + 29.6/3600
	if !nearlyEqual(got, want, 1e-9) {
		t.Errorf("got %v, want %v", got, want)
	}

	got, err = parseCoord(`37D46'29.6"N`, axisLat)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if !nearlyEqual(got, want, 1e-9) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseCoord_WhitespaceDMS(t *testing.T) {
	got, err := parseCoord("37 46 29.6 N", axisLat)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	want := 37 + 46.0/60 + 29.6/3600
	if !nearlyEqual(got, want, 1e-9) {
		t.Errorf("got %v, want %v", got, want)
	}

	// Signed, no hemisphere.
	got, err = parseCoord("-37 46 29.6", axisLat)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if !nearlyEqual(got, -want, 1e-9) {
		t.Errorf("got %v, want %v", got, -want)
	}
}

func TestParseCoord_Errors(t *testing.T) {
	cases := []struct {
		in string
		ax axis
	}{
		{"", axisLat},
		{"not a number", axisLat},
		{"37.7749 E", axisLat},   // E/W invalid for latitude
		{"122.4194 N", axisLon},  // N/S invalid for longitude
		{`37°46'60"N`, axisLat},  // minutes/seconds out of range
		{`37°60'29"N`, axisLat},
		{"37 46", axisLat}, // too few whitespace fields
	}
	for _, c := range cases {
		if _, err := parseCoord(c.in, c.ax); err == nil {
			t.Errorf("parseCoord(%q, %v): expected error, got nil", c.in, c.ax)
		}
	}
}

func TestValidateCoord_RangeErrors(t *testing.T) {
	if err := validateCoord(91, axisLat); err == nil {
		t.Error("expected error for lat 91")
	}
	if err := validateCoord(-91, axisLat); err == nil {
		t.Error("expected error for lat -91")
	}
	if err := validateCoord(181, axisLon); err == nil {
		t.Error("expected error for lon 181")
	}
	if err := validateCoord(-181, axisLon); err == nil {
		t.Error("expected error for lon -181")
	}
	if err := validateCoord(90, axisLat); err != nil {
		t.Errorf("lat 90 unexpectedly rejected: %v", err)
	}
	if err := validateCoord(-180, axisLon); err != nil {
		t.Errorf("lon -180 unexpectedly rejected: %v", err)
	}
}
