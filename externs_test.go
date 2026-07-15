package geocty

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The extern files are scanned here with regexes rather than parsed with functy on
// purpose: this package must not depend on functy (its bytes are opaque to it), and the
// checks only need the names.
var (
	namespaceRE  = regexp.MustCompile(`(?m)^namespace (\w+)\s*$`)
	externDeclRE = regexp.MustCompile(`(?m)^func (\w+)\(`)
)

// declaredExterns returns every declared name, qualified by its file's namespace, and
// how many forms each has.
func declaredExterns(t *testing.T) map[string]int {
	t.Helper()
	declared := make(map[string]int)

	for file, src := range Externs() {
		var ns string
		if m := namespaceRE.FindStringSubmatch(string(src)); m != nil {
			ns = m[1] + "::"
		}
		for _, m := range externDeclRE.FindAllStringSubmatch(string(src), -1) {
			declared[ns+m[1]]++
		}
		require.True(t, strings.HasPrefix(string(src), "//functy:extern\n"),
			"%s must begin with the //functy:extern directive: functy's RegisterExterns "+
				"verifies it rather than forcing the mode, so that the file is also a valid "+
				"standalone .cty that `functy fmt` and `functy symbols` can open", file)
	}
	return declared
}

// wantExterns is every geo function and how many forms it declares. Every function is
// declared: those cty cannot describe at all (a faked-optional or type-dispatched
// argument, a result shape that varies), and those it could describe but for their point
// arguments being `dynamic` — which poisons the cty return type to `dynamic`, hiding the
// return shape entirely. The form count guards against losing an overload.
var wantExterns = map[string]int{
	"geo::inverse":        1,
	"geo::waypoints":      1,
	"geo::area":           1,
	"geo::contains":       1,
	"geo::nearest":        1,
	"geo::line_intersect": 1,
	"geo::point":          4, // string / string+base / lat,lon / lat,lon,base
	"geo::format":         1,
	"geo::destination":    3, // third arg is a distance, a duration, or a target time
	"sky::sun_position":   1,
	"sky::moon_position":  1,
	"sky::moon_phase":     1,
	"sky::sunrise":        1,
	"sky::sunset":         1,
	"sky::solar_noon":     1,
	"sky::solar_midnight": 1,
}

// TestExternsCoverEveryFunction is the drift guard, in both directions. A function with
// no declaration reflects as its (misleading or empty) cty signature; a declaration for a
// function that no longer exists documents something nobody can call.
func TestExternsCoverEveryFunction(t *testing.T) {
	declared := declaredExterns(t)
	funcs := GetGeoFunctions()

	for name, forms := range wantExterns {
		require.Contains(t, funcs, name, "wantExterns names %s(), which is not provided", name)
		assert.Equal(t, forms, declared[name],
			"%s() should have %d form(s) declared, found %d", name, forms, declared[name])
	}
	for name := range declared {
		assert.Contains(t, wantExterns, name, "the externs declare %s(), not in wantExterns", name)
		assert.Contains(t, funcs, name, "the externs declare %s(), which GetGeoFunctions does not provide", name)
	}
	// Every provided function is declared — geo has no honest-cty-signature function to
	// leave out, since every one takes a point.
	for name := range funcs {
		assert.Contains(t, wantExterns, name, "%s() is provided but not declared in the externs", name)
	}
}

// One file per namespace, because a functy source declares at most one — so a name's
// forms cannot be split across files, and neither can a namespace.
func TestOneNamespacePerExternFile(t *testing.T) {
	seen := make(map[string]string)
	for file, src := range Externs() {
		m := namespaceRE.FindAllStringSubmatch(string(src), -1)
		require.Len(t, m, 1, "%s must declare exactly one namespace", file)
		ns := m[0][1]
		if prev, dup := seen[ns]; dup {
			t.Errorf("namespace %q is declared in both %s and %s; functy treats one name "+
				"across two files as a collision, not an overload set", ns, prev, file)
		}
		seen[ns] = file
	}
}

// Every name is namespaced under geo:: or sky::.
func TestNamesAreNamespaced(t *testing.T) {
	for name := range GetGeoFunctions() {
		assert.True(t, strings.HasPrefix(name, "geo::") || strings.HasPrefix(name, "sky::"),
			"%s() is not under the geo:: or sky:: namespace", name)
	}
}

// Every function, and every parameter of every function, must carry a cty description.
// The cty metadata is the only documentation a non-functy cty host can see, and the only
// thing functy's own doc() reads (doc() does not consult the extern).
func TestEverythingIsDescribed(t *testing.T) {
	for name, fn := range GetGeoFunctions() {
		assert.NotEmpty(t, fn.Description(), "%s() has no cty Description", name)

		for _, p := range fn.Params() {
			assert.NotEmpty(t, p.Description, "%s() parameter %q has no Description", name, p.Name)
		}
		if vp := fn.VarParam(); vp != nil {
			assert.NotEmpty(t, vp.Description, "%s() variadic parameter %q has no Description", name, vp.Name)
		}
	}
}
