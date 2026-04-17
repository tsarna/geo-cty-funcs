package geocty

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type axis int

const (
	axisLat axis = iota
	axisLon
)

var (
	// Signed decimal: "37.7749", "-37.7749", "+37.7749", "37".
	reDec = regexp.MustCompile(`^[+-]?[0-9]+(?:\.[0-9]+)?$`)

	// Decimal with trailing hemisphere: "37.7749N", "37.7749 N".
	reDecHemiSuffix = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)\s*([NSEWnsew])$`)

	// Decimal with leading hemisphere: "N 37.7749", "N37.7749".
	reDecHemiPrefix = regexp.MustCompile(`^([NSEWnsew])\s*([0-9]+(?:\.[0-9]+)?)$`)

	// DMS using ° or d/D. Optional leading sign. Optional minutes, seconds, and
	// trailing hemisphere letter. Submatch groups:
	//   1 sign, 2 deg, 3 min, 4 sec, 5 hemi
	reDMSSym = regexp.MustCompile(
		`^([+-])?([0-9]+)\s*[°dD]\s*` +
			`(?:([0-9]+(?:\.[0-9]+)?)\s*'?\s*` +
			`(?:([0-9]+(?:\.[0-9]+)?)\s*"?)?)?` +
			`\s*([NSEWnsew])?$`)
)

// parseCoord parses a coordinate string. ax selects whether the
// string represents a latitude or longitude; hemisphere letters are checked
// against the axis.
func parseCoord(raw string, ax axis) (float64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("empty coordinate string")
	}

	if reDec.MatchString(s) {
		return strconv.ParseFloat(s, 64)
	}
	if m := reDecHemiSuffix.FindStringSubmatch(s); m != nil {
		return parseDecimalHemisphere(m[1], m[2], ax)
	}
	if m := reDecHemiPrefix.FindStringSubmatch(s); m != nil {
		return parseDecimalHemisphere(m[2], m[1], ax)
	}
	if m := reDMSSym.FindStringSubmatch(s); m != nil {
		return parseDMS(m[1], m[2], m[3], m[4], m[5], ax)
	}

	fields := strings.Fields(s)
	switch len(fields) {
	case 3:
		// [deg min sec] — sign may be baked into deg.
		sign, deg := splitSign(fields[0])
		return parseDMS(sign, deg, fields[1], fields[2], "", ax)
	case 4:
		// [deg min sec hemi]
		return parseDMS("", fields[0], fields[1], fields[2], fields[3], ax)
	}

	return 0, fmt.Errorf("could not parse coordinate %q", raw)
}

func parseDecimalHemisphere(num, hemi string, ax axis) (float64, error) {
	v, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q: %w", num, err)
	}
	sign, err := hemisphereSign(hemi, ax)
	if err != nil {
		return 0, err
	}
	return sign * v, nil
}

func parseDMS(signStr, degStr, minStr, secStr, hemi string, ax axis) (float64, error) {
	deg, err := strconv.ParseFloat(degStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid degrees %q: %w", degStr, err)
	}

	min := 0.0
	if minStr != "" {
		min, err = strconv.ParseFloat(minStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid minutes %q: %w", minStr, err)
		}
		if min < 0 || min >= 60 {
			return 0, fmt.Errorf("minutes %v out of range [0, 60)", min)
		}
	}

	sec := 0.0
	if secStr != "" {
		sec, err = strconv.ParseFloat(secStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid seconds %q: %w", secStr, err)
		}
		if sec < 0 || sec >= 60 {
			return 0, fmt.Errorf("seconds %v out of range [0, 60)", sec)
		}
	}

	v := deg + min/60 + sec/3600

	sign := 1.0
	if signStr == "-" {
		sign = -1.0
	}
	if hemi != "" {
		hSign, err := hemisphereSign(hemi, ax)
		if err != nil {
			return 0, err
		}
		if signStr == "-" && hSign != sign {
			return 0, fmt.Errorf("conflicting sign and hemisphere %q", hemi)
		}
		sign = hSign
	}
	return sign * v, nil
}

func splitSign(s string) (sign, rest string) {
	if strings.HasPrefix(s, "+") {
		return "+", s[1:]
	}
	if strings.HasPrefix(s, "-") {
		return "-", s[1:]
	}
	return "", s
}

func hemisphereSign(hemi string, ax axis) (float64, error) {
	h := strings.ToUpper(hemi)
	switch ax {
	case axisLat:
		switch h {
		case "N":
			return 1, nil
		case "S":
			return -1, nil
		}
		return 0, fmt.Errorf("invalid latitude hemisphere %q (expected N or S)", hemi)
	case axisLon:
		switch h {
		case "E":
			return 1, nil
		case "W":
			return -1, nil
		}
		return 0, fmt.Errorf("invalid longitude hemisphere %q (expected E or W)", hemi)
	}
	return 0, fmt.Errorf("unknown axis")
}

// parseCombinedCoord splits a combined "lat,lon" or "lat lon" string into
// two coordinates and parses each. Handles all geo_format output formats:
//
//   - Comma-separated: "37.7749,-122.4194" or "37.7749,-122.4194,11"
//   - DMS with hemisphere: `37°46'29.6"N 122°25'9.8"W`
//   - DMS signed: `37°46'29.6" -122°25'9.8"`
func parseCombinedCoord(s string) (float64, float64, error) {
	s = strings.TrimSpace(s)

	// Comma-separated: split on comma, take first two parts.
	if strings.Contains(s, ",") {
		parts := strings.SplitN(s, ",", 4)
		if len(parts) < 2 {
			return 0, 0, fmt.Errorf("expected lat,lon in %q", s)
		}
		lat, err := parseCoord(parts[0], axisLat)
		if err != nil {
			return 0, 0, fmt.Errorf("lat: %w", err)
		}
		lon, err := parseCoord(parts[1], axisLon)
		if err != nil {
			return 0, 0, fmt.Errorf("lon: %w", err)
		}
		return lat, lon, nil
	}

	// DMS with hemisphere letters: split after N or S followed by whitespace.
	upper := strings.ToUpper(s)
	for i := 0; i < len(upper)-1; i++ {
		if (upper[i] == 'N' || upper[i] == 'S') && upper[i+1] == ' ' {
			latStr := strings.TrimSpace(s[:i+1])
			lonStr := strings.TrimSpace(s[i+2:])
			if lonStr == "" {
				continue
			}
			lat, latErr := parseCoord(latStr, axisLat)
			lon, lonErr := parseCoord(lonStr, axisLon)
			if latErr == nil && lonErr == nil {
				return lat, lon, nil
			}
		}
	}

	// Signed DMS: split after closing " followed by whitespace.
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '"' && s[i+1] == ' ' {
			latStr := strings.TrimSpace(s[:i+1])
			lonStr := strings.TrimSpace(s[i+2:])
			if lonStr == "" {
				continue
			}
			lat, latErr := parseCoord(latStr, axisLat)
			lon, lonErr := parseCoord(lonStr, axisLon)
			if latErr == nil && lonErr == nil {
				return lat, lon, nil
			}
		}
	}

	return 0, 0, fmt.Errorf("could not parse combined coordinate %q", s)
}

func validateCoord(v float64, ax axis) error {
	switch ax {
	case axisLat:
		if v < -90 || v > 90 {
			return fmt.Errorf("latitude %v out of range [-90, 90]", v)
		}
	case axisLon:
		if v < -180 || v > 180 {
			return fmt.Errorf("longitude %v out of range [-180, 180]", v)
		}
	}
	return nil
}
