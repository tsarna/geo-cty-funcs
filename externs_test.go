package geocty

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noExtern are functions deliberately left out of externs.cty. It is empty: every geo
// function is declared. The fixed-arity, fixed-return ones would ordinarily rely on their
// cty metadata, but their point arguments are `dynamic` (objects with arbitrary extras),
// and a dynamic argument poisons cty's return type to `dynamic`, hiding the return shape
// entirely — so even those need a declaration to be legible.
var noExtern = map[string]bool{}

// externDeclRE matches a top-level declaration in externs.cty. The file is parsed here
// with a regex rather than with functy on purpose: this package must not depend on
// functy (its bytes are opaque to it), and the check only needs the name set. A name may
// appear on several lines — these functions are overload sets — so the result is a set.
var externDeclRE = regexp.MustCompile(`(?m)^func (\w+)\(`)

// TestExternsCoverTheRightFunctions is the drift guard, in both directions: every
// function cty cannot describe must be declared, and every function cty *can* describe
// must not be (declaring it would shadow correct, self-maintaining cty metadata with a
// hand-written copy).
func TestExternsCoverTheRightFunctions(t *testing.T) {
	declared := make(map[string]bool)
	for _, m := range externDeclRE.FindAllStringSubmatch(string(Externs()), -1) {
		declared[m[1]] = true
	}

	for name := range GetGeoFunctions() {
		if noExtern[name] {
			assert.False(t, declared[name],
				"%s() is listed in noExtern but externs.cty declares it: pick one", name)
			continue
		}
		assert.True(t, declared[name],
			"%s() is provided by GetGeoFunctions but has no declaration in externs.cty", name)
	}

	funcs := GetGeoFunctions()
	for name := range declared {
		assert.Contains(t, funcs, name,
			"externs.cty declares %s(), which GetGeoFunctions does not provide", name)
	}
}

// The bytes must declare themselves an extern file: functy's RegisterExterns verifies the
// directive rather than forcing the mode, so that this same file is a valid standalone
// .cty that `functy fmt` and `functy symbols` can open.
func TestExternsCarryTheDirective(t *testing.T) {
	require.True(t, strings.HasPrefix(string(Externs()), "//functy:extern\n"),
		"externs.cty must begin with the //functy:extern directive")
}

// Every function, and every parameter of every function, must carry a cty description —
// extern or not. The cty metadata is the only documentation a non-functy cty host can
// see, and the only thing functy's own doc() reads (doc() does not consult the extern),
// so a gap here reads as "exists but undocumented" even where help() shows a full block.
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
