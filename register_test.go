package geocty

import "testing"

func TestGetGeoFunctionsReturnsMap(t *testing.T) {
	funcs := GetGeoFunctions()
	if funcs == nil {
		t.Fatal("GetGeoFunctions returned nil")
	}
}
