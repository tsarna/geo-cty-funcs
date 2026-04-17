package geocty

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"
)

// GeoPointFunc constructs a point object. It accepts three forms:
//
//	geo_point(combined_string)          — parse "lat,lon" or "DMS DMS"
//	geo_point(combined_string, base)    — same, merged onto base
//	geo_point(lat, lon)                 — separate lat/lon (number or string)
//	geo_point(lat, lon, base)           — same, merged onto base
//
// Numbers are treated as signed decimal degrees; strings are parsed as
// decimal, hemisphere-annotated, or DMS.
var GeoPointFunc = function.New(&function.Spec{
	Description: "Constructs a point object from lat/lon, optionally merged onto a base object.",
	Params: []function.Parameter{
		{Name: "first", Type: cty.DynamicPseudoType},
	},
	VarParam: &function.Parameter{
		Name: "rest",
		Type: cty.DynamicPseudoType,
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		if len(args) > 3 {
			return cty.NilType, fmt.Errorf("geo_point takes 1-3 arguments, got %d", len(args))
		}
		attrs := map[string]cty.Type{
			"lat": cty.Number,
			"lon": cty.Number,
		}
		base := findBase(args)
		if base != nil {
			baseType := base.Type()
			if !baseType.IsObjectType() {
				return cty.NilType, fmt.Errorf("base must be an object, got %s", baseType.FriendlyName())
			}
			for k, t := range baseType.AttributeTypes() {
				if k == "lat" || k == "lon" {
					continue
				}
				attrs[k] = t
			}
		}
		return cty.Object(attrs), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		var latVal, lonVal cty.Value
		var err error

		switch {
		case len(args) == 1 && args[0].Type() == cty.String:
			// geo_point("lat,lon")
			latVal, lonVal, err = parseCombinedString(args[0].AsString())
		case len(args) == 2 && args[0].Type() == cty.String && args[1].Type().IsObjectType():
			// geo_point("lat,lon", base)
			latVal, lonVal, err = parseCombinedString(args[0].AsString())
		default:
			// geo_point(lat, lon[, base])
			if len(args) < 2 {
				return cty.NilVal, fmt.Errorf("geo_point requires lat and lon, or a combined coordinate string")
			}
			latVal, err = coerceCoord(args[0], axisLat)
			if err != nil {
				return cty.NilVal, fmt.Errorf("lat: %w", err)
			}
			lonVal, err = coerceCoord(args[1], axisLon)
			if err != nil {
				return cty.NilVal, fmt.Errorf("lon: %w", err)
			}
		}
		if err != nil {
			return cty.NilVal, err
		}

		result := map[string]cty.Value{
			"lat": latVal,
			"lon": lonVal,
		}
		if base := findBase(args); base != nil {
			for k, v := range base.AsValueMap() {
				if k == "lat" || k == "lon" {
					continue
				}
				result[k] = v
			}
		}
		return cty.ObjectVal(result), nil
	},
})

// findBase returns the base object argument if present, or nil.
// Layout: 1 arg = no base; 2 args with object last = base; 3 args = last is base.
func findBase(args []cty.Value) *cty.Value {
	switch len(args) {
	case 2:
		if args[0].Type() == cty.String && args[1].Type().IsObjectType() {
			return &args[1]
		}
	case 3:
		return &args[2]
	}
	return nil
}

func parseCombinedString(s string) (cty.Value, cty.Value, error) {
	lat, lon, err := parseCombinedCoord(s)
	if err != nil {
		return cty.NilVal, cty.NilVal, err
	}
	if err := validateCoord(lat, axisLat); err != nil {
		return cty.NilVal, cty.NilVal, err
	}
	if err := validateCoord(lon, axisLon); err != nil {
		return cty.NilVal, cty.NilVal, err
	}
	return cty.NumberFloatVal(lat), cty.NumberFloatVal(lon), nil
}

// GeoFormatFunc formats a point as a string using one of the named formats
// (decimal, decimal_alt, dms, dms_ascii, dms_signed).
var GeoFormatFunc = function.New(&function.Spec{
	Description: "Formats a point as a string.",
	Params: []function.Parameter{
		{Name: "point", Type: cty.DynamicPseudoType},
	},
	VarParam: &function.Parameter{
		Name: "format",
		Type: cty.String,
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		if len(args) > 2 {
			return cty.NilVal, fmt.Errorf("geo_format takes at most 2 arguments")
		}
		pt := args[0]
		if !pt.Type().IsObjectType() {
			return cty.NilVal, fmt.Errorf("point must be an object, got %s", pt.Type().FriendlyName())
		}
		attrs := pt.AsValueMap()
		lat, err := pointCoord(attrs, "lat", axisLat)
		if err != nil {
			return cty.NilVal, err
		}
		lon, err := pointCoord(attrs, "lon", axisLon)
		if err != nil {
			return cty.NilVal, err
		}

		format := "decimal"
		if len(args) == 2 {
			format = args[1].AsString()
		}

		s, err := formatPoint(lat, lon, attrs, format)
		if err != nil {
			return cty.NilVal, err
		}
		return cty.StringVal(s), nil
	},
})

func coerceCoord(val cty.Value, ax axis) (cty.Value, error) {
	switch val.Type() {
	case cty.Number:
		var f float64
		if err := gocty.FromCtyValue(val, &f); err != nil {
			return cty.NilVal, err
		}
		if err := validateCoord(f, ax); err != nil {
			return cty.NilVal, err
		}
		return val, nil
	case cty.String:
		f, err := parseCoord(val.AsString(), ax)
		if err != nil {
			return cty.NilVal, err
		}
		if err := validateCoord(f, ax); err != nil {
			return cty.NilVal, err
		}
		return cty.NumberFloatVal(f), nil
	}
	return cty.NilVal, fmt.Errorf("must be number or string, got %s", val.Type().FriendlyName())
}

func pointCoord(attrs map[string]cty.Value, key string, ax axis) (float64, error) {
	v, ok := attrs[key]
	if !ok {
		return 0, fmt.Errorf("point missing %q attribute", key)
	}
	if v.Type() != cty.Number {
		return 0, fmt.Errorf("%s must be a number, got %s", key, v.Type().FriendlyName())
	}
	var f float64
	if err := gocty.FromCtyValue(v, &f); err != nil {
		return 0, err
	}
	if err := validateCoord(f, ax); err != nil {
		return 0, err
	}
	return f, nil
}
