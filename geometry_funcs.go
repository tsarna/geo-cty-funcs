package geocty

import (
	"fmt"
	"math"

	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"
)

const earthRadiusM = 6371010.0

// GeoAreaFunc returns the area enclosed by a polygon in square meters.
var GeoAreaFunc = function.New(&function.Spec{
	Description: "Returns the area enclosed by a polygon (list of points) in square meters.",
	Params: []function.Parameter{
		{Name: "polygon", Type: cty.DynamicPseudoType},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		loop, err := ctyListToLoop(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		areaSr := loop.Area()
		areaM2 := math.Abs(areaSr) * earthRadiusM * earthRadiusM
		return cty.NumberFloatVal(areaM2), nil
	},
})

// GeoContainsFunc returns true if point is inside polygon.
var GeoContainsFunc = function.New(&function.Spec{
	Description: "Returns true if the point is inside the polygon.",
	Params: []function.Parameter{
		{Name: "polygon", Type: cty.DynamicPseudoType},
		{Name: "point", Type: cty.DynamicPseudoType},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		loop, err := ctyListToLoop(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		pt, err := ctyToS2Point(args[1])
		if err != nil {
			return cty.NilVal, fmt.Errorf("point: %w", err)
		}
		return cty.BoolVal(loop.ContainsPoint(pt)), nil
	},
})

// GeoNearestFunc returns the closest point on a polygon's perimeter to the
// input point. The result is a bare {lat, lon} object.
var GeoNearestFunc = function.New(&function.Spec{
	Description: "Returns the nearest point on the polygon perimeter to the given point.",
	Params: []function.Parameter{
		{Name: "polygon", Type: cty.DynamicPseudoType},
		{Name: "point", Type: cty.DynamicPseudoType},
	},
	Type: function.StaticReturnType(cty.Object(map[string]cty.Type{
		"lat": cty.Number,
		"lon": cty.Number,
	})),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		loop, err := ctyListToLoop(args[0])
		if err != nil {
			return cty.NilVal, err
		}
		pt, err := ctyToS2Point(args[1])
		if err != nil {
			return cty.NilVal, fmt.Errorf("point: %w", err)
		}

		minDist := s1.InfAngle()
		var closest s2.Point
		for i := 0; i < loop.NumEdges(); i++ {
			edge := loop.Edge(i)
			projected := s2.Project(pt, edge.V0, edge.V1)
			dist := pt.Distance(projected)
			if dist < minDist {
				minDist = dist
				closest = projected
			}
		}
		return s2PointToCty(closest), nil
	},
})

var lineIntersectResultType = cty.List(cty.Object(map[string]cty.Type{
	"lat": cty.Number,
	"lon": cty.Number,
}))

// GeoLineIntersectFunc returns all intersection points between two polylines.
var GeoLineIntersectFunc = function.New(&function.Spec{
	Description: "Returns all intersection points between two polylines.",
	Params: []function.Parameter{
		{Name: "line_a", Type: cty.DynamicPseudoType},
		{Name: "line_b", Type: cty.DynamicPseudoType},
	},
	Type: function.StaticReturnType(lineIntersectResultType),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		lineA, err := ctyListToPoints(args[0])
		if err != nil {
			return cty.NilVal, fmt.Errorf("line_a: %w", err)
		}
		lineB, err := ctyListToPoints(args[1])
		if err != nil {
			return cty.NilVal, fmt.Errorf("line_b: %w", err)
		}
		if len(lineA) < 2 {
			return cty.NilVal, fmt.Errorf("line_a must have at least 2 points")
		}
		if len(lineB) < 2 {
			return cty.NilVal, fmt.Errorf("line_b must have at least 2 points")
		}

		var intersections []cty.Value
		for i := 0; i < len(lineA)-1; i++ {
			for j := 0; j < len(lineB)-1; j++ {
				a0, a1 := lineA[i], lineA[i+1]
				b0, b1 := lineB[j], lineB[j+1]
				if s2.CrossingSign(a0, a1, b0, b1) == s2.Cross {
					ix := s2.Intersection(a0, a1, b0, b1)
					intersections = append(intersections, s2PointToCty(ix))
				}
			}
		}
		if len(intersections) == 0 {
			return cty.ListValEmpty(lineIntersectResultType.ElementType()), nil
		}
		return cty.ListVal(intersections), nil
	},
})

// ctyListToLoop extracts a list of point objects and builds an S2 loop.
// Requires at least 3 points.
func ctyListToLoop(val cty.Value) (*s2.Loop, error) {
	pts, err := ctyListToPoints(val)
	if err != nil {
		return nil, err
	}
	if len(pts) < 3 {
		return nil, fmt.Errorf("polygon requires at least 3 points, got %d", len(pts))
	}
	loop := s2.LoopFromPoints(pts)
	loop.Normalize()
	return loop, nil
}

// ctyListToPoints extracts a list of cty point objects into S2 points.
func ctyListToPoints(val cty.Value) ([]s2.Point, error) {
	if !val.Type().IsListType() && !val.Type().IsTupleType() {
		return nil, fmt.Errorf("expected a list of points, got %s", val.Type().FriendlyName())
	}
	elems := val.AsValueSlice()
	pts := make([]s2.Point, len(elems))
	for i, elem := range elems {
		p, err := ctyToS2Point(elem)
		if err != nil {
			return nil, fmt.Errorf("point[%d]: %w", i, err)
		}
		pts[i] = p
	}
	return pts, nil
}

// ctyToS2Point extracts lat/lon from a cty object and returns an S2 point.
func ctyToS2Point(val cty.Value) (s2.Point, error) {
	if !val.Type().IsObjectType() {
		return s2.Point{}, fmt.Errorf("expected object with lat/lon, got %s", val.Type().FriendlyName())
	}
	attrs := val.AsValueMap()
	latVal, ok := attrs["lat"]
	if !ok {
		return s2.Point{}, fmt.Errorf("missing lat attribute")
	}
	lonVal, ok := attrs["lon"]
	if !ok {
		return s2.Point{}, fmt.Errorf("missing lon attribute")
	}
	var lat, lon float64
	if err := gocty.FromCtyValue(latVal, &lat); err != nil {
		return s2.Point{}, fmt.Errorf("lat: %w", err)
	}
	if err := gocty.FromCtyValue(lonVal, &lon); err != nil {
		return s2.Point{}, fmt.Errorf("lon: %w", err)
	}
	return s2.PointFromLatLng(s2.LatLngFromDegrees(lat, lon)), nil
}

// s2PointToCty converts an S2 point to a bare {lat, lon} cty object.
func s2PointToCty(p s2.Point) cty.Value {
	ll := s2.LatLngFromPoint(p)
	return cty.ObjectVal(map[string]cty.Value{
		"lat": cty.NumberFloatVal(ll.Lat.Degrees()),
		"lon": cty.NumberFloatVal(ll.Lng.Degrees()),
	})
}
