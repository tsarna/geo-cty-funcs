package geocty

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"
)

// GeoPointFunc constructs a point object from lat/lon values, optionally
// merging onto a base object. Numbers are treated as signed decimal degrees;
// strings are parsed as decimal, hemisphere-annotated, or DMS.
var GeoPointFunc = function.New(&function.Spec{
	Description: "Constructs a point object from lat/lon, optionally merged onto a base object.",
	Params: []function.Parameter{
		{Name: "lat", Type: cty.DynamicPseudoType},
		{Name: "lon", Type: cty.DynamicPseudoType},
	},
	VarParam: &function.Parameter{
		Name: "base",
		Type: cty.DynamicPseudoType,
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		if len(args) > 3 {
			return cty.NilType, fmt.Errorf("geo_point takes at most 3 arguments")
		}
		attrs := map[string]cty.Type{
			"lat": cty.Number,
			"lon": cty.Number,
		}
		if len(args) == 3 {
			baseType := args[2].Type()
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
		latVal, err := coerceCoord(args[0], axisLat)
		if err != nil {
			return cty.NilVal, fmt.Errorf("lat: %w", err)
		}
		lonVal, err := coerceCoord(args[1], axisLon)
		if err != nil {
			return cty.NilVal, fmt.Errorf("lon: %w", err)
		}

		result := map[string]cty.Value{
			"lat": latVal,
			"lon": lonVal,
		}
		if len(args) == 3 {
			for k, v := range args[2].AsValueMap() {
				if k == "lat" || k == "lon" {
					continue
				}
				result[k] = v
			}
		}
		return cty.ObjectVal(result), nil
	},
})

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
