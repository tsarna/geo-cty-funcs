package geocty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

func TestIsGeoPoint(t *testing.T) {
	num := cty.NumberFloatVal

	t.Run("bare lat/lon", func(t *testing.T) {
		v := cty.ObjectVal(map[string]cty.Value{"lat": num(40.7), "lon": num(-74)})
		assert.NoError(t, IsGeoPoint(v))
	})

	t.Run("extras are allowed", func(t *testing.T) {
		v := cty.ObjectVal(map[string]cty.Value{
			"lat": num(40.7), "lon": num(-74),
			"alt": num(10), "name": cty.StringVal("home"),
		})
		assert.NoError(t, IsGeoPoint(v), "altitude and other extras must pass")
	})

	t.Run("missing lon", func(t *testing.T) {
		v := cty.ObjectVal(map[string]cty.Value{"lat": num(40.7)})
		assert.ErrorContains(t, IsGeoPoint(v), `"lon"`)
	})

	t.Run("string coordinates are not yet a geopoint", func(t *testing.T) {
		v := cty.ObjectVal(map[string]cty.Value{"lat": cty.StringVal("40.7"), "lon": cty.StringVal("-74")})
		assert.ErrorContains(t, IsGeoPoint(v), "must be a number")
	})

	t.Run("not an object", func(t *testing.T) {
		assert.ErrorContains(t, IsGeoPoint(cty.StringVal("40.7,-74")), "must be an object")
	})
}
