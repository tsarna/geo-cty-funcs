package geocty

import (
	"math"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

// A ~1° × 1° square near the equator. At the equator, 1° lat ≈ 111.32 km,
// 1° lon ≈ 111.32 km, so area ≈ 111.32² ≈ 12,392 km².
var equatorSquare = cty.ListVal([]cty.Value{
	sfPoint(0, 0),
	sfPoint(0, 1),
	sfPoint(1, 1),
	sfPoint(1, 0),
})

// SF neighborhood polygon (small area).
var sfNeighborhood = cty.ListVal([]cty.Value{
	sfPoint(37.780, -122.420),
	sfPoint(37.780, -122.410),
	sfPoint(37.770, -122.410),
	sfPoint(37.770, -122.420),
})

func TestGeoArea_EquatorSquare(t *testing.T) {
	result, err := GeoAreaFunc.Call([]cty.Value{equatorSquare})
	if err != nil {
		t.Fatalf("geo_area: %v", err)
	}
	var area float64
	area = mustFloat(t, result)
	// Expected ≈ 12,308 km² = 1.2308e10 m². Allow ±5%.
	expected := 1.2308e10
	if math.Abs(area-expected)/expected > 0.05 {
		t.Errorf("area = %.3e m², expected ~%.3e (±5%%)", area, expected)
	}
}

func TestGeoArea_SFNeighborhood(t *testing.T) {
	result, err := GeoAreaFunc.Call([]cty.Value{sfNeighborhood})
	if err != nil {
		t.Fatalf("geo_area: %v", err)
	}
	area := mustFloat(t, result)
	// ~0.01° × ~0.01° at 37.7°N. 0.01° lat ≈ 1.113 km, 0.01° lon ≈ 0.881 km.
	// Area ≈ 1.113 × 0.881 ≈ 0.98 km² = 9.8e5 m².
	if area < 5e5 || area > 2e6 {
		t.Errorf("SF neighborhood area = %.0f m², expected ~1e6", area)
	}
}

func TestGeoArea_TooFewPoints(t *testing.T) {
	twoPoints := cty.ListVal([]cty.Value{
		sfPoint(0, 0),
		sfPoint(1, 1),
	})
	if _, err := GeoAreaFunc.Call([]cty.Value{twoPoints}); err == nil {
		t.Error("expected error for fewer than 3 points")
	}
}

func TestGeoContains_Inside(t *testing.T) {
	// Center of the SF neighborhood polygon.
	center := sfPoint(37.775, -122.415)
	result, err := GeoContainsFunc.Call([]cty.Value{sfNeighborhood, center})
	if err != nil {
		t.Fatalf("geo_contains: %v", err)
	}
	if result.False() {
		t.Error("center should be inside polygon")
	}
}

func TestGeoContains_Outside(t *testing.T) {
	outside := sfPoint(38.0, -122.0)
	result, err := GeoContainsFunc.Call([]cty.Value{sfNeighborhood, outside})
	if err != nil {
		t.Fatalf("geo_contains: %v", err)
	}
	if result.True() {
		t.Error("point should be outside polygon")
	}
}

func TestGeoContains_NearVertex(t *testing.T) {
	// A point just inside a vertex — should be considered inside.
	nearVertex := sfPoint(37.7799, -122.4199)
	result, err := GeoContainsFunc.Call([]cty.Value{sfNeighborhood, nearVertex})
	if err != nil {
		t.Fatalf("geo_contains: %v", err)
	}
	if result.False() {
		t.Error("point just inside vertex should be inside polygon")
	}
}

func TestGeoNearest_PointOutside(t *testing.T) {
	// Point due east of the polygon — nearest should be on the east edge.
	outside := sfPoint(37.775, -122.405)
	result, err := GeoNearestFunc.Call([]cty.Value{sfNeighborhood, outside})
	if err != nil {
		t.Fatalf("geo_nearest: %v", err)
	}
	attrs := result.AsValueMap()
	lat := mustFloat(t, attrs["lat"])
	lon := mustFloat(t, attrs["lon"])

	// Nearest should be at approximately (37.775, -122.410) — the projection
	// onto the east edge (lon = -122.410).
	if math.Abs(lat-37.775) > 0.001 {
		t.Errorf("nearest lat = %v, expected ~37.775", lat)
	}
	if math.Abs(lon-(-122.410)) > 0.001 {
		t.Errorf("nearest lon = %v, expected ~-122.410", lon)
	}
}

func TestGeoNearest_PointInside(t *testing.T) {
	// Point inside — nearest is still on the perimeter.
	inside := sfPoint(37.775, -122.415)
	result, err := GeoNearestFunc.Call([]cty.Value{sfNeighborhood, inside})
	if err != nil {
		t.Fatalf("geo_nearest: %v", err)
	}
	attrs := result.AsValueMap()
	lat := mustFloat(t, attrs["lat"])
	lon := mustFloat(t, attrs["lon"])

	// Center of a 0.01° × 0.01° box — nearest edge should be ~0.005° away
	// in either lat or lon.
	distLat := math.Min(math.Abs(lat-37.770), math.Abs(lat-37.780))
	distLon := math.Min(math.Abs(lon-(-122.410)), math.Abs(lon-(-122.420)))
	minDist := math.Min(distLat, distLon)
	if minDist > 0.001 {
		t.Errorf("nearest point (%v, %v) should be on an edge of the polygon", lat, lon)
	}
}

func TestGeoLineIntersect_CrossingLines(t *testing.T) {
	// Two lines forming an X through the equator-square area.
	lineA := cty.ListVal([]cty.Value{
		sfPoint(0, 0),
		sfPoint(1, 1),
	})
	lineB := cty.ListVal([]cty.Value{
		sfPoint(0, 1),
		sfPoint(1, 0),
	})
	result, err := GeoLineIntersectFunc.Call([]cty.Value{lineA, lineB})
	if err != nil {
		t.Fatalf("geo_line_intersect: %v", err)
	}
	points := result.AsValueSlice()
	if len(points) != 1 {
		t.Fatalf("expected 1 intersection, got %d", len(points))
	}
	attrs := points[0].AsValueMap()
	lat := mustFloat(t, attrs["lat"])
	lon := mustFloat(t, attrs["lon"])
	// Intersection should be near (0.5, 0.5).
	if math.Abs(lat-0.5) > 0.01 || math.Abs(lon-0.5) > 0.01 {
		t.Errorf("intersection at (%v, %v), expected ~(0.5, 0.5)", lat, lon)
	}
}

func TestGeoLineIntersect_ParallelLines(t *testing.T) {
	lineA := cty.ListVal([]cty.Value{
		sfPoint(0, 0),
		sfPoint(1, 0),
	})
	lineB := cty.ListVal([]cty.Value{
		sfPoint(0, 1),
		sfPoint(1, 1),
	})
	result, err := GeoLineIntersectFunc.Call([]cty.Value{lineA, lineB})
	if err != nil {
		t.Fatalf("geo_line_intersect: %v", err)
	}
	points := result.AsValueSlice()
	if len(points) != 0 {
		t.Errorf("expected 0 intersections for parallel lines, got %d", len(points))
	}
}

func TestGeoLineIntersect_MultiSegment(t *testing.T) {
	// A zigzag line crossing a straight line twice.
	straight := cty.ListVal([]cty.Value{
		sfPoint(0, 0),
		sfPoint(0, 4),
	})
	zigzag := cty.ListVal([]cty.Value{
		sfPoint(-1, 1),
		sfPoint(1, 2),
		sfPoint(-1, 3),
	})
	result, err := GeoLineIntersectFunc.Call([]cty.Value{straight, zigzag})
	if err != nil {
		t.Fatalf("geo_line_intersect: %v", err)
	}
	points := result.AsValueSlice()
	if len(points) != 2 {
		t.Errorf("expected 2 intersections, got %d", len(points))
	}
}

func TestGeoLineIntersect_TooFewPoints(t *testing.T) {
	onePoint := cty.ListVal([]cty.Value{sfPoint(0, 0)})
	twoPoints := cty.ListVal([]cty.Value{sfPoint(0, 0), sfPoint(1, 1)})
	if _, err := GeoLineIntersectFunc.Call([]cty.Value{onePoint, twoPoints}); err == nil {
		t.Error("expected error for line_a with 1 point")
	}
}
