package geocty

import (
	"fmt"
	"math"
	"strconv"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

func formatPoint(lat, lon float64, attrs map[string]cty.Value, format string) (string, error) {
	switch format {
	case "decimal":
		return trimFloat(lat) + "," + trimFloat(lon), nil
	case "decimal_alt":
		s := trimFloat(lat) + "," + trimFloat(lon)
		if altVal, ok := attrs["alt"]; ok && altVal.Type() == cty.Number {
			var alt float64
			if err := gocty.FromCtyValue(altVal, &alt); err == nil {
				s += "," + trimFloat(alt)
			}
		}
		return s, nil
	case "dms":
		return formatDMSCoord(lat, axisLat, "°", true, false) + " " +
			formatDMSCoord(lon, axisLon, "°", true, false), nil
	case "dms_ascii":
		return formatDMSCoord(lat, axisLat, "d", true, false) + " " +
			formatDMSCoord(lon, axisLon, "d", true, false), nil
	case "dms_signed":
		return formatDMSCoord(lat, axisLat, "°", false, true) + " " +
			formatDMSCoord(lon, axisLon, "°", false, true), nil
	}
	return "", fmt.Errorf("unknown format %q", format)
}

func trimFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func formatDMSCoord(v float64, ax axis, degSym string, hemisphere, signed bool) string {
	deg, min, sec := toDMS(v)
	var prefix, suffix string
	if hemisphere {
		suffix = hemisphereLetter(v, ax)
	} else if signed && v < 0 {
		prefix = "-"
	}
	return fmt.Sprintf("%s%d%s%d'%s\"%s", prefix, deg, degSym, min, formatSec(sec), suffix)
}

// toDMS splits an absolute-valued signed decimal degree into (deg, min, sec),
// rounding sec to the display precision (0.1") and carrying any rollover back
// into min/deg so callers never produce "60.0"" or "60'".
func toDMS(v float64) (int, int, float64) {
	av := math.Abs(v)
	deg := int(av)
	rm := (av - float64(deg)) * 60
	min := int(rm)
	sec := (rm - float64(min)) * 60

	sec = math.Round(sec*10) / 10
	if sec >= 60 {
		sec = 0
		min++
	}
	if min >= 60 {
		min = 0
		deg++
	}
	return deg, min, sec
}

func formatSec(sec float64) string {
	return strconv.FormatFloat(sec, 'f', 1, 64)
}

func hemisphereLetter(v float64, ax axis) string {
	switch ax {
	case axisLat:
		if v >= 0 {
			return "N"
		}
		return "S"
	case axisLon:
		if v >= 0 {
			return "E"
		}
		return "W"
	}
	return ""
}
